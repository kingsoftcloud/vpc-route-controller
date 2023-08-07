package route

import (
	"context"
	"fmt"
	cmap "github.com/orcaman/concurrent-map"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"time"

	"ezone.ksyun.com/code/kce/vpc-route-controller/pkg/controller/helper"
	openstackCfg "ezone.ksyun.com/code/kce/vpc-route-controller/pkg/ksyun/openstack_client/config"
	"ezone.ksyun.com/code/kce/vpc-route-controller/pkg/model"
	"ezone.ksyun.com/code/kce/vpc-route-controller/pkg/util/metric"
)

const (
	updateNodeStatusMaxRetries       = 3
	defaultRouteReconciliationPeriod = 5 * time.Minute
)

func Add(mgr manager.Manager, cfg *openstackCfg.Config) error {
	return add(mgr, newReconciler(mgr, cfg))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, cfg *openstackCfg.Config) *ReconcileRoute {
	recon := &ReconcileRoute{
		client:          mgr.GetClient(),
		scheme:          mgr.GetScheme(),
		record:          mgr.GetEventRecorderFor("route-controller"),
		nodeCache:       cmap.New(),
		configRoutes:    true,
		reconcilePeriod: defaultRouteReconciliationPeriod,
		neutronConfig:   cfg,
	}
	return recon
}

type routeController struct {
	c     controller.Controller
	recon *ReconcileRoute
}

// Start() function will not be called until the resource lock is acquired
func (controller routeController) Start(ctx context.Context) error {
	if controller.recon.configRoutes {
		controller.recon.periodicalSync()
	}
	return controller.c.Start(ctx)
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r *ReconcileRoute) error {
	// Create a new controller
	recoverPanic := true
	c, err := controller.NewUnmanaged(
		"route-controller", mgr,
		controller.Options{
			Reconciler:              r,
			MaxConcurrentReconciles: 1,
			RecoverPanic:            &recoverPanic,
		},
	)
	if err != nil {
		return err
	}

	// Watch for changes to primary resource AutoRepair
	err = c.Watch(
		&source.Kind{
			Type: &corev1.Node{},
		},
		&handler.EnqueueRequestForObject{},
		&predicateForNodeEvent{},
	)
	if err != nil {
		return err
	}

	return mgr.Add(&routeController{c: c, recon: r})
}

// ReconcileRoute implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileRoute{}

// ReconcileRoute reconciles a AutoRepair object
type ReconcileRoute struct {
	client client.Client
	scheme *runtime.Scheme

	// configuration fields
	reconcilePeriod time.Duration
	configRoutes    bool

	nodeCache cmap.ConcurrentMap

	//record event recorder
	record record.EventRecorder

	neutronConfig *openstackCfg.Config
}

func (r *ReconcileRoute) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	// if do not need route, skip all node events
	if !r.configRoutes {
		return reconcile.Result{}, nil
	}

	reconcileNode := &corev1.Node{}
	err := r.client.Get(context.TODO(), request.NamespacedName, reconcileNode)
	if err != nil {
		if errors.IsNotFound(err) {
			if o, ok := r.nodeCache.Get(request.Name); ok {
				if route, ok := o.(*model.Route); ok {
					start := time.Now()
					var errList []error
					if err = deleteRouteForInstance(ctx, r.neutronConfig, route.DestinationCIDR); err != nil {
						errList = append(errList, err)
						klog.Errorf("error delete route entry for delete node %s route %v, error: %v", request.Name, route, err)
					} else {
						klog.Infof("successfully delete route entry for node %s route %s", request.Name, route)
					}
					metric.RouteLatency.WithLabelValues("delete").Observe(metric.MsSince(start))
					if aggrErr := utilerrors.NewAggregate(errList); aggrErr == nil {
						r.nodeCache.Remove(request.Name)
					} else {
						// requeue for remove error
						return reconcile.Result{}, aggrErr
					}
				}
			}
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	err = r.syncCloudRoute(ctx, reconcileNode)
	if err != nil {
		klog.Errorf("add route for node %s failed, err: %s", reconcileNode.Name, err.Error())
		nodeRef := &corev1.ObjectReference{
			Kind:      "Node",
			Name:      reconcileNode.Name,
			UID:       types.UID(reconcileNode.Name),
			Namespace: "",
		}
		r.record.Event(nodeRef, corev1.EventTypeWarning, helper.FailedSyncRoute, "sync cloud route failed")
	}
	// no need to retry, reconcileForCluster() will reconcile routes periodically
	return reconcile.Result{}, nil
}

func (r *ReconcileRoute) syncCloudRoute(ctx context.Context, node *corev1.Node) error {
	if !needSyncRoute(node) {
		return nil
	}

	_, ipv4RouteCidr, err := getIPv4RouteForNode(node)
	if err != nil || ipv4RouteCidr == "" {
		klog.Warningf("node %s parse podCIDR %s error, skip creating route", node.Name, node.Spec.PodCIDR)
		if err1 := r.updateNetworkingCondition(ctx, node, false); err1 != nil {
			klog.Errorf("route, update network condition error: %v", err1)
		}
		return err
	}

	var routeErr []error
	routeErr = append(routeErr, r.addRouteForNode(ctx, ipv4RouteCidr, node, nil))
	if utilerrors.NewAggregate(routeErr) != nil {
		err := r.updateNetworkingCondition(ctx, node, false)
		if err != nil {
			klog.Errorf("update network condition for node %s, error: %v", node.Name, err.Error())
		}
		return utilerrors.NewAggregate(routeErr)
	} else {
		return r.updateNetworkingCondition(ctx, node, true)
	}
}

func (r *ReconcileRoute) addRouteForNode(ctx context.Context, ipv4Cidr string, node *corev1.Node, cachedRouteEntry []*model.Route) error {
	var err error
	instanceId := getNodeInstanceId(ctx, r.neutronConfig, node)
	if len(instanceId) == 0 {
		return fmt.Errorf("cannot find instance uuid.")
	}
	nodeRef := &corev1.ObjectReference{
		Kind:      "Node",
		Name:      node.Name,
		UID:       types.UID(instanceId),
		Namespace: "",
	}

	route, findErr := findRoute(ctx, r.neutronConfig, ipv4Cidr, cachedRouteEntry)
	if findErr != nil {
		klog.Errorf("error found exist route for instance: %v, %v", nodeRef.UID, findErr)
		r.record.Event(
			nodeRef,
			corev1.EventTypeWarning,
			"DescriberRouteFailed",
			fmt.Sprintf("Describe Route Failed for %s reason: %s", ipv4Cidr, helper.GetLogMessage(findErr)),
		)
		return nil
	}

	// route not found, try to create route
	if route == nil || route.DestinationCIDR != ipv4Cidr {
		klog.Infof("create routes for node %s: %v - %v", node.Name, nodeRef.UID, ipv4Cidr)
		start := time.Now()
		route, err = createRouteForInstance(ctx, r.neutronConfig, string(nodeRef.UID), ipv4Cidr)
		if err != nil {
			klog.Errorf("error create route for node %v : instance id [%v], err: %s", node.Name, nodeRef.UID, err.Error())
			r.record.Event(
				nodeRef,
				corev1.EventTypeWarning,
				helper.FailedCreateRoute,
				fmt.Sprintf("Error creating route entry : %s", helper.GetLogMessage(err)),
			)
		} else {
			klog.Infof("Created route for %s - %s successfully", node.Name, ipv4Cidr)
			r.record.Event(
				nodeRef,
				corev1.EventTypeNormal,
				helper.SucceedCreateRoute,
				fmt.Sprintf("Created route for %s -> %s successfully", node.Name, ipv4Cidr),
			)
		}
		metric.RouteLatency.WithLabelValues("create").Observe(metric.MsSince(start))
	}
	if route != nil {
		r.nodeCache.SetIfAbsent(node.Name, route)
	}
	return err
}

func (r *ReconcileRoute) updateNetworkingCondition(ctx context.Context, node *corev1.Node, routeCreated bool) error {
	networkCondition, ok := helper.FindCondition(node.Status.Conditions, corev1.NodeNetworkUnavailable)
	if routeCreated && ok && networkCondition.Status == corev1.ConditionFalse {
		klog.Infof("set node %v with NodeNetworkUnavailable=false was canceled because it is already set", node.Name)
		return nil
	}

	if !routeCreated && ok && networkCondition.Status == corev1.ConditionTrue {
		klog.Infof("set node %v with NodeNetworkUnavailable=true was canceled because it is already set", node.Name)
		return nil
	}

	klog.Infof("Patching node status %v with %v previous condition was:%+v", node.Name, routeCreated, networkCondition)
	var err error
	for i := 0; i < updateNodeStatusMaxRetries; i++ {
		// Patch could also fail, even though the chance is very slim. So we still do
		// patch in the retry loop.
		diff := func(copy runtime.Object) (client.Object, error) {
			nins := copy.(*corev1.Node)
			condition, ok := helper.FindCondition(nins.Status.Conditions, corev1.NodeNetworkUnavailable)
			condition.Type = corev1.NodeNetworkUnavailable
			condition.LastTransitionTime = metav1.Now()
			condition.LastHeartbeatTime = metav1.Now()
			if routeCreated {
				condition.Status = corev1.ConditionFalse
				condition.Reason = "RouteCreated"
				condition.Message = "RouteController created a route"
			} else {
				condition.Status = corev1.ConditionTrue
				condition.Reason = "NoRouteCreated"
				condition.Message = "RouteController failed to create a route"
			}
			if !ok {
				nins.Status.Conditions = append(nins.Status.Conditions, *condition)
			}
			return nins, nil
		}
		err = helper.PatchM(r.client, node, diff, helper.PatchStatus)
		if err == nil {
			return nil
		}
		if !errors.IsConflict(err) {
			klog.Errorf("Error updating node %s: %v", node.Name, err)
			return err
		}
		klog.Infof("Error updating node %s, retrying: %v", node.Name, err)
	}
	klog.Errorf("Error updating node %s: %v", node.Name, err)
	return err
}

func (r *ReconcileRoute) periodicalSync() {
	go wait.Until(r.reconcileForCluster, r.reconcilePeriod, wait.NeverStop)
}

func (r *ReconcileRoute) reconcileForCluster() {
	ctx := context.Background()
	start := time.Now()
	defer func() {
		metric.RouteLatency.WithLabelValues("reconcile").Observe(metric.MsSince(start))
	}()

	nodes, err := r.NodeList()
	if err != nil {
		klog.Errorf("reconcile: error listing nodes: %v", err)
		return
	}

	// Sync for nodes
	if err := r.syncRoutes(ctx, r.neutronConfig, nodes); err != nil {
		klog.Errorf("sync route error: %s", err.Error())
	}

	klog.Infof("sync route successfully.")
}

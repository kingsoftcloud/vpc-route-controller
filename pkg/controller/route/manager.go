package route

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"

	"ezone.ksyun.com/ezone/kce/vpc-route-controller/pkg/controller/helper"
	"ezone.ksyun.com/ezone/kce/vpc-route-controller/pkg/ksyun"
	"ezone.ksyun.com/ezone/kce/vpc-route-controller/pkg/model"
)

var (
	createBackoff = wait.Backoff{
		Duration: 5 * time.Second,
		Steps:    3,
		Factor:   2,
		Jitter:   1,
	}

	routeLock = sync.Mutex{}
)

func createRouteForInstance(ctx context.Context, instanceId, cidr string) (
	*model.Route, error,
) {
	routeLock.Lock()
	defer routeLock.Unlock()
	var (
		route    *model.Route
		innerErr error
		findErr  error
	)
	err := wait.ExponentialBackoff(createBackoff, func() (bool, error) {
		innerErr = ksyun.CreateRoute(ctx, instanceId, cidr)
		if innerErr != nil {
			if strings.Contains(innerErr.Error(), "same with a route") {
				route, findErr = ksyun.FindRoute(ctx, cidr)
				if findErr == nil && route != nil {
					return true, nil
				}
				// fail fast, wait next time reconcile
				klog.Errorf("Backoff creating route: same cidr exsits: %s", innerErr.Error())
				return false, innerErr
			}
			klog.Errorf("Backoff creating route: %s", innerErr.Error())
			return false, nil
		}
		return true, nil
	})

	if err != nil {
		return nil, fmt.Errorf("error create route for node %v, err: %v", instanceId, err)
	}

	if route == nil {
		route, _ = ksyun.FindRoute(ctx, cidr)
	}

	return route, nil
}

func deleteRouteForInstance(ctx context.Context, cidr string) error {
	routeLock.Lock()
	defer routeLock.Unlock()
	return ksyun.DeleteRoute(ctx, cidr)
}

func (r *ReconcileRoute) syncRoutes(ctx context.Context, nodes *v1.NodeList) error {
	routes, err := ksyun.ListRoutes(ctx)
	if err != nil {
		return fmt.Errorf("error listing routes: %v", err)
	}

	for _, route := range routes {
		if conflictWithNodes(ctx, route, nodes) {
			if err = deleteRouteForInstance(ctx, route.DestinationCIDR); err != nil {
				klog.Errorf("Could not delete conflict route %s %s, %s", route.Name, route.DestinationCIDR, err.Error())
				continue
			}
			klog.Infof("Delete conflict route %s, %s SUCCESS.", route.Name, route.DestinationCIDR)
		}
	}

	for _, node := range nodes.Items {
		if !needSyncRoute(&node) {
			continue
		}

		_, ipv4RouteCidr, err := getIPv4RouteForNode(&node)
		if err != nil || ipv4RouteCidr == "" {
			continue
		}

		err = r.addRouteForNode(ctx, ipv4RouteCidr, &node, routes)
		if err != nil {
			continue
		}

		if err := r.updateNetworkingCondition(ctx, &node, true); err != nil {
			klog.Errorf("update node %s network condition err: %s", node.Name, err.Error())
		}
	}
	return nil
}

func conflictWithNodes(ctx context.Context, route *model.Route, nodes *v1.NodeList) bool {
	for _, node := range nodes.Items {
		ipv4Cidr, _, err := getIPv4RouteForNode(&node)
		if err != nil {
			klog.Errorf("error get ipv4 cidr from node: %v", node.Name)
			continue
		}
		if ipv4Cidr == nil {
			continue
		}
		equal, contains, err := containsRoute(ipv4Cidr, route.DestinationCIDR)
		if err != nil {
			klog.Errorf("error get conflict state from node: %v and route: %v", node.Name, route)
			continue
		}
		instanceId := getNodeInstanceId(ctx, &node)
		if contains || (equal && route.InstanceId != instanceId) {
			klog.Warningf("conflict route with node %v(%v) found, route: %+v", node.Name, ipv4Cidr, route)
			return true
		}

	}
	return false
}

func findRoute(ctx context.Context, cidr string, cachedRoutes []*model.Route) (*model.Route, error) {
	if cidr == "" {
		return nil, fmt.Errorf("empty query condition")
	}
	if len(cachedRoutes) != 0 {
		for _, route := range cachedRoutes {
			if cidr != "" {
				if route.DestinationCIDR == cidr {
					return route, nil
				}
			}
		}
		return nil, nil
	}
	return ksyun.FindRoute(ctx, cidr)
}

func containsRoute(outside *net.IPNet, insideRoute string) (containsEqual bool, realContains bool, err error) {
	if outside == nil {
		// outside is nil, contains all route
		return true, true, nil
	}
	_, cidr, err := net.ParseCIDR(insideRoute)
	if err != nil {
		return false, false, fmt.Errorf("ignoring route %s, unparsable CIDR: %v", insideRoute, err)
	}

	if outside.String() == insideRoute {
		return true, false, nil
	}

	lastIP := make([]byte, len(cidr.IP))
	for i := range lastIP {
		lastIP[i] = cidr.IP[i] | ^cidr.Mask[i]
	}
	if !outside.Contains(cidr.IP) || !outside.Contains(lastIP) {
		return false, false, nil
	}
	return true, true, nil
}

func needSyncRoute(node *v1.Node) bool {
	if helper.HasExcludeLabel(node) {
		klog.Infof("node %s has exclude label, skip creating route", node.Name)
		return false
	}

	readyCondition, ok := helper.FindCondition(node.Status.Conditions, v1.NodeReady)
	if ok && readyCondition.Status == v1.ConditionUnknown {
		klog.Infof("node %s is in unknown status, skip creating route", node.Name)
		return false
	}

	if node.DeletionTimestamp != nil {
		klog.Infof("node %s has deletionTimestamp, skip creating route", node.Name)
		return false
	}

	return true
}

func getIPv4RouteForNode(node *v1.Node) (*net.IPNet, string, error) {
	var (
		ipv4CIDR    *net.IPNet
		ipv4CIDRStr string
		err         error
	)
	for _, podCidr := range append(node.Spec.PodCIDRs, node.Spec.PodCIDR) {
		if podCidr != "" {
			_, ipv4CIDR, err = net.ParseCIDR(podCidr)
			if err != nil {
				return nil, "", fmt.Errorf("invalid pod cidr on node spec: %v", podCidr)
			}
			ipv4CIDRStr = ipv4CIDR.String()
			if len(ipv4CIDR.Mask) == net.IPv4len {
				ipv4CIDRStr = ipv4CIDR.String()
				break
			}
		}
	}
	return ipv4CIDR, ipv4CIDRStr, nil
}

func (r *ReconcileRoute) NodeList() (*v1.NodeList, error) {
	nodes := &v1.NodeList{}
	err := r.client.List(context.TODO(), nodes)
	if err != nil {
		return nil, err
	}
	var mnodes []v1.Node
	for _, node := range nodes.Items {
		if helper.HasExcludeLabel(&node) {
			continue
		}
		mnodes = append(mnodes, node)
	}
	nodes.Items = mnodes
	return nodes, nil
}

func getNodeInstanceId(ctx context.Context, node *v1.Node) string {
	annotations := node.Annotations
	if _, ok := annotations["appengine.sdns.ksyun.com/instance-uuid"]; ok {
		return annotations["appengine.sdns.ksyun.com/instance-uuid"]
	}

	if _, ok := annotations["kce.sdns.ksyun.com/instanceId"]; ok {
		return annotations["kce.sdns.ksyun.com/instanceId"]
	}

	if node.Spec.ProviderID != "" {
		return node.Spec.ProviderID
	}

	/*
	for _, addr := range node.Status.Addresses {
		if addr.Type == "InternalIP" {
			nodeIP := addr.Address
			instanceId, err := ksyun.GetInstanceIdFromIP(ctx, nodeIP)
			if err != nil {
				return ""
			}
			return instanceId
		}
	}*/

	return ""
}

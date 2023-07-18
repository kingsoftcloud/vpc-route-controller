package route

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"reflect"

	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type predicateForNodeEvent struct {
	predicate.Funcs
}

func (sp *predicateForNodeEvent) Update(e event.UpdateEvent) bool {
	oldNode, ok1 := e.ObjectOld.(*v1.Node)
	newNode, ok2 := e.ObjectNew.(*v1.Node)
	if ok1 && ok2 {
		_, ok1 := oldNode.Annotations["appengine.sdns.ksyun.com/instance-uuid"]
		newId1, ok2 := newNode.Annotations["appengine.sdns.ksyun.com/instance-uuid"]
		_, ok3 := oldNode.Annotations["kce.sdns.ksyun.com/instanceId"]
		newId2, ok4 := newNode.Annotations["kce.sdns.ksyun.com/instanceId"]
		if !ok2 && !ok4 {
			return false
		}
		if !ok1 && ok2 {
			klog.Infof("node uuid changed: %s instance uuid changed: %v - %v", oldNode.Name, "", newId1)
			return true
		}
		if !ok3 && ok4 {
			klog.Infof("node uuid changed: %s instance uuid changed: %v - %v", oldNode.Name, "", newId2)
			return true
		}


		if oldNode.UID != newNode.UID {
			klog.Infof("node changed: %s UIDChanged: %v - %v", oldNode.Name, oldNode.UID, newNode.UID)
			return true
		}
		if oldNode.Spec.PodCIDR != newNode.Spec.PodCIDR {
			klog.Infof("node changed: %s Pod CIDR Changed: %v - %v", oldNode.Name, oldNode.Spec.PodCIDR, newNode.Spec.PodCIDR)
			return true
		}
		if !reflect.DeepEqual(oldNode.Spec.PodCIDRs, newNode.Spec.PodCIDRs) {
			klog.Infof("node changed: %s Pod CIDRs Changed: %v - %v", oldNode.Name, oldNode.Spec.PodCIDRs, newNode.Spec.PodCIDRs)
			return true
		}

		return false
	}
	return true
}

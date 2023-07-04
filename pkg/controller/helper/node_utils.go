package helper

import (
	"context"
	"encoding/json"
	"fmt"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	PatchAll    = "all"
	PatchSpec   = "spec"
	PatchStatus = "status"
)

const (
	LabelNodeTypeVK = "virtual-kubelet"

	LabelNodeExcludeNode           = "service.ksyun.com/exclude-node"
	LabelNodeExcludeNodeDeprecated = "service.beta.kubernetes.io/exclude-node"
)

func PatchM(mclient client.Client, target client.Object, getter func(runtime.Object) (client.Object, error), resource string,
) error {
	err := mclient.Get(
		context.TODO(),
		client.ObjectKey{
			Name:      target.GetName(),
			Namespace: target.GetNamespace(),
		}, target,
	)
	if err != nil {
		return fmt.Errorf("get origin object: %s", err.Error())
	}

	ntarget, err := getter(target.DeepCopyObject())
	if err != nil {
		return fmt.Errorf("get object diff patch: %s", err.Error())
	}
	oldData, err := json.Marshal(target)
	if err != nil {
		return fmt.Errorf("ensure marshal: %s", err.Error())
	}
	newData, err := json.Marshal(ntarget)
	if err != nil {
		return fmt.Errorf("ensure marshal: %s", err.Error())
	}
	patchBytes, patchErr := strategicpatch.CreateTwoWayMergePatch(oldData, newData, target)
	if patchErr != nil {
		return fmt.Errorf("create merge patch: %s", patchErr.Error())
	}

	if string(patchBytes) == "{}" {
		return nil
	}

	klog.Infof("try to patch %s/%s, %s ", target.GetNamespace(), target.GetName(), string(patchBytes))
	if resource == PatchSpec || resource == PatchAll {
		err := mclient.Patch(
			context.TODO(), ntarget,
			client.RawPatch(types.StrategicMergePatchType, patchBytes),
		)
		if err != nil {
			return fmt.Errorf("patch spec: %s", err.Error())
		}
	}

	if resource == PatchStatus || resource == PatchAll {
		return mclient.Status().Patch(
			context.TODO(), ntarget,
			client.RawPatch(types.StrategicMergePatchType, patchBytes),
		)
	}
	return nil
}

func FindCondition(conds []v1.NodeCondition, conditionType v1.NodeConditionType) (*v1.NodeCondition, bool) {
	var retCon *v1.NodeCondition
	for i := range conds {
		if conds[i].Type == conditionType {
			if retCon == nil || retCon.LastHeartbeatTime.Before(&conds[i].LastHeartbeatTime) {
				retCon = &conds[i]
			}
		}
	}

	if retCon == nil {
		return &v1.NodeCondition{}, false
	} else {
		return retCon, true
	}
}

// GetNodeCondition will get pointer to Node's existing condition.
// returns nil if no matching condition found.
func GetNodeCondition(node *v1.Node, conditionType v1.NodeConditionType) *v1.NodeCondition {
	for i := range node.Status.Conditions {
		if node.Status.Conditions[i].Type == conditionType {
			return &node.Status.Conditions[i]
		}
	}
	return nil
}

func HasExcludeLabel(node *v1.Node) bool {
	if _, exclude := node.Labels[LabelNodeExcludeNodeDeprecated]; exclude {
		return true
	}
	if _, exclude := node.Labels[LabelNodeExcludeNode]; exclude {
		return true
	}
	return false
}

func FindNodeByNodeName(nodes []v1.Node, nodeName string) *v1.Node {
	for _, n := range nodes {
		if n.Name == nodeName {
			return &n
		}
	}
	klog.Infof("node %s not found ", nodeName)
	return nil
}

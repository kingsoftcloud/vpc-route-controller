package annotation

import (
	"fmt"
	"io"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"net/http"
)

const (
	// KCE1NodeAnnotationInstanceUUIDKey KCE1.0 node instance id annotation key
	KCE1NodeAnnotationInstanceUUIDKey = "appengine.sdns.ksyun.com/instance-uuid"
	// KCE2NodeAnnotationInstanceUUIDKey KCE2.0 node instance id annotation key
	KCE2NodeAnnotationInstanceUUIDKey = "kce.sdns.ksyun.com/instanceId"
)

const (
	metadataUrl = "http://11.255.255.100:8775/latest/meta-data/"
)

// IsExistedInstanceUUIDKey Check if the key exists
func IsExistedInstanceUUIDKey(node *v1.Node) bool {
	if node.Annotations != nil {
		if _, existed := node.Annotations[KCE1NodeAnnotationInstanceUUIDKey]; existed {
			return true
		}
		if _, existed := node.Annotations[KCE2NodeAnnotationInstanceUUIDKey]; existed {
			return true
		}
	}
	return false
}

// SetInstanceUUID Set keys for node annotations, node annotations may be nil
func SetInstanceUUID(node *v1.Node, instanceId string) *v1.Node {
	var annotations map[string]string
	if node.Annotations != nil {
		annotations = node.Annotations
	} else {
		annotations = make(map[string]string)
	}
	annotations[KCE2NodeAnnotationInstanceUUIDKey] = instanceId
	node.Annotations = annotations
	return node
}

func getMetadataInstanceId() (string, error) {
	return getMetadata("instance-id")
}

func getMetadata(path string) (string, error) {
	url := metadataUrl + path
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	regionByte, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	// If failed, often 404
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Failed to obtain from metadata interface, accress path: %s, return: %s ", url, string(regionByte))
	}
	return string(regionByte), nil
}

// EnsureNodeInstanceId Adding instanceId annotation to nodes,
// If instanceId annotation already exist, skip directly
func EnsureNodeInstanceId(nodeName string) error {
	klog.Infoln("Call EnsureNodeInstanceId, node:", nodeName)
	c, err := NewController()
	if err != nil {
		return err
	}
	node, err := c.GetNode(nodeName)
	if err != nil {
		return err
	}
	if IsExistedInstanceUUIDKey(node) {
		klog.Infoln("Current node already annotation InstanceUUID, annotation:", node.Annotations)
		return nil
	}
	instanceId, err := getMetadataInstanceId()
	if err != nil {
		return err
	}
	klog.Infoln("Node instance Id:", instanceId)
	node = SetInstanceUUID(node, instanceId)
	klog.Infoln("Node set annotations:", node.Annotations)
	_, err = c.UpdateNode(node)
	if err != nil {
		return err
	}
	klog.Infoln("Node update completed")
	return nil
}

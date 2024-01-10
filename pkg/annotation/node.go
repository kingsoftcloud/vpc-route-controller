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
	// KCE1NodeAnnotationZoneKey KCE1.0 node placement zone annotation key
	KCE1NodeAnnotationZoneKey = "appengine.sdns.ksyun.com/zone"
	// KCE2NodeAnnotationZoneKey KCE2.0 node placement zone annotation key
	KCE2NodeAnnotationZoneKey = "kce.sdns.ksyun.com/zone"
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

func IsExistedZoneKey(node *v1.Node) bool {
	if node.Annotations != nil {
		if _, existed := node.Annotations[KCE1NodeAnnotationZoneKey]; existed {
			return true
		}
		if _, existed := node.Annotations[KCE2NodeAnnotationZoneKey]; existed {
			return true
		}
	}
	return false
}

// SetInstanceUUID Set keys for node annotations, node annotations may be nil
func SetInstanceUUID(node *v1.Node, instanceId string) *v1.Node {
	return setAnnotation(node, KCE2NodeAnnotationInstanceUUIDKey, instanceId)
}

func SetZone(node *v1.Node, Zone string) *v1.Node {
	return setAnnotation(node, KCE2NodeAnnotationZoneKey, Zone)
}

func setAnnotation(node *v1.Node, key, value string) *v1.Node {
	var annotations map[string]string
	if node.Annotations != nil {
		annotations = node.Annotations
	} else {
		annotations = make(map[string]string)
	}
	annotations[key] = value
	node.Annotations = annotations
	return node
}

func getMetadataInstanceId() (string, error) {
	return getMetadata("instance-id")
}

func getMetadataZone() (string, error) {
	return getMetadata("placement/zone")
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
func (c *Controller) EnsureNodeInstanceId(nodeName string) error {
	klog.Infoln("Call EnsureNodeInstanceId, node:", nodeName)
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

func (c *Controller) EnsureNodeZone(nodeName string) error {
	klog.Infoln("Call EnsureNodeZone, node:", nodeName)
	node, err := c.GetNode(nodeName)
	if err != nil {
		return err
	}
	if IsExistedZoneKey(node) {
		klog.Infoln("Current node already annotation Zone, annotation:", node.Annotations)
		return nil
	}
	zone, err := getMetadataZone()
	if err != nil {
		return err
	}
	klog.Infoln("Node placement zone:", zone)
	node = SetZone(node, zone)
	klog.Infoln("Node set annotations:", node.Annotations)
	_, err = c.UpdateNode(node)
	if err != nil {
		return err
	}
	klog.Infoln("Node update completed")
	return nil
}

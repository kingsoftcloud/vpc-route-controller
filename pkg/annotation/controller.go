package annotation

import (
	"context"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type Controller struct {
	Client *kubernetes.Clientset
}

func NewController() (*Controller, error) {
	// /etc/kubernetes/admin.conf
	config, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	c := &Controller{Client: clientset}
	return c, nil
}

func (c *Controller) GetNode(nodeName string) (*v1.Node, error) {
	return c.Client.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
}

func (c *Controller) UpdateNode(node *v1.Node) (*v1.Node, error) {
	return c.Client.CoreV1().Nodes().Update(context.TODO(), node, metav1.UpdateOptions{})
}

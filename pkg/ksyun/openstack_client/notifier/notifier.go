package notifier

import (
	"context"
	"k8s.io/klog"
	nethttp "net/http"

	"ezone.ksyun.com/ezone/kce/vpc-route-controller/pkg/http"
	openTypes "ezone.ksyun.com/ezone/kce/vpc-route-controller/pkg/ksyun/openstack_client/types"
)

const (
	defaultAlarmUrl = "http://alarm.inner.sdns.ksyun.com/alarm/receptor"
	defaultProduct  = "CONTAINER_NETWORK"
)

type NotifierClient struct {
	AlarmUrl string
	Product  string
}

func NewNotifierClient() *NotifierClient {
	return &NotifierClient{
		AlarmUrl: defaultAlarmUrl,
		Product:  defaultProduct,
	}
}

func (notifier *NotifierClient) Notify(ctx context.Context, messages []openTypes.NotifyMessage) {
	httpClient := http.NewClient(&nethttp.Client{})

	for _, message := range messages {
		body := map[string]string{
			"name":     message.Name,
			"product":  notifier.Product,
			"priority": message.Priority,
			"content":  message.Content,
			"no_deal":  message.NoDeal,
		}
		klog.Infof("sending message to onepiece: %v", body)

		_, err := httpClient.PostForm(notifier.AlarmUrl, body)
		if err != nil {
			klog.Errorf("error sending message to onepiece: %v", err)
			continue
		}
	}

	return
}

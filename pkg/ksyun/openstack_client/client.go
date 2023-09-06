package openstack_client

import (
	"context"
	"ezone.ksyun.com/ezone/kce/vpc-route-controller/pkg/ksyun/openstack_client/alarm"
	"ezone.ksyun.com/ezone/kce/vpc-route-controller/pkg/ksyun/openstack_client/config"
	"ezone.ksyun.com/ezone/kce/vpc-route-controller/pkg/ksyun/openstack_client/neutron"
	//"ezone.ksyun.com/ezone/kce/vpc-route-controller/pkg/ksyun/openstack_client/notifier"
	"ezone.ksyun.com/ezone/kce/vpc-route-controller/pkg/ksyun/openstack_client/nova"
)

func Route(ctx context.Context, conf *config.Config) (*neutron.RouteClient, error) {
	return neutron.NewRouteClient(ctx, conf)
}

func Server(ctx context.Context, conf *config.Config) (*nova.ServerClient, error) {
	return nova.NewServerClient(ctx, conf)
}

/*func Notifier() *notifier.NotifierClient {
	return notifier.NewNotifierClient()
}*/

func Alarm(ctx context.Context, conf *config.Config) *alarm.AlarmClient {
	return alarm.NewAlarmClient(ctx, conf)
}

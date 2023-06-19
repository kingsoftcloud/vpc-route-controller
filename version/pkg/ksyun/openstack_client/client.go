package openstack_client

import (
	"context"
	"newgit.op.ksyun.com/kce/vpc-route-controller/pkg/ksyun/openstack_client/config"
	"newgit.op.ksyun.com/kce/vpc-route-controller/pkg/ksyun/openstack_client/neutron"
)

func Route(ctx context.Context, conf *config.Config) (*neutron.RouteClient, error) {
	return neutron.NewRouteClient(ctx, conf)
}

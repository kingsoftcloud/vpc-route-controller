package ksyun

import (
	"encoding/json"
	"fmt"
	"os"

	"golang.org/x/net/context"
	log "k8s.io/klog/v2"

	openstack_client "ezone.ksyun.com/ezone/kce/vpc-route-controller/pkg/ksyun/openstack_client"
	"ezone.ksyun.com/ezone/kce/vpc-route-controller/pkg/ksyun/openstack_client/config"
	openstackTypes "ezone.ksyun.com/ezone/kce/vpc-route-controller/pkg/ksyun/openstack_client/types"
	"ezone.ksyun.com/ezone/kce/vpc-route-controller/pkg/model"

	"github.com/kingsoftcloud/aksk-provider/env"
	"github.com/kingsoftcloud/aksk-provider/file"
)

const (
	defaultRouteType       = "Host"
	defaultNetworkEndpoint = "http://internal.api.ksyun.com"
)

var (
	DefaultCipherKey string
)

func GetInstanceIdFromIP(ctx context.Context, cfg *config.Config, privateIP string) (string, error) {
	s, err := openstack_client.Server(ctx, cfg)
	if err != nil {
		return "", err
	}

	getInstances := &openstackTypes.InstanceArgs{
		DomainId:          cfg.VpcID,
		InstanceType:      defaultRouteType,
		InstancePrivateIP: privateIP,
	}

	log.V(9).Infof("Check ksc nova instance args: %v \n", getInstances)

	result, err := s.DescribeInstances(getInstances)
	if err != nil {
		log.Errorf("Error get instance %s: %s .\n", privateIP, getErrorString(err))
		mesg := openstackTypes.NotifyMessage{
			Name:     "GetInstanceIdFromIP",
			Priority: "2",
			NoDeal:   "1",
			Content:  fmt.Sprintf("region: %s, cluster: %s, plugin: vpc-route-controller,  error: %s", cfg.Region, cfg.ClusterUUID, err.Error()),
		}

		var mesgs []openstackTypes.NotifyMessage
		mesgs = append(mesgs, mesg)
		n := openstack_client.Notifier()
		n.Notify(ctx, mesgs)

		return "", err
	}
	return result.Id, nil
}

func ListRoutes(ctx context.Context, cfg *config.Config) ([]*model.Route, error) {
	var result []*model.Route
	r, err := openstack_client.Route(ctx, cfg)
	if err != nil {
		return result, err
	}

	getRoutes := &openstackTypes.RouteArgs{
		DomainId:     cfg.VpcID,
		InstanceType: defaultRouteType,
	}

	log.Infof("Check ksc vpc route args: %v \n", getRoutes)

	routes, err := r.ListRoutes(getRoutes)
	if err != nil {
		log.Errorf("Error CheckRouteEntry: %s .\n", getErrorString(err))

		mesg := openstackTypes.NotifyMessage{
			Name:     "ListRoutes",
			Priority: "2",
			NoDeal:   "1",
			Content:  fmt.Sprintf("region: %s, cluster: %s, plugin: vpc-route-controller,  error: %s", cfg.Region, cfg.ClusterUUID, err.Error()),
		}

		var mesgs []openstackTypes.NotifyMessage
		mesgs = append(mesgs, mesg)
		n := openstack_client.Notifier()
		n.Notify(ctx, mesgs)

		return result, err
	}

	for _, r := range routes {
		if r.DestinationCIDR == "0.0.0.0/0" {
			continue
		}

		gatewayId := ""
		if  len(r.NextHopset) != 0 {
			gatewayId = r.NextHopset[0].GatewayId
		}
		m := &model.Route{
			Name:            fmt.Sprintf("%s-%s", r.RouteId, r.DestinationCIDR),
			DestinationCIDR: r.DestinationCIDR,
			RouteId:         r.RouteId,
			InstanceId:      gatewayId,
		}

		result = append(result, m)
	}
	return result, nil
}

func FindRoute(ctx context.Context, cfg *config.Config, cidr string) (*model.Route, error) {
	r, err := openstack_client.Route(ctx, cfg)
	if err != nil {
		return nil, err
	}

	getRoutes := &openstackTypes.RouteArgs{
		DomainId:     cfg.VpcID,
		InstanceType: defaultRouteType,
		CidrBlock:    cidr,
	}

	log.Infof("Check ksc vpc route args: %v \n", getRoutes)

	routes, err := r.GetRoutes(getRoutes)
	if err != nil {
		log.Errorf("Error CheckRouteEntry: %s .\n", getErrorString(err))
		mesg := openstackTypes.NotifyMessage{
			Name:     "FindRoute",
			Priority: "2",
			NoDeal:   "1",
			Content:  fmt.Sprintf("region: %s, cluster: %s, plugin: vpc-route-controller,  error: %s", cfg.Region, cfg.ClusterUUID, err.Error()),
		}

		var mesgs []openstackTypes.NotifyMessage
		mesgs = append(mesgs, mesg)
		n := openstack_client.Notifier()
		n.Notify(ctx, mesgs)

		return nil, err
	}

	if len(routes) == 0 {
		return nil, nil
	}

	return &model.Route{
		Name:            fmt.Sprintf("%s-%s", routes[0].RouteId, routes[0].DestinationCIDR),
		DestinationCIDR: routes[0].DestinationCIDR,
		RouteId:         routes[0].RouteId,
		InstanceId:      routes[0].NextHopset[0].GatewayId,
	}, nil
}

func DeleteRoute(ctx context.Context, cfg *config.Config, cidr string) error {
	r, err := openstack_client.Route(ctx, cfg)
	if err != nil {
		return err
	}

	route, err := FindRoute(ctx, cfg, cidr)
	if err != nil {
		return err
	}
	if route != nil {
		log.Infof("vpc id %s delete route id: %s", cfg.VpcID, route.RouteId)
		if err := r.DeleteRoute(route.RouteId); err != nil {
			log.Errorf("Error deleteRoute: %s . \n", getErrorString(err))

			mesg := openstackTypes.NotifyMessage{
				Name:     "DeleteRoute",
				Priority: "2",
				NoDeal:   "1",
				Content:  fmt.Sprintf("region: %s, cluster: %s, plugin: vpc-route-controller,  error: %s", cfg.Region, cfg.ClusterUUID, err.Error()),
			}

			var mesgs []openstackTypes.NotifyMessage
			mesgs = append(mesgs, mesg)
			n := openstack_client.Notifier()
			n.Notify(ctx, mesgs)

			return fmt.Errorf("Error deleteRoute: %s . \n", getErrorString(err))
		}
	}

	return nil
}

func CreateRoute(ctx context.Context, cfg *config.Config, instanceId, cidr string) error {
	log.Infof("begin to create route: vpc %s, instance %s, cidr %s", cfg.VpcID, instanceId, cidr)

	r, err := openstack_client.Route(ctx, cfg)
	if err != nil {
		return err
	}

	createRoute := &openstackTypes.RouteArgs{
		DomainId:     cfg.VpcID,
		InstanceId:   instanceId,
		InstanceType: defaultRouteType,
		CidrBlock:    cidr,
	}

	id, err := r.CreateRoute(createRoute)
	if err != nil {
		mesg := openstackTypes.NotifyMessage{
			Name:     "CreateRoute",
			Priority: "2",
			NoDeal:   "1",
			Content:  fmt.Sprintf("region: %s, cluster: %s, plugin: vpc-route-controller,  error: %s", cfg.Region, cfg.ClusterUUID, err.Error()),
		}

		var mesgs []openstackTypes.NotifyMessage
		mesgs = append(mesgs, mesg)
		n := openstack_client.Notifier()
		n.Notify(ctx, mesgs)

		return fmt.Errorf("Error createRoute: %s . \n", getErrorString(err))
	}

	if err := r.WaitForAllRouteEntriesAvailable(id, 60); err != nil {
		return fmt.Errorf("Error not found Route: %s . \n", getErrorString(err))

	}

	return nil
}

func getErrorString(e error) string {
	if e == nil {
		return ""
	}
	return e.Error()
}

func GetNeutronConfig() (*config.Config, error) {
	var c config.Config

	content := os.Getenv("NET_CONF")
	if content == "" {
		return nil, fmt.Errorf("net config is null.")
	}

	if err := json.Unmarshal([]byte(content), &c); err != nil {
		return nil, fmt.Errorf("json unmarshal %s error: %v", content, err)
	}

	switch c.AkskType {
	case "env":
		c.AkskProvider = env.NewEnvAKSKProvider(c.Encrypt, DefaultCipherKey)
	case "file":
		c.AkskProvider = file.NewFileAKSKProvider(c.AkskFilePath, DefaultCipherKey)
	default:
		return nil, fmt.Errorf("please set aksk type.")
	}

	return &c, nil
}

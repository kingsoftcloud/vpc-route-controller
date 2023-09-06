package ksyun

import (
	"encoding/json"
	"fmt"
	"os"

	"golang.org/x/net/context"
	log "k8s.io/klog/v2"

	openstack_client "ezone.ksyun.com/ezone/kce/vpc-route-controller/pkg/ksyun/openstack_client"
	"ezone.ksyun.com/ezone/kce/vpc-route-controller/pkg/ksyun/openstack_client/alarm"
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
	Cfg              *config.Config
	err              error
)

func init() {
	Cfg, err = GetNeutronConfig()
	if err != nil {
		log.Error(err, "failed to get neutron config")
		os.Exit(1)
	}
}

func GetInstanceIdFromIP(ctx context.Context, privateIP string) (string, error) {
	s, err := openstack_client.Server(ctx, Cfg)
	if err != nil {
		return "", err
	}

	getInstances := &openstackTypes.InstanceArgs{
		DomainId:          Cfg.VpcID,
		InstanceType:      defaultRouteType,
		InstancePrivateIP: privateIP,
	}

	log.V(9).Infof("Check ksc nova instance args: %v \n", getInstances)

	result, err := s.DescribeInstances(getInstances)
	if err != nil {
		log.Errorf("Error get instance %s: %s .\n", privateIP, getErrorString(err))

		if Cfg.AlarmEnabled {
			mesg := openstackTypes.AlarmArgs{
				Name:     "GetInstanceIdFromIP",
				Priority: "2",
				Product:  alarm.DefaultProduct,
				NoDeal:   "1",
				Content:  fmt.Sprintf("region: %s, cluster: %s, plugin: vpc-route-controller,  error: %s", Cfg.Region, Cfg.ClusterUUID, err.Error()),
			}

			alarmClient := openstack_client.Alarm(ctx, Cfg)
			alarmClient.CreateAlarm(mesg)
		}

		return "", err
	}
	return result.Id, nil
}

func ListRoutes(ctx context.Context) ([]*model.Route, error) {
	var result []*model.Route
	r, err := openstack_client.Route(ctx, Cfg)
	if err != nil {
		return result, err
	}

	getRoutes := &openstackTypes.RouteArgs{
		DomainId:     Cfg.VpcID,
		InstanceType: defaultRouteType,
	}

	log.Infof("Check ksc vpc route args: %v \n", getRoutes)

	routes, err := r.ListRoutes(getRoutes)
	if err != nil {
		log.Errorf("Error CheckRouteEntry: %s .\n", getErrorString(err))

		if Cfg.AlarmEnabled {
			mesg := openstackTypes.AlarmArgs{
				Name:     "ListRoutes",
				Priority: "2",
				Product:  alarm.DefaultProduct,
				NoDeal:   "1",
				Content:  fmt.Sprintf("region: %s, cluster: %s, plugin: vpc-route-controller,  error: %s", Cfg.Region, Cfg.ClusterUUID, err.Error()),
			}

			alarmClient := openstack_client.Alarm(ctx, Cfg)
			alarmClient.CreateAlarm(mesg)
		}

		return result, err
	}

	if Cfg.AlarmEnabled {
		mesg := openstackTypes.AlarmArgs{
			Name:     "ListRoutes",
			Priority: "2",
			Product:  alarm.DefaultProduct,
			NoDeal:   "1",
			Content:  "eeeee",
		}
		alarmClient := openstack_client.Alarm(ctx, Cfg)
		alarmClient.CreateAlarm(mesg)
	}

	for _, r := range routes {
		if r.DestinationCIDR == "0.0.0.0/0" {
			continue
		}

		gatewayId := ""
		if len(r.NextHopset) != 0 {
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

func FindRoute(ctx context.Context, cidr string) (*model.Route, error) {
	r, err := openstack_client.Route(ctx, Cfg)
	if err != nil {
		return nil, err
	}

	getRoutes := &openstackTypes.RouteArgs{
		DomainId:     Cfg.VpcID,
		InstanceType: defaultRouteType,
		CidrBlock:    cidr,
	}

	log.Infof("Check ksc vpc route args: %v \n", getRoutes)

	routes, err := r.GetRoutes(getRoutes)
	if err != nil {
		log.Errorf("Error CheckRouteEntry: %s .\n", getErrorString(err))

		if Cfg.AlarmEnabled {
			mesg := openstackTypes.AlarmArgs{
				Name:     "FindRoute",
				Priority: "2",
				Product:  alarm.DefaultProduct,
				NoDeal:   "1",
				Content:  fmt.Sprintf("region: %s, cluster: %s, plugin: vpc-route-controller,  error: %s", Cfg.Region, Cfg.ClusterUUID, err.Error()),
			}

			alarmClient := openstack_client.Alarm(ctx, Cfg)
			alarmClient.CreateAlarm(mesg)
		}

		return nil, err
	}

	if len(routes) == 0 {
		return nil, nil
	}

	gatewayId := ""
	if len(routes[0].NextHopset) != 0 {
		gatewayId = routes[0].NextHopset[0].GatewayId
	}

	return &model.Route{
		Name:            fmt.Sprintf("%s-%s", routes[0].RouteId, routes[0].DestinationCIDR),
		DestinationCIDR: routes[0].DestinationCIDR,
		RouteId:         routes[0].RouteId,
		InstanceId:      gatewayId,
	}, nil
}

func DeleteRoute(ctx context.Context, cidr string) error {
	r, err := openstack_client.Route(ctx, Cfg)
	if err != nil {
		return err
	}

	route, err := FindRoute(ctx, cidr)
	if err != nil {
		return err
	}
	if route != nil {
		log.Infof("vpc id %s delete route id: %s", Cfg.VpcID, route.RouteId)
		if err := r.DeleteRoute(route.RouteId); err != nil {
			log.Errorf("Error deleteRoute: %s . \n", getErrorString(err))

			if Cfg.AlarmEnabled {
				mesg := openstackTypes.AlarmArgs{
					Name:     "DeleteRoute",
					Priority: "2",
					Product:  alarm.DefaultProduct,
					NoDeal:   "1",
					Content:  fmt.Sprintf("region: %s, cluster: %s, plugin: vpc-route-controller,  error: %s", Cfg.Region, Cfg.ClusterUUID, err.Error()),
				}

				alarmClient := openstack_client.Alarm(ctx, Cfg)
				alarmClient.CreateAlarm(mesg)
			}

			return fmt.Errorf("Error deleteRoute: %s . \n", getErrorString(err))
		}
	}

	return nil
}

func CreateRoute(ctx context.Context, instanceId, cidr string) error {
	log.Infof("begin to create route: vpc %s, instance %s, cidr %s", Cfg.VpcID, instanceId, cidr)

	r, err := openstack_client.Route(ctx, Cfg)
	if err != nil {
		return err
	}

	createRoute := &openstackTypes.RouteArgs{
		DomainId:     Cfg.VpcID,
		InstanceId:   instanceId,
		InstanceType: defaultRouteType,
		CidrBlock:    cidr,
	}

	id, err := r.CreateRoute(createRoute)
	if err != nil {
		if Cfg.AlarmEnabled {
			mesg := openstackTypes.AlarmArgs{
				Name:     "CreateRoute",
				Priority: "2",
				Product:  alarm.DefaultProduct,
				NoDeal:   "1",
				Content:  fmt.Sprintf("region: %s, cluster: %s, plugin: vpc-route-controller,  error: %s", Cfg.Region, Cfg.ClusterUUID, err.Error()),
			}

			alarmClient := openstack_client.Alarm(ctx, Cfg)
			alarmClient.CreateAlarm(mesg)
		}

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

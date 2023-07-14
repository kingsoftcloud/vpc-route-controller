package neutron

import (
	"context"
	"encoding/json"
	"fmt"
	"k8s.io/klog"
	"net/url"
	"sync"
	"strings"
	"time"

	prvd "newgit.op.ksyun.com/kce/aksk-provider"
	kopHttp "newgit.op.ksyun.com/kce/vpc-route-controller/pkg/http"
	"newgit.op.ksyun.com/kce/vpc-route-controller/pkg/ksyun/openstack_client/config"
	openTypes "newgit.op.ksyun.com/kce/vpc-route-controller/pkg/ksyun/openstack_client/types"
	"newgit.op.ksyun.com/kce/vpc-route-controller/pkg/ksyun/openstack_client/utils"
)

const (
	defaultVersion         = "2016-03-04"
	defautlServerName      = "vpc"
	DefaultTimeout         = 60
	DefaultWaitForInterval = 5
)

type RouteClient struct {
	conf         *config.Config
	client       *kopHttp.KopClient
	tenantID     string
	headers      map[string]string
	lock         sync.Mutex
	akskProvider prvd.AKSKProvider
}

func NewRouteClient(ctx context.Context, conf *config.Config) (*RouteClient, error) {
	if len(conf.NetworkEndpoint) == 0 {
		conf.NetworkEndpoint = config.DefaultNetworkEndpoint
	}
	dataClient := kopHttp.NewKopClient(ctx)
	/*if len(conf.Token) == 0 {
	        conf.Token = fmt.Sprintf("%s:%s", conf.UserID, conf.TenantID)
	}*/

	headers := make(map[string]string)
	//headers["X-Auth-Project-Id"] = conf.TenantID
	//headers["X-Auth-Token"] = conf.Token
	headers["User-Agent"] = "vpc-route-controller"
	headers["Content-Type"] = "application/json"
	headers["Accept"] = "application/json"
	headers["X-Auth-User-Tag"] = "docker"

	routeClient := &RouteClient{
		conf:    conf,
		headers: headers,
		client:  dataClient,
		//tenantID: conf.TenantID,
		akskProvider: conf.AkskProvider,
	}

	return routeClient, nil
}

func (c *RouteClient) CreateRoute(args *openTypes.RouteArgs) (string, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	aksk, err := c.akskProvider.GetAKSK()
	if err != nil {
		return "", err
	}

	action := url.Values{
		"Action":               []string{"CreateRoute"},
		"Version":              []string{defaultVersion},
		"VpcId":                []string{args.DomainId},
		"RouteType":            []string{args.InstanceType},
		"InstanceId":           []string{args.InstanceId},
		"DestinationCidrBlock": []string{args.CidrBlock},
	}
	klog.Infof("create neutron route : %s", c.conf.NetworkEndpoint)
	if len(aksk.SecurityToken) != 0 {
		c.headers["X-Ksc-Security-Token"] = aksk.SecurityToken
	}
	c.client.SetEndpoint(c.conf.NetworkEndpoint)
	c.client.SetHeader(c.headers)
	c.client.SetBody(args)
	c.client.SetUrlQuery("", action)
	c.client.SetMethod(kopHttp.POST)
	c.client.SetSigner(defautlServerName, c.conf.Region, aksk.AK, aksk.SK)
	data, err := c.client.Go()
	if err != nil {
		if strings.Contains(err.Error(), "SecurityTokenExpired") {
			aksk, err := c.akskProvider.ReloadAKSK()
			if err != nil {
				return "", fmt.Errorf("kop create route %v and reload aksk err: %v", args, err)
			}
			if len(aksk.SecurityToken) != 0 {
				c.headers["X-Ksc-Security-Token"] = aksk.SecurityToken
			}
			c.client.SetSigner(defautlServerName, c.conf.Region, aksk.AK, aksk.SK)
			data, err = c.client.Go()
			if err != nil {
				return "", fmt.Errorf("retry kop create route %v after reloading aksk err: %v", args, err)
			}
		} else {
			return "", fmt.Errorf("kop create route %v err: %v", args, err)
		}
	}

	response := new(openTypes.CreateRouteResponse)
	if err := json.Unmarshal(data, response); err != nil {
		return "", err
	}
	return response.RouteId, nil
}

func (c *RouteClient) DeleteRoute(id string) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	aksk, err := c.akskProvider.GetAKSK()
	if err != nil {
		return err
	}

	action := url.Values{
		"Action":  []string{"DeleteRoute"},
		"Version": []string{defaultVersion},
		"RouteId": []string{id},
	}
	klog.Infof("delete neutron route : %s", c.conf.NetworkEndpoint)
	if len(aksk.SecurityToken) != 0 {
		c.headers["X-Ksc-Security-Token"] = aksk.SecurityToken
	}
	c.client.SetEndpoint(c.conf.NetworkEndpoint)
	c.client.SetHeader(c.headers)
	c.client.SetUrlQuery("", action)
	c.client.SetMethod(kopHttp.DELETE)
	c.client.SetSigner(defautlServerName, c.conf.Region, aksk.AK, aksk.SK)
	if _, err := c.client.Go(); err != nil {
		if strings.Contains(err.Error(), "SecurityTokenExpired") {
			aksk, err := c.akskProvider.ReloadAKSK()
			if err != nil {
				return fmt.Errorf("kop delete route %v and reload aksk err: %v", id, err)
			}
			if len(aksk.SecurityToken) != 0 {
				c.headers["X-Ksc-Security-Token"] = aksk.SecurityToken
			}
			c.client.SetSigner(defautlServerName, c.conf.Region, aksk.AK, aksk.SK)
			_, err = c.client.Go()
			if err != nil {
				return fmt.Errorf("retry kop delete route %v after reloading aksk err: %v", id, err)
			}
		} else {
			return fmt.Errorf("kop delete route %v err: %v", id, err)
		}
	}
	return nil
}

func (c *RouteClient) ListRoutes(args *openTypes.RouteArgs) ([]openTypes.RouteSetType, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	aksk, err := c.akskProvider.GetAKSK()
	if err != nil {
		return nil, err
	}

	action := url.Values{
		"Action":           []string{"DescribeRoutes"},
		"Version":          []string{defaultVersion},
		"Filter.1.Name":    []string{"vpc-id"},
		"Filter.1.Value.1": []string{args.DomainId},
		"Filter.2.Name":    []string{"route-type"},
		"Filter.2.Value.1": []string{args.InstanceType},
	}
	klog.Infof("list neutron route : %s", c.conf.NetworkEndpoint)
	if len(aksk.SecurityToken) != 0 {
		c.headers["X-Ksc-Security-Token"] = aksk.SecurityToken
	}
	c.client.SetEndpoint(c.conf.NetworkEndpoint)
	c.client.SetHeader(c.headers)
	c.client.SetUrlQuery("", action)
	c.client.SetMethod(kopHttp.GET)
	c.client.SetSigner(defautlServerName, c.conf.Region, aksk.AK, aksk.SK)
	data, err := c.client.Go()
	if err != nil {
		if strings.Contains(err.Error(), "SecurityTokenExpired") {
			aksk, err := c.akskProvider.ReloadAKSK()
			if err != nil {
				return nil, fmt.Errorf("kop list routes %v and reload aksk err: %v", args, err)
			}
			if len(aksk.SecurityToken) != 0 {
				c.headers["X-Ksc-Security-Token"] = aksk.SecurityToken
			}
			c.client.SetSigner(defautlServerName, c.conf.Region, aksk.AK, aksk.SK)
			data, err = c.client.Go()
			if err != nil {
				return nil, fmt.Errorf("retry kop list routes %v after reloading aksk err: %v", args, err)
			}
		} else {
			return nil, fmt.Errorf("kop list routes %v err: %v", args, err)
		}
	}

	response := new(openTypes.GetRoutesResponse)
	err = json.Unmarshal(data, response)
	if err != nil {
		return nil, fmt.Errorf("json unmarshal %s err: %v", data, err)
	}
	return response.RouteSet, nil
}

func (c *RouteClient) GetRoutes(args *openTypes.RouteArgs) ([]openTypes.RouteSetType, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	aksk, err := c.akskProvider.GetAKSK()
	if err != nil {
		return nil, err
	}

	action := url.Values{
		"Action":           []string{"DescribeRoutes"},
		"Version":          []string{defaultVersion},
		"Filter.1.Name":    []string{"vpc-id"},
		"Filter.1.Value.1": []string{args.DomainId},
		"Filter.2.Name":    []string{"route-type"},
		"Filter.2.Value.1": []string{args.InstanceType},
		"Filter.3.Name":    []string{"destination-cidr-block"},
		"Filter.3.Value.1": []string{args.CidrBlock},
	}
	klog.Infof("get neutron route : %s", c.conf.NetworkEndpoint)
	if len(aksk.SecurityToken) != 0 {
		c.headers["X-Ksc-Security-Token"] = aksk.SecurityToken
	}
	c.client.SetEndpoint(c.conf.NetworkEndpoint)
	c.client.SetHeader(c.headers)
	c.client.SetUrlQuery("", action)
	c.client.SetMethod(kopHttp.GET)
	c.client.SetSigner(defautlServerName, c.conf.Region, aksk.AK, aksk.SK)
	data, err := c.client.Go()
	if err != nil {
		if strings.Contains(err.Error(), "SecurityTokenExpired") {
			aksk, err := c.akskProvider.ReloadAKSK()
			if err != nil {
				return nil, fmt.Errorf("kop get routes %v and reload aksk err: %v", args, err)
			}
			if len(aksk.SecurityToken) != 0 {
				c.headers["X-Ksc-Security-Token"] = aksk.SecurityToken
			}
			c.client.SetSigner(defautlServerName, c.conf.Region, aksk.AK, aksk.SK)
			data, err = c.client.Go()
			if err != nil {
				return nil, fmt.Errorf("retry kop get routes %v after reloading aksk err: %v", args, err)
			}
		} else {
			return nil, fmt.Errorf("kop get routes %v err: %v", args, err)
		}
	}

	response := new(openTypes.GetRoutesResponse)
	err = json.Unmarshal(data, response)
	if err != nil {
		return nil, fmt.Errorf("json unmarshal %s err: %v", data, err)
	}
	return response.RouteSet, nil
}

func (c *RouteClient) GetRoute(id string) (*openTypes.RouteSetType, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	aksk, err := c.akskProvider.GetAKSK()
	if err != nil {
		return nil, err
	}

	action := url.Values{
		"Action":    []string{"DescribeRoutes"},
		"Version":   []string{defaultVersion},
		"RouteId.1": []string{id},
	}
	klog.Infof("DescribeRoute neutron route : %s", c.conf.NetworkEndpoint)
	if len(aksk.SecurityToken) != 0 {
		c.headers["X-Ksc-Security-Token"] = aksk.SecurityToken
	}
	c.client.SetEndpoint(c.conf.NetworkEndpoint)
	c.client.SetHeader(c.headers)
	c.client.SetUrlQuery("", action)
	c.client.SetMethod(kopHttp.GET)
	c.client.SetSigner(defautlServerName, c.conf.Region, aksk.AK, aksk.SK)
	data, err := c.client.Go()
	if err != nil {
		if strings.Contains(err.Error(), "SecurityTokenExpired") {
			aksk, err := c.akskProvider.ReloadAKSK()
			if err != nil {
				return nil, fmt.Errorf("kop get route %v and reload aksk err: %v", id, err)
			}
			if len(aksk.SecurityToken) != 0 {
				c.headers["X-Ksc-Security-Token"] = aksk.SecurityToken
			}
			c.client.SetSigner(defautlServerName, c.conf.Region, aksk.AK, aksk.SK)
			data, err = c.client.Go()
			if err != nil {
				return nil, fmt.Errorf("retry kop get route %v after reloading aksk err: %v", id, err)
			}
		} else {
			return nil, fmt.Errorf("kop get route %v err: %v", id, err)
		}
	}
	response := new(openTypes.DescribeRouteResponse)
	err = json.Unmarshal([]byte(data), response)
	if err != nil {
		return nil, fmt.Errorf("json unmarshal %s err: %v", data, err)
	}
	return &response.RouteSet[0], nil
}

// WaitForAllRouteEntriesAvailable waits for all route entries to Available status
func (c *RouteClient) WaitForAllRouteEntriesAvailable(vrouterId string, timeout int) error {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	for {
		success := true
		route, err := c.GetRoute(vrouterId)
		if err != nil || len(route.RouteId) == 0 {
			success = false
		}

		if success {
			break
		} else {
			timeout = timeout - DefaultWaitForInterval
			if timeout <= 0 {
				return utils.GetClientErrorFromString("Timeout", "")
			}
			time.Sleep(DefaultWaitForInterval * time.Second)
		}
	}
	return nil
}

package nova

import (
	"context"
	"encoding/json"
	"fmt"
	"k8s.io/klog"
	"net/url"
	"strings"
	"sync"

	prvd "ezone.ksyun.com/code/kce/aksk-provider"
	kopHttp "ezone.ksyun.com/code/kce/vpc-route-controller/pkg/http"
	"ezone.ksyun.com/code/kce/vpc-route-controller/pkg/ksyun/openstack_client/config"
	openTypes "ezone.ksyun.com/code/kce/vpc-route-controller/pkg/ksyun/openstack_client/types"
)

const (
	defaultVersion    = "2016-03-04"
	defaultServerName = "kec"
)

type ServerClient struct {
	conf         *config.Config
	client       *kopHttp.KopClient
	tenantID     string
	headers      map[string]string
	lock         sync.Mutex
	akskProvider prvd.AKSKProvider
}

func NewServerClient(ctx context.Context, conf *config.Config) (*ServerClient, error) {
	if len(conf.NetworkEndpoint) == 0 {
		conf.NetworkEndpoint = config.DefaultNetworkEndpoint
	}
	dataClient := kopHttp.NewKopClient(ctx)

	headers := make(map[string]string)
	headers["User-Agent"] = "vpc-route-controller"
	headers["Content-Type"] = "application/json"
	headers["Accept"] = "application/json"
	//headers["X-Auth-User-Tag"] = "docker"

	serverClient := &ServerClient{
		conf:    conf,
		headers: headers,
		client:  dataClient,
		//tenantID: conf.TenantID,
		akskProvider: conf.AkskProvider,
	}

	return serverClient, nil
}

func (n *ServerClient) DescribeInstances(args *openTypes.InstanceArgs) (*openTypes.Instance, error) {
	n.lock.Lock()
	defer n.lock.Unlock()

	aksk, err := n.akskProvider.GetAKSK()
	if err != nil {
		return nil, err
	}

	action := url.Values{
		"Action":           []string{"DescribeInstances"},
		"Version":          []string{defaultVersion},
		"Filter.1.Name":    []string{"vpc-id"},
		"Filter.1.Value.1": []string{args.DomainId},
		"Filter.2.Name":    []string{"private-ip-address"},
		"Filter.2.Value.1": []string{args.InstancePrivateIP},
	}
	klog.Infof("get nova instance: %s", n.conf.NetworkEndpoint)
	if len(aksk.SecurityToken) != 0 {
		n.headers["X-Ksc-Security-Token"] = aksk.SecurityToken
	}
	n.client.SetEndpoint(n.conf.NetworkEndpoint)
	n.client.SetHeader(n.headers)
	n.client.SetUrlQuery("", action)
	n.client.SetMethod(kopHttp.GET)
	n.client.SetSigner(defaultServerName, n.conf.Region, aksk.AK, aksk.SK)
	data, err := n.client.Go()
	if err != nil {
		if strings.Contains(err.Error(), "SecurityTokenExpired") {
			aksk, err := n.akskProvider.ReloadAKSK()
			if err != nil {
				return nil, fmt.Errorf("kop get instances %v and reload aksk err: %v", args, err)
			}
			if len(aksk.SecurityToken) != 0 {
				n.headers["X-Ksc-Security-Token"] = aksk.SecurityToken
			}
			n.client.SetSigner(defaultServerName, n.conf.Region, aksk.AK, aksk.SK)
			data, err = n.client.Go()
			if err != nil {
				return nil, fmt.Errorf("retry kop get instances %v after reloading aksk err: %v", args, err)
			}
		} else {
			return nil, fmt.Errorf("kop get instances %v err: %v", args, err)
		}
	}

	response := new(openTypes.GetInstancesResponse)
	err = json.Unmarshal(data, response)
	if err != nil {
		return nil, fmt.Errorf("json unmarshal %s err: %v", data, err)
	}
	return &(response.InstancesSet[0]), nil
}

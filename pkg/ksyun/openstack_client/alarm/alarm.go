package alarm

import (
	"context"
	"fmt"
	"k8s.io/klog"
	"net/url"
	"strings"
	"sync"

	kopHttp "ezone.ksyun.com/ezone/kce/vpc-route-controller/pkg/http"
	"ezone.ksyun.com/ezone/kce/vpc-route-controller/pkg/ksyun/openstack_client/config"
	openTypes "ezone.ksyun.com/ezone/kce/vpc-route-controller/pkg/ksyun/openstack_client/types"
	//prvd "github.com/kingsoftcloud/aksk-provider"
)

const (
	defaultVersion         = "2019-01-30"
	defaultServerName      = "alarm"
	DefaultTimeout         = 60
	DefaultWaitForInterval = 5

	DefaultProduct = "CONTAINER_NETWORK"
)

var (
	AKForAlarm string
	SKForAlarm string
)

type AlarmClient struct {
	conf     *config.Config
	client   *kopHttp.KopClient
	tenantID string
	headers  map[string]string
	lock     sync.Mutex
	//akskProvider prvd.AKSKProvider
	ak string
	sk string
}

func NewAlarmClient(ctx context.Context, conf *config.Config) *AlarmClient {
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
	//headers["X-Auth-User-Tag"] = "docker"

	alarmClient := &AlarmClient{
		conf:    conf,
		headers: headers,
		client:  dataClient,
		//tenantID: conf.TenantID,
		//akskProvider: conf.AkskProvider,
		ak: AKForAlarm,
		sk: SKForAlarm,
	}

	return alarmClient
}

func (c *AlarmClient) CreateAlarm(message openTypes.AlarmArgs) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	/*aksk, err := c.akskProvider.GetAKSK()
	if err != nil {
		return err
	}*/

	action := url.Values{
		"Action":  []string{"AlarmReceptor"},
		"Version": []string{defaultVersion},
	}
	klog.Infof("create alarm : %s", c.conf.NetworkEndpoint)
	/*if len(aksk.SecurityToken) != 0 {
		c.headers["X-Ksc-Security-Token"] = aksk.SecurityToken
	}*/

	c.client.SetEndpoint(c.conf.NetworkEndpoint)
	c.client.SetHeader(c.headers)
	c.client.SetBody(message)
	c.client.SetUrlQuery("", action)
	c.client.SetMethod(kopHttp.POST)
	c.client.SetSigner(defaultServerName, c.conf.Region, AKForAlarm, SKForAlarm)
	_, err := c.client.Go()
	if err != nil {
		if strings.Contains(err.Error(), "SecurityTokenExpired") {
			/*aksk, err := c.akskProvider.ReloadAKSK()
			if err != nil {
				return fmt.Errorf("kop create alarm %v and reload aksk err: %v", body, err)
			}
			if len(aksk.SecurityToken) != 0 {
				c.headers["X-Ksc-Security-Token"] = aksk.SecurityToken
			}*/
			c.client.SetSigner(defaultServerName, c.conf.Region, AKForAlarm, SKForAlarm)
			_, err = c.client.Go()
			if err != nil {
				return fmt.Errorf("retry kop create alarm %v after reloading aksk err: %v", message, err)
			}
		} else {
			return fmt.Errorf("kop create alarm %v err: %v", message, err)
		}
	}

	return nil
}

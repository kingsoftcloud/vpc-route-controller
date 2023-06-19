package aksk

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
	"newgit.op.ksyun.com/kce/vpc-route-controller/pkg/aksk/config/types"
	"newgit.op.ksyun.com/kce/vpc-route-controller/pkg/ksyun/openstack_client/config"
	"newgit.op.ksyun.com/kce/vpc-route-controller/pkg/util"
)

const (
	AKSKConfigMapType = "configmap"
	AKSKSecretType    = "secret"

	DefaultAKSKNamespace = "kube-system"
	DefaultAKSKConfigMap = "user-temp-aksk"
	DefaultAKSKSecret    = "kce-security-token"
)

var (
	DefaultAESKey string
)

// GetAKSK get aksk from k8s or local file
func GetAKSK(kubeconfig string, aksk *types.AKSK, c *config.Config) error {
	ctx := context.Background()
	switch c.AkskType {

	case AKSKConfigMapType:
		return GetAKSKConfigMap(ctx, kubeconfig, aksk, c)
	case AKSKSecretType:
		return GetAKSKSecret(ctx, kubeconfig, aksk, c)
	default:
		if c.AK != "" && c.SK != "" {
			aksk.SetAKSK(c.AK, c.SK)
			if c.SecurityToken != "" {
				aksk.SetSecurityToken(c.SecurityToken)
			}
			return nil
		}

		if os.Getenv("AK") != "" && os.Getenv("SK") != "" {
			aksk.SetAKSK(os.Getenv("AK"), os.Getenv("SK"))
			if os.Getenv("SecurityToken") != "" {
				aksk.SetSecurityToken(os.Getenv("SecurityToken"))
			}
			return nil
		}
		return fmt.Errorf("no available aksk.")
	}
}

// GetAKSKK8S get aksk from k8s
func GetAKSKConfigMap(ctx context.Context, kubeconfig string, aksk *types.AKSK, c *config.Config) error {
	// set up the client config
	var clientConfig *rest.Config
	var err error
	if len(kubeconfig) > 0 {
		loadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig}
		loader := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{})

		clientConfig, err = loader.ClientConfig()
	} else {
		clientConfig, err = rest.InClusterConfig()
	}
	if err != nil {
		return fmt.Errorf("unable to construct lister client config: %v", err)
	}

	// set up the informers
	kubeClient, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		return fmt.Errorf("unable to construct lister client: %v", err)
	}

	if c.AkskName == "" {
		c.AkskName = DefaultAKSKConfigMap
	}
	if c.AkskNamespace == "" {
		c.AkskNamespace = DefaultAKSKNamespace
	}
	cm, err := kubeClient.CoreV1().ConfigMaps(c.AkskNamespace).Get(ctx, DefaultAKSKConfigMap, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get config maps: %s", err)
	}

	// set ak sk
	if err := aksk.SetAKSK(cm.Data["ak"], cm.Data["sk"]); err != nil {
		return err
	}

	// set securityToken
	if err := aksk.SetSecurityToken(cm.Data["securityToken"]); err != nil {
		return err
	}

	// set region
	if err := aksk.SetRegion(cm.Data["region"]); err != nil {
		klog.Errorf("not found region from k8s user-temp-aksk configmap")
	}
	return nil
}

// GetAKSKK8S get aksk from k8s
func GetAKSKSecret(ctx context.Context, kubeconfig string, aksk *types.AKSK, c *config.Config) error {
	// set up the client config
	var clientConfig *rest.Config
	var err error
	if len(kubeconfig) > 0 {
		loadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig}
		loader := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{})

		clientConfig, err = loader.ClientConfig()
	} else {
		clientConfig, err = rest.InClusterConfig()
	}
	if err != nil {
		return fmt.Errorf("unable to construct lister client config: %v", err)
	}

	// set up the informers
	kubeClient, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		return fmt.Errorf("unable to construct lister client: %v", err)
	}

	sc, err := kubeClient.CoreV1().Secrets(DefaultAKSKNamespace).Get(ctx, DefaultAKSKSecret, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get config maps: %s", err)
	}

	if c.Key == "" {
		c.Key = DefaultAESKey
	}
	data := util.AesDecrypt(string(sc.Data["kce-security-token"]), c.Key)
	if err = json.Unmarshal([]byte(data), aksk); err != nil {
		return fmt.Errorf("json unmarshal macvlan-conf error: %v", err)
	}

	return nil
}

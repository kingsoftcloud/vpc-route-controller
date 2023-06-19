package types

import (
	"fmt"

	"k8s.io/klog"
)

const (
	DefautlServerName = "kce"
	DefautlVersion    = "2018-03-14"
	DefaultAKSKFile   = "/opt/app-agent/arrangement/clusterinfo"
)

type Domain struct {
	Domain string `json:"domain"`
	Ip     string `json:"ip"`
}

type AKSK struct {
	AK            string `json:"ak"`
	SK            string `json:"sk"`
	Expired_at    string `json:"expired_at"`
	SecurityToken string `json:"securityToken"`
	Region        string `json:"region"`
	ServerName    string `json:"server_name"`
	Version       string `json:"version"`
	KceApiServer  string `json:"kceapi_server"`
}

func (k *AKSK) SetAKSK(ak, sk string) error {
	if len(ak) == 0 || len(sk) == 0 {
		return fmt.Errorf("not set kop ak or sk")
	}

	k.AK = ak
	k.SK = sk
	return nil
}

func (k *AKSK) SetSecurityToken(securityToken string) error {
	if len(securityToken) == 0 {
		klog.Errorf("not found securityToken from k8s user-temp-aksk configmap")
	}

	k.SecurityToken = securityToken
	return nil
}

func (k *AKSK) SetRegion(region string) error {
	if len(region) == 0 {
		return fmt.Errorf("not found region")
	}
	k.Region = region
	return nil
}

func (k *AKSK) SetServerName(name string) error {
	if len(name) == 0 {
		k.ServerName = DefautlServerName
	}
	k.ServerName = name
	return nil
}

func (k *AKSK) SetVersion(version string) error {
	if len(version) == 0 {
		k.Version = DefautlVersion
	}
	k.Version = version
	return nil
}

func (k *AKSK) GetAKSK() (string, string, error) {
	if len(k.AK) == 0 {
		return "", "", fmt.Errorf("not found cluster ak")
	}
	if len(k.SK) == 0 {
		return "", "", fmt.Errorf("not found cluster sk")
	}

	return k.AK, k.SK, nil
}

func (k *AKSK) GetSecurityToken() (string, error) {
	if len(k.SecurityToken) == 0 {
		return "", fmt.Errorf("not found cluster SecurityToken")
	}
	return k.SecurityToken, nil
}

func (k *AKSK) GetRegion() (string, error) {
	if len(k.Region) == 0 {
		return "", fmt.Errorf("not found kop region")
	}

	return k.Region, nil
}

func (k *AKSK) GetServerName() string {
	if len(k.ServerName) == 0 {
		k.ServerName = DefautlServerName
	}

	return k.ServerName
}

func (k *AKSK) GetVersion() string {
	if len(k.Version) == 0 {
		k.Version = DefautlVersion
	}

	return k.Version
}

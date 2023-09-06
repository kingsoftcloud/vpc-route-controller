package config

import (
	prvd "github.com/kingsoftcloud/aksk-provider"
	"k8s.io/apimachinery/pkg/util/wait"
)

var (
	DefaultNetworkEndpoint = "http://internal.api.ksyun.com"
)

type Config struct {
	// Kop endpoint address
	NetworkEndpoint string `json:"network_endpoint"`
	// k8s cluster uuid info from appengine
	ClusterUUID string `json:"cluster_uuid"`
	// nova tenant id
	TenantID string `json:"tenant_id"`
	// nova user id
	UserID string `json:"user_id"`
	// nova token id
	Token string `json:"token"`
	// vpc id
	VpcID string `json:"vpc_id"`
	// nova region
	Region string `json:"region"`
	// kube config file
	Kubeconfig string `json:"kubeconfig"`
	// auth open api
	Auth bool `json:"auth"`
	// aksk type
	AkskType string `json:"aksk_type"`
	// X-Request-ID
	AK            string            `json:"ak"`
	SK            string            `json:"sk"`
	SecurityToken string            `json:"securityToken"`
	AkskProvider  prvd.AKSKProvider `json:"aksk_provider"`
	AkskFilePath  string            `json:"aksk_file_path"`
	Encrypt       bool              `json:"encrypt"`

	InstanceIdFrom string `json:"instance_id_from"`

	RequestIdPrefix string `json:"requestIdPrefix"`

	// http client headers
	Headers map[string]string `json:"headers"`

	// http client backoff
	Backoff *wait.Backoff `json:"backoff"`

	AlarmEnabled bool `json: "alarm_enabled"`
}

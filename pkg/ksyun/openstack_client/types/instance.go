package types

type NovaVif struct {
	IsPrimary  bool   `json:"is_primary"`
	VpcUuid    string `json:"vpc_uuid"`
	VpcIp      string `json:"vpc_ip"`
	VifUuid    string `json:"vif_uuid"`
	MacAddress string `json:"mac_address"`
	VpcVnetId  string `json:"vpc_vnet_id"`
	VpcSgId    string `json:"vpc_sg_id"`

	/*"is_primary":true,
	          "vpc_uuid":"b13a9c33-5c09-4235-90d3-1b7a4b7bc69f",
	          "vpc_ip":"10.20.20.2",
	          "vif_uuid":"5b08529f-36d5-48fa-a144-cf300d30ea47",
	          "mac_address":"fa:16:3e:22:9f:fe",
	          "vpc_vnet_id":"791e6d71-4865-48a5-9d2f-b843e9aa8eb0",
	          "vpc_sg_ids":[
	          "11eb5fff-c86f-459c-8e1e-255109638779"
	  ],
	          "vpc_sg_id":"11eb5fff-c86f-459c-8e1e-255109638779"
	*/
}

type Instance struct {
	/*
	   "id": "c7986596-d741-4192-b79e-721cf8380635",
	   "hostname"： "hostname",
	   "hostId": "4b9f9ea41eb2b47ee4c6df601d731ad40696f886e7bf3320fe574680",
	   "OS-EXT-SRV-ATTR:host": "sh01-cp-compute002202.sh01.ksyun.com",
	   "key_name": null,
	   "OS-EXT-SRV-ATTR:hypervisor_hostname": "sh01-cp-compute002202.sh01.ksyun.com",
	   "name": "test_eph_create_vm",
	*/
	Id   string `json:"InstanceId"`
	Name string `json:"InstanceName"`
	/*Status             string    `json:"OS-EXT-STS:vm_state"`
	  Host               string    `json:"OS-EXT-SRV-ATTR:host"`
	  HypervisorHostname string    `json:"OS-EXT-SRV-ATTR:hypervisor_hostname"`
	  UserId             string    `json:"user_id"`
	  TenantId           string    `json:"tenant_id"`
	  Vifs               []NovaVif `json:"vifs"`*/
}

type InstanceArgs struct {
	DomainId          string `json:"domain_id"`
	InstanceId        string `json:"instance_id"`
	InstanceType      string `json:"instance_type"`
	InstancePrivateIP string `json:"instance_private_ip"`
}

type GetInstancesResponse struct {
	Response
	InstancesSet []Instance `json:"InstancesSet"`
}

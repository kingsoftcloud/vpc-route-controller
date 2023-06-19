package types

import "fmt"

type Response struct {
	RequestId string `json:"RequestId"`
}

type Vpc struct {
	VpcId                 string `json:"VpcId"`
	VpcName               string `json:"VpcName"`
	CidrBlock             string `json:"CidrBlock"`
	IsDefault             bool   `json:"IsDefault"`
	ProvidedIpv6CidrBlock bool   `json:"ProvidedIpv6CidrBlock"`
}

type VpcResp struct {
	Response
	Vpcs []Vpc `json:"VpcSet"`
}

type Domain struct {
	Name string `json:"name"`
	Id   string `json:"id"`
	Ip   string `json:"ip"`
	Mask int    `json:"mask"`
}

type DomainType struct {
	Domain Domain `json:"domain"`
}

func (d *DomainType) ConVpc() *Vpc {
	return &Vpc{
		VpcId:     d.Domain.Id,
		VpcName:   d.Domain.Name,
		CidrBlock: fmt.Sprintf("%s/%d", d.Domain.Ip, d.Domain.Mask),
	}
}

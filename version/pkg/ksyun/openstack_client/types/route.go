package types

import (
	"fmt"
)

type NextHop struct {
	GatewayId string `json:"GatewayId"`
}

type RouteSetType struct {
	RouteId         string    `json:"RouteId"`
	VpcId           string    `json:"VpcId"`
	RouteType       string    `json:"RouteType"`
	DestinationCIDR string    `json:"DestinationCidrBlock"`
	NextHopset      []NextHop `json:"NextHopset"`
}

type RouteId struct {
	Id string `json:"id"`
}

type RouteBase struct {
	DomainId     string `json:"domain_id"`
	InstanceId   string `json:"instance_id"`
	InstanceType string `json:"instance_type"`
	Ip           string `json:"ip"`
	Mask         int    `json:"mask"`
}

type Route struct {
	RouteId
	RouteBase
}

type RouteResp struct {
	Route Route `json:"route"`
}

type RouteIdResp struct {
	Route RouteId `json:"route"`
}

type RouteBaseResp struct {
	Route RouteBase `json:"route"`
}

func (r *RouteResp) ConRoute() *RouteSetType {

	return &RouteSetType{
		RouteId:         r.Route.Id,
		VpcId:           r.Route.DomainId,
		RouteType:       r.Route.InstanceType,
		DestinationCIDR: fmt.Sprintf("%s/%d", r.Route.Ip, r.Route.Mask),
	}
}

type RoutesResp struct {
	Routes []Route `json:"routes"`
}

func (r *RoutesResp) ConRoutes() []RouteSetType {

	rss := make([]RouteSetType, 0)
	for _, route := range r.Routes {
		rs := RouteSetType{
			RouteId:         route.Id,
			VpcId:           route.DomainId,
			RouteType:       route.InstanceType,
			DestinationCIDR: fmt.Sprintf("%s/%d", route.Ip, route.Mask),
		}

		rss = append(rss, rs)
	}

	return rss
}

type RouteArgs struct {
	DomainId     string `json:"domain_id"`
	InstanceId   string `json:"instance_id"`
	InstanceType string `json:"instance_type"`
	CidrBlock    string `json:"cidrblock"`
}

type CreateRouteResponse struct {
	Response
	RouteId string `json:"RouteId"`
}

type DelRouteResponse struct {
	Response
	Return bool `json:"Return"`
}

type GetRoutesResponse struct {
	Response
	RouteSet []RouteSetType `json:"RouteSet"`
}

type DescribeRouteResponse struct {
	Response
	RouteSet []RouteSetType `json:"RouteSet"`
}

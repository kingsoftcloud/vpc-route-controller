package utils

import (
	"fmt"
	openTypes "newgit.op.ksyun.com/kce/vpc-route-controller/pkg/ksyun/openstack_client/types"
)

type ErrorResponse struct {
	Detail  string `json:"detail"`
	Type    string `json:"type"`
	Message string `json:"message"`
}

// An Error represents a custom error for Aliyun API failure response
type Error struct {
	openTypes.Response
	StatusCode    int           //Status Code of HTTP Response
	ErrorResponse ErrorResponse `json:"NeutronError"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("Ksyun API Error: RequestId: %s Status Code: %d Type: %s Message: %s Detail: %s", e.RequestId, e.StatusCode, e.ErrorResponse.Type, e.ErrorResponse.Message, e.ErrorResponse.Detail)
}

func GetClientErrorFromString(str string, req string) error {
	errors := ErrorResponse{
		Type:    "vpcRouteControllerFailure",
		Message: str,
	}
	rErrors := Error{
		ErrorResponse: errors,
		StatusCode:    -1,
	}
	rErrors.RequestId = req
	return &rErrors
}

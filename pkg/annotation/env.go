package annotation

import (
	"fmt"
	"os"
)

const (
	NodeName = "NODE_NAME"
)

func GetNodeName() (string, error) {
	val := os.Getenv(NodeName)
	if val == "" {
		return "", fmt.Errorf("unable to get the name of the node deployed by the pod,"+
			" check whether the downward api in the deployment file is used incorrectly, env %s lack", NodeName)
	}
	return val, nil
}

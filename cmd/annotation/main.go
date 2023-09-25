package main

import (
	"ezone.ksyun.com/ezone/kce/vpc-route-controller/pkg/annotation"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"os"
	"time"
)

const (
	checkInterval = 20 * time.Second
)

func main() {
	klog.Infoln("Start node annotation check")
	if err := wait.PollImmediateInfinite(checkInterval, Run); err != nil {
		klog.Infoln("wait PollImmediateInfinite err:", err)
		os.Exit(1)
	}
	klog.Infoln("Check task completed, program exit")
}

func Run() (bool, error) {
	// If the node name is not passed through the download API,
	// it is considered a startup error and an error is returned,
	// os.Exit(1)
	nodeName, err := annotation.GetNodeName()
	if err != nil {
		klog.Infoln(err)
		return false, err
	}
	klog.Infoln("Current node name:", nodeName)
	// loop until successful or serious error
	if err := annotation.EnsureNodeInstanceId(nodeName); err != nil {
		klog.Infof("failed to EnsureNodeInstanceId, %v", err)
		return false, nil
	}
	return true, nil
}

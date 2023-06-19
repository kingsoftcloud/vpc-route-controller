package main

import (
	"fmt"
	"os"
	"runtime"

	"k8s.io/client-go/rest"
	"k8s.io/klog/v2/klogr"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	ctrlCfg "newgit.op.ksyun.com/kce/vpc-route-controller/pkg/config"
	"newgit.op.ksyun.com/kce/vpc-route-controller/pkg/controller"
	"newgit.op.ksyun.com/kce/vpc-route-controller/pkg/ksyun"
	"newgit.op.ksyun.com/kce/vpc-route-controller/version"
)

var log = klogr.New()

func printVersion() {
	log.Info(fmt.Sprintf("Cloud Controller Manager Version: %s, git commit: %s, build date: %s",
		version.Version, version.GitCommit, version.BuildDate))
	log.Info(fmt.Sprintf("Go version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
}

func main() {
	err := ctrlCfg.ControllerCFG.LoadControllerConfig()
	if err != nil {
		log.Error(err, "unable to load controller config")
		os.Exit(1)
	}

	printVersion()

	// Get a config to talk to the api-server
	cfg := config.GetConfigOrDie()
	cfg.QPS = ctrlCfg.ControllerCFG.RuntimeConfig.QPS
	cfg.Burst = ctrlCfg.ControllerCFG.RuntimeConfig.Burst
	cfg.ContentConfig = rest.ContentConfig{
		ContentType: "application/vnd.kubernetes.protobuf",
	}

	mgr, err := manager.New(cfg, ctrlCfg.BuildRuntimeOptions(ctrlCfg.ControllerCFG.RuntimeConfig))
	if err != nil {
		log.Error(err, "failed to create manager")
		os.Exit(1)
	}

	conf, err := ksyun.GetNeutronConfig()
	if err != nil {
		log.Error(err, "failed to get neutron config")
		os.Exit(1)
	}

	log.Info("Registering Components.")
	if err := controller.AddToManager(mgr, ctrlCfg.ControllerCFG.Controllers, conf); err != nil {
		log.Error(err, "add controller: %s", err.Error())
		os.Exit(1)
	} else {
		log.Info(fmt.Sprintf("Loaded controllers: %v", ctrlCfg.ControllerCFG.Controllers))
	}

	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Error(err, "Manager exited non-zero: %s", err.Error())
		os.Exit(1)
	}
}

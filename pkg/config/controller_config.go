package config

import (
	"flag"
	"github.com/spf13/pflag"
	"k8s.io/cloud-provider/config"
	"os"
	"time"
)

const (
	flagControllers                  = "controllers"
	flagRouteReconciliationPeriod    = "route-reconciliation-period"
	defaultRouteReconciliationPeriod = 5 * time.Minute
)

var ControllerCFG = &ControllerConfig{}

// Flag stores the configuration for global usage
type ControllerConfig struct {
	config.KubeCloudSharedConfiguration
	Controllers []string
	LogLevel    int

	RuntimeConfig RuntimeConfig
}

func (cfg *ControllerConfig) BindFlags(fs *pflag.FlagSet) {
	fs.StringSliceVar(&cfg.Controllers, flagControllers, []string{"route"}, "A list of controllers to enable.")
	fs.DurationVar(&cfg.RouteReconciliationPeriod.Duration, flagRouteReconciliationPeriod, defaultRouteReconciliationPeriod,
		"The period for reconciling routes created for nodes by cloud provider. The minimum value is 1 minute")
	cfg.RuntimeConfig.BindFlags(fs)
}

// Validate the controller configuration
func (cfg *ControllerConfig) Validate() error {
	if cfg.RouteReconciliationPeriod.Duration < 1*time.Minute {
		cfg.RouteReconciliationPeriod.Duration = 1 * time.Minute
	}
	return nil
}

func (cfg *ControllerConfig) LoadControllerConfig() error {
	fs := pflag.NewFlagSet("", pflag.ExitOnError)
	fs.AddGoFlagSet(flag.CommandLine)
	cfg.BindFlags(fs)

	if err := fs.Parse(os.Args); err != nil {
		return err
	}

	if err := cfg.Validate(); err != nil {
		return err
	}

	return nil
}

package controller

import (
	"fmt"
	"newgit.op.ksyun.com/kce/vpc-route-controller/pkg/controller/route"
	"newgit.op.ksyun.com/kce/vpc-route-controller/pkg/ksyun/openstack_client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func init() {
	controllerMap = map[string]func(manager.Manager, *config.Config) error{
		"route": route.Add,
	}
}

// controllerMap is a list of functions to add all Controllers to the Manager.
var controllerMap map[string]func(manager.Manager, *config.Config) error

// AddToManager adds selected Controllers to the Manager.
func AddToManager(m manager.Manager, enabledControllers []string, c *config.Config) error {
	for _, cont := range enabledControllers {
		if f, ok := controllerMap[cont]; ok {
			if err := f(m, c); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("cannot find controller %s", cont)
		}
	}

	return nil
}

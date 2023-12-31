package controller

import (
	"ezone.ksyun.com/ezone/kce/vpc-route-controller/pkg/controller/route"
	"fmt"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func init() {
	controllerMap = map[string]func(manager.Manager) error{
		"route": route.Add,
	}
}

// controllerMap is a list of functions to add all Controllers to the Manager.
var controllerMap map[string]func(manager.Manager) error

// AddToManager adds selected Controllers to the Manager.
func AddToManager(m manager.Manager, enabledControllers []string) error {
	for _, cont := range enabledControllers {
		if f, ok := controllerMap[cont]; ok {
			if err := f(m); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("cannot find controller %s", cont)
		}
	}

	return nil
}

package controllers

import (
	"sync"
)

var (
	controllersMux sync.Mutex
	controllers    []Controller
)

func registerController(controller Controller) {
	controllersMux.Lock()
	defer controllersMux.Unlock()
	controllers = append(controllers, controller)
}

func GetControllers() []Controller {
	return controllers
}

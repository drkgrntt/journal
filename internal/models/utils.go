package models

import (
	"sync"
)

var (
	modelsMux sync.Mutex
	models    []interface{}
)

func registerModel(model interface{}) {
	modelsMux.Lock()
	defer modelsMux.Unlock()
	models = append(models, model)
}

func GetModels() []interface{} {
	return models
}

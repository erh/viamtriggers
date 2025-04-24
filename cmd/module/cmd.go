package main

import (
	"go.viam.com/rdk/module"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/services/generic"

	"github.com/erh/viamtriggers"
)

func main() {
	module.ModularMain(
		resource.APIModel{generic.API, viamtriggers.TriggersMovementMotion},
		resource.APIModel{generic.API, viamtriggers.SunsetLightModel},
	)
}

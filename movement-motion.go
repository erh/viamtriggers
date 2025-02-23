package viamtriggers

import (
	"context"
	"fmt"
	"math"
	"time"

	"go.viam.com/rdk/components/movementsensor"
	"go.viam.com/rdk/components/switch"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/services/generic"
)

var TriggersMovementMotion = family.WithModel("movement-motion")

func init() {
	resource.RegisterService(generic.API, TriggersMovementMotion,
		resource.Registration[resource.Resource, *Config]{
			Constructor: newMovementMotion,
		},
	)
}

type PositionConfig struct {
	Motion uint32
	Idle   uint32
}

type Config struct {
	Sensor      string
	Switch      string
	Position    *PositionConfig
	IdleMinutes float64 `json:"idle-minutes"`
	Threshold   float64
}

func (cfg *Config) Validate(path string) ([]string, error) {
	if cfg.Sensor == "" {
		return nil, fmt.Errorf("need a sensor")
	}
	if cfg.Switch == "" {
		return nil, fmt.Errorf("need a sensor")
	}
	return []string{cfg.Sensor, cfg.Switch}, nil
}

func (cfg *Config) threshold() float64 {
	if cfg.Threshold > 0 {
		return cfg.Threshold
	}
	return 10
}

func (cfg *Config) positionMotion() uint32 {
	if cfg.Position != nil {
		return cfg.Position.Motion
	}
	return 1
}

func (cfg *Config) positionIdle() uint32 {
	if cfg.Position != nil {
		return cfg.Position.Idle
	}
	return 0
}

type triggerMovementMotion struct {
	resource.AlwaysRebuild

	name   resource.Name
	logger logging.Logger
	conf   *Config

	ms        movementsensor.MovementSensor
	theSwitch toggleswitch.Switch

	backgroundContext context.Context
	backgroundCancel  context.CancelFunc
}

func newMovementMotion(ctx context.Context, deps resource.Dependencies, rawConf resource.Config, logger logging.Logger) (resource.Resource, error) {

	conf, err := resource.NativeConfig[*Config](rawConf)
	if err != nil {
		return nil, err
	}

	return NewMovementMotion(ctx, deps, rawConf.ResourceName(), conf, logger)
}

func NewMovementMotion(ctx context.Context, deps resource.Dependencies, name resource.Name, conf *Config, logger logging.Logger) (resource.Resource, error) {

	mm := &triggerMovementMotion{
		name:   name,
		conf:   conf,
		logger: logger,
	}

	mm.backgroundContext = context.Background()
	mm.backgroundContext, mm.backgroundCancel = context.WithCancel(mm.backgroundContext)

	r, err := deps.Lookup(movementsensor.Named(conf.Sensor))
	if err != nil {
		return nil, err
	}

	mm.ms = r.(movementsensor.MovementSensor)

	r, err = deps.Lookup(toggleswitch.Named(conf.Switch))
	if err != nil {
		return nil, err
	}

	mm.theSwitch = r.(toggleswitch.Switch)

	go mm.run()

	return mm, nil
}

func (mm *triggerMovementMotion) run() {
	lastMotion := time.UnixMilli(0)
	triggered := false

	for mm.backgroundContext.Err() == nil {
		start := time.Now()

		av, err := mm.ms.AngularVelocity(mm.backgroundContext, nil)
		if err != nil {
			mm.logger.Warnf("error getting AngularVelocity: %v", err)
			time.Sleep(time.Minute)
			continue
		}

		max := math.Abs(av.X)
		max = math.Max(max, math.Abs(av.Y))
		max = math.Max(max, math.Abs(av.Z))

		mm.logger.Debugf("angularvelocity %v", av)
		mm.logger.Debugf("max: %f", max)

		if max >= mm.conf.threshold() {
			mm.logger.Infof("movement detecteded max: %f, setting to: %d", max, mm.conf.positionMotion())
			err := mm.theSwitch.SetPosition(mm.backgroundContext, mm.conf.positionMotion(), nil)
			if err != nil {
				mm.logger.Warnf("error setting position after motion %v", err)
			} else if mm.conf.IdleMinutes > 0 {
				lastMotion = time.Now()
				triggered = true
			}
		}

		if mm.conf.IdleMinutes > 0 && time.Since(lastMotion) > time.Duration(mm.conf.IdleMinutes*float64(time.Minute)) {
			if triggered {
				mm.logger.Infof("idling after motion")
			} else {
				mm.logger.Debugf("idling from timeout")
			}

			err := mm.theSwitch.SetPosition(mm.backgroundContext, mm.conf.positionIdle(), nil)
			if err != nil {
				mm.logger.Warnf("error setting position for idle %v", err)
			} else {
				lastMotion = time.Now()
			}
		}

		sleepTime := time.Millisecond - time.Since(start)
		mm.logger.Debugf("sleeping %v", sleepTime)
		time.Sleep(sleepTime)
	}
}

func (mm *triggerMovementMotion) Name() resource.Name {
	return mm.name
}

func (mm *triggerMovementMotion) DoCommand(ctx context.Context, cmd map[string]interface{}) (map[string]interface{}, error) {
	return nil, nil
}

func (mm *triggerMovementMotion) Close(ctx context.Context) error {
	if mm.backgroundCancel != nil {
		mm.backgroundCancel()
		mm.backgroundCancel = nil
	}
	return nil
}

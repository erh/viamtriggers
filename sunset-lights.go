package viamtriggers

import (
	"context"
	"fmt"
	"time"

	"github.com/nathan-osman/go-sunrise"

	"go.viam.com/rdk/components/switch"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/services/generic"
)

var SunsetLightModel = family.WithModel("sunset-lights")

func init() {
	resource.RegisterService(generic.API, SunsetLightModel,
		resource.Registration[resource.Resource, *SunsetLightConfig]{
			Constructor: newSunsetLight,
		},
	)
}

type SunsetLightConfig struct {
	Lat    float64
	Lng    float64
	Switch string
}

func (cfg *SunsetLightConfig) Validate(path string) ([]string, error) {
	if cfg.Switch == "" {
		return nil, fmt.Errorf("need a sensor")
	}
	if cfg.Lat == 0 || cfg.Lng == 0 {
		return nil, fmt.Errorf("need lat and lng")
	}
	return []string{cfg.Switch}, nil
}

type sunsetLight struct {
	resource.AlwaysRebuild

	name   resource.Name
	logger logging.Logger
	conf   *SunsetLightConfig

	theSwitch toggleswitch.Switch

	backgroundContext context.Context
	backgroundCancel  context.CancelFunc
}

func newSunsetLight(ctx context.Context, deps resource.Dependencies, rawConf resource.Config, logger logging.Logger) (resource.Resource, error) {

	conf, err := resource.NativeConfig[*SunsetLightConfig](rawConf)
	if err != nil {
		return nil, err
	}

	return NewSunsetLight(ctx, deps, rawConf.ResourceName(), conf, logger)
}

func NewSunsetLight(ctx context.Context, deps resource.Dependencies, name resource.Name, conf *SunsetLightConfig, logger logging.Logger) (resource.Resource, error) {

	sl := &sunsetLight{
		name:   name,
		conf:   conf,
		logger: logger,
	}

	sl.backgroundContext = context.Background()
	sl.backgroundContext, sl.backgroundCancel = context.WithCancel(sl.backgroundContext)

	r, err := deps.Lookup(toggleswitch.Named(conf.Switch))
	if err != nil {
		return nil, err
	}

	sl.theSwitch = r.(toggleswitch.Switch)

	go sl.run()

	return sl, nil
}

func (sl *sunsetLight) doLoop(ctx context.Context, now time.Time) error {
	_, set := sunrise.SunriseSunset(
		sl.conf.Lat,
		sl.conf.Lng,
		now.Year(),
		now.Month(),
		now.Day(),
	)

	sl.logger.Infof("set: %v", set)

	if now.Before(set) {
		sl.logger.Infof("durring day, turn lights off")
		return sl.theSwitch.SetPosition(sl.backgroundContext, 0, nil)
	}

	sl.logger.Infof("after sunset, turn lights on")
	return sl.theSwitch.SetPosition(sl.backgroundContext, 1, nil)
}

func (sl *sunsetLight) run() {

	for sl.backgroundContext.Err() == nil {
		start := time.Now()

		err := sl.doLoop(sl.backgroundContext, start)
		if err != nil {
			sl.logger.Warnf("error in doLoop: %v", err)
		}

		sleepTime := time.Minute - time.Since(start)
		sl.logger.Debugf("sleeping %v", sleepTime)
		time.Sleep(sleepTime)
	}
}

func (sl *sunsetLight) Name() resource.Name {
	return sl.name
}

func (sl *sunsetLight) DoCommand(ctx context.Context, cmd map[string]interface{}) (map[string]interface{}, error) {
	return nil, nil
}

func (sl *sunsetLight) Close(ctx context.Context) error {
	if sl.backgroundCancel != nil {
		sl.backgroundCancel()
		sl.backgroundCancel = nil
	}
	return nil
}

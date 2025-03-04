package config

import (
	"github.com/johandrevandeventer/devices-api-server/internal/config/app"
	"github.com/johandrevandeventer/devices-api-server/internal/config/system"
)

type Config struct {
	System *system.SystemConfig `mapstructure:"system" yaml:"system"`
	App    *app.AppConfig       `mapstructure:"app" yaml:"app"`
}

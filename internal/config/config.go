package config

import (
	"github.com/wanderer69/flow_processor/pkg/dao"
)

type Config struct {
	AppPort uint   `envconfig:"APP_PORT" default:"8888"`
	AppEnv  string `envconfig:"APP_ENV" default:"prod"`

	dao.ConfigDAO
}

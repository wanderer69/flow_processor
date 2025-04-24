package config

import (
	"github.com/wanderer69/flow_processor/pkg/dao"
)

type Config struct {
	AppWebPort  uint   `envconfig:"APP_WEB_PORT" default:"8888"`
	AppGRPCPort uint   `envconfig:"APP_GRPC_PORT" default:"8889"`
	AppEnv      string `envconfig:"APP_ENV" default:"prod"`

	dao.ConfigDAO
}

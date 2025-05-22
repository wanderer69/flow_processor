package config

import (
	"github.com/wanderer69/flow_processor/pkg/dao"
)

type Config struct {
	AppWebPort       int    `envconfig:"APP_WEB_PORT" default:"8888"`
	AppGRPCPort      int    `envconfig:"APP_GRPC_PORT" default:"8889"`
	AppFrontGRPCPort int    `envconfig:"APP_FRONT_GRPC_PORT" default:"8890"`
	AppEnv           string `envconfig:"APP_ENV" default:"test"`

	dao.ConfigDAO
	ExternalProcessDuration int `envconfig:"EXTERNAL_PROCESS_DURATION_MSEC" default:"1"`
	TimerProcessDuration    int `envconfig:"TIMER_PROCESS_DURATION_MSEC" default:"1"`
	TopicProcessDuration    int `envconfig:"TOPIC_PROCESS_DURATION_MSEC" default:"1"`
	GlobalProcessDuration   int `envconfig:"GLOBAL_PROCESS_DURATION_MSEC" default:"1"`
	WebProcessDuration      int `envconfig:"WEB_PROCESS_DURATION_MSEC" default:"1"`
	ClientProcessDuration   int `envconfig:"CLIENT_PROCESS_DURATION_MSEC" default:"1"`
}

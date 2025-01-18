package config

import (
	"github.com/wanderer69/flow_processor/pkg/dao"
)

type Config struct {
	AppPort                  uint   `envconfig:"APP_PORT" default:"8888"`
	AuthJwtSecret            string `envconfig:"AUTH_JWT_SECRET" required:"true"`
	AuthJwtTokenLifeTime     uint   `envconfig:"AUTH_JWT_TOKEN_LIFETIME_MIN" required:"true"`
	AuthRefreshTokenLifeTime uint   `envconfig:"AUTH_REFRESH_TOKEN_LIFETIME_HOUR" required:"true"`
	AppEnv                   string `envconfig:"APP_ENV" default:"prod"`

	dao.ConfigDAO
}

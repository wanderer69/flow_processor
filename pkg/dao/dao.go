package dao

import (
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DAO struct {
	db *gorm.DB
}

type ConfigDAO struct {
	DSN                   string        `envconfig:"POSTGRE_DSN" required:"true"`
	MaxIdleConnections    uint          `envconfig:"POSTGRE_MAX_IDLE_CONS" default:"10"`
	MaxOpenConnections    uint          `envconfig:"POSTGRE_MAX_OPEN_CONS" default:"10"`
	MaxLifetimeConnection time.Duration `envconfig:"POSTGRE_MAX_LIFETIME_CON" default:"0"`
}

func (dao *DAO) DB() *gorm.DB {
	return dao.db
}

func InitDAO(cnf ConfigDAO, l logger.Interface) (*DAO, error) {
	db, err := gorm.Open(
		postgres.Open(cnf.DSN),
		&gorm.Config{
			Logger: l,
		},
	)
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	if cnf.MaxIdleConnections != 0 {
		sqlDB.SetMaxIdleConns(int(cnf.MaxIdleConnections))
	}
	if cnf.MaxOpenConnections != 0 {
		sqlDB.SetMaxOpenConns(int(cnf.MaxOpenConnections))
	}
	if cnf.MaxLifetimeConnection != 0 {
		sqlDB.SetConnMaxLifetime(cnf.MaxLifetimeConnection)
	}

	return &DAO{db: db}, nil
}

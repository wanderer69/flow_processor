package tests

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/pressly/goose"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type DAO struct {
	db  *gorm.DB
	SQL *sql.DB
}

const containerLivetimeSeconds = 1800

func (dao *DAO) DB() *gorm.DB {
	return dao.db
}

func upInDocker() *sql.DB {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}
	Network, _ := pool.NetworksByName("bridge")
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgis/postgis",
		//NetworkID:  []*dockertest.Network{&Network[0]},
		Tag: "13-master",
		Env: []string{
			"POSTGRES_PASSWORD=password",
			"POSTGRES_USER=user",
			"POSTGRES_DB=dbname",
			"listen_addresses = '*'",
		},
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	Host := resource.GetIPInNetwork(&Network[0])
	dsn := fmt.Sprintf("postgres://user:password@%s:5432/dbname?sslmode=disable", Host)

	log.Println("Connecting to database on url: ", dsn)
	_ = resource.Expire(containerLivetimeSeconds)

	pool.MaxWait = containerLivetimeSeconds * time.Second
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("could not connect to docker: %s", err)
	}
	if err = pool.Retry(func() error {
		sqlDB, err = sql.Open("pgx", dsn)
		if err != nil {
			log.Fatalf("could not connect to docker: %s", err)
		}
		return sqlDB.Ping()
	}); err != nil {
		log.Fatalf("could not connect to docker: %s", err)
	}

	return sqlDB
}

func InitDAO(migrationsPath string) (*DAO, error) {
	var (
		sqlDB *sql.DB
		err   error
	)

	extDSN := os.Getenv("EXTERNAL_POSTGRE_DSN")
	if extDSN != "" {
		fmt.Println("extDSN:", extDSN)
		sqlDB, err = sql.Open("pgx", extDSN)
		if err != nil {
			log.Fatalf("could not connect to postgres: %s", err)
		}
	} else {
		sqlDB = upInDocker()
	}

	db, err := gorm.Open(
		postgres.New(postgres.Config{
			Conn: sqlDB,
		}),
		&gorm.Config{},
	)
	if err != nil {
		log.Fatalf("could open gorm: %s", err)
	}

	pwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	migrationsPath = fmt.Sprintf("%s/%s", pwd, migrationsPath)
	if err = goose.Up(sqlDB, migrationsPath); err != nil {
		log.Fatalf("failed up migrations: %s", err)
	}

	return &DAO{db: db, SQL: sqlDB}, nil
}

package app

import (
	"context"
	"embed"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/wanderer69/flow_processor/internal/config"
	camunda7convertor "github.com/wanderer69/flow_processor/pkg/camunda_7_convertor"
	clientconnector "github.com/wanderer69/flow_processor/pkg/client_connector"
	externalactivation "github.com/wanderer69/flow_processor/pkg/external_activation"
	externaltopic "github.com/wanderer69/flow_processor/pkg/external_topic"
	internalformat "github.com/wanderer69/flow_processor/pkg/internal_format"
	"github.com/wanderer69/flow_processor/pkg/loader"
	"github.com/wanderer69/flow_processor/pkg/process"
	"github.com/wanderer69/flow_processor/pkg/store"
	"github.com/wanderer69/flow_processor/pkg/timer"
	"go.uber.org/zap"
)

type Application struct {
	processRepository  processRepository
	diagrammRepository diagrammRepository
	e                  *echo.Echo
	cnf                config.Config
}

func NewApplication(
	processRepository processRepository,
	diagrammRepository diagrammRepository,
) *Application {
	return &Application{
		processRepository:  processRepository,
		diagrammRepository: diagrammRepository,
	}
}

func (a *Application) Init(fsSPA embed.FS, fsVersion embed.FS, cnf config.Config) error {
	ctx := context.Background()
	logger := zap.L()
	a.cnf = cnf
	topicClient := externaltopic.NewExternalTopic()
	timerClient := timer.NewTimer()
	externalActivationClient := externalactivation.NewExternalActivation()
	camunda7ConvertorClient := camunda7convertor.NewConverterClient()
	internalFormatClient := internalformat.NewInternalFormat()
	loader := loader.NewLoader(camunda7ConvertorClient, internalFormatClient)
	storeClient := store.NewStore(loader, a.processRepository, a.diagrammRepository)

	stop := make(chan struct{})

	pe := process.NewProcessExecutor(topicClient, timerClient, externalActivationClient, storeClient, stop)

	pe.SetLogger(ctx, func(ctx context.Context, msg string) error {
		fmt.Printf("%v\r\n", msg)
		return nil
	})

	port := cnf.AppPort
	go func() {
		err := clientconnector.ServerConnect(int(port), topicClient, externalActivationClient, pe)
		if err != nil {
			logger.Error("ServerConnect", zap.Error(err))
		}
	}()

	time.Sleep(time.Second)

	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{echo.GET, echo.PUT, echo.POST, echo.DELETE},
	}))

	e.Any("/health-check/", func(c echo.Context) error {
		c.Response().Writer.WriteHeader(http.StatusOK)
		return nil
	})

	e.Any("/version/", func(c echo.Context) error {
		file, err := fsVersion.ReadFile("version/ver.html")
		result := file
		state := http.StatusOK
		if err != nil {
			state = http.StatusBadGateway
			result = []byte(fmt.Sprintf("%v", err))
		}
		c.Response().Writer.Write(result)
		return c.JSON(state, nil)
	})

	e.Any("/spa/", func(c echo.Context) error {
		u := c.Request().URL
		l := strings.Split(u.Path, "/spa/")
		if len(l) > 0 {
			file, err := fsSPA.ReadFile("spa/index.html")
			result := file
			state := http.StatusOK
			if err != nil {
				state = http.StatusBadGateway
				result = []byte(fmt.Sprintf("%v", err))
			}
			c.Response().Writer.Write(result)
			return c.JSON(state, nil)
		}
		result := []byte("{ not found}")
		state := http.StatusNotFound
		c.Response().Writer.Write(result)
		return c.JSON(state, nil)
	})
	a.e = e
	return nil
}

func (a *Application) Start() error {
	a.e.Server.ReadHeaderTimeout = time.Second * 5
	return a.e.Start(fmt.Sprintf("0.0.0.0:%d", a.cnf.AppPort))
}

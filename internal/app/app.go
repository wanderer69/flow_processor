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
	frontconnector "github.com/wanderer69/flow_processor/internal/gateway/front_connector"
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
	frontUser          frontUser
	e                  *echo.Echo
	cnf                config.Config
}

func NewApplication(
	processRepository processRepository,
	diagrammRepository diagrammRepository,
	frontUser frontUser,
) *Application {
	return &Application{
		processRepository:  processRepository,
		diagrammRepository: diagrammRepository,
		frontUser:          frontUser,
	}
}

func (a *Application) Init(fsSPA embed.FS, fsVersion embed.FS, cnf config.Config) error {
	ctx := context.Background()
	logger := zap.L()
	a.cnf = cnf
	topicClient := externaltopic.NewExternalTopic(cnf.TopicProcessDuration)
	timerClient := timer.NewTimer(cnf.TimerProcessDuration)
	externalActivationClient := externalactivation.NewExternalActivation(cnf.ExternalProcessDuration)
	camunda7ConvertorClient := camunda7convertor.NewConverterClient()
	internalFormatClient := internalformat.NewInternalFormat()
	loader := loader.NewLoader(camunda7ConvertorClient, internalFormatClient)
	storeClient := store.NewStore(loader, a.processRepository, a.diagrammRepository)

	stop := make(chan struct{})

	pe := process.NewProcessExecutor(topicClient, timerClient, externalActivationClient, storeClient, stop, cnf.GlobalProcessDuration)

	pe.SetLogger(ctx, func(ctx context.Context, msg string) error {
		fmt.Printf("%v\r\n", msg)
		return nil
	})
	go func() {
		err := clientconnector.ServerConnect(int(cnf.AppGRPCPort), topicClient, externalActivationClient, pe, cnf.ClientProcessDuration)
		if err != nil {
			logger.Error("ServerConnect", zap.Error(err))
		}
	}()

	go func() {
		err := frontconnector.ServerConnectWeb(int(cnf.AppFrontGRPCPort), pe, storeClient, a.frontUser, fsSPA, cnf.WebProcessDuration)
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

	e.Any("/", func(c echo.Context) error {
		u := c.Request().URL
		l := strings.Split(u.Path, "/spa/")
		fmt.Printf("/l - %v\r\n", l)
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
	/*
		e.Any("/spa/*", func(c echo.Context) error {
			u := c.Request().URL
			l := strings.Split(strings.Trim(u.Path, " \r\n"), "/spa/")
			fmt.Printf(">%v l - %v %v\r\n", u.Path, l, len(l))
			if len(l) == 2 {
				fileName := l[1]
				if len(l[1]) == 0 {
					fileName = "index.html"
				}
				fmt.Printf("fileName %v\r\n", fileName)
				file, err := fsSPA.ReadFile("spa/" + fileName)
				result := file
				state := http.StatusOK
				if err != nil {
					state = http.StatusBadGateway
					result = []byte(fmt.Sprintf("%v", err))
				}

				c.Response().Writer.WriteHeader(state)
				c.Response().Writer.Write(result)
				//fmt.Printf("file %v\r\n", string(result))

				//return c.JSON(state, nil)
				return nil

			}
			result := []byte("{ not found}")
			state := http.StatusNotFound
			c.Response().Writer.WriteHeader(state)
			c.Response().Writer.Write(result)
			// return c.JSON(state, nil)
			return nil
		})
	*/
	a.e = e
	return nil
}

func (a *Application) Start() error {
	a.e.Server.ReadHeaderTimeout = time.Second * 5
	return a.e.Start(fmt.Sprintf("0.0.0.0:%d", a.cnf.AppWebPort))
}

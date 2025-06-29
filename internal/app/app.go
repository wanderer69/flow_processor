package app

import (
	"context"
	"embed"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
	"google.golang.org/grpc"

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
)

type Application struct {
	processRepository  processRepository
	diagrammRepository diagrammRepository
	frontUser          frontUser
	e                  *echo.Echo
	cnf                config.Config

	HttpServer *http.Server
	GrpcServer *grpc.Server
	appCancel  context.CancelFunc
	start      chan struct{}
}

func NewApplication(
	processRepository processRepository,
	diagrammRepository diagrammRepository,
	frontUser frontUser,
	cnf config.Config,
) *Application {
	return &Application{
		processRepository:  processRepository,
		diagrammRepository: diagrammRepository,
		frontUser:          frontUser,
		cnf:                cnf,
	}
}

func (a *Application) Init(appCtx context.Context, appCancel context.CancelFunc, fsSPA embed.FS, fsVersion embed.FS) error {
	logger := zap.L()

	topicClient := externaltopic.NewExternalTopic(a.cnf.TopicProcessDuration)
	timerClient := timer.NewTimer(a.cnf.TimerProcessDuration)
	externalActivationClient := externalactivation.NewExternalActivation(a.cnf.ExternalProcessDuration)
	camunda7ConvertorClient := camunda7convertor.NewConverterClient()
	internalFormatClient := internalformat.NewInternalFormat()
	loader := loader.NewLoader(camunda7ConvertorClient, internalFormatClient)
	storeClient := store.NewStore(loader, a.processRepository, a.diagrammRepository)

	stop := make(chan struct{})

	pe := process.NewProcessExecutor(topicClient, timerClient, externalActivationClient, storeClient, stop, a.cnf.GlobalProcessDuration)

	pe.SetLogger(appCtx, func(ctx context.Context, msg string) error {
		logger.Info("process", zap.String("message", msg))
		return nil
	})

	go func() {
		err := clientconnector.ServerConnect(int(a.cnf.AppGRPCPort), topicClient, externalActivationClient, pe, a.cnf.ClientProcessDuration)
		if err != nil {
			logger.Error("ServerConnect", zap.Error(err))
		}
	}()

	/*
		go func() {
			err := frontconnector.ServerConnectWeb(int(cnf.AppFrontGRPCPort), pe, storeClient, a.frontUser, fsSPA, cnf.WebProcessDuration)
			if err != nil {
				logger.Error("ServerConnect", zap.Error(err))
			}
		}()
	*/

	a.appCancel = appCancel

	e := echo.New()
	e.HideBanner = true

	//Setup HTTP / Websocket server
	router := http.NewServeMux()
	router.Handle("/", http.FileServer(http.Dir("./spa")))
	a.start = make(chan struct{}, 1)
	grpcServer, err := frontconnector.ServerConnectWS(appCtx, appCancel, router, pe, storeClient, a.frontUser, a.cnf.WebProcessDuration, a.start)
	if err != nil {
		logger.Error("ServerConnectWS", zap.Error(err))
	}
	a.GrpcServer = grpcServer

	httpServer := &http.Server{Addr: fmt.Sprintf(":%d", a.cnf.AppFrontGRPCPort), Handler: router}
	a.HttpServer = httpServer

	time.Sleep(time.Second)

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

/*
	type helloServer struct {
		pb.UnimplementedGreeterServer
	}

	func (*helloServer) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
		fmt.Printf("sayHello: %v\r\n", in.Name)
		return &pb.HelloReply{Message: "Hello " + in.GetName()}, nil
	}
*/
func (a *Application) Start() {
	logger := zap.L()

	/*
		go func() {
				//App context setup
				appCtx, appCancel := context.WithCancel(context.Background())
				defer appCancel()

				//Setup HTTP / Websocket server
				router := http.NewServeMux()

				wsl := wasmws.NewWebSocketListener(appCtx)
				router.HandleFunc("/grpc-proxy", wsl.ServeHTTP)
				router.Handle("/", http.FileServer(http.Dir("./spa")))
				httpServer := &http.Server{Addr: ":8188", Handler: router}
				//Run HTTP server
				go func() {
					defer appCancel()
					log.Printf("ERROR: HTTP Listen and Server failed; Details: %s", httpServer.ListenAndServe())
				}()

				//gRPC setup
				creds, err := credentials.NewServerTLSFromFile("cert.pem", "key.pem")
				if err != nil {
					log.Fatalf("Failed to contruct gRPC TSL credentials from {cert,key}.pem: %s", err)
				}
				grpcServer := grpc.NewServer(grpc.Creds(creds))
				pb.RegisterGreeterServer(grpcServer, new(helloServer))
				//Run gRPC server
				go func() {
					defer appCancel()

					if err := grpcServer.Serve(wsl); err != nil {
						log.Printf("ERROR: Failed to serve gRPC connections; Details: %s", err)
					}
				}()

				//Handle signals
				sigs := make(chan os.Signal, 1)
				signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
				go func() {
					log.Printf("INFO: Received shutdown signal: %s", <-sigs)
					appCancel()
				}()

				//Shutdown
				<-appCtx.Done()
				shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), time.Second*2)
				defer shutdownCancel()

				grpcShutdown := make(chan struct{}, 1)
				go func() {
					grpcServer.GracefulStop()
					grpcShutdown <- struct{}{}
				}()

				httpServer.Shutdown(shutdownCtx)
				select {
				case <-grpcShutdown:
				case <-shutdownCtx.Done():
					grpcServer.Stop()
				}
		}()
	*/
	//Run HTTP server
	go func() {
		defer a.appCancel()
		err := a.HttpServer.ListenAndServe()
		logger.Error("Start: HTTP Listen and Server failed", zap.Error(err))
	}()
	a.start <- struct{}{}

	a.e.Server.ReadHeaderTimeout = time.Second * 5
	go func() {
		defer a.appCancel()
		err := a.e.Start(fmt.Sprintf("0.0.0.0:%d", a.cnf.AppWebPort))
		logger.Error("Start: echo Listen and Server failed", zap.Error(err))
	}()
}

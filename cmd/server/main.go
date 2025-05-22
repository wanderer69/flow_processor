package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	gLogger "gorm.io/gorm/logger"

	"github.com/kelseyhightower/envconfig"

	"github.com/wanderer69/flow_processor/cmd/server/spa"
	"github.com/wanderer69/flow_processor/internal/app"
	"github.com/wanderer69/flow_processor/internal/config"
	diagrammRepository "github.com/wanderer69/flow_processor/internal/repository/diagramm"
	frontUserRepository "github.com/wanderer69/flow_processor/internal/repository/front_user"
	processRepository "github.com/wanderer69/flow_processor/internal/repository/process"
	"github.com/wanderer69/flow_processor/pkg/dao"
)

//go:embed spa
var fsSPA embed.FS

//go:embed version
var fsVersion embed.FS

func main() {
	cnf := config.Config{}
	ctx, ctxCancel := context.WithCancel(context.Background())

	if err := envconfig.Process("", &cnf); err != nil {
		envconfig.Usage("", &cnf)
		panic(fmt.Errorf("ошибка парсинга переменных окружения: %w", err))
	}
	fmt.Printf("%#v\r\n", cnf)

	config := zap.NewProductionEncoderConfig()
	if cnf.AppEnv == "dev" {
		config = zap.NewDevelopmentEncoderConfig()
	}
	config.EncodeTime = zapcore.ISO8601TimeEncoder
	fileEncoder := zapcore.NewJSONEncoder(config)

	stdout := zapcore.AddSync(os.Stdout)
	level := zap.NewAtomicLevelAt(zap.InfoLevel)

	core := zapcore.NewTee(
		zapcore.NewCore(fileEncoder, stdout, level),
	)
	zapLogger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	defer zapLogger.Sync()

	undo := zap.ReplaceGlobals(zapLogger)
	defer undo()

	zapLogger.Info("Starting service flow processor")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT)
	signal.Notify(sigCh, syscall.SIGTERM)

	sqlLogLevel := gLogger.Warn
	lg := gLogger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		gLogger.Config{
			SlowThreshold:             time.Millisecond * 300,
			LogLevel:                  sqlLogLevel,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)
	dao, err := dao.InitDAO(cnf.ConfigDAO, lg)
	if err != nil {
		panic(fmt.Errorf("error database initialization: %w", err))
	}
	gm := dao.DB()
	sql, err := gm.DB()
	if err != nil {
		panic(fmt.Errorf("error dao initialization: %w", err))
	}
	defer sql.Close()
	zapLogger.Info("Database connected")

	processRepo := processRepository.NewRepository(dao)
	diagrammRepo := diagrammRepository.NewRepository(dao)
	frontUserRepo := frontUserRepository.NewRepository(dao)
	application := app.NewApplication(processRepo, diagrammRepo, frontUserRepo)
	err = application.Init(spa.SPA, fsVersion, cnf)
	if err != nil {
		panic(fmt.Errorf("error application initialization: %w", err))
	}

	defer func() {
		ctxCancel()
		signal.Stop(sigCh)
		close(sigCh)
	}()

	go func() {
		err := application.Start()
		zapLogger.Error("failed start health-check", zap.Error(err))
		panic(err)
	}()

	zapLogger.Info("Service started")
	for {
		ticker := time.NewTicker(time.Duration(10) * time.Second)
		select {
		case <-ticker.C:
			runtime.Gosched()
			// Получаем стэк для каждой горутины
			var buf []byte
			for i := 0; i < runtime.NumCPU(); i++ {
				n := runtime.Stack(buf, true)
				if n > 2 {
					// Показываем стэк, можно выводить в файл или обрабатывать как угодно
					fmt.Printf("%s\r\n", string(buf))
				}
				buf = buf[:0]
			}

		case <-sigCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

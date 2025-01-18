package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	gLogger "gorm.io/gorm/logger"

	"github.com/kelseyhightower/envconfig"

	"github.com/wanderer69/flow_processor/internal/app"
	"github.com/wanderer69/flow_processor/internal/config"
	diagrammRepository "github.com/wanderer69/flow_processor/internal/repository/diagramm"
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

	config := zap.NewProductionEncoderConfig()
	defaultLogLevel := zapcore.WarnLevel
	if cnf.AppEnv == "dev" {
		config = zap.NewDevelopmentEncoderConfig()
		defaultLogLevel = zapcore.DebugLevel
	}
	config.EncodeTime = zapcore.ISO8601TimeEncoder
	fileEncoder := zapcore.NewJSONEncoder(config)
	logFile, _ := os.OpenFile("log.json", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	writer := zapcore.AddSync(logFile)

	//	consoleEncoder := zapcore.NewConsoleEncoder(config)
	stdout := zapcore.AddSync(os.Stdout)
	level := zap.NewAtomicLevelAt(zap.InfoLevel)

	core := zapcore.NewTee(
		zapcore.NewCore(fileEncoder, writer, defaultLogLevel),
		//		zapcore.NewCore(consoleEncoder, stdout, level),
		zapcore.NewCore(fileEncoder, stdout, level),
	)
	zapLogger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	defer zapLogger.Sync()

	undo := zap.ReplaceGlobals(zapLogger)
	defer undo()

	zapLogger.Info("Started service pushbroker")

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
	zapLogger.Info("database connected")

	processRepo := processRepository.NewRepository(dao)
	diagrammRepo := diagrammRepository.NewRepository(dao)
	application := app.NewApplication(processRepo, diagrammRepo)
	err = application.Init(fsSPA, fsVersion, cnf)
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

	for {
		select {
		case <-sigCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

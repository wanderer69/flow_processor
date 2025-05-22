package spa_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	frontclient "github.com/wanderer69/flow_processor/pkg/front_client"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type AppT struct {
	currentToken string
	client       *frontclient.FrontClient
}

const (
	GSClientConnect    = 0
	GSLogin            = 1
	GSFillProcesses    = 2
	GSFillProcessFlows = 3

	GSError = 100
)

func TestMain(t *testing.T) {
	ctx := context.Background()
	fmt.Println("Flow processor SPA start")

	at := AppT{}

	config := zap.NewDevelopmentEncoderConfig()
	//	}
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

	done := make(chan bool)

	globalState := 0

	client := frontclient.NewFrontClient("localhost", 8090)
	isError := false
	for {
		fmt.Printf("-> %v\r\n", globalState)
		switch globalState {
		case GSClientConnect:
			at.client = client
			globalState = GSLogin
		case GSLogin:
			token, err := client.Login(ctx, "user", "password")
			if err != nil {
				fmt.Printf("login error: %v", err.Error())
				globalState = GSError
				continue
			}
			at.currentToken = token
		case GSError:
			isError = true
		}
		time.Sleep(time.Duration(5) * time.Millisecond)
		if isError {
			break
		}
	}

	<-done
}

//go:build js && wasm
// +build js,wasm

package spa

import (
	"context"
	"fmt"
	"os"
	"syscall/js"
	"time"

	frontclient "github.com/wanderer69/flow_processor/pkg/front_client"
	object "github.com/wanderer69/js_object"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type AppT struct {
	Jsoa []*object.JSObject
	Jsod map[string]*object.JSObject

	currentToken string
	client       *frontclient.FrontClient
}

func (at AppT) TableRowClickCallBack1(this js.Value, args []js.Value, jso *object.JSObject) interface{} {
	fmt.Printf("TableRowClickCallBack1\r\n")
	for i := range args {
		fmt.Printf("-- %v\r\n", args[i].Type().String())
		switch args[i].Type().String() {
		case "string":
			fmt.Printf("%v\r\n", args[i].String())
		case "number":
			fmt.Printf("%v\r\n", args[i].Int())
		case "float":
			fmt.Printf("%v\r\n", args[i].Float())
		case "bool":
			fmt.Printf("%v\r\n", args[i].Bool())
		}
	}
	table1 := at.Jsod["table_1"]
	if table1 != nil {
		fmt.Printf("table1 %v\r\n", table1)
	}
	return nil
}

func (at AppT) TableRowClickCallBack2(this js.Value, args []js.Value, jso *object.JSObject) interface{} {
	fmt.Printf("TableRowClickCallBack2\r\n")
	for i := range args {
		fmt.Printf("-- %v\r\n", args[i].Type().String())
		switch args[i].Type().String() {
		case "string":
			fmt.Printf("%v\r\n", args[i].String())
		case "number":
			fmt.Printf("%v\r\n", args[i].Int())
		case "float":
			fmt.Printf("%v\r\n", args[i].Float())
		case "bool":
			fmt.Printf("%v\r\n", args[i].Bool())
		}
	}
	table2 := at.Jsod["table_2"]
	if table2 != nil {
		fmt.Printf("table2 %v\r\n", table2)
	}
	return nil
}

const (
	GSClientConnect    = 0
	GSLogin            = 1
	GSFillProcesses    = 2
	GSFillProcessFlows = 3

	GSError = 100
)

func main() {
	ctx := context.Background()
	fmt.Println("Flow processor SPA start")

	//	config := zap.NewProductionEncoderConfig()
	//	if cnf.AppEnv == "dev" {
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
	jsoa, jsod := object.CreateDocConstructor()
	at := AppT{}
	at.Jsoa = jsoa
	at.Jsod = jsod
	fmt.Printf("jsoa %v jsod %v\r\n", jsoa, jsod)

	object.BindCallBack(at, jsoa, &at)

	globalState := 0

	client := frontclient.NewFrontClient("127.0.0.1", 8090)
	isError := false
	for {
		fmt.Printf("-> %v\r\n", globalState)
		switch globalState {
		case GSClientConnect:
			at.client = client
			globalState = GSLogin
		case GSLogin:
			token, err := client.LoginWeb(ctx, "user", "password")
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

	table_1 := jsod["table_1"]
	hl1 := []string{"Col 1", " Col 2", "Col 3", "Col 4"}
	ol11 := []string{"Item10", "Item20", "Item30", "Item40"}
	ol21 := []string{"Item11", "Item21", "Item31", "Item41"}
	oll1 := [][]string{ol11, ol21}
	table_1.SetTable(at, hl1, oll1)

	table_2 := jsod["table_2"]
	hl2 := []string{"Col 1", " Col 2", "Col 3", "Col 4"}
	ol12 := []string{"Item10", "Item20", "Item30", "Item40"}
	ol22 := []string{"Item11", "Item21", "Item31", "Item41"}
	oll2 := [][]string{ol12, ol22}
	table_2.SetTable(at, hl2, oll2)

	/*
	   bb_cb := func(this js.Value, args []js.Value) interface{} {
	         text_1.SetValue("333333")
	         return nil
	   }
	   button_1.SetCallBack("click", bb_cb)
	*/
	<-done
}

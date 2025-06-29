//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"strings"
	"syscall/js"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	wasmws "github.com/wanderer69/flow_processor/cmd/spa/wsjs"
	frontclient "github.com/wanderer69/flow_processor/pkg/front_client"
	object "github.com/wanderer69/js_object"
)

type Command struct {
	Cmd string
	Arg string
}
type AppT struct {
	Jsoa []*object.JSObject
	Jsod map[string]*object.JSObject

	currentToken string
	client       *frontclient.FrontClient
	cmd          chan *Command
	done         chan bool
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
	at.cmd <- &Command{
		Cmd: "GetListProcessFlow",
		Arg: "0",
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

func (at AppT) TableClickCallBack(this js.Value, args []js.Value, jso *object.JSObject) interface{} {
	fmt.Printf("TableClickCallBack\r\n")
	fmt.Printf("%#v\r\n", args)
	table1 := at.Jsod["table_1"]
	if table1 != nil {
	}
	return nil
}

func (at AppT) TableRowClickCallBack(this js.Value, args []js.Value, jso *object.JSObject) interface{} {
	fmt.Printf("TableRowClickCallBack\r\n")
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

const (
	GSLogin            = 0
	GSClientConnect    = 1
	GSFillProcesses    = 2
	GSFillProcessFlows = 3

	GSWaitCmd = 80
	GSSuccess = 90
	GSError   = 100
)

// jwtCredentials satisfies the PerRPCCredentials interface.
type jwtCredentials struct {
	getToken func() string
}

// GetRequestMetadata will be called on every RPC call and returns a map which is used to build the request metadata.
func (j *jwtCredentials) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	token := j.getToken() // get your token from the context, or wherever it's stored

	// return metadata map for RPC call
	return map[string]string{
		"authorization": token,
	}, nil
}

func (j *jwtCredentials) RequireTransportSecurity() bool {
	return false
}

func main() {
	appCtx, appCancel := context.WithCancel(context.Background())
	defer appCancel()

	fmt.Println("Flow processor SPA start")

	//	config := zap.NewProductionEncoderConfig()
	config := zap.NewDevelopmentEncoderConfig()
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

	jsoa, jsod := object.CreateDocConstructor()
	at := AppT{
		Jsoa: jsoa,
		Jsod: jsod,
		cmd:  make(chan *Command, 10),
	}

	object.BindCallBack(at, jsoa, &at)

	globalState := 0
	prevGlobalState := 0

	jwtCreds := &jwtCredentials{
		getToken: func() string {
			return at.currentToken
		},
	}

	//Connect to remote gRPC server
	const websocketURL = "ws://localhost:8090/grpc-proxy"
	creds := credentials.NewTLS(&tls.Config{
		InsecureSkipVerify: true,
	})
	conn, err := grpc.NewClient("passthrough:///"+websocketURL,
		grpc.WithContextDialer(wasmws.GRPCDialer),
		grpc.WithDisableRetry(),
		grpc.WithTransportCredentials(creds),
		grpc.WithPerRPCCredentials(jwtCreds),
	)
	if err != nil {
		log.Fatalf("Could not gRPC dial: %s; Details: %s", websocketURL, err)
	}
	defer conn.Close()

	table2 := jsod["table_1"]
	table1 := jsod["table_2"]

	at.client = frontclient.NewFrontClient("127.0.0.1", 0, conn)
	isSuccess := false
	subscriberPong := make(chan string)
	subscriberProcessEvent := make(chan *frontclient.Args)

	cnt := 3
	tiker := time.NewTicker(time.Duration(1) * time.Second)
	for {
		//fmt.Printf("-> %v\r\n", globalState)
		switch globalState {
		case GSLogin:
			token, err := at.client.Login(appCtx, "user", "password")
			if err != nil {
				fmt.Printf("login error: %v", err.Error())
				globalState = GSError
				continue
			}
			at.currentToken = token
			globalState = GSClientConnect
		case GSClientConnect:
			prevGlobalState = globalState
			connected := make(chan bool)
			go func() {
				at.done, err = at.client.Connect(appCtx, connected)
				if err != nil {
					fmt.Printf("connect error: %v", err.Error())
					globalState = GSError
				}
			}()
			<-connected
			//fmt.Printf("Connected!\r\n")
			at.client.SubscribePong(appCtx, subscriberPong)
			at.client.SubscribeProcessEvent(appCtx, subscriberProcessEvent)
			globalState = GSWaitCmd
		case GSError:
			time.Sleep(time.Duration(500) * time.Millisecond)
			globalState = prevGlobalState
		case GSSuccess:
			isSuccess = true
		case GSWaitCmd:
			select {
			case <-tiker.C:
				fmt.Printf("Ping\r\n")
				msg := uuid.NewString()
				result, err := at.client.Ping(appCtx, msg)
				if err != nil {
					fmt.Printf("connect error: %v", err.Error())
					globalState = GSError
					continue
				}
				if result != msg {
					globalState = GSLogin
				}
				cnt--
				if cnt == 0 {
					cnt = 3
					at.cmd <- &Command{
						Cmd: "GetListProcesses",
					}
				}
				/*
					result, err := at.client.Pong(appCtx)
					if err != nil {
						fmt.Printf("connect error: %v", err.Error())
						globalState = GSError
						continue
					}
					if !result {
						msg := uuid.NewString()
						result, err := at.client.Ping(appCtx, msg)
						if err != nil {
							fmt.Printf("connect error: %v", err.Error())
							globalState = GSError
							continue
						}
						if result != msg {
							globalState = GSLogin
						}
					}
				*/
			case pong := <-subscriberPong:
				fmt.Printf("pong: %v\r\n", pong)
			case processEvent := <-subscriberProcessEvent:
				fmt.Printf("process event: %#v\r\n", processEvent)
			case cmd := <-at.cmd:
				fmt.Printf("command: %#v\r\n", cmd)
				switch cmd.Cmd {
				case "GetListProcesses":
					processes, err := at.client.GetListProcesses(appCtx)
					if err != nil {
						fmt.Printf("get list processes error: %v", err.Error())
						globalState = GSError
						continue
					}
					hl1 := []string{"Application ID", "Process ID", "State"}

					oll1 := [][]string{}
					for i := range processes {
						ol11 := []string{
							processes[i].ApplicationId,
							processes[i].ProcessId,
							processes[i].State,
						}
						oll1 = append(oll1, ol11)
					}
					if len(processes) == 0 {
						ol11 := []string{"Item10", "Item20", "Item30"}
						ol21 := []string{"Item11", "Item21", "Item31"}
						oll1 = [][]string{ol11, ol21}
					}
					table1.Object.Set("innerHTML", "")
					table1.SetTable(at, hl1, oll1)
				case "GetListProcessFlow":
					flows, err := at.client.GetListProcessFlows(appCtx, cmd.Arg)
					if err != nil {
						fmt.Printf("get list process flows error: %v", err.Error())
						globalState = GSError
						continue
					}
					hl2 := []string{"Execute", "Process state ID", "state", "states"}
					oll2 := [][]string{}
					for i := range flows {
						ol11 := []string{
							flows[i].Execute,
							flows[i].ProcessStateData,
							flows[i].State,
							strings.Join(flows[i].ProcessStates, ", "),
						}
						oll2 = append(oll2, ol11)
					}
					if len(flows) == 0 {
						ol12 := []string{"Item10", "Item20", "Item30", "Item40"}
						ol22 := []string{"Item11", "Item21", "Item31", "Item41"}
						oll2 = [][]string{ol12, ol22}
					}

					table1.Object.Set("innerHTML", "")
					table2.SetTable(at, hl2, oll2)
				}
			}
		}
		if isSuccess {
			break
		}

	}
}

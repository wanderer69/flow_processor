package frontclient

import (
	"context"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/wanderer69/flow_processor/pkg/entity"
	pb "github.com/wanderer69/flow_processor/pkg/proto/front"
)

type Callback func(
	ctx context.Context,
	processID string,
	processName string,
	topicName string,
	messages []*entity.Message,
	variables []*entity.Variable,
) error

type Args struct {
	RecievedMsg string
	ProcessName string
	ProcessID   string
	TopicName   string
	TaskName    string
	Messages    []*entity.Message
	Variables   []*entity.Variable
}

const (
	CTPing             string = "ping"
	CTListProcesses    string = "list processes"
	CTProcessEvent     string = "process event"
	CTListProcessFlows string = "list process flows"
)

type Call struct {
	Token     string
	ProcessID string
	CallType  string
	Args      Args
	next      *Call
	Msg       string
}

type FrontClient struct {
	url  string
	port int
	conn *grpc.ClientConn

	fnProcessEventByProcessID map[string]Callback
	idCounter                 int32

	toProcess chan *Call

	pingResponse             chan *pb.Response
	listProcessesResponse    chan *pb.Response
	listProcessFlowsResponse chan *pb.Response
	processEventResponse     chan *pb.Response

	send chan *pb.Request

	durationWaitAnswer time.Duration

	mu          *sync.Mutex
	callRoot    *Call
	callCurrent *Call

	currentToken      string
	muCalProcesses    *sync.Mutex
	muCalProcessFlows *sync.Mutex
}

func NewFrontClient(url string, port int, conn *grpc.ClientConn) *FrontClient {
	result := &FrontClient{
		url:                       url,
		port:                      port,
		conn:                      conn,
		fnProcessEventByProcessID: make(map[string]Callback),
		mu:                        &sync.Mutex{},
		toProcess:                 make(chan *Call),
		send:                      make(chan *pb.Request),
		durationWaitAnswer:        10 * time.Second,

		pingResponse:             make(chan *pb.Response),
		listProcessesResponse:    make(chan *pb.Response),
		listProcessFlowsResponse: make(chan *pb.Response),
		processEventResponse:     make(chan *pb.Response),
	}
	go func() {
		for call := range result.toProcess {
			result.mu.Lock()
			if result.callRoot == nil {
				result.callRoot = call
				result.callCurrent = result.callRoot
			} else {
				call.next = result.callCurrent
				result.callCurrent = call
			}
			result.mu.Unlock()
		}
	}()

	go func() {
		for {
			time.Sleep(time.Duration(5) * time.Millisecond)
			result.mu.Lock()
			if result.callRoot == nil {
				result.mu.Unlock()
				continue
			}
			if result.callCurrent == nil {
				result.mu.Unlock()
				continue
			}
			call := result.callRoot
			result.callRoot = call.next
			if result.callRoot == nil {
				result.callCurrent = nil
			}
			result.mu.Unlock()
			switch call.CallType {
			case CTPing:
				msg := &pb.Request{
					PingRequest: &pb.PingRequest{
						Msg: call.Msg,
					},
				}
				result.send <- msg
			case CTListProcesses:
				msg := &pb.Request{
					ListProcessesRequest: &pb.ListProcessesRequest{},
				}
				result.send <- msg
			case CTListProcessFlows:
				msg := &pb.Request{
					ListProcessFlowsRequest: &pb.ListProcessFlowsRequest{
						ProcessId: call.ProcessID,
					},
				}
				result.send <- msg
			}
		}
	}()

	return result
}

func (pc FrontClient) Pong(ctx context.Context) (bool, error) {
	logger := zap.L()
	logger.Info("Pong")

	if !pc.muCalProcesses.TryLock() {
		return false, fmt.Errorf("locked")
	}
	defer pc.muCalProcesses.Unlock()

	msg := "ping"
	pc.toProcess <- &Call{
		Msg:      msg,
		CallType: CTPing,
	}
	ticker := time.NewTicker(pc.durationWaitAnswer)
	result := false
	isComplete := false
	for {
		select {
		case <-ticker.C:
			return false, fmt.Errorf("timeout")
		case resp := <-pc.pingResponse:
			if resp.PingResponse.Msg == msg {
				result = true
			}
		}
		if isComplete {
			break
		}
	}
	return result, nil
}

func (pc FrontClient) SubscribePong(ctx context.Context, subscriber chan string) error {
	logger := zap.L()
	logger.Info("SubscribePong")

	go func() {
		for resp := range pc.pingResponse {
			if resp.PingResponse != nil {
				subscriber <- resp.PingResponse.Msg
			}
		}
	}()
	return nil
}

func (pc FrontClient) SubscribeProcessEvent(ctx context.Context, subscriber chan *Args) error {
	logger := zap.L()
	logger.Info("SubscribeProcessEvent")

	go func() {
		for resp := range pc.pingResponse {
			if resp.ProcessEventResponse != nil {
				msgs := []*entity.Message{}
				vars := []*entity.Variable{}
				if resp.ProcessEventResponse.ProcessFlow != nil {
					for i := range resp.ProcessEventResponse.ProcessFlow.Messages {
						fields := []*entity.Field{}
						for j := range resp.ProcessEventResponse.ProcessFlow.Messages[i].Fields {
							field := &entity.Field{
								Name:  resp.ProcessEventResponse.ProcessFlow.Messages[i].Fields[j].Name,
								Type:  resp.ProcessEventResponse.ProcessFlow.Messages[i].Fields[j].Type,
								Value: resp.ProcessEventResponse.ProcessFlow.Messages[i].Fields[j].Value,
							}
							fields = append(fields, field)
						}
						msg := &entity.Message{
							Name:   resp.ProcessEventResponse.ProcessFlow.Messages[i].Name,
							Fields: fields,
						}
						msgs = append(msgs, msg)
					}
					for i := range resp.ProcessEventResponse.ProcessFlow.Variables {
						varl := &entity.Variable{
							Name:  resp.ProcessEventResponse.ProcessFlow.Variables[i].Name,
							Type:  resp.ProcessEventResponse.ProcessFlow.Variables[i].Type,
							Value: resp.ProcessEventResponse.ProcessFlow.Variables[i].Value,
						}
						vars = append(vars, varl)
					}
				}
				subscriber <- &Args{
					ProcessName: resp.ProcessEventResponse.ProcessFlow.ProcessName,
					ProcessID:   resp.ProcessEventResponse.ProcessFlow.ProcessId,
					Messages:    msgs,
					Variables:   vars,
				}
			}

		}
	}()
	return nil
}

type Process struct {
	ProcessId     string
	ApplicationId string
	State         string
}

func (pc FrontClient) ListProcesses(ctx context.Context) ([]*Process, error) {
	logger := zap.L()
	logger.Info("ListProcesses")
	processes := []*Process{}

	if !pc.muCalProcesses.TryLock() {
		return nil, fmt.Errorf("locked")
	}
	defer pc.muCalProcesses.Unlock()

	pc.toProcess <- &Call{
		Token:    pc.currentToken,
		CallType: CTListProcesses,
	}
	ticker := time.NewTicker(pc.durationWaitAnswer)
	isComplete := false
	for {
		select {
		case <-ticker.C:
			return nil, fmt.Errorf("timeout")
		case resp := <-pc.listProcessesResponse:
			if resp.ListProcessesResponse.Result == "Ok" {
				for i := range resp.ListProcessesResponse.Processes {
					process := &Process{
						ProcessId:     resp.ListProcessesResponse.Processes[i].ProcessId,
						ApplicationId: resp.ListProcessesResponse.Processes[i].ApplicationId,
						State:         resp.ListProcessesResponse.Processes[i].State,
					}
					processes = append(processes, process)
				}
				isComplete = true
			}
		}
		if isComplete {
			break
		}
	}
	return processes, nil
}

type ProcessFlow struct {
	ProcessId   string
	ProcessName string
	FlowId      string
	State       string
	TopicName   string
	Messages    []*entity.Message
	Variables   []*entity.Variable
}

func (pc FrontClient) ListProcessFlows(ctx context.Context, processID string) ([]*entity.ProcessExecutorStateItem, error) {
	logger := zap.L()
	logger.Info("ListProcessFlows")

	if !pc.muCalProcessFlows.TryLock() {
		return nil, fmt.Errorf("locked")
	}
	defer pc.muCalProcessFlows.Unlock()

	processFlows := []*entity.ProcessExecutorStateItem{}

	pc.toProcess <- &Call{
		Token:     pc.currentToken,
		CallType:  CTListProcessFlows,
		ProcessID: processID,
	}
	ticker := time.NewTicker(pc.durationWaitAnswer)
	isComplete := false
	for {
		select {
		case <-ticker.C:
			return nil, fmt.Errorf("timeout")
		case resp := <-pc.listProcessFlowsResponse:
			if resp.ListProcessFlowsResponse.Result == "Ok" {
				for i := range resp.ListProcessFlowsResponse.ProcessFlows {
					processFlow := &entity.ProcessExecutorStateItem{
						ProcessID:        resp.ListProcessFlowsResponse.ProcessFlows[i].ProcessId,
						State:            resp.ListProcessFlowsResponse.ProcessFlows[i].State,
						ProcessName:      resp.ListProcessFlowsResponse.ProcessFlows[i].State,
						Execute:          resp.ListProcessFlowsResponse.ProcessFlows[i].Execute,
						ProcessStates:    resp.ListProcessFlowsResponse.ProcessFlows[i].ProcessStates,
						ProcessStateData: resp.ListProcessFlowsResponse.ProcessFlows[i].Data,
					}
					processFlows = append(processFlows, processFlow)
				}
				isComplete = true
			}
		}
		if isComplete {
			break
		}
	}
	return processFlows, nil
}

func (pc FrontClient) Login(ctx context.Context, login string, password string) (string, error) {
	logger := zap.L()
	logger.Info("Login")
	if pc.conn == nil {
		conn, err := grpc.NewClient(fmt.Sprintf("%s:%d", pc.url, pc.port), grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			logger.Error("can not connect with server", zap.Error(err))
			return "", err
		}
		pc.conn = conn
	}

	client := pb.NewFrontClientConnectorClient(pc.conn)
	resp, err := client.Login(context.Background(), &pb.LoginRequest{
		Login:    login,
		Password: password,
	})
	if err != nil {
		logger.Error("login error", zap.Error(err))
		return "", err
	}

	return resp.Token, nil
}

func (pc FrontClient) Ping(ctx context.Context, msg string) (string, error) {
	logger := zap.L()
	logger.Info("Ping")
	if pc.conn == nil {
		conn, err := grpc.NewClient(fmt.Sprintf("%s:%d", pc.url, pc.port), grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			logger.Error("can not connect with server", zap.Error(err))
			return "", err
		}
		pc.conn = conn
	}

	client := pb.NewFrontClientConnectorClient(pc.conn)

	resp, err := client.Ping(context.Background(), &pb.PingRequest{
		Msg: msg,
	})
	if err != nil {
		logger.Error("login error", zap.Error(err))
		return "", err
	}

	return resp.Msg, nil
}

func (pc FrontClient) GetListProcesses(ctx context.Context) ([]*Process, error) {
	logger := zap.L()
	logger.Info("GetListProcesses")
	processes := []*Process{}

	if pc.conn == nil {
		conn, err := grpc.NewClient(fmt.Sprintf("%s:%d", pc.url, pc.port), grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			logger.Error("can not connect with server", zap.Error(err))
			return nil, err
		}
		pc.conn = conn
	}

	client := pb.NewFrontClientConnectorClient(pc.conn)

	resp, err := client.ListProcesses(ctx, &pb.ListProcessesRequest{})
	if err != nil {
		logger.Error("list process error", zap.Error(err))
		return nil, err
	}

	if resp.Result == "Ok" {
		for i := range resp.Processes {
			process := &Process{
				ProcessId:     resp.Processes[i].ProcessId,
				ApplicationId: resp.Processes[i].ApplicationId,
				State:         resp.Processes[i].State,
			}
			processes = append(processes, process)
		}
	}
	return processes, nil
}

func (pc FrontClient) GetListProcessFlows(ctx context.Context, processID string) ([]*entity.ProcessExecutorStateItem, error) {
	logger := zap.L()
	logger.Info("GetListProcessFlows")

	if pc.conn == nil {
		conn, err := grpc.NewClient(fmt.Sprintf("%s:%d", pc.url, pc.port), grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			logger.Error("can not connect with server", zap.Error(err))
			return nil, err
		}
		pc.conn = conn
	}

	client := pb.NewFrontClientConnectorClient(pc.conn)

	resp, err := client.ListProcessFlows(ctx, &pb.ListProcessFlowsRequest{
		ProcessId: processID,
	})
	if err != nil {
		logger.Error("list process flow error", zap.Error(err))
		return nil, err
	}

	processFlows := []*entity.ProcessExecutorStateItem{}
	if resp.Result == "Ok" {
		for i := range resp.ProcessFlows {
			processFlow := &entity.ProcessExecutorStateItem{
				ProcessID:        resp.ProcessFlows[i].ProcessId,
				State:            resp.ProcessFlows[i].State,
				ProcessName:      resp.ProcessFlows[i].State,
				Execute:          resp.ProcessFlows[i].Execute,
				ProcessStates:    resp.ProcessFlows[i].ProcessStates,
				ProcessStateData: resp.ProcessFlows[i].Data,
			}
			processFlows = append(processFlows, processFlow)
		}
	}
	return processFlows, nil
}

func (pc *FrontClient) Connect(ctxConn context.Context, connected chan bool) (chan bool, error) {
	logger := zap.L()
	logger.Info("Connect")
	if pc.conn == nil {
		conn, err := grpc.NewClient(fmt.Sprintf("%s:%d", pc.url, pc.port), grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			logger.Error("can not connect with server", zap.Error(err))
			return nil, err
		}
		pc.conn = conn
	}

	client := pb.NewFrontClientConnectorClient(pc.conn)

	stream, err := client.Connect(ctxConn)
	if err != nil {
		logger.Error("open stream error", zap.Error(err))
		return nil, err
	}

	ctx := stream.Context()
	done := make(chan bool)

	go func() {
		for req := range pc.send {
			if req == nil {
				fmt.Printf("empty message\r\n")
				if err = stream.CloseSend(); err != nil {
					logger.Error("send", zap.Error(err))
				}
				return
			}
			pc.idCounter += 1
			req.Id = pc.idCounter
			err = stream.Send(req)
			if err != nil {
				logger.Error("Connect: failed send", zap.Error(err))
				stream.SendMsg(err)
				return
			}
			fmt.Printf("sended!\r\n")

			time.Sleep(time.Duration(5) * time.Microsecond)
		}
	}()

	go func() {
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				close(done)
				return
			}
			if err != nil {
				logger.Error("can not receive", zap.Error(err))
			}
			logger.Info("received", zap.Any("response", resp))
			if resp.PingResponse != nil {
				pc.pingResponse <- resp
			}
			if resp.ListProcessesResponse != nil {
				pc.listProcessesResponse <- resp
			}
			if resp.ListProcessFlowsResponse != nil {
				pc.listProcessFlowsResponse <- resp
			}
			if resp.ProcessEventResponse != nil {
				pc.processEventResponse <- resp
			}
		}
	}()

	go func() {
		<-ctx.Done()
		if err := ctx.Err(); err != nil {
			log.Println(err)
		}
		close(done)
	}()

	connected <- true
	<-done

	return done, nil
}

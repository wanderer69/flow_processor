package integration

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
	pb "github.com/wanderer69/flow_processor/pkg/proto/client"
)

type Callback func(
	ctx context.Context,
	processName string,
	processID string,
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
	CTStartProcess       string = "start process"
	CTConnectToProcess   string = "connect to process"
	CTTopicExecute       string = "topic execute"
	CTTopicComplete      string = "topic complete"
	CTSendMessage        string = "send message"
	CTExternalActivation string = "external activation"
	CTProcessFinished    string = "process finished"
)

type Call struct {
	CallType string
	Args     Args
	next     *Call
}

type ProcessorClient struct {
	url                                     string
	port                                    int
	fnTopicExecuteByProcessNameAndTopicName map[string]Callback
	fnProcessFinishedByProcessName          map[string]Callback
	fnProcessFinishedByProcessID            map[string]Callback
	idCounter                               int32

	toProcess                chan *Call
	startProcessResponse     chan *pb.Response
	connectToProcessResponse chan *pb.Response

	send chan *pb.Request

	durationWaitAnswer time.Duration

	mu              *sync.Mutex
	callRoot        *Call
	callCurrent     *Call
	processDuration int
}

func NewProcessorClient(url string, port int, processDuration int, durationWaitAnswer int) *ProcessorClient {
	result := &ProcessorClient{
		url:                                     url,
		port:                                    port,
		fnTopicExecuteByProcessNameAndTopicName: make(map[string]Callback),
		fnProcessFinishedByProcessName:          make(map[string]Callback),
		fnProcessFinishedByProcessID:            make(map[string]Callback),
		mu:                                      &sync.Mutex{},
		toProcess:                               make(chan *Call),
		startProcessResponse:                    make(chan *pb.Response),
		send:                                    make(chan *pb.Request),
		connectToProcessResponse:                make(chan *pb.Response),
		durationWaitAnswer:                      time.Duration(durationWaitAnswer) * time.Second, // 10
		processDuration:                         processDuration,
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
			time.Sleep(time.Duration(result.processDuration) * time.Millisecond)
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
			case CTStartProcess:
				msg := &pb.Request{
					StartProcessRequest: &pb.StartProcessRequest{
						ProcessName: call.Args.ProcessName,
					},
				}
				result.send <- msg
			case CTConnectToProcess:
				msg := &pb.Request{
					ConnectToProcessRequest: &pb.ConnectToProcessRequest{
						ProcessName: call.Args.ProcessName,
						ProcessId:   call.Args.ProcessID,
					},
				}
				result.send <- msg
			case CTTopicComplete:
				msgs := []*pb.Message{}
				for i := range call.Args.Messages {
					fields := []*pb.Field{}
					for j := range call.Args.Messages[i].Fields {
						field := &pb.Field{
							Name:  call.Args.Messages[i].Fields[j].Name,
							Type:  call.Args.Messages[i].Fields[j].Type,
							Value: call.Args.Messages[i].Fields[j].Value,
						}
						fields = append(fields, field)
					}
					msg := &pb.Message{
						Name:   call.Args.Messages[i].Name,
						Fields: fields,
					}
					msgs = append(msgs, msg)
				}
				vars := []*pb.Variable{}
				for i := range call.Args.Variables {
					varl := &pb.Variable{
						Name:  call.Args.Variables[i].Name,
						Type:  call.Args.Variables[i].Type,
						Value: call.Args.Variables[i].Value,
					}
					vars = append(vars, varl)
				}
				msg := &pb.Request{
					TopicComplete: &pb.TopicComplete{
						ProcessName: call.Args.ProcessName,
						ProcessId:   call.Args.ProcessID,
						TopicName:   call.Args.TopicName,
						Messages:    msgs,
						Variables:   vars,
					},
				}
				result.send <- msg
			case CTSendMessage:
				msgs := []*pb.Message{}
				for i := range call.Args.Messages {
					fields := []*pb.Field{}
					for j := range call.Args.Messages[i].Fields {
						field := &pb.Field{
							Name:  call.Args.Messages[i].Fields[j].Name,
							Type:  call.Args.Messages[i].Fields[j].Type,
							Value: call.Args.Messages[i].Fields[j].Value,
						}
						fields = append(fields, field)
					}
					msg := &pb.Message{
						Name:   call.Args.Messages[i].Name,
						Fields: fields,
					}
					msgs = append(msgs, msg)
				}
				vars := []*pb.Variable{}
				for i := range call.Args.Variables {
					varl := &pb.Variable{
						Name:  call.Args.Variables[i].Name,
						Type:  call.Args.Variables[i].Type,
						Value: call.Args.Variables[i].Value,
					}
					vars = append(vars, varl)
				}
				msg := &pb.Request{
					SendMessage: &pb.SendMessage{
						ProcessName: call.Args.ProcessName,
						ProcessId:   call.Args.ProcessID,
						TopicName:   call.Args.TopicName,
						Messages:    msgs,
						Variables:   vars,
					},
				}
				result.send <- msg
			case CTExternalActivation:
				msgs := []*pb.Message{}
				for i := range call.Args.Messages {
					fields := []*pb.Field{}
					for j := range call.Args.Messages[i].Fields {
						field := &pb.Field{
							Name:  call.Args.Messages[i].Fields[j].Name,
							Type:  call.Args.Messages[i].Fields[j].Type,
							Value: call.Args.Messages[i].Fields[j].Value,
						}
						fields = append(fields, field)
					}
					msg := &pb.Message{
						Name:   call.Args.Messages[i].Name,
						Fields: fields,
					}
					msgs = append(msgs, msg)
				}
				vars := []*pb.Variable{}
				for i := range call.Args.Variables {
					varl := &pb.Variable{
						Name:  call.Args.Variables[i].Name,
						Type:  call.Args.Variables[i].Type,
						Value: call.Args.Variables[i].Value,
					}
					vars = append(vars, varl)
				}
				msg := &pb.Request{
					ExternalActivation: &pb.ExternalActivation{
						ProcessName: call.Args.ProcessName,
						ProcessId:   call.Args.ProcessID,
						TaskName:    call.Args.TaskName,
						Messages:    msgs,
						Variables:   vars,
					},
				}
				result.send <- msg
			case CTTopicExecute:
				fn, ok := result.fnTopicExecuteByProcessNameAndTopicName[call.Args.ProcessName+call.Args.TopicName]
				if !ok {
					fmt.Printf("Error!!!!!!!!!!!!")
				}
				go func() {
					ctx := context.Background()
					err := fn(ctx, call.Args.ProcessName, call.Args.ProcessID, call.Args.TopicName, call.Args.Messages, call.Args.Variables)
					if err != nil {
						fmt.Printf("failed call topic handler")
					}
				}()
			case CTProcessFinished:
				fn, ok := result.fnProcessFinishedByProcessID[call.Args.ProcessID]
				if ok {
					go func() {
						ctx := context.Background()
						err := fn(ctx, call.Args.ProcessName, call.Args.ProcessID, call.Args.TopicName, call.Args.Messages, call.Args.Variables)
						if err != nil {
							fmt.Printf("failed call topic handler")
						}
					}()
				}
			}
		}
	}()

	return result
}

func (pc ProcessorClient) SetCallback(ctx context.Context, processName string, topicName string, fn Callback) error {
	logger := zap.L()
	logger.Info("SetCallback")
	conn, err := grpc.NewClient(fmt.Sprintf("%s:%d", pc.url, pc.port), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Error("can not connect with server", zap.Error(err))
		return err
	}
	client := pb.NewClientConnectorClient(conn)
	resp, err := client.SetHandler(ctx, &pb.SetHandlerRequest{
		ProcessName: processName,
		TopicName:   topicName,
	})
	if err != nil {
		logger.Error("failed call set handler", zap.Error(err))
		return err
	}

	if resp.Error != nil {
		logger.Error("error when call set handler", zap.String("error", *resp.Error))
		return err
	}
	pc.fnTopicExecuteByProcessNameAndTopicName[processName+topicName] = fn
	return nil
}

func (pc ProcessorClient) AddProcess(ctx context.Context, processRaw string) error {
	logger := zap.L()
	logger.Info("AddProcess")
	conn, err := grpc.NewClient(fmt.Sprintf("%s:%d", pc.url, pc.port), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Error("can not connect with server", zap.Error(err))
		return err
	}
	client := pb.NewClientConnectorClient(conn)
	resp, err := client.AddProcess(ctx, &pb.AddProcessRequest{
		Process: processRaw,
	})
	if err != nil {
		logger.Error("failed call add process", zap.Error(err))
		return err
	}

	if resp.Error != nil {
		logger.Error("error when call add process", zap.String("error", *resp.Error))
		return err
	}
	return nil
}

func (pc ProcessorClient) StartProcess(ctx context.Context, processName string, vars []*entity.Variable) (string, error) {
	logger := zap.L()
	logger.Info("StartProcess")

	pc.toProcess <- &Call{
		CallType: CTStartProcess,
		Args: Args{
			ProcessName: processName,
			Variables:   vars,
		},
	}
	processID := ""
	ticker := time.NewTicker(pc.durationWaitAnswer)
	isComplete := false
	for {
		select {
		case <-ticker.C:
			return "", fmt.Errorf("timeout")
		case resp := <-pc.startProcessResponse:
			processID = resp.StartProcessResponse.ProcessId
			isComplete = true
		}
		if isComplete {
			break
		}
	}
	handler, ok := pc.fnProcessFinishedByProcessName[processName]
	if ok {
		pc.fnProcessFinishedByProcessID[processID] = handler
	}
	return processID, nil
}

func (pc ProcessorClient) ConnectToProcess(ctx context.Context, processName string, processID string) (string, error) {
	logger := zap.L()
	logger.Info("ConnectToProcess")

	pc.toProcess <- &Call{
		CallType: CTConnectToProcess,
		Args: Args{
			ProcessName: processName,
			ProcessID:   processID,
		},
	}

	ticker := time.NewTicker(pc.durationWaitAnswer)
	isComplete := false
	for {
		select {
		case <-ticker.C:
			return "", fmt.Errorf("timeout")
		case resp := <-pc.connectToProcessResponse:
			processID = resp.ConnectToProcessResponse.ProcessId
			isComplete = true
		}
		if isComplete {
			break
		}
	}

	return processID, nil
}

func (pc ProcessorClient) TopicComplete(ctx context.Context, processName string, processID string, topicName string, msgs []*entity.Message, vars []*entity.Variable) {
	logger := zap.L()
	logger.Info("TopicComplete")

	pc.toProcess <- &Call{
		CallType: CTTopicComplete,
		Args: Args{
			ProcessName: processName,
			ProcessID:   processID,
			TopicName:   topicName,
			Messages:    msgs,
			Variables:   vars,
		},
	}
}

func (pc ProcessorClient) SendMessage(ctx context.Context, processName string, processID string, topicName string, msgs []*entity.Message, vars []*entity.Variable) {
	logger := zap.L()
	logger.Info("SendMessage")

	pc.toProcess <- &Call{
		CallType: CTSendMessage,
		Args: Args{
			ProcessName: processName,
			ProcessID:   processID,
			TopicName:   topicName,
			Messages:    msgs,
			Variables:   vars,
		},
	}
}

func (pc ProcessorClient) ExternalActivation(ctx context.Context, processName string, processID string, taskName string, msgs []*entity.Message, vars []*entity.Variable) {
	logger := zap.L()
	logger.Info("ExternalActivation")

	pc.toProcess <- &Call{
		CallType: CTExternalActivation,
		Args: Args{
			ProcessName: processName,
			ProcessID:   processID,
			TaskName:    taskName,
			Messages:    msgs,
			Variables:   vars,
		},
	}
}

func (pc ProcessorClient) SetProcessFinished(ctx context.Context, processName string, fn Callback) error {
	logger := zap.L()
	logger.Info("SetProcessFinished")
	pc.fnProcessFinishedByProcessName[processName] = fn
	return nil
}

func (pc ProcessorClient) GetList(ctx context.Context, processName string) ([]string, error) {
	logger := zap.L()
	logger.Info("GetList")
	return []string{}, nil
}

func (pc *ProcessorClient) Connect(processName string, connected chan bool) error {
	logger := zap.L()
	logger.Info("Connect")
	conn, err := grpc.NewClient(fmt.Sprintf("%s:%d", pc.url, pc.port), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Error("can not connect with server", zap.Error(err))
		return err
	}

	client := pb.NewClientConnectorClient(conn)

	stream, err := client.Connect(context.Background())
	if err != nil {
		logger.Error("open stream error", zap.Error(err))
		return err
	}

	ctx := stream.Context()
	done := make(chan bool)

	go func() {
		for req := range pc.send {
			if req == nil {
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

			time.Sleep(time.Duration(pc.processDuration) * time.Microsecond)
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
			if resp.StartProcessResponse != nil {
				pc.startProcessResponse <- resp
			}
			if resp.ConnectToProcessResponse != nil {
				pc.connectToProcessResponse <- resp
			}
			if resp.TopicExecute != nil {
				msgs := []*entity.Message{}
				for i := range resp.TopicExecute.Messages {
					fields := []*entity.Field{}
					for j := range resp.TopicExecute.Messages[i].Fields {
						field := &entity.Field{
							Name:  resp.TopicExecute.Messages[i].Fields[j].Name,
							Type:  resp.TopicExecute.Messages[i].Fields[j].Type,
							Value: resp.TopicExecute.Messages[i].Fields[j].Value,
						}
						fields = append(fields, field)
					}
					msg := &entity.Message{
						Name:   resp.TopicExecute.Messages[i].Name,
						Fields: fields,
					}
					msgs = append(msgs, msg)
				}
				vars := []*entity.Variable{}
				for i := range resp.TopicExecute.Variables {
					varl := &entity.Variable{
						Name:  resp.TopicExecute.Variables[i].Name,
						Type:  resp.TopicExecute.Variables[i].Type,
						Value: resp.TopicExecute.Variables[i].Value,
					}
					vars = append(vars, varl)
				}

				pc.toProcess <- &Call{
					CallType: CTTopicExecute,
					Args: Args{
						ProcessName: resp.TopicExecute.ProcessName,
						ProcessID:   resp.TopicExecute.ProcessId,
						TopicName:   resp.TopicExecute.TopicName,
						Messages:    msgs,
						Variables:   vars,
					},
				}
			}
			if resp.ProcessFinished != nil {
				vars := []*entity.Variable{}
				for i := range resp.ProcessFinished.Variables {
					varl := &entity.Variable{
						Name:  resp.ProcessFinished.Variables[i].Name,
						Type:  resp.ProcessFinished.Variables[i].Type,
						Value: resp.ProcessFinished.Variables[i].Value,
					}
					vars = append(vars, varl)
				}

				pc.toProcess <- &Call{
					CallType: CTProcessFinished,
					Args: Args{
						ProcessName: resp.ProcessFinished.ProcessName,
						ProcessID:   resp.ProcessFinished.ProcessId,
						Variables:   vars,
					},
				}
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
	return nil
}

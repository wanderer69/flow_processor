package integration

import (
	"context"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/wanderer69/flow_processor/pkg/entity"
	pb "github.com/wanderer69/flow_processor/pkg/proto"
	"go.uber.org/zap"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

/*
type Callback func(
	ctx context.Context,
	id int32,
	recievedMsg string,
	processName string,
	processID string,
	topicName string,
	messages []*entity.Message,
	variables []*entity.Variable,
	sendMsg func(id int32, msg string, processName, processID, topicName string,
		messages []*entity.Message, variables []*entity.Variable, result string, err string) error) (int32, string, string, string, string, string, []*entity.Message, []*entity.Variable, error)
*/
/*
type Callback func(
	ctx context.Context,
	id int32,
	recievedMsg string,
	processName string,
	processID string,
	topicName string,
	messages []*entity.Message,
	variables []*entity.Variable,
) (int32, string, string, string, string, string, []*entity.Message, []*entity.Variable, error)
*/

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
	Messages    []*entity.Message
	Variables   []*entity.Variable
}

const (
	CTStartProcess     string = "start process"
	CTConnectToProcess string = "connect to process"
	CTTopicExecute     string = "topic execute"
	CTTopicComplete    string = "topic complete"
	CTSendMessage      string = "send message"
)

type Call struct {
	CallType string
	Args     Args
	next     *Call
}

type ProcessorClient struct {
	port                                    int
	fnTopicExecuteByProcessNameAndTopicName map[string]Callback
	processConnector                        map[string]string
	idCounter                               int32

	toProcess            chan *Call
	startProcessResponse chan *pb.Response
	//connectToProcessRequest  chan *Call
	connectToProcessResponse chan *pb.Response

	//topicExecute chan *Call

	send chan *pb.Request

	durationWaitAnswer time.Duration

	mu          *sync.Mutex
	callRoot    *Call
	callCurrent *Call
}

func NewProcessorClient(port int) *ProcessorClient {
	result := &ProcessorClient{
		port:                                    port,
		fnTopicExecuteByProcessNameAndTopicName: make(map[string]Callback),
		mu:                                      &sync.Mutex{},
		toProcess:                               make(chan *Call),
		startProcessResponse:                    make(chan *pb.Response),
		send:                                    make(chan *pb.Request),
		//connectToProcessRequest:                 make(chan *Call),
		connectToProcessResponse: make(chan *pb.Response),
		durationWaitAnswer:       10 * time.Second,
		//topicExecute:                            make(chan *Call),
	}
	go func() {
		for {
			select {
			case call := <-result.toProcess:
				result.mu.Lock()
				if result.callRoot == nil {
					result.callRoot = call
					result.callCurrent = result.callRoot
				} else {
					call.next = result.callCurrent
					result.callCurrent = call
				}
				result.mu.Unlock()
				/*
					case exec := <-result.topicExecute:
						result.mu.Lock()
						if result.callRoot == nil {
							result.callRoot = exec
							result.callCurrent = result.callRoot
						} else {
							exec.next = result.callCurrent
							result.callCurrent = exec
						}
						result.mu.Unlock()
					case exec := <-result.topicComplete:
						result.mu.Lock()
						if result.callRoot == nil {
							result.callRoot = exec
							result.callCurrent = result.callRoot
						} else {
							exec.next = result.callCurrent
							result.callCurrent = exec
						}
						result.mu.Unlock()
				*/
			}
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
				/*
					msg := &pb.Request{
						StartProcessRequest: &pb.StartProcessRequest{
							ProcessName: call.Args.ProcessName,
						},
					}
					result.send <- msg
				*/
			}
		}
	}()

	return result
}

func (pc ProcessorClient) SetCallback(ctx context.Context, processName string, topicName string, fn Callback) error {
	logger := zap.L()
	logger.Info("SetCallback")
	// 50005
	conn, err := grpc.NewClient(fmt.Sprintf(":%d", pc.port), grpc.WithTransportCredentials(insecure.NewCredentials()))
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
	logger.Info("StoreProcess")
	// 50005
	conn, err := grpc.NewClient(fmt.Sprintf(":%d", pc.port), grpc.WithTransportCredentials(insecure.NewCredentials()))
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

func (pc ProcessorClient) StartProcess(ctx context.Context, processName string) (string, error) {
	logger := zap.L()
	logger.Info("StartProcess")

	pc.toProcess <- &Call{
		CallType: CTStartProcess,
		Args: Args{
			ProcessName: processName,
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

	//processID := ""
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

func (pc *ProcessorClient) Connect(processName string, connected chan bool) error {
	logger := zap.L()
	logger.Info("Connect")
	// 50005
	conn, err := grpc.NewClient(fmt.Sprintf(":%d", pc.port), grpc.WithTransportCredentials(insecure.NewCredentials()))
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
		for {
			select {
			case req := <-pc.send:
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
			}
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
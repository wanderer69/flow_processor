package clientconnector

import (
	"context"
	"fmt"
	"io"
	"net"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/google/uuid"
	camunda7convertor "github.com/wanderer69/flow_processor/pkg/camunda_7_convertor"
	"github.com/wanderer69/flow_processor/pkg/entity"
	internalformat "github.com/wanderer69/flow_processor/pkg/internal_format"
	"github.com/wanderer69/flow_processor/pkg/loader"
	pb "github.com/wanderer69/flow_processor/pkg/proto"
)

type Handler struct {
	sendExecuteTopic func(
		ctx context.Context,
		processName string,
		processID string,
		topicName string,
		messages []*entity.Message,
		variables []*entity.Variable,
	) error
	/*
		sendConnectToProcessResponse func(
			ctx context.Context,
			processName string,
			processID string,
			result string,
			err string,
		) error
		sendStartProcessResponse func(
			ctx context.Context,
			processName string,
			processID string,
			result string,
			err string,
		) error
	*/
	handlerID string
}

type Server struct {
	pb.UnimplementedClientConnectorServer
	topicClient                       ExternalTopic
	externalActivationClient          ExternalActivation
	processExecutor                   ProcessExecutor
	handlersByProcessNameAndTopicName map[string]*Handler
	idCounter                         int32
}

func NewServer(
	topicClient ExternalTopic,
	externalActivationClient ExternalActivation,
	processExecutor ProcessExecutor,
) *Server {
	return &Server{
		topicClient:                       topicClient,
		externalActivationClient:          externalActivationClient,
		processExecutor:                   processExecutor,
		handlersByProcessNameAndTopicName: make(map[string]*Handler),
	}
}

func (s *Server) AddProcess(ctx context.Context, in *pb.AddProcessRequest) (*pb.AddProcessResponse, error) {
	camunda7Client := camunda7convertor.NewConverterClient()
	internalFormatClient := internalformat.NewInternalFormat()
	ldr := loader.NewLoader(camunda7Client, internalFormatClient)
	process, err := ldr.Load(in.Process)
	if err != nil {
		errorRaw := fmt.Sprintf("Loader: %v", err)
		return &pb.AddProcessResponse{
			Error: &errorRaw,
		}, nil
	}

	s.processExecutor.AddProcess(ctx, process)
	return &pb.AddProcessResponse{}, nil
}

func (s *Server) SetHandler(ctx context.Context, in *pb.SetHandlerRequest) (*pb.SetHandlerResponse, error) {
	logger := zap.L()
	logger.Info("SetHandler")

	if in == nil {
		err := "empty request"
		return &pb.SetHandlerResponse{
			Error: &err,
		}, nil
	}

	handlerID := uuid.NewString()
	s.handlersByProcessNameAndTopicName[in.ProcessName] = &Handler{
		handlerID: handlerID,
	}
	fn := func(processName, processID string, topicName string, msgs []*entity.Message, vars []*entity.Variable) error {
		handler, ok := s.handlersByProcessNameAndTopicName[processName]
		if !ok {
			return fmt.Errorf("topic handler not found : %v %v", processName, topicName)
		}

		if handler.sendExecuteTopic == nil {
			return fmt.Errorf("send func in handler not found : %v %v", processName, topicName)
		}
		s.idCounter += 1
		return handler.sendExecuteTopic(ctx, processName, processID, topicName, msgs, vars)
	}
	s.topicClient.SetTopicHandler(ctx, in.ProcessName, in.TopicName, fn)

	return &pb.SetHandlerResponse{
		HandlerId: handlerID,
	}, nil
}

//ConnectToProcess(ctx context.Context, in *ConnectToProcessRequest, opts ...grpc.CallOption) (*ConnectToProcessResponse, error)

func (s *Server) Connect(srv pb.ClientConnector_ConnectServer) error {
	logger := zap.L()
	logger.Info("Connect")

	ctx := srv.Context()

	sendExecuteTopic := func(
		ctx context.Context,
		processName string,
		processID string,
		topicName string,
		messages []*entity.Message,
		variables []*entity.Variable,
	) error {
		s.idCounter += 1
		msgs := []*pb.Message{}
		for i := range messages {
			flds := []*pb.Field{}
			for j := range messages[i].Fields {
				fld := &pb.Field{
					Name:  messages[i].Fields[j].Name,
					Type:  messages[i].Fields[j].Type,
					Value: messages[i].Fields[j].Value,
				}
				flds = append(flds, fld)
			}
			msg := &pb.Message{
				Name:   messages[i].Name,
				Fields: flds,
			}
			msgs = append(msgs, msg)
		}
		vars := []*pb.Variable{}
		for i := range variables {
			v := &pb.Variable{
				Name:  variables[i].Name,
				Type:  variables[i].Type,
				Value: variables[i].Value,
			}
			vars = append(vars, v)
		}
		resp := pb.Response{
			Id:  s.idCounter,
			Msg: "",
			TopicExecute: &pb.TopicExecute{
				ProcessName: processName,
				ProcessId:   processID,
				TopicName:   topicName,
				Messages:    msgs,
				Variables:   vars,
			},
		}
		if err := srv.Send(&resp); err != nil {
			logger.Info("Connect: done", zap.Error(err))
			return err
		}
		return nil
	}

	//state := 0
	for {
		// exit if context is done
		// or continue
		select {
		case <-ctx.Done():
			err := ctx.Err()
			logger.Info("Connect: done", zap.Error(err))
			return err
		default:
		}

		// receive data from stream
		req, err := srv.Recv()
		if err == io.EOF {
			// return will close stream from server side
			logger.Info("Connect: exit")
			return nil
		}
		if err != nil {
			logger.Info("Connect: failed recieve", zap.Error(err))
			continue
		}
		/*
			// анализ в зависимости от состояния - state = 0 может быть либо подключение к процессу либо старт процесса state = 1 ответ от топика или отправка сообщения
			if req.ConnectToProcessRequest != nil {

			}
		*/
		if req.StartProcessRequest != nil {
			// подключаем обработчики
			handler, ok := s.handlersByProcessNameAndTopicName[req.StartProcessRequest.ProcessName]
			if !ok {
				logger.Info("Connect: handler not found", zap.String("process_name", req.StartProcessRequest.ProcessName))
				continue
			}
			handler.sendExecuteTopic = sendExecuteTopic

			errorResult := ""
			result := "Ok"
			processID, err := s.processExecutor.StartProcess(ctx, req.StartProcessRequest.ProcessName, nil)
			if err != nil {
				result = "Error"
				errorResult = fmt.Sprintf("failed start process %v", err)
			}
			s.idCounter += 1

			resp := pb.Response{
				Id:  s.idCounter,
				Msg: "",
				StartProcessResponse: &pb.StartProcessResponse{
					ProcessName: req.StartProcessRequest.ProcessName,
					ProcessId:   processID,
					Result:      result,
					Error:       &errorResult,
				},
			}
			if err := srv.Send(&resp); err != nil {
				logger.Info("Connect: done", zap.Error(err))
				return err
			}
		}
		if req.TopicComplete != nil {
			msgs := []*entity.Message{}
			for i := range req.TopicComplete.Messages {
				flds := []*entity.Field{}
				for j := range req.TopicComplete.Messages[i].Fields {
					fld := &entity.Field{
						Name:  req.TopicComplete.Messages[i].Fields[j].Name,
						Type:  req.TopicComplete.Messages[i].Fields[j].Type,
						Value: req.TopicComplete.Messages[i].Fields[j].Value,
					}
					flds = append(flds, fld)
				}
				msg := &entity.Message{
					Name:   req.TopicComplete.Messages[i].Name,
					Fields: flds,
				}
				msgs = append(msgs, msg)
			}
			vars := []*entity.Variable{}
			for i := range req.TopicComplete.Variables {
				v := &entity.Variable{
					Name:  req.TopicComplete.Variables[i].Name,
					Type:  req.TopicComplete.Variables[i].Type,
					Value: req.TopicComplete.Variables[i].Value,
				}
				vars = append(vars, v)
			}
			s.topicClient.CompleteTopic(ctx, req.TopicComplete.ProcessName, req.TopicComplete.ProcessId, req.TopicComplete.TopicName,
				msgs, vars)
		}

		if req.SendMessage != nil {
			msgs := []*entity.Message{}
			for i := range req.SendMessage.Messages {
				flds := []*entity.Field{}
				for j := range req.SendMessage.Messages[i].Fields {
					fld := &entity.Field{
						Name:  req.SendMessage.Messages[i].Fields[j].Name,
						Type:  req.SendMessage.Messages[i].Fields[j].Type,
						Value: req.SendMessage.Messages[i].Fields[j].Value,
					}
					flds = append(flds, fld)
				}
				msg := &entity.Message{
					Name:   req.SendMessage.Messages[i].Name,
					Fields: flds,
				}
				msgs = append(msgs, msg)
			}
			vars := []*entity.Variable{}
			for i := range req.SendMessage.Variables {
				v := &entity.Variable{
					Name:  req.SendMessage.Variables[i].Name,
					Type:  req.SendMessage.Variables[i].Type,
					Value: req.SendMessage.Variables[i].Value,
				}
				vars = append(vars, v)
			}
			s.topicClient.CompleteTopic(ctx, req.SendMessage.ProcessName, req.SendMessage.ProcessId, req.SendMessage.TopicName,
				msgs, vars)
		}

		/*
			if s.fn != nil {
				// обработка данных стрима
				id, msg, err := s.fn(ctx, req.Id, req.Msg, send)
				if err != nil {
					logger.Info("Connect: failed recieve", zap.Error(err))
					continue
				}
				if len(msg) > 0 {
					err = send(id, msg)
					if err != nil {
						logger.Info("Connect: failed recieve", zap.Error(err))
						continue
					}
				}
			}
		*/
		time.Sleep(time.Duration(1) * time.Millisecond)
	}
}

func ServerConnect(port int, topicClient ExternalTopic, externalActivationClient ExternalActivation, processExecutor ProcessExecutor) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	srv := NewServer(topicClient, externalActivationClient, processExecutor)
	pb.RegisterClientConnectorServer(s, srv)

	if err := s.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve: %v", err)
	}

	return nil
}

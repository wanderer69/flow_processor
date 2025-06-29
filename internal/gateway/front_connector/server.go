package frontconnector

import (
	//"github.com/wanderer69/flow_processor/pkg/process"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/wanderer69/flow_processor/pkg/entity"
	pb "github.com/wanderer69/flow_processor/pkg/proto/front"
	wasmws "github.com/wanderer69/flow_processor/pkg/ws"
)

type Token struct {
	Token     string
	Login     string
	CreatedAt time.Time
}

type Server struct {
	pb.UnimplementedFrontClientConnectorServer
	processExecutor ProcessExecutor
	storeClient     StoreClient
	idCounter       int32
	frontUser       FrontUser

	sessionTokensByLogin map[string]*Token
	sessionTokensByToken map[string]*Token
	processDuration      int
	token                string
}

func NewServer(
	processExecutor ProcessExecutor,
	storeClient StoreClient,
	frontUser FrontUser,
	processDuration int,
) *Server {
	return &Server{
		processExecutor:      processExecutor,
		storeClient:          storeClient,
		frontUser:            frontUser,
		sessionTokensByLogin: make(map[string]*Token),
		sessionTokensByToken: make(map[string]*Token),
		processDuration:      processDuration,
	}
}

func (s *Server) GetToken() string {
	return s.token
}

func (s *Server) Login(ctx context.Context, in *pb.LoginRequest) (*pb.LoginResponse, error) {
	logger := zap.L()
	logger.Info("Login")

	result := &pb.LoginResponse{}
	err := s.frontUser.Get(ctx, in.Login, in.Password)
	if err != nil {
		errMsg := err.Error()
		result.Error = &errMsg
		return result, nil
	}
	tokenData := uuid.NewString()
	s.token = tokenData
	token := &Token{
		Token:     tokenData,
		Login:     in.Login,
		CreatedAt: time.Now(),
	}

	s.sessionTokensByLogin[in.Login] = token
	s.sessionTokensByToken[tokenData] = token
	result.Token = tokenData

	return result, nil
}

func (s *Server) Ping(ctx context.Context, in *pb.PingRequest) (*pb.PingResponse, error) {
	logger := zap.L()
	logger.Info("Ping")

	result := &pb.PingResponse{Msg: in.Msg}

	return result, nil
}

func (s *Server) ListProcesses(ctx context.Context, in *pb.ListProcessesRequest) (*pb.ListProcessesResponse, error) {
	logger := zap.L()
	logger.Info("ListProcesses")

	errorResult := ""
	result := "Ok"
	storeProcesses, err := s.storeClient.LoadProcessStates(ctx, s.processExecutor.GetProcessExecutor().UUID)
	if err != nil {
		errorResult = err.Error()
		result = "Error"
	}

	processes := []*pb.Process{}
	for i := range storeProcesses {
		process := pb.Process{
			ProcessId: storeProcesses[i].ProcessID,
			State:     storeProcesses[i].State,
		}
		processes = append(processes, &process)
	}
	listProcessesResponse := &pb.ListProcessesResponse{
		Processes: processes,
		Error:     &errorResult,
		Result:    result,
	}

	return listProcessesResponse, nil
}

func (s *Server) ListProcessFlows(ctx context.Context, in *pb.ListProcessFlowsRequest) (*pb.ListProcessFlowsResponse, error) {
	logger := zap.L()
	logger.Info("ListProcessFlows")

	errorResult := ""
	result := "Ok"
	processStates, err := s.storeClient.LoadProcessStates(ctx, s.processExecutor.GetProcessExecutor().UUID)
	if err != nil {
		errorResult = err.Error()
		result = "Error"
	}

	processes := []*pb.ProcessFlow{}

	for i := range processStates {
		process := pb.ProcessFlow{
			ProcessId:   processStates[i].ProcessID,
			State:       processStates[i].State,
			ProcessName: processStates[i].ProcessName,
			Execute:     processStates[i].Execute,
			Data:        processStates[i].ProcessStateData,
		}
		processes = append(processes, &process)
	}

	listProcessFlowsResponse := &pb.ListProcessFlowsResponse{
		ProcessFlows: processes,
		Error:        &errorResult,
		Result:       result,
	}

	return listProcessFlowsResponse, nil
}

func (s *Server) Connect(srv pb.FrontClientConnector_ConnectServer) error {
	logger := zap.L()
	logger.Info("Connect")

	ctx := srv.Context()

	sendPingResponse := func(
		msg string,
	) error {
		s.idCounter += 1
		resp := pb.Response{
			Id:  s.idCounter,
			Msg: "",
			PingResponse: &pb.PingResponse{
				Msg: msg,
			},
		}
		if err := srv.Send(&resp); err != nil {
			logger.Info("Connect: done", zap.Error(err))
			return err
		}
		return nil
	}

	sendListProcessesResponse := func(
		storeProcesses []*entity.ProcessExecutorStateItem,
		result string,
		errorMsg string,
	) error {
		s.idCounter += 1
		processes := []*pb.Process{}
		for i := range storeProcesses {
			process := pb.Process{
				ProcessId: storeProcesses[i].ProcessID,
				State:     storeProcesses[i].State,
			}
			processes = append(processes, &process)
		}
		listProcessesResponse := pb.ListProcessesResponse{
			Processes: processes,
			Error:     &errorMsg,
			Result:    result,
		}
		resp := pb.Response{
			Id:                    s.idCounter,
			Msg:                   "",
			ListProcessesResponse: &listProcessesResponse,
		}
		if err := srv.Send(&resp); err != nil {
			logger.Info("Connect: done", zap.Error(err))
			return err
		}
		return nil
	}

	sendListProcessFlowsResponse := func(
		processStates []*entity.ProcessExecutorStateItem,
		result string,
		errorMsg string,
	) error {
		s.idCounter += 1
		processes := []*pb.ProcessFlow{}

		for i := range processStates {
			process := pb.ProcessFlow{
				ProcessId:   processStates[i].ProcessID,
				State:       processStates[i].State,
				ProcessName: processStates[i].ProcessName,
				Execute:     processStates[i].Execute,
				Data:        processStates[i].ProcessStateData,
			}
			processes = append(processes, &process)
		}

		listProcessFlowsResponse := pb.ListProcessFlowsResponse{
			ProcessFlows: processes,
			Error:        &errorMsg,
			Result:       result,
		}
		resp := pb.Response{
			Id:                       s.idCounter,
			Msg:                      "",
			ListProcessFlowsResponse: &listProcessFlowsResponse,
		}
		if err := srv.Send(&resp); err != nil {
			logger.Info("Connect: done", zap.Error(err))
			return err
		}
		return nil
	}

	for {
		// exit if context is done
		// or continue
		select {
		case <-ctx.Done():
			err := ctx.Err()
			logger.Info("Connect: done", zap.Error(err))
			return err
		case finishedProcessData := <-s.processExecutor.GetStopped():
			resp := pb.Response{
				Id:  s.idCounter,
				Msg: "",
				ProcessEventResponse: &pb.ProcessEventResponse{
					ProcessId: finishedProcessData.ProcessID,
					// State: process.St,
				},
			}
			if err := srv.Send(&resp); err != nil {
				logger.Info("Connect: send failed", zap.Error(err))
			}
			logger.Info("Connect: finished")
		default:
		}

		fmt.Printf("%#v, %v\r\n", ctx, time.Now())
		err := srv.Send(&pb.Response{
			Id:  int32(0) >> 1,
			Msg: "start",
		})
		if err != nil {
			logger.Info("Connect: failed send", zap.Error(err))
			continue
		}
		// receive data from stream
		req, err := srv.Recv()
		if err == io.EOF {
			// return will close stream from server side
			logger.Info("Connect: exit")
			return nil
		}
		fmt.Printf("%#v, %v\r\n", ctx, time.Now())
		if err != nil {
			logger.Info("Connect: failed recieve", zap.Error(err))
			continue
		}

		if req.PingRequest != nil {
			err = sendPingResponse(req.PingRequest.Msg)
			if err != nil {
				logger.Info("Connect: failed send", zap.Error(err))
				continue
			}
		}

		if req.ListProcessesRequest != nil {
			errorResult := ""
			result := "Ok"
			processExecutorStateItems, err := s.storeClient.LoadProcessStates(ctx, s.processExecutor.GetProcessExecutor().UUID)
			if err != nil {
				errorResult = err.Error()
				result = "Error"
			}
			err = sendListProcessesResponse(processExecutorStateItems, result, errorResult)
			if err != nil {
				logger.Info("Connect: failed send", zap.Error(err))
				continue
			}
		}

		if req.ListProcessFlowsRequest != nil {
			errorResult := ""
			result := "Ok"
			processExecutorStateItems, err := s.storeClient.LoadProcessStates(ctx, s.processExecutor.GetProcessExecutor().UUID)
			if err != nil {
				errorResult = err.Error()
				result = "Error"
			}
			err = sendListProcessFlowsResponse(processExecutorStateItems, result, errorResult)
			if err != nil {
				logger.Info("Connect: failed send", zap.Error(err))
				continue
			}
		}

		time.Sleep(time.Duration(s.processDuration) * time.Millisecond)
	}
}

func ServerConnect(port int, processExecutor ProcessExecutor, storeClient StoreClient, frontUser FrontUser, processDuration int) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	srv := NewServer(processExecutor, storeClient, frontUser, processDuration)
	pb.RegisterFrontClientConnectorServer(s, srv)

	if err := s.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve: %v", err)
	}

	return nil
}

var (
	errMissingMetadata = status.Errorf(codes.InvalidArgument, "missing metadata")
	errInvalidToken    = status.Errorf(codes.Unauthenticated, "invalid token")
)

func valid(expected string, authorization []string) bool {
	if len(authorization) < 1 {
		return false
	}
	token := strings.TrimPrefix(authorization[0], "Bearer ")
	return token == expected
}

func ServerConnectWS(
	appCtx context.Context,
	appCancel context.CancelFunc,
	router *http.ServeMux,
	processExecutor ProcessExecutor,
	storeClient StoreClient,
	frontUser FrontUser,
	processDuration int,
	start chan struct{},
) (*grpc.Server, error) {
	logger := zap.L()

	srv := NewServer(processExecutor, storeClient, frontUser, processDuration)

	//Setup HTTP / Websocket server
	wsl := wasmws.NewWebSocketListener(appCtx)
	router.HandleFunc("/grpc-proxy", wsl.ServeHTTP)

	//gRPC setup
	creds, err := credentials.NewServerTLSFromFile("cert.pem", "key.pem")
	if err != nil {
		logger.Error("ServerConnectWS: failed get TSL credentials from {cert,key}.pem", zap.Error(err))
		return nil, fmt.Errorf("failed to contruct gRPC TSL credentials from {cert,key}.pem: %w", err)
	}
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(
			func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
				if info.FullMethod != "/frontprocessorclient.FrontClientConnector/Login" &&
					info.FullMethod != "/frontprocessorclient.FrontClientConnector/Ping" {
					md, ok := metadata.FromIncomingContext(ctx)
					if !ok {
						return nil, errMissingMetadata
					}

					if !valid(srv.GetToken(), md["authorization"]) {
						return nil, errInvalidToken
					}
				}
				// Continue execution of handler after ensuring a valid token.
				return handler(ctx, req)
			}), grpc.Creds(creds))
	pb.RegisterFrontClientConnectorServer(grpcServer, srv)

	//Run gRPC server
	go func() {
		defer appCancel()
		<-start
		if err := grpcServer.Serve(wsl); err != nil {
			logger.Error("ServerConnectWS: Failed to serve gRPC connections", zap.Error(err))
		}
	}()

	return grpcServer, nil
}

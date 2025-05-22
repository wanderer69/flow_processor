package frontconnector

import (
	//"github.com/wanderer69/flow_processor/pkg/process"
	"context"
	"embed"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/wanderer69/flow_processor/pkg/entity"
	pb "github.com/wanderer69/flow_processor/pkg/proto/front"
	"go.uber.org/zap"
	"google.golang.org/grpc"
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

func (s *Server) Connect(srv pb.FrontClientConnector_ConnectServer) error {
	logger := zap.L()
	logger.Info("Connect")

	ctx := srv.Context()

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
		//		ctx context.Context,
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

func ServerConnectWeb(port int, processExecutor ProcessExecutor, storeClient StoreClient, frontUser FrontUser, fsSPA embed.FS, processDuration int) error {
	logger := zap.L()
	gs := grpc.NewServer()
	srv := NewServer(processExecutor, storeClient, frontUser, processDuration)
	pb.RegisterFrontClientConnectorServer(gs, srv)
	wrappedServer := grpcweb.WrapServer(gs)

	handler := func(resp http.ResponseWriter, req *http.Request) {
		// Redirect gRPC and gRPC-Web requests to the gRPC-Web Websocket Proxy server
		if req.ProtoMajor == 2 && strings.Contains(req.Header.Get("Content-Type"), "application/grpc") {
			wrappedServer.ServeHTTP(resp, req)
			return
		}

		// Serve the WASM client
		wasmContentTypeSetter(http.FileServer(http.FS(fsSPA))).ServeHTTP(resp, req)
	}

	addr := "localhost:10000"
	httpsSrv := &http.Server{
		Addr:    addr,
		Handler: http.HandlerFunc(handler),
		// Some security settings
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	logger.Info("Serving on https://" + addr)
	return httpsSrv.ListenAndServe()
}

func wasmContentTypeSetter(fn http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if strings.Contains(req.URL.Path, ".wasm") {
			w.Header().Set("content-type", "application/wasm")
		} else {
			if strings.Contains(req.URL.Path, ".css") {
				w.Header().Set("content-type", "text/css")
			}
		}
		fn.ServeHTTP(w, req)
	}
}

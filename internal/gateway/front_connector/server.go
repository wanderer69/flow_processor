package frontconnector

import (
	//"github.com/wanderer69/flow_processor/pkg/process"
	"context"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/wanderer69/flow_processor/pkg/entity"
	pb "github.com/wanderer69/flow_processor/pkg/proto/front"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type Server struct {
	pb.UnimplementedClientConnectorServer
	processExecutor ProcessExecutor
	storeClient     StoreClient
	idCounter       int32
}

func NewServer(
	processExecutor ProcessExecutor,
	storeClient StoreClient,
) *Server {
	return &Server{
		processExecutor: processExecutor,
		storeClient:     storeClient,
	}
}

func (s *Server) Login(ctx context.Context, in *pb.LoginRequest) (*pb.LoginResponse, error) {
	errMsg := "not implemented"
	return &pb.LoginResponse{
		Error: &errMsg,
	}, nil
}

func (s *Server) Connect(srv pb.ClientConnector_ConnectServer) error {
	logger := zap.L()
	logger.Info("Connect")

	ctx := srv.Context()

	sendListProcessesResponse := func(
		//		ctx context.Context,
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
				//FlowId: ,
				ProcessId: processStates[i].ProcessID,
				State:     processStates[i].State,
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

		time.Sleep(time.Duration(1) * time.Millisecond)
	}
}

func ServerConnect(port int, processExecutor ProcessExecutor, storeClient StoreClient) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	srv := NewServer(processExecutor, storeClient)
	pb.RegisterClientConnectorServer(s, srv)

	if err := s.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve: %v", err)
	}

	return nil
}

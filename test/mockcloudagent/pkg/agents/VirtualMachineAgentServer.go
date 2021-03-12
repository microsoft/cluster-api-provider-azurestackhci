// Copyright (c) Microsoft Corporation.
// Licensed under the Apache v2.0 license.

package agents

import (
	"context"

	pb "github.com/microsoft/moc/rpc/cloudagent/compute"
	pbcom "github.com/microsoft/moc/rpc/common"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

type VirtualMachineAgentServer struct {
	InvokeFunc  func(context context.Context, req *pb.VirtualMachineRequest) (*pb.VirtualMachineResponse, error)
	OperateFunc func(context context.Context, req *pb.VirtualMachineOperationRequest) (res *pb.VirtualMachineOperationResponse, err error)
}

func (s *VirtualMachineAgentServer) Invoke(context context.Context, req *pb.VirtualMachineRequest) (*pb.VirtualMachineResponse, error) {
	invokeFunc := s.InvokeFunc
	if invokeFunc != nil {
		return invokeFunc(context, req)
	}

	switch req.OperationType {
	case pbcom.Operation_GET,
		pbcom.Operation_POST,
		pbcom.Operation_DELETE:
		return nil, status.Errorf(codes.Unavailable, "Operation type not implemented: %v", req.OperationType)

	default:
		return nil, status.Errorf(codes.Unavailable, "Invalid operation type specified: %v", req.OperationType)
	}
}

func (s *VirtualMachineAgentServer) Operate(context context.Context, req *pb.VirtualMachineOperationRequest) (res *pb.VirtualMachineOperationResponse, err error) {
	operateFunc := s.OperateFunc
	if operateFunc != nil {
		return operateFunc(context, req)
	}

	switch req.OperationType {
	case pbcom.VirtualMachineOperation_START,
		pbcom.VirtualMachineOperation_STOP,
		pbcom.VirtualMachineOperation_RESET:
		return nil, status.Errorf(codes.Unavailable, "Operation type not implemented: %v", req.OperationType)

	default:
		return nil, status.Errorf(codes.Unavailable, "Invalid operation type specified: %v", req.OperationType)
	}
}

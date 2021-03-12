// Copyright (c) Microsoft Corporation.
// Licensed under the Apache v2.0 license.

package agents

import (
	context "context"

	pb "github.com/microsoft/moc/rpc/cloudagent/network"
	pbcom "github.com/microsoft/moc/rpc/common"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

type VirtualNetworkAgentServer struct {
	InvokeFunc func(context context.Context, req *pb.VirtualNetworkRequest) (*pb.VirtualNetworkResponse, error)
}

func (s *VirtualNetworkAgentServer) Invoke(context context.Context, req *pb.VirtualNetworkRequest) (*pb.VirtualNetworkResponse, error) {
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

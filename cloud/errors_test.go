/*
Copyright 2020 The Kubernetes Authors.
Portions Copyright © Microsoft Corporation.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package azurestackhci

import (
	"errors"
	"testing"

	perrors "github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestMocUnreachable(t *testing.T) {
	// Mirrors the real MOC DNS/transport dial failure observed in the field: a gRPC
	// codes.Unavailable connection error, including the wrapped form that reaches the
	// controller through the networkinterfaces service + VM reconciler call chain.
	grpcUnavailable := status.Error(codes.Unavailable,
		`connection error: desc = "transport: Error while dialing: dial tcp: lookup host.local: i/o timeout"`)

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil error", err: nil, want: false},
		{name: "raw gRPC Unavailable", err: grpcUnavailable, want: true},
		{
			name: "wrapped gRPC Unavailable (service + reconciler wrap)",
			err:  perrors.Wrapf(perrors.Wrap(grpcUnavailable, "unable to create VM network interface"), "failed to create nic %s", "nic0"),
			want: true,
		},
		{name: "gRPC NotFound is not unreachable", err: status.Error(codes.NotFound, "not found"), want: false},
		{name: "gRPC DeadlineExceeded is not unreachable", err: status.Error(codes.DeadlineExceeded, "deadline"), want: false},
		// Safety: a plain (non-gRPC) error with the SAME DNS-timeout text must NOT be classified
		// as MOC unreachable. Only a real gRPC codes.Unavailable status qualifies — this is what
		// makes the classification deterministic rather than a fragile string match.
		{name: "plain error with same dial text is not unreachable", err: errors.New(`transport: Error while dialing: dial tcp: lookup host.local: i/o timeout`), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MocUnreachable(tt.err); got != tt.want {
				t.Errorf("MocUnreachable(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

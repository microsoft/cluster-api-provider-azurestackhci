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
	perrors "github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ResourceNotFound parses the error to check if its a resource not found
func ResourceNotFound(err error) bool {
	if e, ok := status.FromError(err); ok && e.Code() == codes.NotFound {
		return true
	}
	return false
}

// ResourceAlreadyExists parses the error to check if its a resource already exists
func ResourceAlreadyExists(err error) bool {
	if e, ok := status.FromError(err); ok && e.Code() == codes.AlreadyExists {
		return true
	}
	return false
}

// MocUnreachable parses the error to check if MOC is unreachable, i.e. the gRPC call to the MOC
// agent failed with codes.Unavailable (for example a DNS/transport dial failure such as
// `transport: Error while dialing: dial tcp: lookup <host>: i/o timeout`). The check runs on the
// unwrapped cause because gRPC status inspection does not see through pkg/errors wrapping, and the
// MOC client errors reach this layer wrapped by the service/reconciler call chain.
func MocUnreachable(err error) bool {
	if err == nil {
		return false
	}
	if e, ok := status.FromError(perrors.Cause(err)); ok && e.Code() == codes.Unavailable {
		return true
	}
	return false
}

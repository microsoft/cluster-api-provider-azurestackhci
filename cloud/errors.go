/*
Copyright 2019 The Kubernetes Authors.

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

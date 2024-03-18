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

package nodes

import (
	"context"
	"errors"
	"fmt"
)

type Spec struct {
	Location string
}

func (s *Service) GetCount(ctx context.Context, spec interface{}) (int, error) {
	logger := s.Scope.GetLogger()

	nodeSpec, ok := spec.(*Spec)
	if !ok {
		return 0, errors.New("invalid node specification")
	}

	nodes, err := s.Client.Get(ctx, nodeSpec.Location, "")
	if err != nil {
		return 0, err
	}

	if nodes == nil {
		logger.Info("Empty node resources")
		return 0, nil
	}

	return len(*nodes), nil
}

func (s *Service) Reconcile(ctx context.Context, spec interface{}) error {
	s.Scope.GetLogger().Info("Reconciling nodes is not supported")
	return fmt.Errorf("Reconciling nodes is not supported")
}

func (s *Service) Delete(ctx context.Context, spec interface{}) error {
	s.Scope.GetLogger().Info("Deleting nodes is not supported")
	return fmt.Errorf("Deleting nodes is not supported")
}

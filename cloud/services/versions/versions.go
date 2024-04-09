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

package versions

import (
	"context"
	"fmt"
)

type Spec struct{}

type VersionPair struct {
	WssdCloudAgentVersion string
	MocVersion            string
}

func (s *Service) Get(ctx context.Context) (*VersionPair, error) {
	version, mocversion, err := s.Client.GetVersion(ctx)
	if err != nil {
		s.Scope.GetLogger().Error(err, "Unable to get moc version")
		return nil, err
	}
	return &VersionPair{
		WssdCloudAgentVersion: version,
		MocVersion:            mocversion,
	}, nil

}

func (s *Service) Reconcile(ctx context.Context, spec interface{}) error {
	s.Scope.GetLogger().Info("Reconciling version is not supported")
	return fmt.Errorf("Reconciling version is not supported")
}

func (s *Service) Delete(ctx context.Context, spec interface{}) error {
	s.Scope.GetLogger().Info("Deleting version is not supported")
	return fmt.Errorf("Deleting version is not supported")
}

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

package health

import (
	"context"
	"fmt"
)

func (s *Service) GetMocDeploymentID(ctx context.Context) string {
	deploymentID, err := s.Client.GetDeploymentId(ctx)
	if err != nil {
		s.Scope.GetLogger().Error(err, "Unable to get moc deployment id")
		return ""
	}
	return deploymentID

}

func (s *Service) Reconcile(ctx context.Context, spec interface{}) error {
	s.Scope.GetLogger().Info("Reconciling health is not supported")
	return fmt.Errorf("Reconciling health is not supported")
}

func (s *Service) Delete(ctx context.Context, spec interface{}) error {
	s.Scope.GetLogger().Info("Deleting health is not supported")
	return fmt.Errorf("Deleting health is not supported")
}

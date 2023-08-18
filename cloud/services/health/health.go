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

	"k8s.io/klog/v2"
)

func (s *Service) GetMocDeploymentId(ctx context.Context) string {
	deploymentId, err := s.Client.GetDeploymentId(ctx)
	if err != nil {
		klog.Error("Unable to get moc deployment id. ", err)
		return ""
	}
	return deploymentId

}

func (s *Service) Reconcile(ctx context.Context, spec interface{}) error {
	klog.V(2).Infof("Reconciling health is not supported")
	return fmt.Errorf("Reconciling health is not supported")
}

func (s *Service) Delete(ctx context.Context, spec interface{}) error {
	klog.V(2).Infof("Deleting health is not supported")
	return fmt.Errorf("Deleting health is not supported")
}

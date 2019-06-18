// Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package infrastructure

import (
	"context"
	"path/filepath"

	kubevirtv1alpha1 "github.com/gardener/gardener-extensions/controllers/provider-kubevirt/pkg/apis/kubevirt/v1alpha1"
	"github.com/gardener/gardener-extensions/controllers/provider-kubevirt/pkg/internal"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	// ChartsPath is the path to the charts
	ChartsPath = filepath.Join("controllers", "provider-kubevirt", "charts")
	// InternalChartsPath is the path to the internal charts
	InternalChartsPath = filepath.Join(ChartsPath, "internal")
	// StatusTypeMeta is the TypeMeta of the GCP InfrastructureStatus
	StatusTypeMeta = metav1.TypeMeta{
		APIVersion: kubevirtv1alpha1.SchemeGroupVersion.String(),
		Kind:       "InfrastructureStatus",
	}
)

// GetCredentialsFromInfrastructure retrieves the ServiceAccount from the Secret referenced in the given Infrastructure.
func GetCredentialsFromInfrastructure(ctx context.Context, c client.Client, config *extensionsv1alpha1.Infrastructure) (*internal.Credentials, error) {
	return internal.GetCredentials(ctx, c, config.Spec.SecretRef)
}

// ComputeStatus computes the status based on the Terraformer and the given InfrastructureConfig.
func ComputeStatus(config *kubevirtv1alpha1.InfrastructureConfig) (*kubevirtv1alpha1.InfrastructureStatus, error) {
	return &kubevirtv1alpha1.InfrastructureStatus{
		TypeMeta: metav1.TypeMeta{
			APIVersion: kubevirtv1alpha1.SchemeGroupVersion.String(),
			Kind:       "InfrastructureStatus",
		},
	}, nil
}

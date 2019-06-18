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

package internal

import (
	"context"
	"fmt"

	"github.com/gardener/gardener-extensions/controllers/provider-kubevirt/pkg/kubevirt"
	"github.com/gardener/gardener-extensions/pkg/util"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Credentials contains the necessary KubeVirt credential information.
type Credentials struct {
	Kubeconfig string
	Namespace  string
}

// GetCredentials computes for a given context and infrastructure the corresponding credentials object.
func GetCredentials(ctx context.Context, c client.Client, secretRef corev1.SecretReference) (*Credentials, error) {
	secret, err := util.GetSecretByRef(ctx, c, secretRef)
	if err != nil {
		return nil, err
	}
	return ExtractCredentials(secret)
}

// GetCredentialsForNamespaceAndName computes for a given context and namespace and name the corresponding credentials object.
func GetCredentialsForNamespaceAndName(ctx context.Context, c client.Client, namespace, name string) (*Credentials, error) {
	providerSecret := &corev1.Secret{}
	if err := c.Get(ctx, kutil.Key(namespace, name), providerSecret); err != nil {
		return nil, err
	}
	return ExtractCredentials(providerSecret)
}

// ExtractCredentials generates a credentials object for a given provider secret.
func ExtractCredentials(secret *corev1.Secret) (*Credentials, error) {
	if secret.Data == nil {
		return nil, fmt.Errorf("secret does not contain any data")
	}
	kubeconfig, err := getRequired(secret.Data, kubevirt.Kubeconfig)
	if err != nil {
		return nil, err
	}
	namespace, err := getRequired(secret.Data, kubevirt.Namespace)
	if err != nil {
		return nil, err
	}

	return &Credentials{
		Kubeconfig: kubeconfig,
		Namespace:  namespace,
	}, nil
}

// getRequired checks if the provided map has a valid value for a corresponding key.
func getRequired(data map[string][]byte, key string) (string, error) {
	value, ok := data[key]
	if !ok {
		return "", fmt.Errorf("map %v does not contain key %s", data, key)
	}
	if len(value) == 0 {
		return "", fmt.Errorf("key %s may not be empty", key)
	}
	return string(value), nil
}

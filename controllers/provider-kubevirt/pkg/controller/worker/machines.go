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

package worker

import (
	"context"
	"fmt"
	"path/filepath"

	apiskubevirt "github.com/gardener/gardener-extensions/controllers/provider-kubevirt/pkg/apis/kubevirt"
	kubevirtapi "github.com/gardener/gardener-extensions/controllers/provider-kubevirt/pkg/apis/kubevirt"
	"github.com/gardener/gardener-extensions/controllers/provider-kubevirt/pkg/internal"
	"github.com/gardener/gardener-extensions/controllers/provider-kubevirt/pkg/kubevirt"
	extensionscontroller "github.com/gardener/gardener-extensions/pkg/controller"
	"github.com/gardener/gardener-extensions/pkg/controller/worker"
	"github.com/gardener/gardener-extensions/pkg/util"

	machinev1alpha1 "github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
)

// MachineClassKind yields the name of the kubevirt machine class.
func (w *workerDelegate) MachineClassKind() string {
	return "kubevirtMachineClass"
}

// MachineClassList yields a newly initialized kubevirtMachineClassList object.
func (w *workerDelegate) MachineClassList() runtime.Object {
	return &machinev1alpha1.KubeVirtMachineClass{}
}

// DeployMachineClasses generates and creates the kubevirt specific machine classes.
func (w *workerDelegate) DeployMachineClasses(ctx context.Context) error {
	if w.machineClasses == nil {
		if err := w.generateMachineConfig(ctx); err != nil {
			return err
		}
	}
	return w.seedChartApplier.ApplyChart(ctx, filepath.Join(kubevirt.InternalChartsPath, "machineclass"), w.worker.Namespace, "machineclass", map[string]interface{}{"machineClasses": w.machineClasses}, nil)
}

// GenerateMachineDeployments generates the configuration for the desired machine deployments.
func (w *workerDelegate) GenerateMachineDeployments(ctx context.Context) (worker.MachineDeployments, error) {
	if w.machineDeployments == nil {
		if err := w.generateMachineConfig(ctx); err != nil {
			return nil, err
		}
	}
	return w.machineDeployments, nil
}

func (w *workerDelegate) generateMachineClassSecretData(ctx context.Context) (map[string][]byte, error) {
	credentials, err := internal.GetCredentials(ctx, w.client, w.worker.Spec.SecretRef)
	if err != nil {
		return nil, err
	}

	return map[string][]byte{
		machinev1alpha1.KubeVirtKubeConfig: []byte(credentials.KubeConfig),
		machinev1alpha1.KubeVirtNamespace:  []byte(credentials.Namespace),
	}, nil
}

func (w *workerDelegate) generateMachineConfig(ctx context.Context) error {
	var (
		machineDeployments = worker.MachineDeployments{}
		machineClasses     []map[string]interface{}
		machineImages      []apiskubevirt.MachineImage
	)

	machineClassSecretData, err := w.generateMachineClassSecretData(ctx)
	if err != nil {
		return err
	}

	shootVersionMajorMinor, err := util.VersionMajorMinor(w.cluster.Shoot.Spec.Kubernetes.Version)
	if err != nil {
		return err
	}

	infrastructureStatus := &kubevirtapi.InfrastructureStatus{}
	if _, _, err := w.decoder.Decode(w.worker.Spec.InfrastructureProviderStatus.Raw, nil, infrastructureStatus); err != nil {
		return err
	}

	for _, pool := range w.worker.Spec.Pools {
		zoneLen := len(pool.Zones)

		machineImage, err := w.findMachineImage(pool.MachineImage.Name, pool.MachineImage.Version, w.cluster.CloudProfile.Name)
		if err != nil {
			return err
		}
		machineImages = appendMachineImage(machineImages, apiskubevirt.MachineImage{
			Name:         pool.MachineImage.Name,
			Version:      pool.MachineImage.Version,
			CloudProfile: w.cluster.CloudProfile.Name,
			Image:        machineImage,
		})

		for zoneIndex, zone := range pool.Zones {
			machineClassSpec := map[string]interface{}{
				"region":           w.worker.Spec.Region,
				"availabilityZone": zone,
				"machineType":      pool.MachineType,
				"imageName":        machineImage,
				"podNetworkCidr":   extensionscontroller.GetPodNetwork(w.cluster.Shoot),
				"tags": map[string]string{
					fmt.Sprintf("kubernetes.io-cluster-%s", w.worker.Namespace): "1",
					"kubernetes.io-role-node":                                   "1",
				},
				"secret": map[string]interface{}{
					"cloudConfig": string(pool.UserData),
				},
			}

			var (
				machineClassSpecHash = worker.MachineClassHash(machineClassSpec, shootVersionMajorMinor)
				deploymentName       = fmt.Sprintf("%s-%s-z%d", w.worker.Namespace, pool.Name, zoneIndex+1)
				className            = fmt.Sprintf("%s-%s", deploymentName, machineClassSpecHash)
			)

			machineDeployments = append(machineDeployments, worker.MachineDeployment{
				Name:           deploymentName,
				ClassName:      className,
				SecretName:     className,
				Minimum:        worker.DistributeOverZones(zoneIndex, pool.Minimum, zoneLen),
				Maximum:        worker.DistributeOverZones(zoneIndex, pool.Maximum, zoneLen),
				MaxSurge:       worker.DistributePositiveIntOrPercent(zoneIndex, pool.MaxSurge, zoneLen, pool.Maximum),
				MaxUnavailable: worker.DistributePositiveIntOrPercent(zoneIndex, pool.MaxUnavailable, zoneLen, pool.Minimum),
				Labels:         pool.Labels,
				Annotations:    pool.Annotations,
				Taints:         pool.Taints,
			})

			machineClassSpec["name"] = className
			machineClassSpec["secret"].(map[string]interface{})[kubevirt.Namespace] = string(machineClassSecretData[machinev1alpha1.KubeVirtNamespace])
			machineClassSpec["secret"].(map[string]interface{})[kubevirt.KubeConfig] = string(machineClassSecretData[machinev1alpha1.KubeVirtKubeConfig])

			machineClasses = append(machineClasses, machineClassSpec)
		}
	}

	w.machineDeployments = machineDeployments
	w.machineClasses = machineClasses
	w.machineImages = machineImages

	return nil
}

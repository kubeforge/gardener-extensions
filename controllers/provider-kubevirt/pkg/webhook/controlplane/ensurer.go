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

package controlplane

import (
	"context"

	"github.com/gardener/gardener-extensions/controllers/provider-kubevirt/pkg/kubevirt"
	extensionswebhook "github.com/gardener/gardener-extensions/pkg/webhook"
	"github.com/gardener/gardener-extensions/pkg/webhook/controlplane"
	"github.com/gardener/gardener-extensions/pkg/webhook/controlplane/genericmutator"
	gardencorev1alpha1 "github.com/gardener/gardener/pkg/apis/core/v1alpha1"

	"github.com/coreos/go-systemd/unit"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	kubeletconfigv1beta1 "k8s.io/kubelet/config/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewEnsurer creates a new controlplane ensurer.
func NewEnsurer(logger logr.Logger) genericmutator.Ensurer {
	return &ensurer{
		logger: logger.WithName("kubevirt-controlplane-ensurer"),
	}
}

type ensurer struct {
	genericmutator.NoopEnsurer
	client client.Client
	logger logr.Logger
}

// InjectClient injects the given client into the ensurer.
func (e *ensurer) InjectClient(client client.Client) error {
	e.client = client
	return nil
}

// EnsureKubeAPIServerDeployment ensures that the kube-apiserver deployment conforms to the provider requirements.
func (e *ensurer) EnsureKubeAPIServerDeployment(ctx context.Context, dep *appsv1.Deployment) error {
	template := &dep.Spec.Template
	ps := &template.Spec
	if c := extensionswebhook.ContainerWithName(ps.Containers, "kube-apiserver"); c != nil {
		ensureKubeAPIServerCommandLineArgs(c)
		ensureVolumeMounts(c)
	}
	ensureVolumes(ps)
	return e.ensureChecksumAnnotations(ctx, &dep.Spec.Template, dep.Namespace)
}

// EnsureKubeControllerManagerDeployment ensures that the kube-controller-manager deployment conforms to the provider requirements.
func (e *ensurer) EnsureKubeControllerManagerDeployment(ctx context.Context, dep *appsv1.Deployment) error {
	template := &dep.Spec.Template
	ps := &template.Spec
	if c := extensionswebhook.ContainerWithName(ps.Containers, "kube-controller-manager"); c != nil {
		ensureKubeControllerManagerCommandLineArgs(c)
		ensureVolumeMounts(c)
	}
	ensureKubeControllerManagerAnnotations(template)
	ensureVolumes(ps)
	return e.ensureChecksumAnnotations(ctx, &dep.Spec.Template, dep.Namespace)
}

func ensureKubeAPIServerCommandLineArgs(c *corev1.Container) {
	c.Command = extensionswebhook.EnsureStringWithPrefix(c.Command, "--cloud-provider=", "kubevirt")
	c.Command = extensionswebhook.EnsureStringWithPrefix(c.Command, "--cloud-config=",
		"/etc/kubernetes/cloudprovider/cloudprovider.conf")
	c.Command = extensionswebhook.EnsureStringWithPrefixContains(c.Command, "--enable-admission-plugins=",
		"PersistentVolumeLabel", ",")
	c.Command = extensionswebhook.EnsureNoStringWithPrefixContains(c.Command, "--disable-admission-plugins=",
		"PersistentVolumeLabel", ",")
}

func ensureKubeControllerManagerCommandLineArgs(c *corev1.Container) {
	c.Command = extensionswebhook.EnsureStringWithPrefix(c.Command, "--cloud-provider=", "external")
	c.Command = extensionswebhook.EnsureStringWithPrefix(c.Command, "--cloud-config=",
		"/etc/kubernetes/cloudprovider/cloudprovider.conf")
	c.Command = extensionswebhook.EnsureStringWithPrefix(c.Command, "--external-cloud-volume-plugin=", "kubevirt")
}

func ensureKubeControllerManagerAnnotations(t *corev1.PodTemplateSpec) {
	t.Labels = extensionswebhook.EnsureAnnotationOrLabel(t.Labels, gardencorev1alpha1.LabelNetworkPolicyToPublicNetworks, gardencorev1alpha1.LabelNetworkPolicyAllowed)
	t.Labels = extensionswebhook.EnsureAnnotationOrLabel(t.Labels, gardencorev1alpha1.LabelNetworkPolicyToPrivateNetworks, gardencorev1alpha1.LabelNetworkPolicyAllowed)
	t.Labels = extensionswebhook.EnsureAnnotationOrLabel(t.Labels, gardencorev1alpha1.LabelNetworkPolicyToBlockedCIDRs, gardencorev1alpha1.LabelNetworkPolicyAllowed)
}

var (
	cloudProviderConfigKubeControllerManagerVolumeMount = corev1.VolumeMount{
		Name:      kubevirt.CloudProviderConfigKubeControllerManagerName,
		MountPath: "/etc/kubernetes/cloudprovider",
	}
	cloudProviderConfigKubeControllerManagerVolume = corev1.Volume{
		Name: kubevirt.CloudProviderConfigKubeControllerManagerName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: kubevirt.CloudProviderConfigKubeControllerManagerName},
			},
		},
	}
)

func ensureVolumeMounts(c *corev1.Container) {
	c.VolumeMounts = extensionswebhook.EnsureVolumeMountWithName(c.VolumeMounts, cloudProviderConfigKubeControllerManagerVolumeMount)
}

func ensureVolumes(ps *corev1.PodSpec) {
	ps.Volumes = extensionswebhook.EnsureVolumeWithName(ps.Volumes, cloudProviderConfigKubeControllerManagerVolume)
}

func (e *ensurer) ensureChecksumAnnotations(ctx context.Context, template *corev1.PodTemplateSpec, namespace string) error {
	return controlplane.EnsureConfigMapChecksumAnnotation(ctx, template, e.client, namespace, kubevirt.CloudProviderConfigCloudControllerManagerName)
}

// EnsureKubeletServiceUnitOptions ensures that the kubelet.service unit options conform to the provider requirements.
func (e *ensurer) EnsureKubeletServiceUnitOptions(ctx context.Context, opts []*unit.UnitOption) ([]*unit.UnitOption, error) {
	if opt := extensionswebhook.UnitOptionWithSectionAndName(opts, "Service", "ExecStart"); opt != nil {
		command := extensionswebhook.DeserializeCommandLine(opt.Value)
		command = ensureKubeletCommandLineArgs(command)
		opt.Value = extensionswebhook.SerializeCommandLine(command, 1, " \\\n    ")
	}

	opts = extensionswebhook.EnsureUnitOption(opts, &unit.UnitOption{
		Section: "Service",
		Name:    "ExecStartPre",
		Value:   `/bin/sh -c 'hostnamectl set-hostname $(cat /etc/hostname | cut -d '.' -f 1)'`,
	})
	return opts, nil
}

func ensureKubeletCommandLineArgs(command []string) []string {
	command = extensionswebhook.EnsureStringWithPrefix(command, "--cloud-provider=", "kubevirt")
	command = extensionswebhook.EnsureStringWithPrefix(command, "--cloud-config=", "/var/lib/kubelet/cloudprovider.conf")
	return command
}

// EnsureKubeletConfiguration ensures that the kubelet configuration conforms to the provider requirements.
func (e *ensurer) EnsureKubeletConfiguration(ctx context.Context, kubeletConfig *kubeletconfigv1beta1.KubeletConfiguration) error {
	// Make sure CSI-related feature gates are not enabled
	// TODO Leaving these enabled shouldn't do any harm, perhaps remove this code when properly tested?
	delete(kubeletConfig.FeatureGates, "VolumeSnapshotDataSource")
	delete(kubeletConfig.FeatureGates, "CSINodeInfo")
	delete(kubeletConfig.FeatureGates, "CSIDriverRegistry")
	return nil
}

// ShouldProvisionKubeletCloudProviderConfig returns true if the cloud provider config file should be added to the kubelet configuration.
func (e *ensurer) ShouldProvisionKubeletCloudProviderConfig() bool {
	return true
}

// EnsureKubeletCloudProviderConfig ensures that the cloud provider config file conforms to the provider requirements.
func (e *ensurer) EnsureKubeletCloudProviderConfig(ctx context.Context, data *string, namespace string) error {
	// Get `cloud-provider-config` ConfigMap
	var cm corev1.ConfigMap
	err := e.client.Get(ctx, kutil.Key(namespace, kubevirt.CloudProviderConfigKubeControllerManagerName), &cm)
	if err != nil {
		if apierrors.IsNotFound(err) {
			e.logger.Info("configmap not found", "name", kubevirt.CloudProviderConfigKubeControllerManagerName, "namespace", namespace)
			return nil
		}
		return errors.Wrapf(err, "could not get configmap '%s/%s'", namespace, kubevirt.CloudProviderConfigKubeControllerManagerName)
	}

	// Check if the data has "cloudprovider.conf" key
	if cm.Data == nil || cm.Data[kubevirt.CloudProviderConfigMapKey] == "" {
		return nil
	}

	// Overwrite data variable
	*data = cm.Data[kubevirt.CloudProviderConfigMapKey]
	return nil
}

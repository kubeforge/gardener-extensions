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

package azure

import "path/filepath"

const (
	// Name is the name of the Azure provider.
	Name = "provider-azure"
	// StorageProviderName is the name of the Azure storage provider.
	StorageProviderName = "ABS"

	// TerraformerImageName is the name of the Terraformer image.
	TerraformerImageName = "terraformer"
	// MachineControllerManagerImageName is the name of the MachineControllerManager image.
	MachineControllerManagerImageName = "machine-controller-manager"
	// ETCDBackupRestoreImageName is the name of the etcd backup and restore image.
	ETCDBackupRestoreImageName = "etcd-backup-restore"

	// MachineControllerManagerName is a constant for the name of the machine-controller-manager.
	MachineControllerManagerName = "machine-controller-manager"
	HyperkubeImageName           = "hyperkube"

	// SubscriptionIDKey is the key for the subscription ID
	SubscriptionIDKey = "subscriptionID"
	// TenantIDKey is the key for the tenant id
	TenantIDKey = "tenantID"
	// ClientIDKey is the key for the client id
	ClientIDKey = "clientID"
	// ClientSecretKey is the key for the client secret
	ClientSecretKey = "clientSecret"

	// StorageAccount is a constant for the key in a cloud provider secret and backup secret that holds the Azure account name.
	StorageAccount = "storage-account"
	// StorageKey is a constant for the key in a cloud provider secret and backup secret that holds the Azure secret storage access key.
	StorageKey = "storage-key"

	// BucketName is a constant for the key in a backup secret that holds the bucket name.
	// The bucket name is written to the backup secret by Gardener as a temporary solution.
	// TODO In the future, the bucket name should come from a BackupBucket resource (see https://github.com/gardener/gardener/blob/master/docs/proposals/02-backupinfra.md)
	BucketName = "bucketName"

	// CloudProviderConfigName is the name of the configmap containing the cloud provider config.
	CloudProviderConfigName = "cloud-provider-config"
	// BackupSecretName is the name of the secret containing the credentials for storing the backups of Shoot clusters.
	BackupSecretName = "etcd-backup"
)

var (
	// ChartsPath is the path to the charts
	ChartsPath = filepath.Join("controllers", Name, "charts")
	// InternalChartsPath is the path to the internal charts
	InternalChartsPath = filepath.Join(ChartsPath, "internal")
)

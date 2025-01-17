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

package app

import (
	"context"
	"fmt"
	"os"

	azureinstall "github.com/gardener/gardener-extensions/controllers/provider-azure/pkg/apis/azure/install"
	"github.com/gardener/gardener-extensions/controllers/provider-azure/pkg/azure"
	azurecmd "github.com/gardener/gardener-extensions/controllers/provider-azure/pkg/cmd"
	azurecontrolplane "github.com/gardener/gardener-extensions/controllers/provider-azure/pkg/controller/controlplane"
	azureinfrastructure "github.com/gardener/gardener-extensions/controllers/provider-azure/pkg/controller/infrastructure"
	azureworker "github.com/gardener/gardener-extensions/controllers/provider-azure/pkg/controller/worker"
	azurecontrolplanebackup "github.com/gardener/gardener-extensions/controllers/provider-azure/pkg/webhook/controlplanebackup"
	azurecontrolplaneexposure "github.com/gardener/gardener-extensions/controllers/provider-azure/pkg/webhook/controlplaneexposure"
	"github.com/gardener/gardener-extensions/pkg/controller"
	controllercmd "github.com/gardener/gardener-extensions/pkg/controller/cmd"
	"github.com/gardener/gardener-extensions/pkg/controller/infrastructure"
	"github.com/gardener/gardener-extensions/pkg/controller/worker"
	webhookcmd "github.com/gardener/gardener-extensions/pkg/webhook/cmd"

	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// NewControllerManagerCommand creates a new command for running a Azure provider controller.
func NewControllerManagerCommand(ctx context.Context) *cobra.Command {
	var (
		restOpts = &controllercmd.RESTOptions{}
		mgrOpts  = &controllercmd.ManagerOptions{
			LeaderElection:          true,
			LeaderElectionID:        controllercmd.LeaderElectionNameID(azure.Name),
			LeaderElectionNamespace: os.Getenv("LEADER_ELECTION_NAMESPACE"),
		}
		configFileOpts = &azurecmd.ConfigOptions{}

		// options for the controlplane controller
		controlPlaneCtrlOpts = &controllercmd.ControllerOptions{
			MaxConcurrentReconciles: 5,
		}

		// options for the infrastructure controller
		infraCtrlOpts = &controllercmd.ControllerOptions{
			MaxConcurrentReconciles: 5,
		}
		infraReconcileOpts = &infrastructure.ReconcilerOptions{
			IgnoreOperationAnnotation: true,
		}
		infraCtrlOptsUnprefixed = controllercmd.NewOptionAggregator(infraCtrlOpts, infraReconcileOpts)

		// options for the worker controller
		workerCtrlOpts = &controllercmd.ControllerOptions{
			MaxConcurrentReconciles: 5,
		}
		workerReconcileOpts = &worker.Options{
			DeployCRDs: true,
		}
		workerCtrlOptsUnprefixed = controllercmd.NewOptionAggregator(workerCtrlOpts, workerReconcileOpts)

		controllerSwitches   = azurecmd.ControllerSwitchOptions()
		webhookSwitches      = azurecmd.WebhookSwitchOptions()
		webhookServerOptions = &webhookcmd.ServerOptions{
			Port:             7890,
			CertDir:          "/tmp/cert",
			Mode:             webhookcmd.ServiceMode,
			Name:             "webhooks",
			Namespace:        os.Getenv("WEBHOOK_CONFIG_NAMESPACE"),
			ServiceSelectors: "{}",
			Host:             "localhost",
		}
		webhookOptions = webhookcmd.NewAddToManagerOptions(azure.Name, webhookServerOptions, webhookSwitches)

		aggOption = controllercmd.NewOptionAggregator(
			restOpts,
			mgrOpts,
			controllercmd.PrefixOption("controlplane-", controlPlaneCtrlOpts),
			controllercmd.PrefixOption("infrastructure-", &infraCtrlOptsUnprefixed),
			controllercmd.PrefixOption("worker-", &workerCtrlOptsUnprefixed),
			configFileOpts,
			controllerSwitches,
			webhookOptions,
		)
	)

	cmd := &cobra.Command{
		Use: fmt.Sprintf("%s-controller-manager", azure.Name),

		Run: func(cmd *cobra.Command, args []string) {
			if err := aggOption.Complete(); err != nil {
				controllercmd.LogErrAndExit(err, "Error completing options")
			}

			if workerReconcileOpts.Completed().DeployCRDs {
				if err := worker.ApplyMachineResourcesForConfig(ctx, restOpts.Completed().Config); err != nil {
					controllercmd.LogErrAndExit(err, "Error ensuring the machine CRDs")
				}
			}

			mgr, err := manager.New(restOpts.Completed().Config, mgrOpts.Completed().Options())
			if err != nil {
				controllercmd.LogErrAndExit(err, "Could not instantiate manager")
			}

			if err := controller.AddToScheme(mgr.GetScheme()); err != nil {
				controllercmd.LogErrAndExit(err, "Could not update manager scheme")
			}

			if err := azureinstall.AddToScheme(mgr.GetScheme()); err != nil {
				controllercmd.LogErrAndExit(err, "Could not update manager scheme")
			}

			configFileOpts.Completed().ApplyMachineImages(&azureworker.DefaultAddOptions.MachineImages)
			configFileOpts.Completed().ApplyETCDStorage(&azurecontrolplaneexposure.DefaultAddOptions.ETCDStorage)
			configFileOpts.Completed().ApplyETCDBackup(&azurecontrolplanebackup.DefaultAddOptions.ETCDBackup)
			controlPlaneCtrlOpts.Completed().Apply(&azurecontrolplane.Options)
			infraCtrlOpts.Completed().Apply(&azureinfrastructure.DefaultAddOptions.Controller)
			infraReconcileOpts.Completed().Apply(&azureinfrastructure.DefaultAddOptions.IgnoreOperationAnnotation)
			workerCtrlOpts.Completed().Apply(&azureworker.DefaultAddOptions.Controller)

			if err := controllerSwitches.Completed().AddToManager(mgr); err != nil {
				controllercmd.LogErrAndExit(err, "Could not add controllers to manager")
			}

			if err := webhookOptions.Completed().AddToManager(mgr); err != nil {
				controllercmd.LogErrAndExit(err, "Could not add webhooks to manager")
			}

			if err := mgr.Start(ctx.Done()); err != nil {
				controllercmd.LogErrAndExit(err, "Error running manager")
			}
		},
	}

	aggOption.AddFlags(cmd.Flags())

	return cmd
}

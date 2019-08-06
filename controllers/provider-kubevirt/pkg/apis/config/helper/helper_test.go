// Copyright (c) 2018 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package helper_test

import (
	"github.com/gardener/gardener-extensions/controllers/provider-kubevirt/pkg/apis/config"
	. "github.com/gardener/gardener-extensions/controllers/provider-kubevirt/pkg/apis/config/helper"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Helper", func() {
	DescribeTable("#FindImageForCloudProfile",
		func(machineImages []config.MachineImage, imageName, version, cloudProfileName, expectedImage string) {
			image, err := FindImageForCloudProfile(machineImages, imageName, version, cloudProfileName)

			Expect(image).To(Equal(expectedImage))
			if expectedImage != "" {
				Expect(err).NotTo(HaveOccurred())
			} else {
				Expect(err).To(HaveOccurred())
			}
		},

		Entry("list is nil", nil, "ubuntu", "1", "eu-de-1", ""),
		Entry("empty list", []config.MachineImage{}, "ubuntu", "1", "eu-de-1", ""),
		Entry("entry not found (image does not exist)", makeMachineImages("debian", "1", "eu-de-1", "0"), "ubuntu", "1", "eu-de-1", ""),
		Entry("entry not found (version does not exist)", makeMachineImages("ubuntu", "2", "eu-de-1", "0"), "ubuntu", "1", "eu-de-1", ""),
		Entry("entry not found (region does not exist)", makeMachineImages("ubuntu", "1", "us-ca-1", "0"), "ubuntu", "1", "eu-de-1", ""),
		Entry("entry", makeMachineImages("ubuntu", "1", "eu-de-1", "image-1234"), "ubuntu", "1", "eu-de-1", "image-1234"),
	)
})

func makeMachineImages(name, version, region, image string) []config.MachineImage {
	var cloudProfileImageMapping []config.CloudProfileMapping
	if len(region) != 0 && len(image) != 0 {
		cloudProfileImageMapping = append(cloudProfileImageMapping, config.CloudProfileMapping{
			Name:  region,
			Image: image,
		})
	}

	return []config.MachineImage{
		{
			Name:          name,
			Version:       version,
			CloudProfiles: cloudProfileImageMapping,
		},
	}
}

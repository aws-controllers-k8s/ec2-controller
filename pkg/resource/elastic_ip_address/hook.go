// Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

package elastic_ip_address

import (
	svcsdk "github.com/aws/aws-sdk-go/service/ec2"
)

// updateTagSpecificationsInCreateRequest adds
// Tags defined in the Spec to AllocateAddressInput.TagSpecification
// and ensures the ResourceType is always set to 'elastic-ip'
func updateTagSpecificationsInCreateRequest(r *resource,
	input *svcsdk.AllocateAddressInput) {
	desiredTagSpecs := svcsdk.TagSpecification{}
	if r.ko.Spec.Tags != nil {
		requestedTags := []*svcsdk.Tag{}
		for _, desiredTag := range r.ko.Spec.Tags {
			// Add in tags defined in the Spec
			tag := &svcsdk.Tag{}
			if desiredTag.Key != nil && desiredTag.Value != nil {
				tag.SetKey(*desiredTag.Key)
				tag.SetValue(*desiredTag.Value)
			}
			requestedTags = append(requestedTags, tag)
		}
		desiredTagSpecs.SetResourceType("elastic-ip")
		desiredTagSpecs.SetTags(requestedTags)
	}
	input.TagSpecifications = []*svcsdk.TagSpecification{&desiredTagSpecs}
}

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

package instance

import (
	"errors"

	svcsdk "github.com/aws/aws-sdk-go/service/ec2"
)

// addInstanceIDsToTerminateRequest populates the list of InstanceIDs
// in the TerminateInstances request with the resource's InstanceID
// Return error to indicate to callers that the resource is not yet created.
func addInstanceIDsToTerminateRequest(r *resource,
	input *svcsdk.TerminateInstancesInput) error {
	if r.ko.Status.InstanceID == nil {
		return errors.New("InstanceID nil for resource when creating TerminateRequest")
	}
	input.InstanceIds = append(input.InstanceIds, r.ko.Status.InstanceID)
	return nil
}

// updateTagSpecificationsInCreateRequest adds
// Tags defined in the Spec to RunInstancesInput.TagSpecification
// and ensures the ResourceType is always set to 'instance'
func updateTagSpecificationsInCreateRequest(r *resource,
	input *svcsdk.RunInstancesInput) {
	desiredTagSpecs := svcsdk.TagSpecification{}
	if r.ko.Spec.Tags != nil {
		instanceTags := []*svcsdk.Tag{}
		for _, desiredTag := range r.ko.Spec.Tags {
			// Add in tags defined in the Spec
			tag := &svcsdk.Tag{}
			if desiredTag.Key != nil && desiredTag.Value != nil {
				tag.SetKey(*desiredTag.Key)
				tag.SetValue(*desiredTag.Value)
			}
			instanceTags = append(instanceTags, tag)
		}
		desiredTagSpecs.SetResourceType("instance")
		desiredTagSpecs.SetTags(instanceTags)
	}
	input.TagSpecifications = []*svcsdk.TagSpecification{&desiredTagSpecs}
}

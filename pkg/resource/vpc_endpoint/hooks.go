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

package vpc_endpoint

import (
	"errors"

	svcsdk "github.com/aws/aws-sdk-go/service/ec2"
)

// addIDToDeleteRequest adds resource's Vpc Endpoint ID to DeleteRequest.
// Return error to indicate to callers that the resource is not yet created.
func addIDToDeleteRequest(r *resource,
	input *svcsdk.DeleteVpcEndpointsInput) error {
	if r.ko.Status.VPCEndpointID == nil {
		return errors.New("unable to extract VPCEndpointID from resource")
	}
	input.VpcEndpointIds = []*string{r.ko.Status.VPCEndpointID}
	return nil
}

// updateTagSpecificationsInCreateRequest adds
// Tags defined in the Spec to CreateVpcEndpointInput.TagSpecification
// and ensures the ResourceType is always set to 'vpc-endpoint'
func updateTagSpecificationsInCreateRequest(r *resource,
	input *svcsdk.CreateVpcEndpointInput) {
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
		desiredTagSpecs.SetResourceType("vpc-endpoint")
		desiredTagSpecs.SetTags(requestedTags)
	}
	input.TagSpecifications = []*svcsdk.TagSpecification{&desiredTagSpecs}
}

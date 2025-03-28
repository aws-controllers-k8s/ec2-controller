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

package transit_gateway_vpc_attachment

import (
	"fmt"

	ackrequeue "github.com/aws-controllers-k8s/runtime/pkg/requeue"
	"github.com/aws/aws-sdk-go-v2/aws"
	svcsdk "github.com/aws/aws-sdk-go-v2/service/ec2"
	svcsdktypes "github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/aws-controllers-k8s/ec2-controller/pkg/tags"
)

var StatusAvailable = svcsdktypes.TransitGatewayAttachmentStateAvailable

// requeueWaitUntilCanModify returns a `ackrequeue.RequeueNeededAfter` struct
// explaining the cluster cannot be modified until it reaches an active status.
func requeueWaitUntilCanModify(r *resource) *ackrequeue.RequeueNeeded {
	if r.ko.Status.State == nil {
		return nil
	}
	status := *r.ko.Status.State
	return ackrequeue.Needed(
		fmt.Errorf("transitGatewayAttachment is in '%s' and state, cannot be modified until '%s'",
			status, StatusAvailable),
	)
}

// updateTagSpecificationsInCreateRequest adds
// Tags defined in the Spec to CreateDhcpOptionsInput.TagSpecification
// and ensures the ResourceType is always set to 'transit-gateway-vpc-attachment'
func updateTagSpecificationsInCreateRequest(r *resource,
	input *svcsdk.CreateTransitGatewayVpcAttachmentInput) {
	input.TagSpecifications = nil
	desiredTagSpecs := svcsdktypes.TagSpecification{}
	if r.ko.Spec.Tags != nil {
		requestedTags := []svcsdktypes.Tag{}
		for _, desiredTag := range r.ko.Spec.Tags {
			// Add in tags defined in the Spec
			tag := svcsdktypes.Tag{}
			if desiredTag.Key != nil && desiredTag.Value != nil {
				tag.Key = desiredTag.Key
				tag.Value = desiredTag.Value
			}
			requestedTags = append(requestedTags, tag)
		}
		desiredTagSpecs.ResourceType = svcsdktypes.ResourceTypeTransitGatewayAttachment
		desiredTagSpecs.Tags = requestedTags
		input.TagSpecifications = []svcsdktypes.TagSpecification{desiredTagSpecs}
	}
}

func compareSubnetIDs(desiredSubnetIDs, latestSubnetIDs []*string) ([]string, []string) {

	toAdd, toRemove := []string{}, []string{}
	desired, latest := aws.ToStringSlice(desiredSubnetIDs), aws.ToStringSlice(latestSubnetIDs)

	for _, d := range desired {
		found := false
		for _, l := range latest {
			if d == l {
				found = true
				break
			}
		}
		if !found {
			toAdd = append(toAdd, d)
		}
	}

	for _, l := range latest {
		found := false
		for _, d := range desired {
			if l == d {
				found = true
				break
			}
		}
		if !found {
			toRemove = append(toAdd, l)
		}
	}

	return toAdd, toRemove
}

func (rm *resourceManager) checkForMissingRequiredFields(r *resource) bool {
	return r.ko.Status.ID == nil
}

var syncTags = tags.Sync

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
	"context"
	"errors"

	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"

	"github.com/aws-controllers-k8s/ec2-controller/pkg/tags"
	svcsdk "github.com/aws/aws-sdk-go-v2/service/ec2"
	svcsdktypes "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// https://boto3.amazonaws.com/v1/documentation/api/latest/reference/services/ec2/client/describe_vpc_endpoints.html
const (
	StatusPendingAcceptance = "pendingAcceptance"
	StatusPending           = "pending"
	StatusAvailable         = "available"
	StatusDeleting          = "deleting"
	StatusDeleted           = "deleted"
	StatusRejected          = "rejected"
	StatusFailed            = "failed"
)

func vpcEndpointPending(r *resource) bool {
	if r.ko.Status.State == nil {
		return false
	}
	cs := *r.ko.Status.State
	return cs == StatusPending
}

// addIDToDeleteRequest adds resource's Vpc Endpoint ID to DeleteRequest.
// Return error to indicate to callers that the resource is not yet created.
func addIDToDeleteRequest(r *resource,
	input *svcsdk.DeleteVpcEndpointsInput) error {
	if r.ko.Status.VPCEndpointID == nil {
		return errors.New("unable to extract VPCEndpointID from resource")
	}
	input.VpcEndpointIds = []string{*r.ko.Status.VPCEndpointID}
	return nil
}

func (rm *resourceManager) customUpdateVPCEndpoint(
	ctx context.Context,
	desired *resource,
	latest *resource,
	delta *ackcompare.Delta,
) (updated *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.customUpdateVPCEndpoint")
	defer exit(err)

	// Default `updated` to `desired` because it is likely
	// EC2 `modify` APIs do NOT return output, only errors.
	// If the `modify` calls (i.e. `sync`) do NOT return
	// an error, then the update was successful and desired.Spec
	// (now updated.Spec) reflects the latest resource state.
	updated = rm.concreteResource(desired.DeepCopy())

	if delta.DifferentAt("Spec.Tags") {
		if err := tags.Sync(
			ctx, rm.sdkapi, rm.metrics, *latest.ko.Status.VPCEndpointID,
			desired.ko.Spec.Tags, latest.ko.Spec.Tags,
		); err != nil {
			return nil, err
		}
	}

	return updated, nil
}

// updateTagSpecificationsInCreateRequest adds
// Tags defined in the Spec to CreateVpcEndpointInput.TagSpecification
// and ensures the ResourceType is always set to 'vpc-endpoint'
func updateTagSpecificationsInCreateRequest(r *resource,
	input *svcsdk.CreateVpcEndpointInput) {
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
		desiredTagSpecs.ResourceType = "vpc-endpoint"
		desiredTagSpecs.Tags = requestedTags
		input.TagSpecifications = []svcsdktypes.TagSpecification{desiredTagSpecs}
	}
}

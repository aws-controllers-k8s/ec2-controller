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

package vpc_peering_connection

import (
	"fmt"
	"time"

	svcapitypes "github.com/aws-controllers-k8s/ec2-controller/apis/v1alpha1"
	"github.com/aws-controllers-k8s/ec2-controller/pkg/tags"
	ackrequeue "github.com/aws-controllers-k8s/runtime/pkg/requeue"
	svcsdk "github.com/aws/aws-sdk-go/service/ec2"
)

var (
	ErrVPCPeeringConnectionCreating = fmt.Errorf(
		"VPCPeeringConnection in '%v' state, cannot be modified or deleted",
		"creating",
	)
	ErrVPCPeeringConnectionProvisioning = fmt.Errorf(
		"VPCPeeringConnection in '%v' state, cannot be modified or deleted",
		svcsdk.VpcPeeringConnectionStateReasonCodeProvisioning,
	)
	ErrVPCPeeringConnectionDeleting = fmt.Errorf(
		"VPCPeeringConnection in '%v' state, cannot be modified or deleted",
		svcsdk.VpcPeeringConnectionStateReasonCodeDeleting,
	)
)

var (
	requeueWaitWhileDeleting = ackrequeue.NeededAfter(
		ErrVPCPeeringConnectionDeleting,
		5*time.Second,
	)
	requeueWaitWhileCreating = ackrequeue.NeededAfter(
		ErrVPCPeeringConnectionCreating,
		5*time.Second,
	)
	requeueWaitWhileProvisioning = ackrequeue.NeededAfter(
		ErrVPCPeeringConnectionProvisioning,
		5*time.Second,
	)
)

func isVPCPeeringConnectionCreating(r *resource) bool {
	if r.ko.Status.Status == nil || r.ko.Status.Status.Code == nil {
		return false
	}
	status := *r.ko.Status.Status.Code
	return status == string(svcapitypes.VPCPeeringConnectionStateReasonCode_initiating_request)
}

func isVPCPeeringConnectionProvisioning(r *resource) bool {
	if r.ko.Status.Status == nil || r.ko.Status.Status.Code == nil {
		return false
	}
	status := *r.ko.Status.Status.Code
	return status == string(svcapitypes.VPCPeeringConnectionStateReasonCode_provisioning)
}

func isVPCPeeringConnectionDeleting(r *resource) bool {
	if r.ko.Status.Status == nil || r.ko.Status.Status.Code == nil {
		return false
	}
	status := *r.ko.Status.Status.Code
	return status == string(svcapitypes.VPCPeeringConnectionStateReasonCode_deleting)
}

func isVPCPeeringConnectionActive(r *resource) bool {
	if r.ko.Status.Status == nil || r.ko.Status.Status.Code == nil {
		return false
	}
	status := *r.ko.Status.Status.Code
	return status == string(svcapitypes.VPCPeeringConnectionStateReasonCode_active)
}

func isVPCPeeringConnectionPendingAcceptance(r *resource) bool {
	if r.ko.Status.Status == nil || r.ko.Status.Status.Code == nil {
		return false
	}
	status := *r.ko.Status.Status.Code
	return status == string(svcapitypes.VPCPeeringConnectionStateReasonCode_pending_acceptance)
}

// updateTagSpecificationsInCreateRequest adds
// Tags defined in the Spec to CreateVpcPeeringConnectionInput.TagSpecification
// and ensures the ResourceType is always set to 'vpc-peering-connection'
func updateTagSpecificationsInCreateRequest(r *resource,
	input *svcsdk.CreateVpcPeeringConnectionInput) {
	input.TagSpecifications = nil
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
		desiredTagSpecs.SetResourceType("vpc-peering-connection")
		desiredTagSpecs.SetTags(requestedTags)
		input.TagSpecifications = []*svcsdk.TagSpecification{&desiredTagSpecs}
	}
}

var syncTags = tags.Sync

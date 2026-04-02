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

package fleet

import (
	"fmt"
	"time"

	"github.com/aws-controllers-k8s/ec2-controller/pkg/tags"
	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackrequeue "github.com/aws-controllers-k8s/runtime/pkg/requeue"

	svcsdk "github.com/aws/aws-sdk-go-v2/service/ec2"
	svcsdktypes "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// updateTagSpecificationsInCreateRequest adds
// Tags defined in the Spec to CreateFleetInput.TagSpecification
// and ensures the ResourceType is always set to 'fleet'
func updateTagSpecificationsInCreateRequest(r *resource,
	input *svcsdk.CreateFleetInput) {
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
		desiredTagSpecs.ResourceType = "fleet"
		desiredTagSpecs.Tags = requestedTags
		input.TagSpecifications = []svcsdktypes.TagSpecification{desiredTagSpecs}
	}
}

var syncTags = tags.Sync

var computeTagsDelta = tags.ComputeTagsDelta

var (
	ErrFleetDeleting = fmt.Errorf(
		"Fleet in '%v/%v' state, cannot be modified or deleted",
		svcsdktypes.FleetStateCodeDeletedTerminatingInstances, svcsdktypes.FleetStateCodeDeletedRunning,
	)
	requeueWaitWhileDeleting = ackrequeue.NeededAfter(
		ErrFleetDeleting,
		5*time.Second,
	)
)

// addIDToDeleteRequest adds resource's Fleet ID to DeleteRequest.
// Return error to indicate to callers that the resource is not yet created.
func addIDToDeleteRequest(r *resource,
	input *svcsdk.DeleteFleetsInput) error {
	if r.ko.Status.FleetID == nil {
		return fmt.Errorf("unable to extract FleetID from resource")
	}
	input.FleetIds = []string{*r.ko.Status.FleetID}
	return nil
}

// fleetDeleting returns true if the supplied Fleet is in the process
// of being deleted
func fleetDeleting(r *resource) bool {
	if r.ko.Status.FleetState == nil {
		return false
	}
	fleetState := *r.ko.Status.FleetState
	if fleetState == string(svcsdktypes.FleetStateCodeDeletedTerminatingInstances) || fleetState == string(svcsdktypes.FleetStateCodeDeletedRunning) {
		return true
	}
	return false
}

// fleetDeleted checks if the Fleet is fully deleted
func fleetDeleted(r *resource) bool {
	if r.ko.Status.FleetState == nil {
		return false
	}

	fleetState := *r.ko.Status.FleetState
	return fleetState == string(svcsdktypes.FleetStateCodeDeleted)
}

// If immutable fields are defined in AWS but not in spec, use AWS values
// This applies when adopting existing fleets
func customPreCompare(
	delta *ackcompare.Delta,
	a *resource,
	b *resource,
) {
	if a.ko.Spec.SpotOptions == nil {
		a.ko.Spec.SpotOptions = b.ko.Spec.SpotOptions
	}
	if a.ko.Spec.OnDemandOptions == nil {
		a.ko.Spec.OnDemandOptions = b.ko.Spec.OnDemandOptions
	}

}

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
	"context"
	"fmt"
	"time"

	svcapitypes "github.com/aws-controllers-k8s/ec2-controller/apis/v1alpha1"
	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackrequeue "github.com/aws-controllers-k8s/runtime/pkg/requeue"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
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

// syncTags used to keep tags in sync by calling Create and Delete API's
func (rm *resourceManager) syncTags(
	ctx context.Context,
	desired *resource,
	latest *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.syncTags")
	defer func(err error) {
		exit(err)
	}(err)

	resourceId := []*string{latest.ko.Status.VPCPeeringConnectionID}

	toAdd, toDelete := computeTagsDelta(
		desired.ko.Spec.Tags, latest.ko.Spec.Tags,
	)

	if len(toDelete) > 0 {
		rlog.Debug("removing tags from vpc peering connection resource", "tags", toDelete)
		_, err = rm.sdkapi.DeleteTagsWithContext(
			ctx,
			&svcsdk.DeleteTagsInput{
				Resources: resourceId,
				Tags:      rm.sdkTags(toDelete),
			},
		)
		rm.metrics.RecordAPICall("UPDATE", "DeleteTags", err)
		if err != nil {
			return err
		}

	}

	if len(toAdd) > 0 {
		rlog.Debug("adding tags to vpc peering connection resource", "tags", toAdd)
		_, err = rm.sdkapi.CreateTagsWithContext(
			ctx,
			&svcsdk.CreateTagsInput{
				Resources: resourceId,
				Tags:      rm.sdkTags(toAdd),
			},
		)
		rm.metrics.RecordAPICall("UPDATE", "CreateTags", err)
		if err != nil {
			return err
		}
	}

	return nil
}

// sdkTags converts *svcapitypes.Tag array to a *svcsdk.Tag array
func (rm *resourceManager) sdkTags(
	tags []*svcapitypes.Tag,
) (sdktags []*svcsdk.Tag) {

	for _, i := range tags {
		sdktag := rm.newTag(*i)
		sdktags = append(sdktags, sdktag)
	}

	return sdktags
}

// computeTagsDelta returns tags to be added and removed from the resource
func computeTagsDelta(
	desired []*svcapitypes.Tag,
	latest []*svcapitypes.Tag,
) (toAdd []*svcapitypes.Tag, toDelete []*svcapitypes.Tag) {

	desiredTags := map[string]string{}
	for _, tag := range desired {
		desiredTags[*tag.Key] = *tag.Value
	}

	latestTags := map[string]string{}
	for _, tag := range latest {
		latestTags[*tag.Key] = *tag.Value
	}

	for _, tag := range desired {
		val, ok := latestTags[*tag.Key]
		if !ok || val != *tag.Value {
			toAdd = append(toAdd, tag)
		}
	}

	for _, tag := range latest {
		_, ok := desiredTags[*tag.Key]
		if !ok {
			toDelete = append(toDelete, tag)
		}
	}

	return toAdd, toDelete

}

// compareTags is a custom comparison function for comparing lists of Tag
// structs where the order of the structs in the list is not important.
func compareTags(
	delta *ackcompare.Delta,
	a *resource,
	b *resource,
) {
	if len(a.ko.Spec.Tags) != len(b.ko.Spec.Tags) {
		delta.Add("Spec.Tags", a.ko.Spec.Tags, b.ko.Spec.Tags)
	} else if len(a.ko.Spec.Tags) > 0 {
		addedOrUpdated, removed := computeTagsDelta(a.ko.Spec.Tags, b.ko.Spec.Tags)
		if len(addedOrUpdated) != 0 || len(removed) != 0 {
			delta.Add("Spec.Tags", a.ko.Spec.Tags, b.ko.Spec.Tags)
		}
	}
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

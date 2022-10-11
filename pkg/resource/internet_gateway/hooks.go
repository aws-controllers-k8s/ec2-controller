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

package internet_gateway

import (
	"context"

	svcapitypes "github.com/aws-controllers-k8s/ec2-controller/apis/v1alpha1"
	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	svcsdk "github.com/aws/aws-sdk-go/service/ec2"
)

func (rm *resourceManager) customUpdateInternetGateway(
	ctx context.Context,
	desired *resource,
	latest *resource,
	delta *ackcompare.Delta,
) (updated *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.customUpdateInternetGateway")
	defer func(err error) {
		exit(err)
	}(err)

	// Default `updated` to `desired` because it is likely
	// EC2 `modify` APIs do NOT return output, only errors.
	// If the `modify` calls (i.e. `sync`) do NOT return
	// an error, then the update was successful and desired.Spec
	// (now updated.Spec) reflects the latest resource state.
	updated = desired

	if delta.DifferentAt("Spec.VPC") {
		if latest.ko.Spec.VPC != nil {
			if err = rm.detachFromVPC(ctx, *latest.ko.Spec.VPC, *latest.ko.Status.InternetGatewayID); err != nil {
				return nil, err
			}
		}
		if desired.ko.Spec.VPC != nil {
			if err = rm.attachToVPC(ctx, desired); err != nil {
				return nil, err
			}
		}
	}

	if delta.DifferentAt("Spec.Tags") {
		if err = rm.syncTags(ctx, desired, latest); err != nil {
			return nil, err
		}
	}

	return updated, nil
}

// getAttachedVPC will attempt to find the VPCID for any VPC that the
// InternetGateway is currently attached to. If it is not attached, or is
// actively being detached, then it will return nil.
func (rm *resourceManager) getAttachedVPC(
	ctx context.Context,
	latest *resource,
) (vpcID *string, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.getAttachedVPC")
	defer func(err error) {
		exit(err)
	}(err)

	// InternetGateways can only be attached to a single VPC at a time - even
	// though attachments is a slice. Attaching is almost instant, but if the
	// request returns that it is in `Attaching` status still, we can assume
	// that it will be attached in the near future and does not need to be
	// updated.
	for _, att := range latest.ko.Status.Attachments {
		// There is no `AttachmentStatusAvailable` - so we can just check by
		// using negative logic with the constants we have, instead
		if *att.State != svcsdk.AttachmentStatusDetached &&
			*att.State != svcsdk.AttachmentStatusDetaching {
			return att.VPCID, nil
		}
	}

	return nil, nil
}

func (rm *resourceManager) attachToVPC(
	ctx context.Context,
	desired *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.attachToVPC")
	defer func(err error) {
		exit(err)
	}(err)

	input := &svcsdk.AttachInternetGatewayInput{
		InternetGatewayId: desired.ko.Status.InternetGatewayID,
		VpcId:             desired.ko.Spec.VPC,
	}

	_, err = rm.sdkapi.AttachInternetGatewayWithContext(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "AttachInternetGateway", err)
	if err != nil {
		return err
	}

	return nil
}

func (rm *resourceManager) detachFromVPC(
	ctx context.Context,
	vpcID string,
	igwID string,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.detachFromVPC")
	defer func(err error) {
		exit(err)
	}(err)

	input := &svcsdk.DetachInternetGatewayInput{
		InternetGatewayId: &igwID,
		VpcId:             &vpcID,
	}

	_, err = rm.sdkapi.DetachInternetGatewayWithContext(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "DetachInternetGateway", err)
	if err != nil {
		return err
	}

	return nil
}

// syncTags used to keep tags in sync by calling Create and Delete Tags API's
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

	resourceId := []*string{latest.ko.Status.InternetGatewayID}

	toAdd, toDelete := computeTagsDelta(
		desired.ko.Spec.Tags, latest.ko.Spec.Tags,
	)

	if len(toDelete) > 0 {
		rlog.Debug("removing tags from InternetGateway resource", "tags", toDelete)
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
		rlog.Debug("adding tags to InternetGateway resource", "tags", toAdd)
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

// updateTagSpecificationsInCreateRequest adds
// Tags defined in the Spec to CreateInternetGatewayInput.TagSpecification
// and ensures the ResourceType is always set to 'internet-gateway'
func updateTagSpecificationsInCreateRequest(r *resource,
	input *svcsdk.CreateInternetGatewayInput) {
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
		desiredTagSpecs.SetResourceType("internet-gateway")
		desiredTagSpecs.SetTags(requestedTags)
		input.TagSpecifications = []*svcsdk.TagSpecification{&desiredTagSpecs}
	}
}

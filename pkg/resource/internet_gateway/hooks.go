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

	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	ackutils "github.com/aws-controllers-k8s/runtime/pkg/util"
	svcsdk "github.com/aws/aws-sdk-go-v2/service/ec2"
	svcsdktypes "github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/aws-controllers-k8s/ec2-controller/pkg/tags"
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
	updated = rm.concreteResource(desired.DeepCopy())

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

	if delta.DifferentAt("Spec.RouteTables") {
		if err = rm.updateRouteTableAssociations(ctx, desired, latest, delta); err != nil {
			return nil, err
		}
	}

	if delta.DifferentAt("Spec.Tags") {
		if err := tags.Sync(
			ctx, rm.sdkapi, rm.metrics, *latest.ko.Status.InternetGatewayID,
			desired.ko.Spec.Tags, latest.ko.Spec.Tags,
		); err != nil {
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
		if *att.State != string(svcsdktypes.AttachmentStatusDetached) &&
			*att.State != string(svcsdktypes.AttachmentStatusDetaching) {
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

	_, err = rm.sdkapi.AttachInternetGateway(ctx, input)
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

	_, err = rm.sdkapi.DetachInternetGateway(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "DetachInternetGateway", err)
	if err != nil {
		return err
	}

	return nil
}

// updateTagSpecificationsInCreateRequest adds
// Tags defined in the Spec to CreateInternetGatewayInput.TagSpecification
// and ensures the ResourceType is always set to 'internet-gateway'
func updateTagSpecificationsInCreateRequest(r *resource,
	input *svcsdk.CreateInternetGatewayInput) {
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
		desiredTagSpecs.ResourceType = "internet-gateway"
		desiredTagSpecs.Tags = requestedTags
		input.TagSpecifications = []svcsdktypes.TagSpecification{desiredTagSpecs}
	}
}

func (rm *resourceManager) createRouteTableAssociations(
	ctx context.Context,
	desired *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.createRouteTableAssociations")
	defer func(err error) {
		exit(err)
	}(err)

	if len(desired.ko.Spec.RouteTables) == 0 {
		return nil
	}

	for _, rt := range desired.ko.Spec.RouteTables {
		if err = rm.associateRouteTable(ctx, *rt, *desired.ko.Status.InternetGatewayID); err != nil {
			return err
		}
	}

	return nil
}

func (rm *resourceManager) updateRouteTableAssociations(
	ctx context.Context,
	desired *resource,
	latest *resource,
	delta *ackcompare.Delta,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.updateRouteTableAssociations")
	defer func(err error) {
		exit(err)
	}(err)

	existingRTs, err := rm.getRouteTableAssociations(ctx, latest)
	if err != nil {
		return err
	}

	toAdd := make([]*string, 0)
	toDelete := make([]svcsdktypes.RouteTableAssociation, 0)

	for _, rt := range existingRTs {
		if !ackutils.InStringPs(*rt.RouteTableId, desired.ko.Spec.RouteTables) {
			toDelete = append(toDelete, rt)
		}
	}

	for _, rt := range desired.ko.Spec.RouteTables {
		if !inAssociations(*rt, existingRTs) {
			toAdd = append(toAdd, rt)
		}
	}

	for _, t := range toDelete {
		rlog.Debug("disassocationg route table from internet gateway", "route_table_id", *t.RouteTableId, "association_id", *t.RouteTableAssociationId)
		if err = rm.disassociateRouteTable(ctx, *t.RouteTableAssociationId); err != nil {
			return err
		}
	}
	for _, rt := range toAdd {
		rlog.Debug("associating route table with internet gateway", "route_table_id", *rt)
		if err = rm.associateRouteTable(ctx, *rt, *latest.ko.Status.InternetGatewayID); err != nil {
			return err
		}
	}

	return nil
}

// associateRouteTable will associate a RouteTable (using its RouteTableID) with
// the given internet gateway (using its internet gateway ID)
func (rm *resourceManager) associateRouteTable(
	ctx context.Context,
	rtID string,
	igwID string,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.associateRouteTable")
	defer func(err error) {
		exit(err)
	}(err)

	input := &svcsdk.AssociateRouteTableInput{
		RouteTableId: &rtID,
		GatewayId:    &igwID,
	}

	_, err = rm.sdkapi.AssociateRouteTable(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "AssociateRouteTable", err)
	if err != nil {
		return err
	}

	return nil
}

// disassociateRouteTable will disassociate a RouteTable from a internet
// gateway based on its association ID, which is given by the API in the
// output of the Associate* command but can also be found in the
// description of the route table under the list of its associations.
func (rm *resourceManager) disassociateRouteTable(
	ctx context.Context,
	associationID string,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.disassociateRouteTable")
	defer func(err error) {
		exit(err)
	}(err)

	input := &svcsdk.DisassociateRouteTableInput{
		AssociationId: &associationID,
	}

	_, err = rm.sdkapi.DisassociateRouteTable(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "DisassociateRouteTable", err)
	if err != nil {
		return err
	}

	return nil
}

// getRouteTableAssociations finds all of the route tables that are associated
// with the current internet gateway and returns a list of each of the
// association resources.
func (rm *resourceManager) getRouteTableAssociations(
	ctx context.Context,
	res *resource,
) (rtAssociations []svcsdktypes.RouteTableAssociation, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.getRouteTableAssociations")
	defer func(err error) {
		exit(err)
	}(err)

	input := &svcsdk.DescribeRouteTablesInput{}

	for {
		resp, err := rm.sdkapi.DescribeRouteTables(ctx, input)
		if err != nil || resp == nil {
			break
		}

		rm.metrics.RecordAPICall("GET", "DescribeRouteTables", err)
		for _, rt := range resp.RouteTables {
			// Find the association for the current internet gateway
			for _, rtAssociation := range rt.Associations {
				if rtAssociation.GatewayId == nil || res.ko.Status.InternetGatewayID == nil {
					continue
				}
				if *rtAssociation.GatewayId == *res.ko.Status.InternetGatewayID && rtAssociation.AssociationState.State == svcsdktypes.RouteTableAssociationStateCodeAssociated {
					rtAssociations = append(rtAssociations, rtAssociation)
					break
				}
			}
		}
		if resp.NextToken == nil || *resp.NextToken == "" {
			break
		}
		input.NextToken = resp.NextToken
	}
	return rtAssociations, nil
}

// inAssociations returns true if a route table ID is present in the list of
// route table assocations.
func inAssociations(
	rtID string,
	assocs []svcsdktypes.RouteTableAssociation,
) bool {
	for _, a := range assocs {
		if *a.RouteTableId == rtID {
			return true
		}
	}
	return false
}

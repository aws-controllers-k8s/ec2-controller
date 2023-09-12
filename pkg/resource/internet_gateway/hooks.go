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
	ackutils "github.com/aws-controllers-k8s/runtime/pkg/util"
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
	toDelete := make([]*svcsdk.RouteTableAssociation, 0)

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

	_, err = rm.sdkapi.AssociateRouteTableWithContext(ctx, input)
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

	_, err = rm.sdkapi.DisassociateRouteTableWithContext(ctx, input)
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
) (rtAssociations []*svcsdk.RouteTableAssociation, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.getRouteTableAssociations")
	defer func(err error) {
		exit(err)
	}(err)

	input := &svcsdk.DescribeRouteTablesInput{}

	for {
		resp, err := rm.sdkapi.DescribeRouteTablesWithContext(ctx, input)
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
				if *rtAssociation.GatewayId == *res.ko.Status.InternetGatewayID && *rtAssociation.AssociationState.State == "associated" {
					rtAssociations = append(rtAssociations, rtAssociation)
					break
				}
			}
		}
		if resp.NextToken == nil || *resp.NextToken == "" {
			break
		}
		input.SetNextToken(*resp.NextToken)
	}
	return rtAssociations, nil
}

// inAssociations returns true if a route table ID is present in the list of
// route table assocations.
func inAssociations(
	rtID string,
	assocs []*svcsdk.RouteTableAssociation,
) bool {
	for _, a := range assocs {
		if *a.RouteTableId == rtID {
			return true
		}
	}
	return false
}

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

package subnet

import (
	"context"

	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	ackutils "github.com/aws-controllers-k8s/runtime/pkg/util"
	svcsdk "github.com/aws/aws-sdk-go/service/ec2"

	"github.com/aws-controllers-k8s/ec2-controller/pkg/tags"
)

func (rm *resourceManager) customUpdateSubnet(
	ctx context.Context,
	desired *resource,
	latest *resource,
	delta *ackcompare.Delta,
) (updated *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.customUpdateSubnet")
	defer func(err error) {
		exit(err)
	}(err)

	// Default `updated` to `desired` because it is likely
	// EC2 `modify` APIs do NOT return output, only errors.
	// If the `modify` calls (i.e. `sync`) do NOT return
	// an error, then the update was successful and desired.Spec
	// (now updated.Spec) reflects the latest resource state.
	updated = rm.concreteResource(desired.DeepCopy())

	if delta.DifferentAt("Spec.RouteTables") {
		if err = rm.updateRouteTableAssociations(ctx, desired, latest, delta); err != nil {
			return nil, err
		}
	}

	if delta.DifferentAt("Spec.Tags") {
		if err := tags.Sync(
			ctx, rm.sdkapi, rm.metrics, *latest.ko.Status.SubnetID,
			desired.ko.Spec.Tags, latest.ko.Spec.Tags,
		); err != nil {
			return nil, err
		}
	}

	// These fields must be edited one at a time
	if delta.DifferentAt("Spec.AssignIPv6AddressOnCreation") {
		if err = rm.updateSubnetAttribute(ctx, desired, "AssignIPv6AddressOnCreation"); err != nil {
			return nil, err
		}
	}
	if delta.DifferentAt("Spec.CustomerOwnedIPv4Pool") {
		if err = rm.updateSubnetAttribute(ctx, desired, "CustomerOwnedIPv4Pool"); err != nil {
			return nil, err
		}
		// Spec/Status fields may need updating as a result of modifying another field;
		// However, a call to sdkFind is unneccessary because only a small subset of fields
		// need updating AND their values are known (logically should be desired.Spec values, if no errors)
		boolp := false
		if desired.ko.Spec.CustomerOwnedIPv4Pool != nil {
			boolp = true
			updated.ko.Status.MapCustomerOwnedIPOnLaunch = &boolp
		} else {
			updated.ko.Status.MapCustomerOwnedIPOnLaunch = &boolp
		}
	}
	if delta.DifferentAt("Spec.EnableDNS64") {
		if err = rm.updateSubnetAttribute(ctx, desired, "EnableDNS64"); err != nil {
			return nil, err
		}
	}
	if delta.DifferentAt("Spec.EnableResourceNameDNSAAAARecord") {
		if err = rm.updateSubnetAttribute(ctx, desired, "EnableResourceNameDNSAAAARecord"); err != nil {
			return nil, err
		}
		updated.ko.Status.PrivateDNSNameOptionsOnLaunch.EnableResourceNameDNSAAAARecord = desired.ko.Spec.EnableResourceNameDNSAAAARecord
	}
	if delta.DifferentAt("Spec.EnableResourceNameDNSARecord") {
		if err = rm.updateSubnetAttribute(ctx, desired, "EnableResourceNameDNSARecord"); err != nil {
			return nil, err
		}
		updated.ko.Status.PrivateDNSNameOptionsOnLaunch.EnableResourceNameDNSARecord = desired.ko.Spec.EnableResourceNameDNSARecord
	}
	if delta.DifferentAt("Spec.HostnameType") {
		if err = rm.updateSubnetAttribute(ctx, desired, "HostnameType"); err != nil {
			return nil, err
		}
		updated.ko.Status.PrivateDNSNameOptionsOnLaunch.HostnameType = desired.ko.Spec.HostnameType
	}
	if delta.DifferentAt("Spec.MapPublicIPOnLaunch") {
		if err = rm.updateSubnetAttribute(ctx, desired, "MapPublicIPOnLaunch"); err != nil {
			return nil, err
		}
	}

	return updated, nil
}

// updateSubnetAttribute creates a request based on
// deltaFieldName and calls ModifySubnetAttribute API
// Note, this API can only edit one field at a time
func (rm *resourceManager) updateSubnetAttribute(
	ctx context.Context,
	r *resource,
	deltaFieldName string,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.updateSubnetAttributes")
	defer func(err error) {
		exit(err)
	}(err)

	input := &svcsdk.ModifySubnetAttributeInput{
		SubnetId: r.ko.Status.SubnetID,
	}

	defaultAttrValue := svcsdk.AttributeBooleanValue{}
	defaultAttrValue.SetValue(false)
	switch deltaFieldName {
	case "AssignIPv6AddressOnCreation":
		input.SetAssignIpv6AddressOnCreation(&defaultAttrValue)
		if r.ko.Spec.AssignIPv6AddressOnCreation != nil {
			input.AssignIpv6AddressOnCreation.Value = r.ko.Spec.AssignIPv6AddressOnCreation
		}
	case "CustomerOwnedIPv4Pool":
		input.SetMapCustomerOwnedIpOnLaunch(&defaultAttrValue)
		if r.ko.Spec.CustomerOwnedIPv4Pool != nil {
			input.MapCustomerOwnedIpOnLaunch.SetValue(true)
			input.CustomerOwnedIpv4Pool = r.ko.Spec.CustomerOwnedIPv4Pool
		}
	case "EnableDNS64":
		input.SetEnableDns64(&defaultAttrValue)
		if r.ko.Spec.EnableDNS64 != nil {
			input.EnableDns64.Value = r.ko.Spec.EnableDNS64
		}
	case "EnableResourceNameDNSAAAARecord":
		input.SetEnableResourceNameDnsAAAARecordOnLaunch(&defaultAttrValue)
		if r.ko.Spec.EnableResourceNameDNSAAAARecord != nil {
			input.EnableResourceNameDnsAAAARecordOnLaunch.Value = r.ko.Spec.EnableResourceNameDNSAAAARecord
		}
	case "EnableResourceNameDNSARecord":
		input.SetEnableResourceNameDnsARecordOnLaunch(&defaultAttrValue)
		if r.ko.Spec.EnableResourceNameDNSARecord != nil {
			input.EnableResourceNameDnsARecordOnLaunch.Value = r.ko.Spec.EnableResourceNameDNSARecord
		}
	case "HostnameType":
		input.SetPrivateDnsHostnameTypeOnLaunch("ip-name")
		if r.ko.Spec.HostnameType != nil {
			input.PrivateDnsHostnameTypeOnLaunch = r.ko.Spec.HostnameType
		}
	case "MapPublicIPOnLaunch":
		input.SetMapPublicIpOnLaunch(&defaultAttrValue)
		if r.ko.Spec.MapPublicIPOnLaunch != nil {
			input.MapPublicIpOnLaunch.Value = r.ko.Spec.MapPublicIPOnLaunch
		}
	}

	_, err = rm.sdkapi.ModifySubnetAttributeWithContext(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "ModifySubnetAttribute", err)
	if err != nil {
		return err
	}

	return nil
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
		if err = rm.associateRouteTable(ctx, *rt, *desired.ko.Status.SubnetID); err != nil {
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
		rlog.Debug("disassocationg route table from subnet", "route_table_id", *t.RouteTableId, "association_id", *t.RouteTableAssociationId)
		if err = rm.disassociateRouteTable(ctx, *t.RouteTableAssociationId); err != nil {
			return err
		}
	}
	for _, rt := range toAdd {
		rlog.Debug("associating route table with subnet", "route_table_id", *rt)
		if err = rm.associateRouteTable(ctx, *rt, *latest.ko.Status.SubnetID); err != nil {
			return err
		}
	}

	return nil
}

// associateRouteTable will associate a RouteTable (using its RouteTableID) with
// the given Subnet (using its SubnetID)
func (rm *resourceManager) associateRouteTable(
	ctx context.Context,
	rtID string,
	subnetID string,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.associateRouteTable")
	defer func(err error) {
		exit(err)
	}(err)

	input := &svcsdk.AssociateRouteTableInput{
		RouteTableId: &rtID,
		SubnetId:     &subnetID,
	}

	_, err = rm.sdkapi.AssociateRouteTableWithContext(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "AssociateRouteTable", err)
	if err != nil {
		return err
	}

	return nil
}

// disassociateRouteTable will disassociate a RouteTable from a Subnet based on
// its association ID, which is given by the API in the output of the Associate*
// command but can also be found in the description of the route table under the
// list of its associations.
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
// with the current subnet and returns a list of each of the association
// resources.
func (rm *resourceManager) getRouteTableAssociations(
	ctx context.Context,
	res *resource,
) (assocs []*svcsdk.RouteTableAssociation, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.getAssociatedRouteTables")
	defer func(err error) {
		exit(err)
	}(err)

	input := &svcsdk.DescribeRouteTablesInput{
		Filters: []*svcsdk.Filter{
			{
				Name:   toStrPtr("association.subnet-id"),
				Values: []*string{res.ko.Status.SubnetID},
			},
		},
	}

	for {
		resp, err := rm.sdkapi.DescribeRouteTablesWithContext(ctx, input)
		if err != nil || resp == nil {
			break
		}
		rm.metrics.RecordAPICall("GET", "DescribeRouteTables", err)
		for _, rt := range resp.RouteTables {
			var assoc *svcsdk.RouteTableAssociation
			// Find the association for the current subnet
			for _, as := range rt.Associations {
				if *as.SubnetId == *res.ko.Status.SubnetID {
					assoc = as
					break
				}
			}

			// If we can't find the assocation, something has gone wrong because
			// our filter on the input was meant to have caught this.
			if assoc == nil {
				continue
			}

			assocs = append(assocs, assoc)
		}
		if resp.NextToken == nil || *resp.NextToken == "" {
			break
		}
		input.SetNextToken(*resp.NextToken)
	}
	return assocs, nil
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

func toStrPtr(str string) *string {
	return &str
}

// updateTagSpecificationsInCreateRequest adds
// Tags defined in the Spec to CreateSubnetInput.TagSpecification
// and ensures the ResourceType is always set to 'subnet'
func updateTagSpecificationsInCreateRequest(r *resource,
	input *svcsdk.CreateSubnetInput) {
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
		desiredTagSpecs.SetResourceType("subnet")
		desiredTagSpecs.SetTags(requestedTags)
		input.TagSpecifications = []*svcsdk.TagSpecification{&desiredTagSpecs}
	}
}

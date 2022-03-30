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

	ko := desired.ko.DeepCopy()
	rm.setStatusDefaults(ko)

	if delta.DifferentAt("Spec.RouteTables") {
		if err = rm.updateRouteTableAssociations(ctx, desired, latest, delta); err != nil {
			return nil, err
		}
	}

	return &resource{ko}, nil
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

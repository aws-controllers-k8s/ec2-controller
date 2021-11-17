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

// Code generated by ack-generate. DO NOT EDIT.

package route_table

import (
	"context"
	"reflect"
	"strings"

	ackv1alpha1 "github.com/aws-controllers-k8s/runtime/apis/core/v1alpha1"
	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackcondition "github.com/aws-controllers-k8s/runtime/pkg/condition"
	ackerr "github.com/aws-controllers-k8s/runtime/pkg/errors"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	"github.com/aws/aws-sdk-go/aws"
	svcsdk "github.com/aws/aws-sdk-go/service/ec2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	svcapitypes "github.com/aws-controllers-k8s/ec2-controller/apis/v1alpha1"
)

// Hack to avoid import errors during build...
var (
	_ = &metav1.Time{}
	_ = strings.ToLower("")
	_ = &aws.JSONValue{}
	_ = &svcsdk.EC2{}
	_ = &svcapitypes.RouteTable{}
	_ = ackv1alpha1.AWSAccountID("")
	_ = &ackerr.NotFound
	_ = &ackcondition.NotManagedMessage
	_ = &reflect.Value{}
)

// sdkFind returns SDK-specific information about a supplied resource
func (rm *resourceManager) sdkFind(
	ctx context.Context,
	r *resource,
) (latest *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.sdkFind")
	defer exit(err)
	// If any required fields in the input shape are missing, AWS resource is
	// not created yet. Return NotFound here to indicate to callers that the
	// resource isn't yet created.
	if rm.requiredFieldsMissingFromReadManyInput(r) {
		return nil, ackerr.NotFound
	}

	input, err := rm.newListRequestPayload(r)
	if err != nil {
		return nil, err
	}
	var resp *svcsdk.DescribeRouteTablesOutput
	resp, err = rm.sdkapi.DescribeRouteTablesWithContext(ctx, input)
	rm.metrics.RecordAPICall("READ_MANY", "DescribeRouteTables", err)
	if err != nil {
		if awsErr, ok := ackerr.AWSError(err); ok && awsErr.Code() == "UNKNOWN" {
			return nil, ackerr.NotFound
		}
		return nil, err
	}

	// Merge in the information we read from the API call above to the copy of
	// the original Kubernetes object we passed to the function
	ko := r.ko.DeepCopy()

	found := false
	for _, elem := range resp.RouteTables {
		if elem.Associations != nil {
			f0 := []*svcapitypes.RouteTableAssociation{}
			for _, f0iter := range elem.Associations {
				f0elem := &svcapitypes.RouteTableAssociation{}
				if f0iter.AssociationState != nil {
					f0elemf0 := &svcapitypes.RouteTableAssociationState{}
					if f0iter.AssociationState.State != nil {
						f0elemf0.State = f0iter.AssociationState.State
					}
					if f0iter.AssociationState.StatusMessage != nil {
						f0elemf0.StatusMessage = f0iter.AssociationState.StatusMessage
					}
					f0elem.AssociationState = f0elemf0
				}
				if f0iter.GatewayId != nil {
					f0elem.GatewayID = f0iter.GatewayId
				}
				if f0iter.Main != nil {
					f0elem.Main = f0iter.Main
				}
				if f0iter.RouteTableAssociationId != nil {
					f0elem.RouteTableAssociationID = f0iter.RouteTableAssociationId
				}
				if f0iter.RouteTableId != nil {
					f0elem.RouteTableID = f0iter.RouteTableId
				}
				if f0iter.SubnetId != nil {
					f0elem.SubnetID = f0iter.SubnetId
				}
				f0 = append(f0, f0elem)
			}
			ko.Status.Associations = f0
		} else {
			ko.Status.Associations = nil
		}
		if elem.OwnerId != nil {
			ko.Status.OwnerID = elem.OwnerId
		} else {
			ko.Status.OwnerID = nil
		}
		if elem.PropagatingVgws != nil {
			f2 := []*svcapitypes.PropagatingVGW{}
			for _, f2iter := range elem.PropagatingVgws {
				f2elem := &svcapitypes.PropagatingVGW{}
				if f2iter.GatewayId != nil {
					f2elem.GatewayID = f2iter.GatewayId
				}
				f2 = append(f2, f2elem)
			}
			ko.Status.PropagatingVGWs = f2
		} else {
			ko.Status.PropagatingVGWs = nil
		}
		if elem.RouteTableId != nil {
			ko.Status.RouteTableID = elem.RouteTableId
		} else {
			ko.Status.RouteTableID = nil
		}
		if elem.Routes != nil {
			f4 := []*svcapitypes.CreateRouteInput{}
			for _, f4iter := range elem.Routes {
				f4elem := &svcapitypes.CreateRouteInput{}
				if f4iter.CarrierGatewayId != nil {
					f4elem.CarrierGatewayID = f4iter.CarrierGatewayId
				}
				if f4iter.DestinationCidrBlock != nil {
					f4elem.DestinationCIDRBlock = f4iter.DestinationCidrBlock
				}
				if f4iter.DestinationIpv6CidrBlock != nil {
					f4elem.DestinationIPv6CIDRBlock = f4iter.DestinationIpv6CidrBlock
				}
				if f4iter.DestinationPrefixListId != nil {
					f4elem.DestinationPrefixListID = f4iter.DestinationPrefixListId
				}
				if f4iter.EgressOnlyInternetGatewayId != nil {
					f4elem.EgressOnlyInternetGatewayID = f4iter.EgressOnlyInternetGatewayId
				}
				if f4iter.GatewayId != nil {
					f4elem.GatewayID = f4iter.GatewayId
				}
				if f4iter.InstanceId != nil {
					f4elem.InstanceID = f4iter.InstanceId
				}
				if f4iter.LocalGatewayId != nil {
					f4elem.LocalGatewayID = f4iter.LocalGatewayId
				}
				if f4iter.NatGatewayId != nil {
					f4elem.NATGatewayID = f4iter.NatGatewayId
				}
				if f4iter.NetworkInterfaceId != nil {
					f4elem.NetworkInterfaceID = f4iter.NetworkInterfaceId
				}
				if f4iter.TransitGatewayId != nil {
					f4elem.TransitGatewayID = f4iter.TransitGatewayId
				}
				if f4iter.VpcPeeringConnectionId != nil {
					f4elem.VPCPeeringConnectionID = f4iter.VpcPeeringConnectionId
				}
				f4 = append(f4, f4elem)
			}
			ko.Spec.Routes = f4
		} else {
			ko.Spec.Routes = nil
		}
		if elem.Tags != nil {
			f5 := []*svcapitypes.Tag{}
			for _, f5iter := range elem.Tags {
				f5elem := &svcapitypes.Tag{}
				if f5iter.Key != nil {
					f5elem.Key = f5iter.Key
				}
				if f5iter.Value != nil {
					f5elem.Value = f5iter.Value
				}
				f5 = append(f5, f5elem)
			}
			ko.Status.Tags = f5
		} else {
			ko.Status.Tags = nil
		}
		if elem.VpcId != nil {
			ko.Spec.VPCID = elem.VpcId
		} else {
			ko.Spec.VPCID = nil
		}
		found = true
		break
	}
	if !found {
		return nil, ackerr.NotFound
	}

	rm.setStatusDefaults(ko)

	if found {
		rm.addRoutesToStatus(ko, resp.RouteTables[0])
	}

	return &resource{ko}, nil
}

// requiredFieldsMissingFromReadManyInput returns true if there are any fields
// for the ReadMany Input shape that are required but not present in the
// resource's Spec or Status
func (rm *resourceManager) requiredFieldsMissingFromReadManyInput(
	r *resource,
) bool {
	return r.ko.Status.RouteTableID == nil

}

// newListRequestPayload returns SDK-specific struct for the HTTP request
// payload of the List API call for the resource
func (rm *resourceManager) newListRequestPayload(
	r *resource,
) (*svcsdk.DescribeRouteTablesInput, error) {
	res := &svcsdk.DescribeRouteTablesInput{}

	if r.ko.Status.RouteTableID != nil {
		f4 := []*string{}
		f4 = append(f4, r.ko.Status.RouteTableID)
		res.SetRouteTableIds(f4)
	}

	return res, nil
}

// sdkCreate creates the supplied resource in the backend AWS service API and
// returns a copy of the resource with resource fields (in both Spec and
// Status) filled in with values from the CREATE API operation's Output shape.
func (rm *resourceManager) sdkCreate(
	ctx context.Context,
	desired *resource,
) (created *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.sdkCreate")
	defer exit(err)
	input, err := rm.newCreateRequestPayload(ctx, desired)
	if err != nil {
		return nil, err
	}

	var resp *svcsdk.CreateRouteTableOutput
	_ = resp
	resp, err = rm.sdkapi.CreateRouteTableWithContext(ctx, input)
	rm.metrics.RecordAPICall("CREATE", "CreateRouteTable", err)
	if err != nil {
		return nil, err
	}
	// Merge in the information we read from the API call above to the copy of
	// the original Kubernetes object we passed to the function
	ko := desired.ko.DeepCopy()

	if resp.RouteTable.Associations != nil {
		f0 := []*svcapitypes.RouteTableAssociation{}
		for _, f0iter := range resp.RouteTable.Associations {
			f0elem := &svcapitypes.RouteTableAssociation{}
			if f0iter.AssociationState != nil {
				f0elemf0 := &svcapitypes.RouteTableAssociationState{}
				if f0iter.AssociationState.State != nil {
					f0elemf0.State = f0iter.AssociationState.State
				}
				if f0iter.AssociationState.StatusMessage != nil {
					f0elemf0.StatusMessage = f0iter.AssociationState.StatusMessage
				}
				f0elem.AssociationState = f0elemf0
			}
			if f0iter.GatewayId != nil {
				f0elem.GatewayID = f0iter.GatewayId
			}
			if f0iter.Main != nil {
				f0elem.Main = f0iter.Main
			}
			if f0iter.RouteTableAssociationId != nil {
				f0elem.RouteTableAssociationID = f0iter.RouteTableAssociationId
			}
			if f0iter.RouteTableId != nil {
				f0elem.RouteTableID = f0iter.RouteTableId
			}
			if f0iter.SubnetId != nil {
				f0elem.SubnetID = f0iter.SubnetId
			}
			f0 = append(f0, f0elem)
		}
		ko.Status.Associations = f0
	} else {
		ko.Status.Associations = nil
	}
	if resp.RouteTable.OwnerId != nil {
		ko.Status.OwnerID = resp.RouteTable.OwnerId
	} else {
		ko.Status.OwnerID = nil
	}
	if resp.RouteTable.PropagatingVgws != nil {
		f2 := []*svcapitypes.PropagatingVGW{}
		for _, f2iter := range resp.RouteTable.PropagatingVgws {
			f2elem := &svcapitypes.PropagatingVGW{}
			if f2iter.GatewayId != nil {
				f2elem.GatewayID = f2iter.GatewayId
			}
			f2 = append(f2, f2elem)
		}
		ko.Status.PropagatingVGWs = f2
	} else {
		ko.Status.PropagatingVGWs = nil
	}
	if resp.RouteTable.RouteTableId != nil {
		ko.Status.RouteTableID = resp.RouteTable.RouteTableId
	} else {
		ko.Status.RouteTableID = nil
	}
	if resp.RouteTable.Routes != nil {
		f4 := []*svcapitypes.CreateRouteInput{}
		for _, f4iter := range resp.RouteTable.Routes {
			f4elem := &svcapitypes.CreateRouteInput{}
			if f4iter.CarrierGatewayId != nil {
				f4elem.CarrierGatewayID = f4iter.CarrierGatewayId
			}
			if f4iter.DestinationCidrBlock != nil {
				f4elem.DestinationCIDRBlock = f4iter.DestinationCidrBlock
			}
			if f4iter.DestinationIpv6CidrBlock != nil {
				f4elem.DestinationIPv6CIDRBlock = f4iter.DestinationIpv6CidrBlock
			}
			if f4iter.DestinationPrefixListId != nil {
				f4elem.DestinationPrefixListID = f4iter.DestinationPrefixListId
			}
			if f4iter.EgressOnlyInternetGatewayId != nil {
				f4elem.EgressOnlyInternetGatewayID = f4iter.EgressOnlyInternetGatewayId
			}
			if f4iter.GatewayId != nil {
				f4elem.GatewayID = f4iter.GatewayId
			}
			if f4iter.InstanceId != nil {
				f4elem.InstanceID = f4iter.InstanceId
			}
			if f4iter.LocalGatewayId != nil {
				f4elem.LocalGatewayID = f4iter.LocalGatewayId
			}
			if f4iter.NatGatewayId != nil {
				f4elem.NATGatewayID = f4iter.NatGatewayId
			}
			if f4iter.NetworkInterfaceId != nil {
				f4elem.NetworkInterfaceID = f4iter.NetworkInterfaceId
			}
			if f4iter.TransitGatewayId != nil {
				f4elem.TransitGatewayID = f4iter.TransitGatewayId
			}
			if f4iter.VpcPeeringConnectionId != nil {
				f4elem.VPCPeeringConnectionID = f4iter.VpcPeeringConnectionId
			}
			f4 = append(f4, f4elem)
		}
		ko.Spec.Routes = f4
	} else {
		ko.Spec.Routes = nil
	}
	if resp.RouteTable.Tags != nil {
		f5 := []*svcapitypes.Tag{}
		for _, f5iter := range resp.RouteTable.Tags {
			f5elem := &svcapitypes.Tag{}
			if f5iter.Key != nil {
				f5elem.Key = f5iter.Key
			}
			if f5iter.Value != nil {
				f5elem.Value = f5iter.Value
			}
			f5 = append(f5, f5elem)
		}
		ko.Status.Tags = f5
	} else {
		ko.Status.Tags = nil
	}
	if resp.RouteTable.VpcId != nil {
		ko.Spec.VPCID = resp.RouteTable.VpcId
	} else {
		ko.Spec.VPCID = nil
	}

	rm.setStatusDefaults(ko)
	rm.addRoutesToStatus(ko, resp.RouteTable)

	if rm.requiredFieldsMissingForCreateRoute(&resource{ko}) {
		return nil, ackerr.NotFound
	}

	if len(desired.ko.Spec.Routes) > 0 {
		//desired routes are overwritten by RouteTable's default route
		ko.Spec.Routes = append(ko.Spec.Routes, desired.ko.Spec.Routes...)
		if err := rm.createRoutes(ctx, &resource{ko}); err != nil {
			return nil, err
		}
	}
	return &resource{ko}, nil
}

// newCreateRequestPayload returns an SDK-specific struct for the HTTP request
// payload of the Create API call for the resource
func (rm *resourceManager) newCreateRequestPayload(
	ctx context.Context,
	r *resource,
) (*svcsdk.CreateRouteTableInput, error) {
	res := &svcsdk.CreateRouteTableInput{}

	if r.ko.Spec.TagSpecifications != nil {
		f0 := []*svcsdk.TagSpecification{}
		for _, f0iter := range r.ko.Spec.TagSpecifications {
			f0elem := &svcsdk.TagSpecification{}
			if f0iter.ResourceType != nil {
				f0elem.SetResourceType(*f0iter.ResourceType)
			}
			if f0iter.Tags != nil {
				f0elemf1 := []*svcsdk.Tag{}
				for _, f0elemf1iter := range f0iter.Tags {
					f0elemf1elem := &svcsdk.Tag{}
					if f0elemf1iter.Key != nil {
						f0elemf1elem.SetKey(*f0elemf1iter.Key)
					}
					if f0elemf1iter.Value != nil {
						f0elemf1elem.SetValue(*f0elemf1iter.Value)
					}
					f0elemf1 = append(f0elemf1, f0elemf1elem)
				}
				f0elem.SetTags(f0elemf1)
			}
			f0 = append(f0, f0elem)
		}
		res.SetTagSpecifications(f0)
	}
	if r.ko.Spec.VPCID != nil {
		res.SetVpcId(*r.ko.Spec.VPCID)
	}

	return res, nil
}

// sdkUpdate patches the supplied resource in the backend AWS service API and
// returns a new resource with updated fields.
func (rm *resourceManager) sdkUpdate(
	ctx context.Context,
	desired *resource,
	latest *resource,
	delta *ackcompare.Delta,
) (*resource, error) {
	return rm.customUpdateRouteTable(ctx, desired, latest, delta)
}

// sdkDelete deletes the supplied resource in the backend AWS service API
func (rm *resourceManager) sdkDelete(
	ctx context.Context,
	r *resource,
) (latest *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.sdkDelete")
	defer exit(err)
	input, err := rm.newDeleteRequestPayload(r)
	if err != nil {
		return nil, err
	}
	var resp *svcsdk.DeleteRouteTableOutput
	_ = resp
	resp, err = rm.sdkapi.DeleteRouteTableWithContext(ctx, input)
	rm.metrics.RecordAPICall("DELETE", "DeleteRouteTable", err)
	return nil, err
}

// newDeleteRequestPayload returns an SDK-specific struct for the HTTP request
// payload of the Delete API call for the resource
func (rm *resourceManager) newDeleteRequestPayload(
	r *resource,
) (*svcsdk.DeleteRouteTableInput, error) {
	res := &svcsdk.DeleteRouteTableInput{}

	if r.ko.Status.RouteTableID != nil {
		res.SetRouteTableId(*r.ko.Status.RouteTableID)
	}

	return res, nil
}

// setStatusDefaults sets default properties into supplied custom resource
func (rm *resourceManager) setStatusDefaults(
	ko *svcapitypes.RouteTable,
) {
	if ko.Status.ACKResourceMetadata == nil {
		ko.Status.ACKResourceMetadata = &ackv1alpha1.ResourceMetadata{}
	}
	if ko.Status.ACKResourceMetadata.OwnerAccountID == nil {
		ko.Status.ACKResourceMetadata.OwnerAccountID = &rm.awsAccountID
	}
	if ko.Status.Conditions == nil {
		ko.Status.Conditions = []*ackv1alpha1.Condition{}
	}
}

// updateConditions returns updated resource, true; if conditions were updated
// else it returns nil, false
func (rm *resourceManager) updateConditions(
	r *resource,
	onSuccess bool,
	err error,
) (*resource, bool) {
	ko := r.ko.DeepCopy()
	rm.setStatusDefaults(ko)

	// Terminal condition
	var terminalCondition *ackv1alpha1.Condition = nil
	var recoverableCondition *ackv1alpha1.Condition = nil
	var syncCondition *ackv1alpha1.Condition = nil
	for _, condition := range ko.Status.Conditions {
		if condition.Type == ackv1alpha1.ConditionTypeTerminal {
			terminalCondition = condition
		}
		if condition.Type == ackv1alpha1.ConditionTypeRecoverable {
			recoverableCondition = condition
		}
		if condition.Type == ackv1alpha1.ConditionTypeResourceSynced {
			syncCondition = condition
		}
	}

	if rm.terminalAWSError(err) || err == ackerr.SecretTypeNotSupported || err == ackerr.SecretNotFound {
		if terminalCondition == nil {
			terminalCondition = &ackv1alpha1.Condition{
				Type: ackv1alpha1.ConditionTypeTerminal,
			}
			ko.Status.Conditions = append(ko.Status.Conditions, terminalCondition)
		}
		var errorMessage = ""
		if err == ackerr.SecretTypeNotSupported || err == ackerr.SecretNotFound {
			errorMessage = err.Error()
		} else {
			awsErr, _ := ackerr.AWSError(err)
			errorMessage = awsErr.Error()
		}
		terminalCondition.Status = corev1.ConditionTrue
		terminalCondition.Message = &errorMessage
	} else {
		// Clear the terminal condition if no longer present
		if terminalCondition != nil {
			terminalCondition.Status = corev1.ConditionFalse
			terminalCondition.Message = nil
		}
		// Handling Recoverable Conditions
		if err != nil {
			if recoverableCondition == nil {
				// Add a new Condition containing a non-terminal error
				recoverableCondition = &ackv1alpha1.Condition{
					Type: ackv1alpha1.ConditionTypeRecoverable,
				}
				ko.Status.Conditions = append(ko.Status.Conditions, recoverableCondition)
			}
			recoverableCondition.Status = corev1.ConditionTrue
			awsErr, _ := ackerr.AWSError(err)
			errorMessage := err.Error()
			if awsErr != nil {
				errorMessage = awsErr.Error()
			}
			recoverableCondition.Message = &errorMessage
		} else if recoverableCondition != nil {
			recoverableCondition.Status = corev1.ConditionFalse
			recoverableCondition.Message = nil
		}
	}
	// Required to avoid the "declared but not used" error in the default case
	_ = syncCondition
	if terminalCondition != nil || recoverableCondition != nil || syncCondition != nil {
		return &resource{ko}, true // updated
	}
	return nil, false // not updated
}

// terminalAWSError returns awserr, true; if the supplied error is an aws Error type
// and if the exception indicates that it is a Terminal exception
// 'Terminal' exception are specified in generator configuration
func (rm *resourceManager) terminalAWSError(err error) bool {
	if err == nil {
		return false
	}
	awsErr, ok := ackerr.AWSError(err)
	if !ok {
		return false
	}
	switch awsErr.Code() {
	case "InvalidVpcID.Malformed",
		"InvalidVpcID.NotFound",
		"InvalidParameterValue":
		return true
	default:
		return false
	}
}

func compareCreateRouteInput(
	a *svcapitypes.CreateRouteInput,
	b *svcapitypes.CreateRouteInput,
) *ackcompare.Delta {
	delta := ackcompare.NewDelta()
	if ackcompare.HasNilDifference(a.CarrierGatewayID, b.CarrierGatewayID) {
		delta.Add("CreateRouteInput.CarrierGatewayID", a.CarrierGatewayID, b.CarrierGatewayID)
	} else if a.CarrierGatewayID != nil && b.CarrierGatewayID != nil {
		if *a.CarrierGatewayID != *b.CarrierGatewayID {
			delta.Add("CreateRouteInput.CarrierGatewayID", a.CarrierGatewayID, b.CarrierGatewayID)
		}
	}
	if ackcompare.HasNilDifference(a.DestinationCIDRBlock, b.DestinationCIDRBlock) {
		delta.Add("CreateRouteInput.DestinationCIDRBlock", a.DestinationCIDRBlock, b.DestinationCIDRBlock)
	} else if a.DestinationCIDRBlock != nil && b.DestinationCIDRBlock != nil {
		if *a.DestinationCIDRBlock != *b.DestinationCIDRBlock {
			delta.Add("CreateRouteInput.DestinationCIDRBlock", a.DestinationCIDRBlock, b.DestinationCIDRBlock)
		}
	}
	if ackcompare.HasNilDifference(a.DestinationIPv6CIDRBlock, b.DestinationIPv6CIDRBlock) {
		delta.Add("CreateRouteInput.DestinationIPv6CIDRBlock", a.DestinationIPv6CIDRBlock, b.DestinationIPv6CIDRBlock)
	} else if a.DestinationIPv6CIDRBlock != nil && b.DestinationIPv6CIDRBlock != nil {
		if *a.DestinationIPv6CIDRBlock != *b.DestinationIPv6CIDRBlock {
			delta.Add("CreateRouteInput.DestinationIPv6CIDRBlock", a.DestinationIPv6CIDRBlock, b.DestinationIPv6CIDRBlock)
		}
	}
	if ackcompare.HasNilDifference(a.DestinationPrefixListID, b.DestinationPrefixListID) {
		delta.Add("CreateRouteInput.DestinationPrefixListID", a.DestinationPrefixListID, b.DestinationPrefixListID)
	} else if a.DestinationPrefixListID != nil && b.DestinationPrefixListID != nil {
		if *a.DestinationPrefixListID != *b.DestinationPrefixListID {
			delta.Add("CreateRouteInput.DestinationPrefixListID", a.DestinationPrefixListID, b.DestinationPrefixListID)
		}
	}
	if ackcompare.HasNilDifference(a.EgressOnlyInternetGatewayID, b.EgressOnlyInternetGatewayID) {
		delta.Add("CreateRouteInput.EgressOnlyInternetGatewayID", a.EgressOnlyInternetGatewayID, b.EgressOnlyInternetGatewayID)
	} else if a.EgressOnlyInternetGatewayID != nil && b.EgressOnlyInternetGatewayID != nil {
		if *a.EgressOnlyInternetGatewayID != *b.EgressOnlyInternetGatewayID {
			delta.Add("CreateRouteInput.EgressOnlyInternetGatewayID", a.EgressOnlyInternetGatewayID, b.EgressOnlyInternetGatewayID)
		}
	}
	if ackcompare.HasNilDifference(a.GatewayID, b.GatewayID) {
		delta.Add("CreateRouteInput.GatewayID", a.GatewayID, b.GatewayID)
	} else if a.GatewayID != nil && b.GatewayID != nil {
		if *a.GatewayID != *b.GatewayID {
			delta.Add("CreateRouteInput.GatewayID", a.GatewayID, b.GatewayID)
		}
	}
	if ackcompare.HasNilDifference(a.InstanceID, b.InstanceID) {
		delta.Add("CreateRouteInput.InstanceID", a.InstanceID, b.InstanceID)
	} else if a.InstanceID != nil && b.InstanceID != nil {
		if *a.InstanceID != *b.InstanceID {
			delta.Add("CreateRouteInput.InstanceID", a.InstanceID, b.InstanceID)
		}
	}
	if ackcompare.HasNilDifference(a.LocalGatewayID, b.LocalGatewayID) {
		delta.Add("CreateRouteInput.LocalGatewayID", a.LocalGatewayID, b.LocalGatewayID)
	} else if a.LocalGatewayID != nil && b.LocalGatewayID != nil {
		if *a.LocalGatewayID != *b.LocalGatewayID {
			delta.Add("CreateRouteInput.LocalGatewayID", a.LocalGatewayID, b.LocalGatewayID)
		}
	}
	if ackcompare.HasNilDifference(a.NATGatewayID, b.NATGatewayID) {
		delta.Add("CreateRouteInput.NATGatewayID", a.NATGatewayID, b.NATGatewayID)
	} else if a.NATGatewayID != nil && b.NATGatewayID != nil {
		if *a.NATGatewayID != *b.NATGatewayID {
			delta.Add("CreateRouteInput.NATGatewayID", a.NATGatewayID, b.NATGatewayID)
		}
	}
	if ackcompare.HasNilDifference(a.NetworkInterfaceID, b.NetworkInterfaceID) {
		delta.Add("CreateRouteInput.NetworkInterfaceID", a.NetworkInterfaceID, b.NetworkInterfaceID)
	} else if a.NetworkInterfaceID != nil && b.NetworkInterfaceID != nil {
		if *a.NetworkInterfaceID != *b.NetworkInterfaceID {
			delta.Add("CreateRouteInput.NetworkInterfaceID", a.NetworkInterfaceID, b.NetworkInterfaceID)
		}
	}
	if ackcompare.HasNilDifference(a.RouteTableID, b.RouteTableID) {
		delta.Add("CreateRouteInput.RouteTableID", a.RouteTableID, b.RouteTableID)
	} else if a.RouteTableID != nil && b.RouteTableID != nil {
		if *a.RouteTableID != *b.RouteTableID {
			delta.Add("CreateRouteInput.RouteTableID", a.RouteTableID, b.RouteTableID)
		}
	}
	if ackcompare.HasNilDifference(a.TransitGatewayID, b.TransitGatewayID) {
		delta.Add("CreateRouteInput.TransitGatewayID", a.TransitGatewayID, b.TransitGatewayID)
	} else if a.TransitGatewayID != nil && b.TransitGatewayID != nil {
		if *a.TransitGatewayID != *b.TransitGatewayID {
			delta.Add("CreateRouteInput.TransitGatewayID", a.TransitGatewayID, b.TransitGatewayID)
		}
	}
	if ackcompare.HasNilDifference(a.VPCEndpointID, b.VPCEndpointID) {
		delta.Add("CreateRouteInput.VPCEndpointID", a.VPCEndpointID, b.VPCEndpointID)
	} else if a.VPCEndpointID != nil && b.VPCEndpointID != nil {
		if *a.VPCEndpointID != *b.VPCEndpointID {
			delta.Add("CreateRouteInput.VPCEndpointID", a.VPCEndpointID, b.VPCEndpointID)
		}
	}
	if ackcompare.HasNilDifference(a.VPCPeeringConnectionID, b.VPCPeeringConnectionID) {
		delta.Add("CreateRouteInput.VPCPeeringConnectionID", a.VPCPeeringConnectionID, b.VPCPeeringConnectionID)
	} else if a.VPCPeeringConnectionID != nil && b.VPCPeeringConnectionID != nil {
		if *a.VPCPeeringConnectionID != *b.VPCPeeringConnectionID {
			delta.Add("CreateRouteInput.VPCPeeringConnectionID", a.VPCPeeringConnectionID, b.VPCPeeringConnectionID)
		}
	}

	return delta
}

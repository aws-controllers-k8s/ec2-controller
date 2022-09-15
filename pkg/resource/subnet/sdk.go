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

package subnet

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	ackv1alpha1 "github.com/aws-controllers-k8s/runtime/apis/core/v1alpha1"
	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackcondition "github.com/aws-controllers-k8s/runtime/pkg/condition"
	ackerr "github.com/aws-controllers-k8s/runtime/pkg/errors"
	ackrequeue "github.com/aws-controllers-k8s/runtime/pkg/requeue"
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
	_ = &svcapitypes.Subnet{}
	_ = ackv1alpha1.AWSAccountID("")
	_ = &ackerr.NotFound
	_ = &ackcondition.NotManagedMessage
	_ = &reflect.Value{}
	_ = fmt.Sprintf("")
	_ = &ackrequeue.NoRequeue{}
)

// sdkFind returns SDK-specific information about a supplied resource
func (rm *resourceManager) sdkFind(
	ctx context.Context,
	r *resource,
) (latest *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.sdkFind")
	defer func() {
		exit(err)
	}()
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
	var resp *svcsdk.DescribeSubnetsOutput
	resp, err = rm.sdkapi.DescribeSubnetsWithContext(ctx, input)
	rm.metrics.RecordAPICall("READ_MANY", "DescribeSubnets", err)
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
	for _, elem := range resp.Subnets {
		if elem.AssignIpv6AddressOnCreation != nil {
			ko.Spec.AssignIPv6AddressOnCreation = elem.AssignIpv6AddressOnCreation
		} else {
			ko.Spec.AssignIPv6AddressOnCreation = nil
		}
		if elem.AvailabilityZone != nil {
			ko.Spec.AvailabilityZone = elem.AvailabilityZone
		} else {
			ko.Spec.AvailabilityZone = nil
		}
		if elem.AvailabilityZoneId != nil {
			ko.Spec.AvailabilityZoneID = elem.AvailabilityZoneId
		} else {
			ko.Spec.AvailabilityZoneID = nil
		}
		if elem.AvailableIpAddressCount != nil {
			ko.Status.AvailableIPAddressCount = elem.AvailableIpAddressCount
		} else {
			ko.Status.AvailableIPAddressCount = nil
		}
		if elem.CidrBlock != nil {
			ko.Spec.CIDRBlock = elem.CidrBlock
		} else {
			ko.Spec.CIDRBlock = nil
		}
		if elem.CustomerOwnedIpv4Pool != nil {
			ko.Spec.CustomerOwnedIPv4Pool = elem.CustomerOwnedIpv4Pool
		} else {
			ko.Spec.CustomerOwnedIPv4Pool = nil
		}
		if elem.DefaultForAz != nil {
			ko.Status.DefaultForAZ = elem.DefaultForAz
		} else {
			ko.Status.DefaultForAZ = nil
		}
		if elem.EnableDns64 != nil {
			ko.Spec.EnableDNS64 = elem.EnableDns64
		} else {
			ko.Spec.EnableDNS64 = nil
		}
		if elem.EnableLniAtDeviceIndex != nil {
			ko.Status.EnableLniAtDeviceIndex = elem.EnableLniAtDeviceIndex
		} else {
			ko.Status.EnableLniAtDeviceIndex = nil
		}
		if elem.Ipv6CidrBlockAssociationSet != nil {
			f9 := []*svcapitypes.SubnetIPv6CIDRBlockAssociation{}
			for _, f9iter := range elem.Ipv6CidrBlockAssociationSet {
				f9elem := &svcapitypes.SubnetIPv6CIDRBlockAssociation{}
				if f9iter.AssociationId != nil {
					f9elem.AssociationID = f9iter.AssociationId
				}
				if f9iter.Ipv6CidrBlock != nil {
					f9elem.IPv6CIDRBlock = f9iter.Ipv6CidrBlock
				}
				if f9iter.Ipv6CidrBlockState != nil {
					f9elemf2 := &svcapitypes.SubnetCIDRBlockState{}
					if f9iter.Ipv6CidrBlockState.State != nil {
						f9elemf2.State = f9iter.Ipv6CidrBlockState.State
					}
					if f9iter.Ipv6CidrBlockState.StatusMessage != nil {
						f9elemf2.StatusMessage = f9iter.Ipv6CidrBlockState.StatusMessage
					}
					f9elem.IPv6CIDRBlockState = f9elemf2
				}
				f9 = append(f9, f9elem)
			}
			ko.Status.IPv6CIDRBlockAssociationSet = f9
		} else {
			ko.Status.IPv6CIDRBlockAssociationSet = nil
		}
		if elem.Ipv6Native != nil {
			ko.Spec.IPv6Native = elem.Ipv6Native
		} else {
			ko.Spec.IPv6Native = nil
		}
		if elem.MapCustomerOwnedIpOnLaunch != nil {
			ko.Status.MapCustomerOwnedIPOnLaunch = elem.MapCustomerOwnedIpOnLaunch
		} else {
			ko.Status.MapCustomerOwnedIPOnLaunch = nil
		}
		if elem.MapPublicIpOnLaunch != nil {
			ko.Spec.MapPublicIPOnLaunch = elem.MapPublicIpOnLaunch
		} else {
			ko.Spec.MapPublicIPOnLaunch = nil
		}
		if elem.OutpostArn != nil {
			ko.Spec.OutpostARN = elem.OutpostArn
		} else {
			ko.Spec.OutpostARN = nil
		}
		if elem.OwnerId != nil {
			ko.Status.OwnerID = elem.OwnerId
		} else {
			ko.Status.OwnerID = nil
		}
		if elem.PrivateDnsNameOptionsOnLaunch != nil {
			f15 := &svcapitypes.PrivateDNSNameOptionsOnLaunch{}
			if elem.PrivateDnsNameOptionsOnLaunch.EnableResourceNameDnsAAAARecord != nil {
				f15.EnableResourceNameDNSAAAARecord = elem.PrivateDnsNameOptionsOnLaunch.EnableResourceNameDnsAAAARecord
			}
			if elem.PrivateDnsNameOptionsOnLaunch.EnableResourceNameDnsARecord != nil {
				f15.EnableResourceNameDNSARecord = elem.PrivateDnsNameOptionsOnLaunch.EnableResourceNameDnsARecord
			}
			if elem.PrivateDnsNameOptionsOnLaunch.HostnameType != nil {
				f15.HostnameType = elem.PrivateDnsNameOptionsOnLaunch.HostnameType
			}
			ko.Status.PrivateDNSNameOptionsOnLaunch = f15
		} else {
			ko.Status.PrivateDNSNameOptionsOnLaunch = nil
		}
		if elem.State != nil {
			ko.Status.State = elem.State
		} else {
			ko.Status.State = nil
		}
		if elem.SubnetArn != nil {
			if ko.Status.ACKResourceMetadata == nil {
				ko.Status.ACKResourceMetadata = &ackv1alpha1.ResourceMetadata{}
			}
			tmpARN := ackv1alpha1.AWSResourceName(*elem.SubnetArn)
			ko.Status.ACKResourceMetadata.ARN = &tmpARN
		}
		if elem.SubnetId != nil {
			ko.Status.SubnetID = elem.SubnetId
		} else {
			ko.Status.SubnetID = nil
		}
		if elem.Tags != nil {
			f19 := []*svcapitypes.Tag{}
			for _, f19iter := range elem.Tags {
				f19elem := &svcapitypes.Tag{}
				if f19iter.Key != nil {
					f19elem.Key = f19iter.Key
				}
				if f19iter.Value != nil {
					f19elem.Value = f19iter.Value
				}
				f19 = append(f19, f19elem)
			}
			ko.Spec.Tags = f19
		} else {
			ko.Spec.Tags = nil
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
	assocs, err := rm.getRouteTableAssociations(ctx, &resource{ko})
	if err != nil {
		return nil, err
	} else {
		ko.Spec.RouteTables = make([]*string, len(assocs))
		for i, assoc := range assocs {
			ko.Spec.RouteTables[i] = assoc.RouteTableId
		}
	}
	return &resource{ko}, nil
}

// requiredFieldsMissingFromReadManyInput returns true if there are any fields
// for the ReadMany Input shape that are required but not present in the
// resource's Spec or Status
func (rm *resourceManager) requiredFieldsMissingFromReadManyInput(
	r *resource,
) bool {
	return r.ko.Status.SubnetID == nil

}

// newListRequestPayload returns SDK-specific struct for the HTTP request
// payload of the List API call for the resource
func (rm *resourceManager) newListRequestPayload(
	r *resource,
) (*svcsdk.DescribeSubnetsInput, error) {
	res := &svcsdk.DescribeSubnetsInput{}

	if r.ko.Status.SubnetID != nil {
		f4 := []*string{}
		f4 = append(f4, r.ko.Status.SubnetID)
		res.SetSubnetIds(f4)
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
	defer func() {
		exit(err)
	}()
	input, err := rm.newCreateRequestPayload(ctx, desired)
	if err != nil {
		return nil, err
	}
	updateTagSpecificationsInCreateRequest(desired, input)

	var resp *svcsdk.CreateSubnetOutput
	_ = resp
	resp, err = rm.sdkapi.CreateSubnetWithContext(ctx, input)
	rm.metrics.RecordAPICall("CREATE", "CreateSubnet", err)
	if err != nil {
		return nil, err
	}
	// Merge in the information we read from the API call above to the copy of
	// the original Kubernetes object we passed to the function
	ko := desired.ko.DeepCopy()

	if resp.Subnet.AssignIpv6AddressOnCreation != nil {
		ko.Spec.AssignIPv6AddressOnCreation = resp.Subnet.AssignIpv6AddressOnCreation
	} else {
		ko.Spec.AssignIPv6AddressOnCreation = nil
	}
	if resp.Subnet.AvailabilityZone != nil {
		ko.Spec.AvailabilityZone = resp.Subnet.AvailabilityZone
	} else {
		ko.Spec.AvailabilityZone = nil
	}
	if resp.Subnet.AvailabilityZoneId != nil {
		ko.Spec.AvailabilityZoneID = resp.Subnet.AvailabilityZoneId
	} else {
		ko.Spec.AvailabilityZoneID = nil
	}
	if resp.Subnet.AvailableIpAddressCount != nil {
		ko.Status.AvailableIPAddressCount = resp.Subnet.AvailableIpAddressCount
	} else {
		ko.Status.AvailableIPAddressCount = nil
	}
	if resp.Subnet.CidrBlock != nil {
		ko.Spec.CIDRBlock = resp.Subnet.CidrBlock
	} else {
		ko.Spec.CIDRBlock = nil
	}
	if resp.Subnet.CustomerOwnedIpv4Pool != nil {
		ko.Spec.CustomerOwnedIPv4Pool = resp.Subnet.CustomerOwnedIpv4Pool
	} else {
		ko.Spec.CustomerOwnedIPv4Pool = nil
	}
	if resp.Subnet.DefaultForAz != nil {
		ko.Status.DefaultForAZ = resp.Subnet.DefaultForAz
	} else {
		ko.Status.DefaultForAZ = nil
	}
	if resp.Subnet.EnableDns64 != nil {
		ko.Spec.EnableDNS64 = resp.Subnet.EnableDns64
	} else {
		ko.Spec.EnableDNS64 = nil
	}
	if resp.Subnet.EnableLniAtDeviceIndex != nil {
		ko.Status.EnableLniAtDeviceIndex = resp.Subnet.EnableLniAtDeviceIndex
	} else {
		ko.Status.EnableLniAtDeviceIndex = nil
	}
	if resp.Subnet.Ipv6CidrBlockAssociationSet != nil {
		f9 := []*svcapitypes.SubnetIPv6CIDRBlockAssociation{}
		for _, f9iter := range resp.Subnet.Ipv6CidrBlockAssociationSet {
			f9elem := &svcapitypes.SubnetIPv6CIDRBlockAssociation{}
			if f9iter.AssociationId != nil {
				f9elem.AssociationID = f9iter.AssociationId
			}
			if f9iter.Ipv6CidrBlock != nil {
				f9elem.IPv6CIDRBlock = f9iter.Ipv6CidrBlock
			}
			if f9iter.Ipv6CidrBlockState != nil {
				f9elemf2 := &svcapitypes.SubnetCIDRBlockState{}
				if f9iter.Ipv6CidrBlockState.State != nil {
					f9elemf2.State = f9iter.Ipv6CidrBlockState.State
				}
				if f9iter.Ipv6CidrBlockState.StatusMessage != nil {
					f9elemf2.StatusMessage = f9iter.Ipv6CidrBlockState.StatusMessage
				}
				f9elem.IPv6CIDRBlockState = f9elemf2
			}
			f9 = append(f9, f9elem)
		}
		ko.Status.IPv6CIDRBlockAssociationSet = f9
	} else {
		ko.Status.IPv6CIDRBlockAssociationSet = nil
	}
	if resp.Subnet.Ipv6Native != nil {
		ko.Spec.IPv6Native = resp.Subnet.Ipv6Native
	} else {
		ko.Spec.IPv6Native = nil
	}
	if resp.Subnet.MapCustomerOwnedIpOnLaunch != nil {
		ko.Status.MapCustomerOwnedIPOnLaunch = resp.Subnet.MapCustomerOwnedIpOnLaunch
	} else {
		ko.Status.MapCustomerOwnedIPOnLaunch = nil
	}
	if resp.Subnet.MapPublicIpOnLaunch != nil {
		ko.Spec.MapPublicIPOnLaunch = resp.Subnet.MapPublicIpOnLaunch
	} else {
		ko.Spec.MapPublicIPOnLaunch = nil
	}
	if resp.Subnet.OutpostArn != nil {
		ko.Spec.OutpostARN = resp.Subnet.OutpostArn
	} else {
		ko.Spec.OutpostARN = nil
	}
	if resp.Subnet.OwnerId != nil {
		ko.Status.OwnerID = resp.Subnet.OwnerId
	} else {
		ko.Status.OwnerID = nil
	}
	if resp.Subnet.PrivateDnsNameOptionsOnLaunch != nil {
		f15 := &svcapitypes.PrivateDNSNameOptionsOnLaunch{}
		if resp.Subnet.PrivateDnsNameOptionsOnLaunch.EnableResourceNameDnsAAAARecord != nil {
			f15.EnableResourceNameDNSAAAARecord = resp.Subnet.PrivateDnsNameOptionsOnLaunch.EnableResourceNameDnsAAAARecord
		}
		if resp.Subnet.PrivateDnsNameOptionsOnLaunch.EnableResourceNameDnsARecord != nil {
			f15.EnableResourceNameDNSARecord = resp.Subnet.PrivateDnsNameOptionsOnLaunch.EnableResourceNameDnsARecord
		}
		if resp.Subnet.PrivateDnsNameOptionsOnLaunch.HostnameType != nil {
			f15.HostnameType = resp.Subnet.PrivateDnsNameOptionsOnLaunch.HostnameType
		}
		ko.Status.PrivateDNSNameOptionsOnLaunch = f15
	} else {
		ko.Status.PrivateDNSNameOptionsOnLaunch = nil
	}
	if resp.Subnet.State != nil {
		ko.Status.State = resp.Subnet.State
	} else {
		ko.Status.State = nil
	}
	if ko.Status.ACKResourceMetadata == nil {
		ko.Status.ACKResourceMetadata = &ackv1alpha1.ResourceMetadata{}
	}
	if resp.Subnet.SubnetArn != nil {
		arn := ackv1alpha1.AWSResourceName(*resp.Subnet.SubnetArn)
		ko.Status.ACKResourceMetadata.ARN = &arn
	}
	if resp.Subnet.SubnetId != nil {
		ko.Status.SubnetID = resp.Subnet.SubnetId
	} else {
		ko.Status.SubnetID = nil
	}
	if resp.Subnet.Tags != nil {
		f19 := []*svcapitypes.Tag{}
		for _, f19iter := range resp.Subnet.Tags {
			f19elem := &svcapitypes.Tag{}
			if f19iter.Key != nil {
				f19elem.Key = f19iter.Key
			}
			if f19iter.Value != nil {
				f19elem.Value = f19iter.Value
			}
			f19 = append(f19, f19elem)
		}
		ko.Spec.Tags = f19
	} else {
		ko.Spec.Tags = nil
	}
	if resp.Subnet.VpcId != nil {
		ko.Spec.VPCID = resp.Subnet.VpcId
	} else {
		ko.Spec.VPCID = nil
	}

	rm.setStatusDefaults(ko)
	if err = rm.createRouteTableAssociations(ctx, &resource{ko}); err != nil {
		return nil, err
	}
	return &resource{ko}, nil
}

// newCreateRequestPayload returns an SDK-specific struct for the HTTP request
// payload of the Create API call for the resource
func (rm *resourceManager) newCreateRequestPayload(
	ctx context.Context,
	r *resource,
) (*svcsdk.CreateSubnetInput, error) {
	res := &svcsdk.CreateSubnetInput{}

	if r.ko.Spec.AvailabilityZone != nil {
		res.SetAvailabilityZone(*r.ko.Spec.AvailabilityZone)
	}
	if r.ko.Spec.AvailabilityZoneID != nil {
		res.SetAvailabilityZoneId(*r.ko.Spec.AvailabilityZoneID)
	}
	if r.ko.Spec.CIDRBlock != nil {
		res.SetCidrBlock(*r.ko.Spec.CIDRBlock)
	}
	if r.ko.Spec.IPv6CIDRBlock != nil {
		res.SetIpv6CidrBlock(*r.ko.Spec.IPv6CIDRBlock)
	}
	if r.ko.Spec.IPv6Native != nil {
		res.SetIpv6Native(*r.ko.Spec.IPv6Native)
	}
	if r.ko.Spec.OutpostARN != nil {
		res.SetOutpostArn(*r.ko.Spec.OutpostARN)
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
	return rm.customUpdateSubnet(ctx, desired, latest, delta)
}

// sdkDelete deletes the supplied resource in the backend AWS service API
func (rm *resourceManager) sdkDelete(
	ctx context.Context,
	r *resource,
) (latest *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.sdkDelete")
	defer func() {
		exit(err)
	}()
	input, err := rm.newDeleteRequestPayload(r)
	if err != nil {
		return nil, err
	}
	var resp *svcsdk.DeleteSubnetOutput
	_ = resp
	resp, err = rm.sdkapi.DeleteSubnetWithContext(ctx, input)
	rm.metrics.RecordAPICall("DELETE", "DeleteSubnet", err)
	return nil, err
}

// newDeleteRequestPayload returns an SDK-specific struct for the HTTP request
// payload of the Delete API call for the resource
func (rm *resourceManager) newDeleteRequestPayload(
	r *resource,
) (*svcsdk.DeleteSubnetInput, error) {
	res := &svcsdk.DeleteSubnetInput{}

	if r.ko.Status.SubnetID != nil {
		res.SetSubnetId(*r.ko.Status.SubnetID)
	}

	return res, nil
}

// setStatusDefaults sets default properties into supplied custom resource
func (rm *resourceManager) setStatusDefaults(
	ko *svcapitypes.Subnet,
) {
	if ko.Status.ACKResourceMetadata == nil {
		ko.Status.ACKResourceMetadata = &ackv1alpha1.ResourceMetadata{}
	}
	if ko.Status.ACKResourceMetadata.Region == nil {
		ko.Status.ACKResourceMetadata.Region = &rm.awsRegion
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
	var termError *ackerr.TerminalError
	if rm.terminalAWSError(err) || err == ackerr.SecretTypeNotSupported || err == ackerr.SecretNotFound || errors.As(err, &termError) {
		if terminalCondition == nil {
			terminalCondition = &ackv1alpha1.Condition{
				Type: ackv1alpha1.ConditionTypeTerminal,
			}
			ko.Status.Conditions = append(ko.Status.Conditions, terminalCondition)
		}
		var errorMessage = ""
		if err == ackerr.SecretTypeNotSupported || err == ackerr.SecretNotFound || errors.As(err, &termError) {
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
	case "InvalidParameterValue":
		return true
	default:
		return false
	}
}

func (rm *resourceManager) newTag(
	c svcapitypes.Tag,
) *svcsdk.Tag {
	res := &svcsdk.Tag{}
	if c.Key != nil {
		res.SetKey(*c.Key)
	}
	if c.Value != nil {
		res.SetValue(*c.Value)
	}

	return res
}

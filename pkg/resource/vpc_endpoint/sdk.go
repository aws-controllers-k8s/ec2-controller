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

package vpc_endpoint

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
	_ = &svcapitypes.VPCEndpoint{}
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
	var resp *svcsdk.DescribeVpcEndpointsOutput
	resp, err = rm.sdkapi.DescribeVpcEndpointsWithContext(ctx, input)
	rm.metrics.RecordAPICall("READ_MANY", "DescribeVpcEndpoints", err)
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
	for _, elem := range resp.VpcEndpoints {
		if elem.CreationTimestamp != nil {
			ko.Status.CreationTimestamp = &metav1.Time{*elem.CreationTimestamp}
		} else {
			ko.Status.CreationTimestamp = nil
		}
		if elem.DnsEntries != nil {
			f1 := []*svcapitypes.DNSEntry{}
			for _, f1iter := range elem.DnsEntries {
				f1elem := &svcapitypes.DNSEntry{}
				if f1iter.DnsName != nil {
					f1elem.DNSName = f1iter.DnsName
				}
				if f1iter.HostedZoneId != nil {
					f1elem.HostedZoneID = f1iter.HostedZoneId
				}
				f1 = append(f1, f1elem)
			}
			ko.Status.DNSEntries = f1
		} else {
			ko.Status.DNSEntries = nil
		}
		if elem.Groups != nil {
			f2 := []*svcapitypes.SecurityGroupIdentifier{}
			for _, f2iter := range elem.Groups {
				f2elem := &svcapitypes.SecurityGroupIdentifier{}
				if f2iter.GroupId != nil {
					f2elem.GroupID = f2iter.GroupId
				}
				if f2iter.GroupName != nil {
					f2elem.GroupName = f2iter.GroupName
				}
				f2 = append(f2, f2elem)
			}
			ko.Status.Groups = f2
		} else {
			ko.Status.Groups = nil
		}
		if elem.LastError != nil {
			f3 := &svcapitypes.LastError{}
			if elem.LastError.Code != nil {
				f3.Code = elem.LastError.Code
			}
			if elem.LastError.Message != nil {
				f3.Message = elem.LastError.Message
			}
			ko.Status.LastError = f3
		} else {
			ko.Status.LastError = nil
		}
		if elem.NetworkInterfaceIds != nil {
			f4 := []*string{}
			for _, f4iter := range elem.NetworkInterfaceIds {
				var f4elem string
				f4elem = *f4iter
				f4 = append(f4, &f4elem)
			}
			ko.Status.NetworkInterfaceIDs = f4
		} else {
			ko.Status.NetworkInterfaceIDs = nil
		}
		if elem.OwnerId != nil {
			ko.Status.OwnerID = elem.OwnerId
		} else {
			ko.Status.OwnerID = nil
		}
		if elem.PolicyDocument != nil {
			ko.Spec.PolicyDocument = elem.PolicyDocument
		} else {
			ko.Spec.PolicyDocument = nil
		}
		if elem.PrivateDnsEnabled != nil {
			ko.Spec.PrivateDNSEnabled = elem.PrivateDnsEnabled
		} else {
			ko.Spec.PrivateDNSEnabled = nil
		}
		if elem.RequesterManaged != nil {
			ko.Status.RequesterManaged = elem.RequesterManaged
		} else {
			ko.Status.RequesterManaged = nil
		}
		if elem.RouteTableIds != nil {
			f9 := []*string{}
			for _, f9iter := range elem.RouteTableIds {
				var f9elem string
				f9elem = *f9iter
				f9 = append(f9, &f9elem)
			}
			ko.Spec.RouteTableIDs = f9
		} else {
			ko.Spec.RouteTableIDs = nil
		}
		if elem.ServiceName != nil {
			ko.Spec.ServiceName = elem.ServiceName
		} else {
			ko.Spec.ServiceName = nil
		}
		if elem.State != nil {
			ko.Status.State = elem.State
		} else {
			ko.Status.State = nil
		}
		if elem.SubnetIds != nil {
			f12 := []*string{}
			for _, f12iter := range elem.SubnetIds {
				var f12elem string
				f12elem = *f12iter
				f12 = append(f12, &f12elem)
			}
			ko.Spec.SubnetIDs = f12
		} else {
			ko.Spec.SubnetIDs = nil
		}
		if elem.Tags != nil {
			f13 := []*svcapitypes.Tag{}
			for _, f13iter := range elem.Tags {
				f13elem := &svcapitypes.Tag{}
				if f13iter.Key != nil {
					f13elem.Key = f13iter.Key
				}
				if f13iter.Value != nil {
					f13elem.Value = f13iter.Value
				}
				f13 = append(f13, f13elem)
			}
			ko.Status.Tags = f13
		} else {
			ko.Status.Tags = nil
		}
		if elem.VpcEndpointId != nil {
			ko.Status.VPCEndpointID = elem.VpcEndpointId
		} else {
			ko.Status.VPCEndpointID = nil
		}
		if elem.VpcEndpointType != nil {
			ko.Spec.VPCEndpointType = elem.VpcEndpointType
		} else {
			ko.Spec.VPCEndpointType = nil
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
	return &resource{ko}, nil
}

// requiredFieldsMissingFromReadManyInput returns true if there are any fields
// for the ReadMany Input shape that are required but not present in the
// resource's Spec or Status
func (rm *resourceManager) requiredFieldsMissingFromReadManyInput(
	r *resource,
) bool {
	return r.ko.Status.VPCEndpointID == nil

}

// newListRequestPayload returns SDK-specific struct for the HTTP request
// payload of the List API call for the resource
func (rm *resourceManager) newListRequestPayload(
	r *resource,
) (*svcsdk.DescribeVpcEndpointsInput, error) {
	res := &svcsdk.DescribeVpcEndpointsInput{}

	if r.ko.Status.VPCEndpointID != nil {
		f4 := []*string{}
		f4 = append(f4, r.ko.Status.VPCEndpointID)
		res.SetVpcEndpointIds(f4)
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

	var resp *svcsdk.CreateVpcEndpointOutput
	_ = resp
	resp, err = rm.sdkapi.CreateVpcEndpointWithContext(ctx, input)
	rm.metrics.RecordAPICall("CREATE", "CreateVpcEndpoint", err)
	if err != nil {
		return nil, err
	}
	// Merge in the information we read from the API call above to the copy of
	// the original Kubernetes object we passed to the function
	ko := desired.ko.DeepCopy()

	if resp.VpcEndpoint.CreationTimestamp != nil {
		ko.Status.CreationTimestamp = &metav1.Time{*resp.VpcEndpoint.CreationTimestamp}
	} else {
		ko.Status.CreationTimestamp = nil
	}
	if resp.VpcEndpoint.DnsEntries != nil {
		f1 := []*svcapitypes.DNSEntry{}
		for _, f1iter := range resp.VpcEndpoint.DnsEntries {
			f1elem := &svcapitypes.DNSEntry{}
			if f1iter.DnsName != nil {
				f1elem.DNSName = f1iter.DnsName
			}
			if f1iter.HostedZoneId != nil {
				f1elem.HostedZoneID = f1iter.HostedZoneId
			}
			f1 = append(f1, f1elem)
		}
		ko.Status.DNSEntries = f1
	} else {
		ko.Status.DNSEntries = nil
	}
	if resp.VpcEndpoint.Groups != nil {
		f2 := []*svcapitypes.SecurityGroupIdentifier{}
		for _, f2iter := range resp.VpcEndpoint.Groups {
			f2elem := &svcapitypes.SecurityGroupIdentifier{}
			if f2iter.GroupId != nil {
				f2elem.GroupID = f2iter.GroupId
			}
			if f2iter.GroupName != nil {
				f2elem.GroupName = f2iter.GroupName
			}
			f2 = append(f2, f2elem)
		}
		ko.Status.Groups = f2
	} else {
		ko.Status.Groups = nil
	}
	if resp.VpcEndpoint.LastError != nil {
		f3 := &svcapitypes.LastError{}
		if resp.VpcEndpoint.LastError.Code != nil {
			f3.Code = resp.VpcEndpoint.LastError.Code
		}
		if resp.VpcEndpoint.LastError.Message != nil {
			f3.Message = resp.VpcEndpoint.LastError.Message
		}
		ko.Status.LastError = f3
	} else {
		ko.Status.LastError = nil
	}
	if resp.VpcEndpoint.NetworkInterfaceIds != nil {
		f4 := []*string{}
		for _, f4iter := range resp.VpcEndpoint.NetworkInterfaceIds {
			var f4elem string
			f4elem = *f4iter
			f4 = append(f4, &f4elem)
		}
		ko.Status.NetworkInterfaceIDs = f4
	} else {
		ko.Status.NetworkInterfaceIDs = nil
	}
	if resp.VpcEndpoint.OwnerId != nil {
		ko.Status.OwnerID = resp.VpcEndpoint.OwnerId
	} else {
		ko.Status.OwnerID = nil
	}
	if resp.VpcEndpoint.PolicyDocument != nil {
		ko.Spec.PolicyDocument = resp.VpcEndpoint.PolicyDocument
	} else {
		ko.Spec.PolicyDocument = nil
	}
	if resp.VpcEndpoint.PrivateDnsEnabled != nil {
		ko.Spec.PrivateDNSEnabled = resp.VpcEndpoint.PrivateDnsEnabled
	} else {
		ko.Spec.PrivateDNSEnabled = nil
	}
	if resp.VpcEndpoint.RequesterManaged != nil {
		ko.Status.RequesterManaged = resp.VpcEndpoint.RequesterManaged
	} else {
		ko.Status.RequesterManaged = nil
	}
	if resp.VpcEndpoint.RouteTableIds != nil {
		f9 := []*string{}
		for _, f9iter := range resp.VpcEndpoint.RouteTableIds {
			var f9elem string
			f9elem = *f9iter
			f9 = append(f9, &f9elem)
		}
		ko.Spec.RouteTableIDs = f9
	} else {
		ko.Spec.RouteTableIDs = nil
	}
	if resp.VpcEndpoint.ServiceName != nil {
		ko.Spec.ServiceName = resp.VpcEndpoint.ServiceName
	} else {
		ko.Spec.ServiceName = nil
	}
	if resp.VpcEndpoint.State != nil {
		ko.Status.State = resp.VpcEndpoint.State
	} else {
		ko.Status.State = nil
	}
	if resp.VpcEndpoint.SubnetIds != nil {
		f12 := []*string{}
		for _, f12iter := range resp.VpcEndpoint.SubnetIds {
			var f12elem string
			f12elem = *f12iter
			f12 = append(f12, &f12elem)
		}
		ko.Spec.SubnetIDs = f12
	} else {
		ko.Spec.SubnetIDs = nil
	}
	if resp.VpcEndpoint.Tags != nil {
		f13 := []*svcapitypes.Tag{}
		for _, f13iter := range resp.VpcEndpoint.Tags {
			f13elem := &svcapitypes.Tag{}
			if f13iter.Key != nil {
				f13elem.Key = f13iter.Key
			}
			if f13iter.Value != nil {
				f13elem.Value = f13iter.Value
			}
			f13 = append(f13, f13elem)
		}
		ko.Status.Tags = f13
	} else {
		ko.Status.Tags = nil
	}
	if resp.VpcEndpoint.VpcEndpointId != nil {
		ko.Status.VPCEndpointID = resp.VpcEndpoint.VpcEndpointId
	} else {
		ko.Status.VPCEndpointID = nil
	}
	if resp.VpcEndpoint.VpcEndpointType != nil {
		ko.Spec.VPCEndpointType = resp.VpcEndpoint.VpcEndpointType
	} else {
		ko.Spec.VPCEndpointType = nil
	}
	if resp.VpcEndpoint.VpcId != nil {
		ko.Spec.VPCID = resp.VpcEndpoint.VpcId
	} else {
		ko.Spec.VPCID = nil
	}

	rm.setStatusDefaults(ko)
	return &resource{ko}, nil
}

// newCreateRequestPayload returns an SDK-specific struct for the HTTP request
// payload of the Create API call for the resource
func (rm *resourceManager) newCreateRequestPayload(
	ctx context.Context,
	r *resource,
) (*svcsdk.CreateVpcEndpointInput, error) {
	res := &svcsdk.CreateVpcEndpointInput{}

	if r.ko.Spec.ClientToken != nil {
		res.SetClientToken(*r.ko.Spec.ClientToken)
	}
	if r.ko.Spec.PolicyDocument != nil {
		res.SetPolicyDocument(*r.ko.Spec.PolicyDocument)
	}
	if r.ko.Spec.PrivateDNSEnabled != nil {
		res.SetPrivateDnsEnabled(*r.ko.Spec.PrivateDNSEnabled)
	}
	if r.ko.Spec.RouteTableIDs != nil {
		f3 := []*string{}
		for _, f3iter := range r.ko.Spec.RouteTableIDs {
			var f3elem string
			f3elem = *f3iter
			f3 = append(f3, &f3elem)
		}
		res.SetRouteTableIds(f3)
	}
	if r.ko.Spec.SecurityGroupIDs != nil {
		f4 := []*string{}
		for _, f4iter := range r.ko.Spec.SecurityGroupIDs {
			var f4elem string
			f4elem = *f4iter
			f4 = append(f4, &f4elem)
		}
		res.SetSecurityGroupIds(f4)
	}
	if r.ko.Spec.ServiceName != nil {
		res.SetServiceName(*r.ko.Spec.ServiceName)
	}
	if r.ko.Spec.SubnetIDs != nil {
		f6 := []*string{}
		for _, f6iter := range r.ko.Spec.SubnetIDs {
			var f6elem string
			f6elem = *f6iter
			f6 = append(f6, &f6elem)
		}
		res.SetSubnetIds(f6)
	}
	if r.ko.Spec.TagSpecifications != nil {
		f7 := []*svcsdk.TagSpecification{}
		for _, f7iter := range r.ko.Spec.TagSpecifications {
			f7elem := &svcsdk.TagSpecification{}
			if f7iter.ResourceType != nil {
				f7elem.SetResourceType(*f7iter.ResourceType)
			}
			if f7iter.Tags != nil {
				f7elemf1 := []*svcsdk.Tag{}
				for _, f7elemf1iter := range f7iter.Tags {
					f7elemf1elem := &svcsdk.Tag{}
					if f7elemf1iter.Key != nil {
						f7elemf1elem.SetKey(*f7elemf1iter.Key)
					}
					if f7elemf1iter.Value != nil {
						f7elemf1elem.SetValue(*f7elemf1iter.Value)
					}
					f7elemf1 = append(f7elemf1, f7elemf1elem)
				}
				f7elem.SetTags(f7elemf1)
			}
			f7 = append(f7, f7elem)
		}
		res.SetTagSpecifications(f7)
	}
	if r.ko.Spec.VPCEndpointType != nil {
		res.SetVpcEndpointType(*r.ko.Spec.VPCEndpointType)
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
	// TODO(jaypipes): Figure this out...
	return nil, ackerr.NotImplemented
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
	if err = addIDToDeleteRequest(r, input); err != nil {
		return nil, ackerr.NotFound
	}
	var resp *svcsdk.DeleteVpcEndpointsOutput
	_ = resp
	resp, err = rm.sdkapi.DeleteVpcEndpointsWithContext(ctx, input)
	rm.metrics.RecordAPICall("DELETE", "DeleteVpcEndpoints", err)
	return nil, err
}

// newDeleteRequestPayload returns an SDK-specific struct for the HTTP request
// payload of the Delete API call for the resource
func (rm *resourceManager) newDeleteRequestPayload(
	r *resource,
) (*svcsdk.DeleteVpcEndpointsInput, error) {
	res := &svcsdk.DeleteVpcEndpointsInput{}

	return res, nil
}

// setStatusDefaults sets default properties into supplied custom resource
func (rm *resourceManager) setStatusDefaults(
	ko *svcapitypes.VPCEndpoint,
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
	case "InvalidVpcId.Malformed",
		"InvalidVpcId.NotFound",
		"InvalidServiceName":
		return true
	default:
		return false
	}
}

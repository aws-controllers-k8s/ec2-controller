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

package elastic_ip_address

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
	_ = &svcapitypes.ElasticIPAddress{}
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
	if r.ko.Status.AllocationID == nil {
		return nil, ackerr.NotFound
	}
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
	if r.ko.Status.AllocationID != nil {
		input.SetAllocationIds([]*string{r.ko.Status.AllocationID})
	} else if r.ko.Status.PublicIP != nil {
		input.SetPublicIps([]*string{r.ko.Status.PublicIP})
	}
	var resp *svcsdk.DescribeAddressesOutput
	resp, err = rm.sdkapi.DescribeAddressesWithContext(ctx, input)
	rm.metrics.RecordAPICall("READ_MANY", "DescribeAddresses", err)
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
	for _, elem := range resp.Addresses {
		if elem.AllocationId != nil {
			if ko.Status.AllocationID != nil {
				if *elem.AllocationId != *ko.Status.AllocationID {
					continue
				}
			}
			ko.Status.AllocationID = elem.AllocationId
		} else {
			ko.Status.AllocationID = nil
		}
		if elem.CarrierIp != nil {
			ko.Status.CarrierIP = elem.CarrierIp
		} else {
			ko.Status.CarrierIP = nil
		}
		if elem.CustomerOwnedIp != nil {
			ko.Status.CustomerOwnedIP = elem.CustomerOwnedIp
		} else {
			ko.Status.CustomerOwnedIP = nil
		}
		if elem.CustomerOwnedIpv4Pool != nil {
			ko.Spec.CustomerOwnedIPv4Pool = elem.CustomerOwnedIpv4Pool
		} else {
			ko.Spec.CustomerOwnedIPv4Pool = nil
		}
		if elem.NetworkBorderGroup != nil {
			ko.Spec.NetworkBorderGroup = elem.NetworkBorderGroup
		} else {
			ko.Spec.NetworkBorderGroup = nil
		}
		if elem.PublicIp != nil {
			ko.Status.PublicIP = elem.PublicIp
		} else {
			ko.Status.PublicIP = nil
		}
		if elem.PublicIpv4Pool != nil {
			ko.Spec.PublicIPv4Pool = elem.PublicIpv4Pool
		} else {
			ko.Spec.PublicIPv4Pool = nil
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
			ko.Spec.Tags = f13
		} else {
			ko.Spec.Tags = nil
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
	return false
}

// newListRequestPayload returns SDK-specific struct for the HTTP request
// payload of the List API call for the resource
func (rm *resourceManager) newListRequestPayload(
	r *resource,
) (*svcsdk.DescribeAddressesInput, error) {
	res := &svcsdk.DescribeAddressesInput{}

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
	// EC2-VPC only supports setting Domain to "vpc"
	input.SetDomain(svcsdk.DomainTypeVpc)

	var resp *svcsdk.AllocateAddressOutput
	_ = resp
	resp, err = rm.sdkapi.AllocateAddressWithContext(ctx, input)
	rm.metrics.RecordAPICall("CREATE", "AllocateAddress", err)
	if err != nil {
		return nil, err
	}
	// Merge in the information we read from the API call above to the copy of
	// the original Kubernetes object we passed to the function
	ko := desired.ko.DeepCopy()

	if resp.AllocationId != nil {
		ko.Status.AllocationID = resp.AllocationId
	} else {
		ko.Status.AllocationID = nil
	}
	if resp.CarrierIp != nil {
		ko.Status.CarrierIP = resp.CarrierIp
	} else {
		ko.Status.CarrierIP = nil
	}
	if resp.CustomerOwnedIp != nil {
		ko.Status.CustomerOwnedIP = resp.CustomerOwnedIp
	} else {
		ko.Status.CustomerOwnedIP = nil
	}
	if resp.CustomerOwnedIpv4Pool != nil {
		ko.Spec.CustomerOwnedIPv4Pool = resp.CustomerOwnedIpv4Pool
	} else {
		ko.Spec.CustomerOwnedIPv4Pool = nil
	}
	if resp.NetworkBorderGroup != nil {
		ko.Spec.NetworkBorderGroup = resp.NetworkBorderGroup
	} else {
		ko.Spec.NetworkBorderGroup = nil
	}
	if resp.PublicIp != nil {
		ko.Status.PublicIP = resp.PublicIp
	} else {
		ko.Status.PublicIP = nil
	}
	if resp.PublicIpv4Pool != nil {
		ko.Spec.PublicIPv4Pool = resp.PublicIpv4Pool
	} else {
		ko.Spec.PublicIPv4Pool = nil
	}

	rm.setStatusDefaults(ko)
	return &resource{ko}, nil
}

// newCreateRequestPayload returns an SDK-specific struct for the HTTP request
// payload of the Create API call for the resource
func (rm *resourceManager) newCreateRequestPayload(
	ctx context.Context,
	r *resource,
) (*svcsdk.AllocateAddressInput, error) {
	res := &svcsdk.AllocateAddressInput{}

	if r.ko.Spec.Address != nil {
		res.SetAddress(*r.ko.Spec.Address)
	}
	if r.ko.Spec.CustomerOwnedIPv4Pool != nil {
		res.SetCustomerOwnedIpv4Pool(*r.ko.Spec.CustomerOwnedIPv4Pool)
	}
	if r.ko.Spec.NetworkBorderGroup != nil {
		res.SetNetworkBorderGroup(*r.ko.Spec.NetworkBorderGroup)
	}
	if r.ko.Spec.PublicIPv4Pool != nil {
		res.SetPublicIpv4Pool(*r.ko.Spec.PublicIPv4Pool)
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
	defer func() {
		exit(err)
	}()
	input, err := rm.newDeleteRequestPayload(r)
	if err != nil {
		return nil, err
	}
	// PublicIP and AllocationID are two ways of identifying the same resource
	// depending on whether they are included as part of EC2-Classic or EC2-VPC,
	// respectively. As EC2-Classic is retired, we should attempt to use the
	// AllocationID field whenever possible.
	if input.PublicIp != nil && input.AllocationId != nil {
		input.PublicIp = nil
	}
	var resp *svcsdk.ReleaseAddressOutput
	_ = resp
	resp, err = rm.sdkapi.ReleaseAddressWithContext(ctx, input)
	rm.metrics.RecordAPICall("DELETE", "ReleaseAddress", err)
	return nil, err
}

// newDeleteRequestPayload returns an SDK-specific struct for the HTTP request
// payload of the Delete API call for the resource
func (rm *resourceManager) newDeleteRequestPayload(
	r *resource,
) (*svcsdk.ReleaseAddressInput, error) {
	res := &svcsdk.ReleaseAddressInput{}

	if r.ko.Status.AllocationID != nil {
		res.SetAllocationId(*r.ko.Status.AllocationID)
	}
	if r.ko.Spec.NetworkBorderGroup != nil {
		res.SetNetworkBorderGroup(*r.ko.Spec.NetworkBorderGroup)
	}
	if r.ko.Status.PublicIP != nil {
		res.SetPublicIp(*r.ko.Status.PublicIP)
	}

	return res, nil
}

// setStatusDefaults sets default properties into supplied custom resource
func (rm *resourceManager) setStatusDefaults(
	ko *svcapitypes.ElasticIPAddress,
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
	case "IdempotentParameterMismatch",
		"InvalidAction",
		"InvalidCharacter",
		"InvalidClientTokenId",
		"InvalidPaginationToken",
		"InvalidParameter",
		"InvalidParameterCombination",
		"InvalidParameterValue",
		"InvalidQueryParameter",
		"MalformedQueryString",
		"MissingAction",
		"MissingAuthenticationToken",
		"MissingParameter",
		"UnknownParameter",
		"UnsupportedInstanceAttribute",
		"UnsupportedOperation",
		"UnsupportedProtocol",
		"ValidationError":
		return true
	default:
		return false
	}
}

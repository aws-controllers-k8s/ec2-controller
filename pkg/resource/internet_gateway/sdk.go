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

package internet_gateway

import (
	"context"
	"errors"
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
	_ = &svcapitypes.InternetGateway{}
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
	var resp *svcsdk.DescribeInternetGatewaysOutput
	resp, err = rm.sdkapi.DescribeInternetGatewaysWithContext(ctx, input)
	rm.metrics.RecordAPICall("READ_MANY", "DescribeInternetGateways", err)
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
	for _, elem := range resp.InternetGateways {
		if elem.Attachments != nil {
			f0 := []*svcapitypes.InternetGatewayAttachment{}
			for _, f0iter := range elem.Attachments {
				f0elem := &svcapitypes.InternetGatewayAttachment{}
				if f0iter.State != nil {
					f0elem.State = f0iter.State
				}
				if f0iter.VpcId != nil {
					f0elem.VPCID = f0iter.VpcId
				}
				f0 = append(f0, f0elem)
			}
			ko.Status.Attachments = f0
		} else {
			ko.Status.Attachments = nil
		}
		if elem.InternetGatewayId != nil {
			ko.Status.InternetGatewayID = elem.InternetGatewayId
		} else {
			ko.Status.InternetGatewayID = nil
		}
		if elem.OwnerId != nil {
			ko.Status.OwnerID = elem.OwnerId
		} else {
			ko.Status.OwnerID = nil
		}
		if elem.Tags != nil {
			f3 := []*svcapitypes.Tag{}
			for _, f3iter := range elem.Tags {
				f3elem := &svcapitypes.Tag{}
				if f3iter.Key != nil {
					f3elem.Key = f3iter.Key
				}
				if f3iter.Value != nil {
					f3elem.Value = f3iter.Value
				}
				f3 = append(f3, f3elem)
			}
			ko.Spec.Tags = f3
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
	vpcID, err := rm.getAttachedVPC(ctx, &resource{ko})
	if err != nil {
		return nil, err
	} else {
		ko.Spec.VPC = vpcID
	}
	return &resource{ko}, nil
}

// requiredFieldsMissingFromReadManyInput returns true if there are any fields
// for the ReadMany Input shape that are required but not present in the
// resource's Spec or Status
func (rm *resourceManager) requiredFieldsMissingFromReadManyInput(
	r *resource,
) bool {
	return r.ko.Status.InternetGatewayID == nil

}

// newListRequestPayload returns SDK-specific struct for the HTTP request
// payload of the List API call for the resource
func (rm *resourceManager) newListRequestPayload(
	r *resource,
) (*svcsdk.DescribeInternetGatewaysInput, error) {
	res := &svcsdk.DescribeInternetGatewaysInput{}

	if r.ko.Status.InternetGatewayID != nil {
		f2 := []*string{}
		f2 = append(f2, r.ko.Status.InternetGatewayID)
		res.SetInternetGatewayIds(f2)
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

	var resp *svcsdk.CreateInternetGatewayOutput
	_ = resp
	resp, err = rm.sdkapi.CreateInternetGatewayWithContext(ctx, input)
	rm.metrics.RecordAPICall("CREATE", "CreateInternetGateway", err)
	if err != nil {
		return nil, err
	}
	// Merge in the information we read from the API call above to the copy of
	// the original Kubernetes object we passed to the function
	ko := desired.ko.DeepCopy()

	if resp.InternetGateway.Attachments != nil {
		f0 := []*svcapitypes.InternetGatewayAttachment{}
		for _, f0iter := range resp.InternetGateway.Attachments {
			f0elem := &svcapitypes.InternetGatewayAttachment{}
			if f0iter.State != nil {
				f0elem.State = f0iter.State
			}
			if f0iter.VpcId != nil {
				f0elem.VPCID = f0iter.VpcId
			}
			f0 = append(f0, f0elem)
		}
		ko.Status.Attachments = f0
	} else {
		ko.Status.Attachments = nil
	}
	if resp.InternetGateway.InternetGatewayId != nil {
		ko.Status.InternetGatewayID = resp.InternetGateway.InternetGatewayId
	} else {
		ko.Status.InternetGatewayID = nil
	}
	if resp.InternetGateway.OwnerId != nil {
		ko.Status.OwnerID = resp.InternetGateway.OwnerId
	} else {
		ko.Status.OwnerID = nil
	}
	if resp.InternetGateway.Tags != nil {
		f3 := []*svcapitypes.Tag{}
		for _, f3iter := range resp.InternetGateway.Tags {
			f3elem := &svcapitypes.Tag{}
			if f3iter.Key != nil {
				f3elem.Key = f3iter.Key
			}
			if f3iter.Value != nil {
				f3elem.Value = f3iter.Value
			}
			f3 = append(f3, f3elem)
		}
		ko.Spec.Tags = f3
	} else {
		ko.Spec.Tags = nil
	}

	rm.setStatusDefaults(ko)
	if ko.Spec.VPC != nil {
		if err = rm.attachToVPC(ctx, &resource{ko}); err != nil {
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
) (*svcsdk.CreateInternetGatewayInput, error) {
	res := &svcsdk.CreateInternetGatewayInput{}

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
	return rm.customUpdateInternetGateway(ctx, desired, latest, delta)
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
	if r.ko.Spec.VPC != nil && r.ko.Status.InternetGatewayID != nil {
		if err = rm.detachFromVPC(ctx, *r.ko.Spec.VPC, *r.ko.Status.InternetGatewayID); err != nil {
			return nil, err
		}
	}
	input, err := rm.newDeleteRequestPayload(r)
	if err != nil {
		return nil, err
	}
	var resp *svcsdk.DeleteInternetGatewayOutput
	_ = resp
	resp, err = rm.sdkapi.DeleteInternetGatewayWithContext(ctx, input)
	rm.metrics.RecordAPICall("DELETE", "DeleteInternetGateway", err)
	return nil, err
}

// newDeleteRequestPayload returns an SDK-specific struct for the HTTP request
// payload of the Delete API call for the resource
func (rm *resourceManager) newDeleteRequestPayload(
	r *resource,
) (*svcsdk.DeleteInternetGatewayInput, error) {
	res := &svcsdk.DeleteInternetGatewayInput{}

	if r.ko.Status.InternetGatewayID != nil {
		res.SetInternetGatewayId(*r.ko.Status.InternetGatewayID)
	}

	return res, nil
}

// setStatusDefaults sets default properties into supplied custom resource
func (rm *resourceManager) setStatusDefaults(
	ko *svcapitypes.InternetGateway,
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
	case "InvalidVpcId.Malformed":
		return true
	default:
		return false
	}
}

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

package capacity_reservation

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
	_ = &svcapitypes.CapacityReservation{}
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
	var resp *svcsdk.DescribeCapacityReservationsOutput
	resp, err = rm.sdkapi.DescribeCapacityReservationsWithContext(ctx, input)
	rm.metrics.RecordAPICall("READ_MANY", "DescribeCapacityReservations", err)
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
	for _, elem := range resp.CapacityReservations {
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
		if elem.AvailableInstanceCount != nil {
			ko.Status.AvailableInstanceCount = elem.AvailableInstanceCount
		} else {
			ko.Status.AvailableInstanceCount = nil
		}
		if elem.CapacityReservationArn != nil {
			if ko.Status.ACKResourceMetadata == nil {
				ko.Status.ACKResourceMetadata = &ackv1alpha1.ResourceMetadata{}
			}
			tmpARN := ackv1alpha1.AWSResourceName(*elem.CapacityReservationArn)
			ko.Status.ACKResourceMetadata.ARN = &tmpARN
		}
		if elem.CapacityReservationFleetId != nil {
			ko.Status.CapacityReservationFleetID = elem.CapacityReservationFleetId
		} else {
			ko.Status.CapacityReservationFleetID = nil
		}
		if elem.CapacityReservationId != nil {
			ko.Status.CapacityReservationID = elem.CapacityReservationId
		} else {
			ko.Status.CapacityReservationID = nil
		}
		if elem.CreateDate != nil {
			ko.Status.CreateDate = &metav1.Time{*elem.CreateDate}
		} else {
			ko.Status.CreateDate = nil
		}
		if elem.EbsOptimized != nil {
			ko.Spec.EBSOptimized = elem.EbsOptimized
		} else {
			ko.Spec.EBSOptimized = nil
		}
		if elem.EndDate != nil {
			ko.Spec.EndDate = &metav1.Time{*elem.EndDate}
		} else {
			ko.Spec.EndDate = nil
		}
		if elem.EndDateType != nil {
			ko.Spec.EndDateType = elem.EndDateType
		} else {
			ko.Spec.EndDateType = nil
		}
		if elem.EphemeralStorage != nil {
			ko.Spec.EphemeralStorage = elem.EphemeralStorage
		} else {
			ko.Spec.EphemeralStorage = nil
		}
		if elem.InstanceMatchCriteria != nil {
			ko.Spec.InstanceMatchCriteria = elem.InstanceMatchCriteria
		} else {
			ko.Spec.InstanceMatchCriteria = nil
		}
		if elem.InstancePlatform != nil {
			ko.Spec.InstancePlatform = elem.InstancePlatform
		} else {
			ko.Spec.InstancePlatform = nil
		}
		if elem.InstanceType != nil {
			ko.Spec.InstanceType = elem.InstanceType
		} else {
			ko.Spec.InstanceType = nil
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
		if elem.PlacementGroupArn != nil {
			ko.Spec.PlacementGroupARN = elem.PlacementGroupArn
		} else {
			ko.Spec.PlacementGroupARN = nil
		}
		if elem.StartDate != nil {
			ko.Status.StartDate = &metav1.Time{*elem.StartDate}
		} else {
			ko.Status.StartDate = nil
		}
		if elem.State != nil {
			ko.Status.State = elem.State
		} else {
			ko.Status.State = nil
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
		if elem.Tenancy != nil {
			ko.Spec.Tenancy = elem.Tenancy
		} else {
			ko.Spec.Tenancy = nil
		}
		if elem.TotalInstanceCount != nil {
			ko.Status.TotalInstanceCount = elem.TotalInstanceCount
		} else {
			ko.Status.TotalInstanceCount = nil
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
	return r.ko.Status.CapacityReservationID == nil

}

// newListRequestPayload returns SDK-specific struct for the HTTP request
// payload of the List API call for the resource
func (rm *resourceManager) newListRequestPayload(
	r *resource,
) (*svcsdk.DescribeCapacityReservationsInput, error) {
	res := &svcsdk.DescribeCapacityReservationsInput{}

	if r.ko.Status.CapacityReservationID != nil {
		f0 := []*string{}
		f0 = append(f0, r.ko.Status.CapacityReservationID)
		res.SetCapacityReservationIds(f0)
	}
	if r.ko.Spec.DryRun != nil {
		res.SetDryRun(*r.ko.Spec.DryRun)
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

	var resp *svcsdk.CreateCapacityReservationOutput
	_ = resp
	resp, err = rm.sdkapi.CreateCapacityReservationWithContext(ctx, input)
	rm.metrics.RecordAPICall("CREATE", "CreateCapacityReservation", err)
	if err != nil {
		return nil, err
	}
	// Merge in the information we read from the API call above to the copy of
	// the original Kubernetes object we passed to the function
	ko := desired.ko.DeepCopy()

	if resp.CapacityReservation.AvailabilityZone != nil {
		ko.Spec.AvailabilityZone = resp.CapacityReservation.AvailabilityZone
	} else {
		ko.Spec.AvailabilityZone = nil
	}
	if resp.CapacityReservation.AvailabilityZoneId != nil {
		ko.Spec.AvailabilityZoneID = resp.CapacityReservation.AvailabilityZoneId
	} else {
		ko.Spec.AvailabilityZoneID = nil
	}
	if resp.CapacityReservation.AvailableInstanceCount != nil {
		ko.Status.AvailableInstanceCount = resp.CapacityReservation.AvailableInstanceCount
	} else {
		ko.Status.AvailableInstanceCount = nil
	}
	if ko.Status.ACKResourceMetadata == nil {
		ko.Status.ACKResourceMetadata = &ackv1alpha1.ResourceMetadata{}
	}
	if resp.CapacityReservation.CapacityReservationArn != nil {
		arn := ackv1alpha1.AWSResourceName(*resp.CapacityReservation.CapacityReservationArn)
		ko.Status.ACKResourceMetadata.ARN = &arn
	}
	if resp.CapacityReservation.CapacityReservationFleetId != nil {
		ko.Status.CapacityReservationFleetID = resp.CapacityReservation.CapacityReservationFleetId
	} else {
		ko.Status.CapacityReservationFleetID = nil
	}
	if resp.CapacityReservation.CapacityReservationId != nil {
		ko.Status.CapacityReservationID = resp.CapacityReservation.CapacityReservationId
	} else {
		ko.Status.CapacityReservationID = nil
	}
	if resp.CapacityReservation.CreateDate != nil {
		ko.Status.CreateDate = &metav1.Time{*resp.CapacityReservation.CreateDate}
	} else {
		ko.Status.CreateDate = nil
	}
	if resp.CapacityReservation.EbsOptimized != nil {
		ko.Spec.EBSOptimized = resp.CapacityReservation.EbsOptimized
	} else {
		ko.Spec.EBSOptimized = nil
	}
	if resp.CapacityReservation.EndDate != nil {
		ko.Spec.EndDate = &metav1.Time{*resp.CapacityReservation.EndDate}
	} else {
		ko.Spec.EndDate = nil
	}
	if resp.CapacityReservation.EndDateType != nil {
		ko.Spec.EndDateType = resp.CapacityReservation.EndDateType
	} else {
		ko.Spec.EndDateType = nil
	}
	if resp.CapacityReservation.EphemeralStorage != nil {
		ko.Spec.EphemeralStorage = resp.CapacityReservation.EphemeralStorage
	} else {
		ko.Spec.EphemeralStorage = nil
	}
	if resp.CapacityReservation.InstanceMatchCriteria != nil {
		ko.Spec.InstanceMatchCriteria = resp.CapacityReservation.InstanceMatchCriteria
	} else {
		ko.Spec.InstanceMatchCriteria = nil
	}
	if resp.CapacityReservation.InstancePlatform != nil {
		ko.Spec.InstancePlatform = resp.CapacityReservation.InstancePlatform
	} else {
		ko.Spec.InstancePlatform = nil
	}
	if resp.CapacityReservation.InstanceType != nil {
		ko.Spec.InstanceType = resp.CapacityReservation.InstanceType
	} else {
		ko.Spec.InstanceType = nil
	}
	if resp.CapacityReservation.OutpostArn != nil {
		ko.Spec.OutpostARN = resp.CapacityReservation.OutpostArn
	} else {
		ko.Spec.OutpostARN = nil
	}
	if resp.CapacityReservation.OwnerId != nil {
		ko.Status.OwnerID = resp.CapacityReservation.OwnerId
	} else {
		ko.Status.OwnerID = nil
	}
	if resp.CapacityReservation.PlacementGroupArn != nil {
		ko.Spec.PlacementGroupARN = resp.CapacityReservation.PlacementGroupArn
	} else {
		ko.Spec.PlacementGroupARN = nil
	}
	if resp.CapacityReservation.StartDate != nil {
		ko.Status.StartDate = &metav1.Time{*resp.CapacityReservation.StartDate}
	} else {
		ko.Status.StartDate = nil
	}
	if resp.CapacityReservation.State != nil {
		ko.Status.State = resp.CapacityReservation.State
	} else {
		ko.Status.State = nil
	}
	if resp.CapacityReservation.Tags != nil {
		f19 := []*svcapitypes.Tag{}
		for _, f19iter := range resp.CapacityReservation.Tags {
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
	if resp.CapacityReservation.Tenancy != nil {
		ko.Spec.Tenancy = resp.CapacityReservation.Tenancy
	} else {
		ko.Spec.Tenancy = nil
	}
	if resp.CapacityReservation.TotalInstanceCount != nil {
		ko.Status.TotalInstanceCount = resp.CapacityReservation.TotalInstanceCount
	} else {
		ko.Status.TotalInstanceCount = nil
	}

	rm.setStatusDefaults(ko)
	return &resource{ko}, nil
}

// newCreateRequestPayload returns an SDK-specific struct for the HTTP request
// payload of the Create API call for the resource
func (rm *resourceManager) newCreateRequestPayload(
	ctx context.Context,
	r *resource,
) (*svcsdk.CreateCapacityReservationInput, error) {
	res := &svcsdk.CreateCapacityReservationInput{}

	if r.ko.Spec.AvailabilityZone != nil {
		res.SetAvailabilityZone(*r.ko.Spec.AvailabilityZone)
	}
	if r.ko.Spec.AvailabilityZoneID != nil {
		res.SetAvailabilityZoneId(*r.ko.Spec.AvailabilityZoneID)
	}
	if r.ko.Spec.ClientToken != nil {
		res.SetClientToken(*r.ko.Spec.ClientToken)
	}
	if r.ko.Spec.DryRun != nil {
		res.SetDryRun(*r.ko.Spec.DryRun)
	}
	if r.ko.Spec.EBSOptimized != nil {
		res.SetEbsOptimized(*r.ko.Spec.EBSOptimized)
	}
	if r.ko.Spec.EndDate != nil {
		res.SetEndDate(r.ko.Spec.EndDate.Time)
	}
	if r.ko.Spec.EndDateType != nil {
		res.SetEndDateType(*r.ko.Spec.EndDateType)
	}
	if r.ko.Spec.EphemeralStorage != nil {
		res.SetEphemeralStorage(*r.ko.Spec.EphemeralStorage)
	}
	if r.ko.Spec.InstanceCount != nil {
		res.SetInstanceCount(*r.ko.Spec.InstanceCount)
	}
	if r.ko.Spec.InstanceMatchCriteria != nil {
		res.SetInstanceMatchCriteria(*r.ko.Spec.InstanceMatchCriteria)
	}
	if r.ko.Spec.InstancePlatform != nil {
		res.SetInstancePlatform(*r.ko.Spec.InstancePlatform)
	}
	if r.ko.Spec.InstanceType != nil {
		res.SetInstanceType(*r.ko.Spec.InstanceType)
	}
	if r.ko.Spec.OutpostARN != nil {
		res.SetOutpostArn(*r.ko.Spec.OutpostARN)
	}
	if r.ko.Spec.PlacementGroupARN != nil {
		res.SetPlacementGroupArn(*r.ko.Spec.PlacementGroupARN)
	}
	if r.ko.Spec.Tenancy != nil {
		res.SetTenancy(*r.ko.Spec.Tenancy)
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
	return rm.customUpdateCapacityReservation(ctx, desired, latest, delta)
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
	// TODO(jaypipes): Figure this out...
	return nil, nil

}

// setStatusDefaults sets default properties into supplied custom resource
func (rm *resourceManager) setStatusDefaults(
	ko *svcapitypes.CapacityReservation,
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
	// No terminal_errors specified for this resource in generator config
	return false
}

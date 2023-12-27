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

package vpc_endpoint_service_configuration

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
	_ = &svcapitypes.VPCEndpointServiceConfiguration{}
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
	var resp *svcsdk.DescribeVpcEndpointServiceConfigurationsOutput
	resp, err = rm.sdkapi.DescribeVpcEndpointServiceConfigurationsWithContext(ctx, input)
	rm.metrics.RecordAPICall("READ_MANY", "DescribeVpcEndpointServiceConfigurations", err)
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
	for _, elem := range resp.ServiceConfigurations {
		if elem.AcceptanceRequired != nil {
			ko.Spec.AcceptanceRequired = elem.AcceptanceRequired
		} else {
			ko.Spec.AcceptanceRequired = nil
		}
		if elem.AvailabilityZones != nil {
			f1 := []*string{}
			for _, f1iter := range elem.AvailabilityZones {
				var f1elem string
				f1elem = *f1iter
				f1 = append(f1, &f1elem)
			}
			ko.Status.AvailabilityZones = f1
		} else {
			ko.Status.AvailabilityZones = nil
		}
		if elem.BaseEndpointDnsNames != nil {
			f2 := []*string{}
			for _, f2iter := range elem.BaseEndpointDnsNames {
				var f2elem string
				f2elem = *f2iter
				f2 = append(f2, &f2elem)
			}
			ko.Status.BaseEndpointDNSNames = f2
		} else {
			ko.Status.BaseEndpointDNSNames = nil
		}
		if elem.GatewayLoadBalancerArns != nil {
			f3 := []*string{}
			for _, f3iter := range elem.GatewayLoadBalancerArns {
				var f3elem string
				f3elem = *f3iter
				f3 = append(f3, &f3elem)
			}
			ko.Spec.GatewayLoadBalancerARNs = f3
		} else {
			ko.Spec.GatewayLoadBalancerARNs = nil
		}
		if elem.ManagesVpcEndpoints != nil {
			ko.Status.ManagesVPCEndpoints = elem.ManagesVpcEndpoints
		} else {
			ko.Status.ManagesVPCEndpoints = nil
		}
		if elem.NetworkLoadBalancerArns != nil {
			f5 := []*string{}
			for _, f5iter := range elem.NetworkLoadBalancerArns {
				var f5elem string
				f5elem = *f5iter
				f5 = append(f5, &f5elem)
			}
			ko.Spec.NetworkLoadBalancerARNs = f5
		} else {
			ko.Spec.NetworkLoadBalancerARNs = nil
		}
		if elem.PayerResponsibility != nil {
			ko.Status.PayerResponsibility = elem.PayerResponsibility
		} else {
			ko.Status.PayerResponsibility = nil
		}
		if elem.PrivateDnsName != nil {
			ko.Spec.PrivateDNSName = elem.PrivateDnsName
		} else {
			ko.Spec.PrivateDNSName = nil
		}
		if elem.PrivateDnsNameConfiguration != nil {
			f8 := &svcapitypes.PrivateDNSNameConfiguration{}
			if elem.PrivateDnsNameConfiguration.Name != nil {
				f8.Name = elem.PrivateDnsNameConfiguration.Name
			}
			if elem.PrivateDnsNameConfiguration.State != nil {
				f8.State = elem.PrivateDnsNameConfiguration.State
			}
			if elem.PrivateDnsNameConfiguration.Type != nil {
				f8.Type = elem.PrivateDnsNameConfiguration.Type
			}
			if elem.PrivateDnsNameConfiguration.Value != nil {
				f8.Value = elem.PrivateDnsNameConfiguration.Value
			}
			ko.Status.PrivateDNSNameConfiguration = f8
		} else {
			ko.Status.PrivateDNSNameConfiguration = nil
		}
		if elem.ServiceId != nil {
			ko.Status.ServiceID = elem.ServiceId
		} else {
			ko.Status.ServiceID = nil
		}
		if elem.ServiceName != nil {
			ko.Status.ServiceName = elem.ServiceName
		} else {
			ko.Status.ServiceName = nil
		}
		if elem.ServiceState != nil {
			ko.Status.ServiceState = elem.ServiceState
		} else {
			ko.Status.ServiceState = nil
		}
		if elem.ServiceType != nil {
			f12 := []*svcapitypes.ServiceTypeDetail{}
			for _, f12iter := range elem.ServiceType {
				f12elem := &svcapitypes.ServiceTypeDetail{}
				if f12iter.ServiceType != nil {
					f12elem.ServiceType = f12iter.ServiceType
				}
				f12 = append(f12, f12elem)
			}
			ko.Status.ServiceType = f12
		} else {
			ko.Status.ServiceType = nil
		}
		if elem.SupportedIpAddressTypes != nil {
			f13 := []*string{}
			for _, f13iter := range elem.SupportedIpAddressTypes {
				var f13elem string
				f13elem = *f13iter
				f13 = append(f13, &f13elem)
			}
			ko.Spec.SupportedIPAddressTypes = f13
		} else {
			ko.Spec.SupportedIPAddressTypes = nil
		}
		if elem.Tags != nil {
			f14 := []*svcapitypes.Tag{}
			for _, f14iter := range elem.Tags {
				f14elem := &svcapitypes.Tag{}
				if f14iter.Key != nil {
					f14elem.Key = f14iter.Key
				}
				if f14iter.Value != nil {
					f14elem.Value = f14iter.Value
				}
				f14 = append(f14, f14elem)
			}
			ko.Spec.Tags = f14
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
) (*svcsdk.DescribeVpcEndpointServiceConfigurationsInput, error) {
	res := &svcsdk.DescribeVpcEndpointServiceConfigurationsInput{}

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

	var resp *svcsdk.CreateVpcEndpointServiceConfigurationOutput
	_ = resp
	resp, err = rm.sdkapi.CreateVpcEndpointServiceConfigurationWithContext(ctx, input)
	rm.metrics.RecordAPICall("CREATE", "CreateVpcEndpointServiceConfiguration", err)
	if err != nil {
		return nil, err
	}
	// Merge in the information we read from the API call above to the copy of
	// the original Kubernetes object we passed to the function
	ko := desired.ko.DeepCopy()

	if resp.ServiceConfiguration.AcceptanceRequired != nil {
		ko.Spec.AcceptanceRequired = resp.ServiceConfiguration.AcceptanceRequired
	} else {
		ko.Spec.AcceptanceRequired = nil
	}
	if resp.ServiceConfiguration.AvailabilityZones != nil {
		f1 := []*string{}
		for _, f1iter := range resp.ServiceConfiguration.AvailabilityZones {
			var f1elem string
			f1elem = *f1iter
			f1 = append(f1, &f1elem)
		}
		ko.Status.AvailabilityZones = f1
	} else {
		ko.Status.AvailabilityZones = nil
	}
	if resp.ServiceConfiguration.BaseEndpointDnsNames != nil {
		f2 := []*string{}
		for _, f2iter := range resp.ServiceConfiguration.BaseEndpointDnsNames {
			var f2elem string
			f2elem = *f2iter
			f2 = append(f2, &f2elem)
		}
		ko.Status.BaseEndpointDNSNames = f2
	} else {
		ko.Status.BaseEndpointDNSNames = nil
	}
	if resp.ServiceConfiguration.GatewayLoadBalancerArns != nil {
		f3 := []*string{}
		for _, f3iter := range resp.ServiceConfiguration.GatewayLoadBalancerArns {
			var f3elem string
			f3elem = *f3iter
			f3 = append(f3, &f3elem)
		}
		ko.Spec.GatewayLoadBalancerARNs = f3
	} else {
		ko.Spec.GatewayLoadBalancerARNs = nil
	}
	if resp.ServiceConfiguration.ManagesVpcEndpoints != nil {
		ko.Status.ManagesVPCEndpoints = resp.ServiceConfiguration.ManagesVpcEndpoints
	} else {
		ko.Status.ManagesVPCEndpoints = nil
	}
	if resp.ServiceConfiguration.NetworkLoadBalancerArns != nil {
		f5 := []*string{}
		for _, f5iter := range resp.ServiceConfiguration.NetworkLoadBalancerArns {
			var f5elem string
			f5elem = *f5iter
			f5 = append(f5, &f5elem)
		}
		ko.Spec.NetworkLoadBalancerARNs = f5
	} else {
		ko.Spec.NetworkLoadBalancerARNs = nil
	}
	if resp.ServiceConfiguration.PayerResponsibility != nil {
		ko.Status.PayerResponsibility = resp.ServiceConfiguration.PayerResponsibility
	} else {
		ko.Status.PayerResponsibility = nil
	}
	if resp.ServiceConfiguration.PrivateDnsName != nil {
		ko.Spec.PrivateDNSName = resp.ServiceConfiguration.PrivateDnsName
	} else {
		ko.Spec.PrivateDNSName = nil
	}
	if resp.ServiceConfiguration.PrivateDnsNameConfiguration != nil {
		f8 := &svcapitypes.PrivateDNSNameConfiguration{}
		if resp.ServiceConfiguration.PrivateDnsNameConfiguration.Name != nil {
			f8.Name = resp.ServiceConfiguration.PrivateDnsNameConfiguration.Name
		}
		if resp.ServiceConfiguration.PrivateDnsNameConfiguration.State != nil {
			f8.State = resp.ServiceConfiguration.PrivateDnsNameConfiguration.State
		}
		if resp.ServiceConfiguration.PrivateDnsNameConfiguration.Type != nil {
			f8.Type = resp.ServiceConfiguration.PrivateDnsNameConfiguration.Type
		}
		if resp.ServiceConfiguration.PrivateDnsNameConfiguration.Value != nil {
			f8.Value = resp.ServiceConfiguration.PrivateDnsNameConfiguration.Value
		}
		ko.Status.PrivateDNSNameConfiguration = f8
	} else {
		ko.Status.PrivateDNSNameConfiguration = nil
	}
	if resp.ServiceConfiguration.ServiceId != nil {
		ko.Status.ServiceID = resp.ServiceConfiguration.ServiceId
	} else {
		ko.Status.ServiceID = nil
	}
	if resp.ServiceConfiguration.ServiceName != nil {
		ko.Status.ServiceName = resp.ServiceConfiguration.ServiceName
	} else {
		ko.Status.ServiceName = nil
	}
	if resp.ServiceConfiguration.ServiceState != nil {
		ko.Status.ServiceState = resp.ServiceConfiguration.ServiceState
	} else {
		ko.Status.ServiceState = nil
	}
	if resp.ServiceConfiguration.ServiceType != nil {
		f12 := []*svcapitypes.ServiceTypeDetail{}
		for _, f12iter := range resp.ServiceConfiguration.ServiceType {
			f12elem := &svcapitypes.ServiceTypeDetail{}
			if f12iter.ServiceType != nil {
				f12elem.ServiceType = f12iter.ServiceType
			}
			f12 = append(f12, f12elem)
		}
		ko.Status.ServiceType = f12
	} else {
		ko.Status.ServiceType = nil
	}
	if resp.ServiceConfiguration.SupportedIpAddressTypes != nil {
		f13 := []*string{}
		for _, f13iter := range resp.ServiceConfiguration.SupportedIpAddressTypes {
			var f13elem string
			f13elem = *f13iter
			f13 = append(f13, &f13elem)
		}
		ko.Spec.SupportedIPAddressTypes = f13
	} else {
		ko.Spec.SupportedIPAddressTypes = nil
	}
	if resp.ServiceConfiguration.Tags != nil {
		f14 := []*svcapitypes.Tag{}
		for _, f14iter := range resp.ServiceConfiguration.Tags {
			f14elem := &svcapitypes.Tag{}
			if f14iter.Key != nil {
				f14elem.Key = f14iter.Key
			}
			if f14iter.Value != nil {
				f14elem.Value = f14iter.Value
			}
			f14 = append(f14, f14elem)
		}
		ko.Spec.Tags = f14
	} else {
		ko.Spec.Tags = nil
	}

	rm.setStatusDefaults(ko)
	return &resource{ko}, nil
}

// newCreateRequestPayload returns an SDK-specific struct for the HTTP request
// payload of the Create API call for the resource
func (rm *resourceManager) newCreateRequestPayload(
	ctx context.Context,
	r *resource,
) (*svcsdk.CreateVpcEndpointServiceConfigurationInput, error) {
	res := &svcsdk.CreateVpcEndpointServiceConfigurationInput{}

	if r.ko.Spec.AcceptanceRequired != nil {
		res.SetAcceptanceRequired(*r.ko.Spec.AcceptanceRequired)
	}
	if r.ko.Spec.GatewayLoadBalancerARNs != nil {
		f1 := []*string{}
		for _, f1iter := range r.ko.Spec.GatewayLoadBalancerARNs {
			var f1elem string
			f1elem = *f1iter
			f1 = append(f1, &f1elem)
		}
		res.SetGatewayLoadBalancerArns(f1)
	}
	if r.ko.Spec.NetworkLoadBalancerARNs != nil {
		f2 := []*string{}
		for _, f2iter := range r.ko.Spec.NetworkLoadBalancerARNs {
			var f2elem string
			f2elem = *f2iter
			f2 = append(f2, &f2elem)
		}
		res.SetNetworkLoadBalancerArns(f2)
	}
	if r.ko.Spec.PrivateDNSName != nil {
		res.SetPrivateDnsName(*r.ko.Spec.PrivateDNSName)
	}
	if r.ko.Spec.SupportedIPAddressTypes != nil {
		f4 := []*string{}
		for _, f4iter := range r.ko.Spec.SupportedIPAddressTypes {
			var f4elem string
			f4elem = *f4iter
			f4 = append(f4, &f4elem)
		}
		res.SetSupportedIpAddressTypes(f4)
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
) (updated *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.sdkUpdate")
	defer func() {
		exit(err)
	}()

	// Only continue if the VPC Endpoint Service is in 'Available' state
	if *latest.ko.Status.ServiceState != "Available" {
		return desired, requeueWaitNotAvailable
	}

	if delta.DifferentAt("Spec.Tags") {
		if err := rm.syncTags(ctx, desired, latest); err != nil {
			// This causes a requeue and the rest of the fields will be synced on the next reconciliation loop
			ackcondition.SetSynced(desired, corev1.ConditionFalse, nil, nil)
			return desired, err
		}
	}

	if delta.DifferentAt("Spec.AllowedPrincipals") {
		var listOfPrincipalsToAdd []*string
		for _, desiredPrincipal := range desired.ko.Spec.AllowedPrincipals {
			for _, latestPrincipal := range latest.ko.Spec.AllowedPrincipals {
				if *desiredPrincipal == *latestPrincipal {
					// Principal already in Allow List, skip
					continue
				}
				// Principal is not in the Allow List, add it to the list of those to add
				listOfPrincipalsToAdd = append(listOfPrincipalsToAdd, desiredPrincipal)
			}
		}
		// Make the AWS API call to add the principals
		if len(listOfPrincipalsToAdd) > 0 {
			modifyPermissionsInput := &svcsdk.ModifyVpcEndpointServicePermissionsInput{
				ServiceId:            latest.ko.Status.ServiceID,
				AddAllowedPrincipals: listOfPrincipalsToAdd,
			}
			_, err := rm.sdkapi.ModifyVpcEndpointServicePermissions(modifyPermissionsInput)
			rm.metrics.RecordAPICall("UPDATE", "ModifyVpcEndpointServicePermissions", err)
			if err != nil {
				return nil, err
			}
		}

		// Remove any principal that is not on the allowed list anymore
		var listOfPrincipalsToRemove []*string
		for _, latestPrincipal := range latest.ko.Spec.AllowedPrincipals {
			for _, desiredPrincipal := range desired.ko.Spec.AllowedPrincipals {
				if *desiredPrincipal == *latestPrincipal {
					// Principal still in Allow List, skip
					continue
				}
				// Principal is not in the Allow List, add it to the list of those to remove
				listOfPrincipalsToRemove = append(listOfPrincipalsToRemove, latestPrincipal)
			}
		}
		// Make the AWS API call to remove the principals
		if len(listOfPrincipalsToRemove) > 0 {
			modifyPermissionsInput := &svcsdk.ModifyVpcEndpointServicePermissionsInput{
				ServiceId:               latest.ko.Status.ServiceID,
				RemoveAllowedPrincipals: listOfPrincipalsToRemove,
			}
			_, err := rm.sdkapi.ModifyVpcEndpointServicePermissions(modifyPermissionsInput)
			rm.metrics.RecordAPICall("UPDATE", "ModifyVpcEndpointServicePermissions", err)
			if err != nil {
				return nil, err
			}
		}
	}

	// Only continue if something other than Tags or certain fields has changed in the Spec
	if !delta.DifferentExcept("Spec.Tags", "Spec.AllowedPrincipals") {
		return desired, nil
	}

	input, err := rm.newUpdateRequestPayload(ctx, desired, delta)
	if err != nil {
		return nil, err
	}

	var resp *svcsdk.ModifyVpcEndpointServiceConfigurationOutput
	_ = resp
	resp, err = rm.sdkapi.ModifyVpcEndpointServiceConfigurationWithContext(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "ModifyVpcEndpointServiceConfiguration", err)
	if err != nil {
		return nil, err
	}
	// Merge in the information we read from the API call above to the copy of
	// the original Kubernetes object we passed to the function
	ko := desired.ko.DeepCopy()

	rm.setStatusDefaults(ko)
	return &resource{ko}, nil
}

// newUpdateRequestPayload returns an SDK-specific struct for the HTTP request
// payload of the Update API call for the resource
func (rm *resourceManager) newUpdateRequestPayload(
	ctx context.Context,
	r *resource,
	delta *ackcompare.Delta,
) (*svcsdk.ModifyVpcEndpointServiceConfigurationInput, error) {
	res := &svcsdk.ModifyVpcEndpointServiceConfigurationInput{}

	if r.ko.Spec.AcceptanceRequired != nil {
		res.SetAcceptanceRequired(*r.ko.Spec.AcceptanceRequired)
	}
	if r.ko.Spec.PrivateDNSName != nil {
		res.SetPrivateDnsName(*r.ko.Spec.PrivateDNSName)
	}
	if r.ko.Status.ServiceID != nil {
		res.SetServiceId(*r.ko.Status.ServiceID)
	}

	return res, nil
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
	if err = addIDToDeleteRequest(r, input); err != nil {
		return nil, ackerr.NotFound
	}
	var resp *svcsdk.DeleteVpcEndpointServiceConfigurationsOutput
	_ = resp
	resp, err = rm.sdkapi.DeleteVpcEndpointServiceConfigurationsWithContext(ctx, input)
	rm.metrics.RecordAPICall("DELETE", "DeleteVpcEndpointServiceConfigurations", err)
	return nil, err
}

// newDeleteRequestPayload returns an SDK-specific struct for the HTTP request
// payload of the Delete API call for the resource
func (rm *resourceManager) newDeleteRequestPayload(
	r *resource,
) (*svcsdk.DeleteVpcEndpointServiceConfigurationsInput, error) {
	res := &svcsdk.DeleteVpcEndpointServiceConfigurationsInput{}

	return res, nil
}

// setStatusDefaults sets default properties into supplied custom resource
func (rm *resourceManager) setStatusDefaults(
	ko *svcapitypes.VPCEndpointServiceConfiguration,
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

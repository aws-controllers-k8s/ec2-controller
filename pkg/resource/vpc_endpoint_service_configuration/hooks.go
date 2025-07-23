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

package vpc_endpoint_service_configuration

import (
	"context"
	"errors"
	"fmt"
	"time"

	ackerr "github.com/aws-controllers-k8s/runtime/pkg/errors"
	ackrequeue "github.com/aws-controllers-k8s/runtime/pkg/requeue"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	ackutil "github.com/aws-controllers-k8s/runtime/pkg/util"
	svcsdk "github.com/aws/aws-sdk-go-v2/service/ec2"

	svcapitypes "github.com/aws-controllers-k8s/ec2-controller/apis/v1alpha1"
	"github.com/aws-controllers-k8s/ec2-controller/pkg/tags"
)

var (
	requeueWaitNotAvailable = ackrequeue.NeededAfter(
		fmt.Errorf("VPCEndpointService is not in %v state yet, requeuing", "Available"),
		5*time.Second,
	)
)

// addIDToDeleteRequest adds resource's Vpc Endpoint Service Configuration ID to DeleteRequest.
// Return error to indicate to callers that the resource is not yet created.
func addIDToDeleteRequest(r *resource,
	input *svcsdk.DeleteVpcEndpointServiceConfigurationsInput) error {
	if r.ko.Status.ServiceID == nil {
		return errors.New("unable to extract ServiceID from resource")
	}
	input.ServiceIds = []string{*r.ko.Status.ServiceID}
	return nil
}

// syncAllowedPrincipals adds & removes allowed principals with the 'ModifyVpcEndpointServicePermissions' API call
func (rm *resourceManager) syncAllowedPrincipals(
	ctx context.Context,
	desired *resource,
	latest *resource,
) (updated *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("syncAllowedPrincipals")
	defer func(err error) {
		exit(err)
	}(err)

	toAdd := []string{}
	toDelete := []string{}

	currentlyAllowedPrincipals := latest.ko.Spec.AllowedPrincipals
	desiredAllowedPrincipals := desired.ko.Spec.AllowedPrincipals

	// Check if any desired allowed principals need to be added
	for _, p := range desiredAllowedPrincipals {
		if !ackutil.InStringPs(*p, currentlyAllowedPrincipals) {
			toAdd = append(toAdd, *p)
		}
	}

	// Check if any currently allowed principals need to be deleted
	for _, p := range currentlyAllowedPrincipals {
		if !ackutil.InStringPs(*p, desiredAllowedPrincipals) {
			toDelete = append(toDelete, *p)
		}
	}

	// Modify the allowed principals
	rlog.Debug("Syncing Allowed Principals", "toAdd", toAdd, "toDelete", toDelete)
	if err = rm.modifyAllowedPrincipals(ctx, latest, toAdd, toDelete); err != nil {
		return desired, err
	}

	return desired, nil
}

// Makes the AWS API call 'ModifyVpcEndpointServicePermissions' to add and/or remove the allowed principals
func (rm *resourceManager) modifyAllowedPrincipals(
	ctx context.Context,
	latest *resource,
	toAdd []string,
	toDelete []string,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("modifyAllowedPrincipals")
	defer func(err error) {
		exit(err)
	}(err)
	modifyPermissionsInput := &svcsdk.ModifyVpcEndpointServicePermissionsInput{
		ServiceId: latest.ko.Status.ServiceID,
	}

	if len(toAdd) > 0 {
		modifyPermissionsInput.AddAllowedPrincipals = toAdd
	}

	if len(toDelete) > 0 {
		modifyPermissionsInput.RemoveAllowedPrincipals = toDelete
	}

	_, err = rm.sdkapi.ModifyVpcEndpointServicePermissions(ctx, modifyPermissionsInput)
	rm.metrics.RecordAPICall("UPDATE", "ModifyVpcEndpointServicePermissions", err)
	if err != nil {
		return err
	}

	return nil
}

// Sets additional fields (not covered by CREATE Op) on the resource's object
func (rm *resourceManager) setAdditionalFields(
	ctx context.Context,
	ko *svcapitypes.VPCEndpointServiceConfiguration,
) (latest *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.setAdditionalFields")
	defer func(err error) { exit(err) }(err)
	permInput := &svcsdk.DescribeVpcEndpointServicePermissionsInput{
		ServiceId: ko.Status.ServiceID,
	}
	var permResp *svcsdk.DescribeVpcEndpointServicePermissionsOutput
	permResp, err = rm.sdkapi.DescribeVpcEndpointServicePermissions(ctx, permInput)
	rm.metrics.RecordAPICall("READ_MANY", "DescribeVpcEndpointServicePermissions", err)
	if err != nil {
		if awsErr, ok := ackerr.AWSError(err); ok && awsErr.ErrorCode() == "UNKNOWN" {
			return nil, ackerr.NotFound
		}
		return nil, err
	}

	if permResp.AllowedPrincipals != nil {
		f0 := []*string{}
		for _, elem := range permResp.AllowedPrincipals {
			if elem.Principal != nil {
				f0 = append(f0, elem.Principal)
			}
		}
		ko.Spec.AllowedPrincipals = f0
	} else {
		ko.Spec.AllowedPrincipals = nil
	}

	return &resource{ko}, nil
}

// checkForMissingRequiredFields validates that all fields required for making a ReadMany
// API call are present in the resource's object. Need to use a custom method a current code-gen
// implementation does not include fields marked with is_primary_key.
func (rm *resourceManager) checkForMissingRequiredFields(r *resource) bool {
	return r.ko.Status.ServiceID == nil
}

var syncTags = tags.Sync

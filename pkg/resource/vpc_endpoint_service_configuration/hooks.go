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

	svcapitypes "github.com/aws-controllers-k8s/ec2-controller/apis/v1alpha1"
	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackerr "github.com/aws-controllers-k8s/runtime/pkg/errors"
	ackrequeue "github.com/aws-controllers-k8s/runtime/pkg/requeue"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	ackutil "github.com/aws-controllers-k8s/runtime/pkg/util"

	svcsdk "github.com/aws/aws-sdk-go/service/ec2"
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
	input.ServiceIds = []*string{r.ko.Status.ServiceID}
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

	toAdd := []*string{}
	toDelete := []*string{}

	currentlyAllowedPrincipals := latest.ko.Spec.AllowedPrincipals
	desiredAllowedPrincipals := desired.ko.Spec.AllowedPrincipals

	// Check if any desired allowed principals need to be added
	for _, p := range desiredAllowedPrincipals {
		if !ackutil.InStringPs(*p, currentlyAllowedPrincipals) {
			toAdd = append(toAdd, p)
		}
	}

	// Check if any currently allowed principals need to be deleted
	for _, p := range currentlyAllowedPrincipals {
		if !ackutil.InStringPs(*p, desiredAllowedPrincipals) {
			toDelete = append(toDelete, p)
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
	toAdd []*string,
	toDelete []*string,
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

	_, err = rm.sdkapi.ModifyVpcEndpointServicePermissionsWithContext(ctx, modifyPermissionsInput)
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
	permResp, err = rm.sdkapi.DescribeVpcEndpointServicePermissionsWithContext(ctx, permInput)
	rm.metrics.RecordAPICall("READ_MANY", "DescribeVpcEndpointServicePermissions", err)
	if err != nil {
		if awsErr, ok := ackerr.AWSError(err); ok && awsErr.Code() == "UNKNOWN" {
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

// syncTags used to keep tags in sync by calling Create and Delete API's
func (rm *resourceManager) syncTags(
	ctx context.Context,
	desired *resource,
	latest *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.syncTags")
	defer func(err error) {
		exit(err)
	}(err)

	resourceId := []*string{latest.ko.Status.ServiceID}

	toAdd, toDelete := computeTagsDelta(
		desired.ko.Spec.Tags, latest.ko.Spec.Tags,
	)

	if len(toDelete) > 0 {
		rlog.Debug("removing tags from VPCEndpoint resource", "tags", toDelete)
		_, err = rm.sdkapi.DeleteTagsWithContext(
			ctx,
			&svcsdk.DeleteTagsInput{
				Resources: resourceId,
				Tags:      rm.sdkTags(toDelete),
			},
		)
		rm.metrics.RecordAPICall("UPDATE", "DeleteTags", err)
		if err != nil {
			return err
		}

	}

	if len(toAdd) > 0 {
		rlog.Debug("adding tags to VPCEndpoint resource", "tags", toAdd)
		_, err = rm.sdkapi.CreateTagsWithContext(
			ctx,
			&svcsdk.CreateTagsInput{
				Resources: resourceId,
				Tags:      rm.sdkTags(toAdd),
			},
		)
		rm.metrics.RecordAPICall("UPDATE", "CreateTags", err)
		if err != nil {
			return err
		}
	}

	return nil
}

// sdkTags converts *svcapitypes.Tag array to a *svcsdk.Tag array
func (rm *resourceManager) sdkTags(
	tags []*svcapitypes.Tag,
) (sdktags []*svcsdk.Tag) {

	for _, i := range tags {
		sdktag := rm.newTag(*i)
		sdktags = append(sdktags, sdktag)
	}

	return sdktags
}

// computeTagsDelta returns tags to be added and removed from the resource
func computeTagsDelta(
	desired []*svcapitypes.Tag,
	latest []*svcapitypes.Tag,
) (toAdd []*svcapitypes.Tag, toDelete []*svcapitypes.Tag) {

	desiredTags := map[string]string{}
	for _, tag := range desired {
		desiredTags[*tag.Key] = *tag.Value
	}

	latestTags := map[string]string{}
	for _, tag := range latest {
		latestTags[*tag.Key] = *tag.Value
	}

	for _, tag := range desired {
		val, ok := latestTags[*tag.Key]
		if !ok || val != *tag.Value {
			toAdd = append(toAdd, tag)
		}
	}

	for _, tag := range latest {
		_, ok := desiredTags[*tag.Key]
		if !ok {
			toDelete = append(toDelete, tag)
		}
	}

	return toAdd, toDelete

}

// compareTags is a custom comparison function for comparing lists of Tag
// structs where the order of the structs in the list is not important.
func compareTags(
	delta *ackcompare.Delta,
	a *resource,
	b *resource,
) {
	if len(a.ko.Spec.Tags) != len(b.ko.Spec.Tags) {
		delta.Add("Spec.Tags", a.ko.Spec.Tags, b.ko.Spec.Tags)
	} else if len(a.ko.Spec.Tags) > 0 {
		addedOrUpdated, removed := computeTagsDelta(a.ko.Spec.Tags, b.ko.Spec.Tags)
		if len(addedOrUpdated) != 0 || len(removed) != 0 {
			delta.Add("Spec.Tags", a.ko.Spec.Tags, b.ko.Spec.Tags)
		}
	}
}

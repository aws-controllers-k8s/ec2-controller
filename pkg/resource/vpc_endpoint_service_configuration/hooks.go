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
	ackrequeue "github.com/aws-controllers-k8s/runtime/pkg/requeue"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"

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
	exit := rlog.Trace("updateAllowedPrincipals")
	defer func(err error) {
		exit(err)
	}(err)

	var listOfPrincipalsToAdd []*string
	var listOfPrincipalsToRemove []*string

	// If the latest list of principals is empty, we want to add all principals
	if len(latest.ko.Spec.AllowedPrincipals) == 0 && len(desired.ko.Spec.AllowedPrincipals) > 0 {
		listOfPrincipalsToAdd = desired.ko.Spec.AllowedPrincipals

		// If the desired list of principals is empty, we want to remove all principals
	} else if len(desired.ko.Spec.AllowedPrincipals) == 0 && len(latest.ko.Spec.AllowedPrincipals) > 0 {
		listOfPrincipalsToRemove = latest.ko.Spec.AllowedPrincipals
		// Otherwise, we'll compare the two lists and add/remove principals as needed
	} else {
		// Add any 'desired' principal that is not on the allowed list
		for _, desiredPrincipal := range desired.ko.Spec.AllowedPrincipals {
			principalToAddAlreadyFound := false
			for _, latestPrincipal := range latest.ko.Spec.AllowedPrincipals {
				if *desiredPrincipal == *latestPrincipal {
					// Principal already in Allow List, skip
					principalToAddAlreadyFound = true
					break
				}
			}
			if !principalToAddAlreadyFound {
				// Desired Principal is not in the Allowed List, add it to the list of those to add
				listOfPrincipalsToAdd = append(listOfPrincipalsToAdd, desiredPrincipal)
			}
		}

		// Remove any 'latest' principal that is not on the allowed list anymore
		for _, latestPrincipal := range latest.ko.Spec.AllowedPrincipals {
			principalToRemoveAlreadyFound := false
			for _, desiredPrincipal := range desired.ko.Spec.AllowedPrincipals {
				if *desiredPrincipal == *latestPrincipal {
					// Principal still in Allow List, skip
					principalToRemoveAlreadyFound = true
					break
				}
			}
			if !principalToRemoveAlreadyFound {
				// Latest Principal is not in the Allowed List, add it to the list of those to remove
				listOfPrincipalsToRemove = append(listOfPrincipalsToRemove, latestPrincipal)
			}
		}

	}

	// Make the AWS API call to update the allowed principals
	if len(listOfPrincipalsToAdd) > 0 || len(listOfPrincipalsToRemove) > 0 {
		modifyPermissionsInput := &svcsdk.ModifyVpcEndpointServicePermissionsInput{
			ServiceId: latest.ko.Status.ServiceID,
		}

		if len(listOfPrincipalsToAdd) > 0 {
			modifyPermissionsInput.AddAllowedPrincipals = listOfPrincipalsToAdd
		}

		if len(listOfPrincipalsToRemove) > 0 {
			modifyPermissionsInput.RemoveAllowedPrincipals = listOfPrincipalsToRemove
		}

		_, err := rm.sdkapi.ModifyVpcEndpointServicePermissions(modifyPermissionsInput)
		rm.metrics.RecordAPICall("UPDATE", "ModifyVpcEndpointServicePermissions", err)
		if err != nil {
			return desired, err
		}
	}
	return desired, nil
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

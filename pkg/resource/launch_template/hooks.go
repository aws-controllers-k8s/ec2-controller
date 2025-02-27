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

package launch_template

import (
	"context"
	"strconv"

	svcapitypes "github.com/aws-controllers-k8s/ec2-controller/apis/v1alpha1"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	svcsdk "github.com/aws/aws-sdk-go-v2/service/ec2"
	svcsdktypes "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func (rm *resourceManager) setDefaultTemplateVersion(r *resource, input *svcsdk.ModifyLaunchTemplateInput) error {

	if r.ko.Spec.DefaultVersionNumber != nil {

		defaultVersion := strconv.FormatInt(*r.ko.Spec.DefaultVersionNumber, 10)
		input.DefaultVersion = &defaultVersion

	}

	return nil
}

// updateTagSpecificationsInCreateRequest adds
// Tags defined in the Spec to CreateLaunchTemplate.TagSpecification
// and ensures the ResourceType is always set to 'launch-template'
func updateTagSpecificationsInCreateRequest(r *resource,
	input *svcsdk.CreateLaunchTemplateInput) {
	input.TagSpecifications = nil
	desiredTagSpecs := svcsdktypes.TagSpecification{}

	if r.ko.Spec.Tags != nil {

		requestedTags := []svcsdktypes.Tag{}
		for _, desiredTag := range r.ko.Spec.Tags {

			// Add in tags defined in the Spec
			tag := svcsdktypes.Tag{}
			if desiredTag.Key != nil && desiredTag.Value != nil {
				{

					tag.Key = desiredTag.Key
					tag.Value = desiredTag.Value

				}
				requestedTags = append(requestedTags, tag)
			}

		}
		desiredTagSpecs.ResourceType = svcsdktypes.ResourceTypeLaunchTemplate
		desiredTagSpecs.Tags = requestedTags
		input.TagSpecifications = []svcsdktypes.TagSpecification{desiredTagSpecs}
	}
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

	resourceId := []string{*latest.ko.Status.LaunchTemplateID}

	toAdd, toDelete := computeTagsDelta(
		desired.ko.Spec.Tags, latest.ko.Spec.Tags,
	)

	if len(toDelete) > 0 {
		rlog.Debug("removing tags from launchtemplate resource", "tags", toDelete)
		_, err = rm.sdkapi.DeleteTags(
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
		rlog.Debug("adding tags to launchtemplate resource", "tags", toAdd)
		_, err = rm.sdkapi.CreateTags(
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
	tags []svcapitypes.Tag,
) (sdktags []svcsdktypes.Tag) {

	for _, i := range tags {
		sdktag := rm.newTag(i)
		sdktags = append(sdktags, *sdktag)
	}

	return sdktags
}

func (rm *resourceManager) newTag(
	c svcapitypes.Tag,
) *svcsdktypes.Tag {
	res := &svcsdktypes.Tag{}
	if c.Key != nil {
		res.Key = c.Key
	}
	if c.Value != nil {
		res.Value = c.Value

	}

	return res
}

// computeTagsDelta returns tags to be added and removed from the resource
func computeTagsDelta(
	desired []*svcapitypes.Tag,
	latest []*svcapitypes.Tag,
) (toAdd []svcapitypes.Tag, toDelete []svcapitypes.Tag) {

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
			toAdd = append(toAdd, *tag)
		}
	}

	for _, tag := range latest {
		_, ok := desiredTags[*tag.Key]
		if !ok {
			toDelete = append(toDelete, *tag)
		}
	}

	return toAdd, toDelete

}

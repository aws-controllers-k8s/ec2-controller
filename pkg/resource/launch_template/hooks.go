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
	"fmt"

	ackerr "github.com/aws-controllers-k8s/runtime/pkg/errors"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	"github.com/aws/aws-sdk-go-v2/aws"
	svcsdk "github.com/aws/aws-sdk-go-v2/service/ec2"
	svcsdktypes "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	svcapitypes "github.com/aws-controllers-k8s/ec2-controller/apis/v1alpha1"
	"github.com/aws-controllers-k8s/ec2-controller/pkg/tags"
)

var syncTags = tags.Sync

// setLatestLaunchTemplateAttributes calls DescribeLaunchTemplateVersions
// API and retrieves information that will be populated in LaunchTemplateData
func (rm *resourceManager) setLatestLaunchTemplateAttributes(
	ctx context.Context,
	r *resource,
	ko *svcapitypes.LaunchTemplate,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.findLaunchTemplateVersion")
	defer func() {
		exit(err)
	}()

	input, err := rm.newListLaunchTemplateVersionRequestPayload(r)
	if err != nil {
		return err
	}

	var resp *svcsdk.DescribeLaunchTemplateVersionsOutput
	resp, err = rm.sdkapi.DescribeLaunchTemplateVersions(ctx, input)
	rm.metrics.RecordAPICall("READ_MANY", "DescribeLaunchTemplateVersions", err)
	if err != nil {
		return err
	}

	for _, elem := range resp.LaunchTemplateVersions {
		if elem.CreateTime != nil {
			ko.Status.CreateTime = &metav1.Time{*elem.CreateTime}
		} else {
			ko.Status.CreateTime = nil
		}
		if elem.CreatedBy != nil {
			ko.Status.CreatedBy = elem.CreatedBy
		} else {
			ko.Status.CreatedBy = nil
		}
		if elem.LaunchTemplateData != nil {
			ko.Spec.Data = rm.setRequestLaunchTemplateData(elem.LaunchTemplateData)
		} else {
			ko.Spec.Data = nil
		}
		if elem.Operator != nil {
			f6 := &svcapitypes.OperatorResponse{}
			if elem.Operator.Managed != nil {
				f6.Managed = elem.Operator.Managed
			}
			if elem.Operator.Principal != nil {
				f6.Principal = elem.Operator.Principal
			}
			ko.Status.Operator = f6
		} else {
			ko.Status.Operator = nil
		}
		if elem.VersionDescription != nil {
			ko.Spec.VersionDescription = elem.VersionDescription
		} else {
			ko.Spec.VersionDescription = nil
		}
	}
	return nil
}

func (rm *resourceManager) newListLaunchTemplateVersionRequestPayload(
	r *resource,
) (*svcsdk.DescribeLaunchTemplateVersionsInput, error) {
	res := &svcsdk.DescribeLaunchTemplateVersionsInput{}

	if r.ko.Status.ID != nil {
		res.LaunchTemplateId = r.ko.Status.ID
	}
	if r.ko.Status.LatestVersion != nil {
		res.Versions = []string{fmt.Sprintf("%d", *r.ko.Status.LatestVersion)}
	}

	return res, nil
}

// updateDefaultVersion patches the supplied resource in the backend AWS service API
func (rm *resourceManager) updateDefaultVersion(
	ctx context.Context,
	desired *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.updateDefaultVersion")
	defer func() {
		exit(err)
	}()

	// TODO (michaelhtm) Not sure if we should check if
	// the defaultVersion is greater than latestVersion
	//
	// My proposal would be to return a terminal error
	// since the launchTemplate's latestVersion will never
	// increase intil we make updates...˘\\/(ヅ)\/˘˘
	// if *desired.ko.Spec.DefaultVersion > *latest.ko.Status.LatestVersion {
	// 	return ackerr.NewTerminalError(fmt.Errorf("desired version number is ahead of the latest version"))
	// }

	ko := desired.ko.DeepCopy()

	input, err := newUpdateLaunchTemplateVersionRequestPayload(&resource{ko})
	if err != nil {
		return err
	}

	var resp *svcsdk.ModifyLaunchTemplateOutput
	resp, err = rm.sdkapi.ModifyLaunchTemplate(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "ModifyLaunchTemplate", err)
	if err != nil {
		return err
	}

	if resp.LaunchTemplate.CreateTime != nil {
		ko.Status.CreateTime = &metav1.Time{*resp.LaunchTemplate.CreateTime}
	} else {
		ko.Status.CreateTime = nil
	}
	if resp.LaunchTemplate.CreatedBy != nil {
		ko.Status.CreatedBy = resp.LaunchTemplate.CreatedBy
	} else {
		ko.Status.CreatedBy = nil
	}
	if resp.LaunchTemplate.DefaultVersionNumber != nil {
		ko.Spec.DefaultVersion = resp.LaunchTemplate.DefaultVersionNumber
	} else {
		ko.Spec.DefaultVersion = nil
	}
	if resp.LaunchTemplate.LatestVersionNumber != nil {
		ko.Status.LatestVersion = resp.LaunchTemplate.LatestVersionNumber
	} else {
		ko.Status.LatestVersion = nil
	}
	if resp.LaunchTemplate.Operator != nil {
		f6 := &svcapitypes.OperatorResponse{}
		if resp.LaunchTemplate.Operator.Managed != nil {
			f6.Managed = resp.LaunchTemplate.Operator.Managed
		}
		if resp.LaunchTemplate.Operator.Principal != nil {
			f6.Principal = resp.LaunchTemplate.Operator.Principal
		}
		ko.Status.Operator = f6
	} else {
		ko.Status.Operator = nil
	}
	if resp.LaunchTemplate.Tags != nil {
		f7 := []*svcapitypes.Tag{}
		for _, f7iter := range resp.LaunchTemplate.Tags {
			f7elem := &svcapitypes.Tag{}
			if f7iter.Key != nil {
				f7elem.Key = f7iter.Key
			}
			if f7iter.Value != nil {
				f7elem.Value = f7iter.Value
			}
			f7 = append(f7, f7elem)
		}
		ko.Spec.Tags = f7
	} else {
		ko.Spec.Tags = nil
	}

	rm.setStatusDefaults(ko)
	return nil
}

func newUpdateLaunchTemplateVersionRequestPayload(
	r *resource,
) (*svcsdk.ModifyLaunchTemplateInput, error) {
	res := &svcsdk.ModifyLaunchTemplateInput{}

	if r.ko.Spec.Name != nil {
		res.LaunchTemplateName = r.ko.Spec.Name
	}
	if r.ko.Spec.DefaultVersion != nil {
		res.DefaultVersion = aws.String(fmt.Sprintf("%d", *r.ko.Spec.DefaultVersion))
	} else {
		return nil, ackerr.NewTerminalError(fmt.Errorf("field DefaultVersion is required"))
	}

	return res, nil
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

func (rm *resourceManager) checkForMissingRequiredFields(r *resource) bool {
	return r.ko.Status.ID == nil
}

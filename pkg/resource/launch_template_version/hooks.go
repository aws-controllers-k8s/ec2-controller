package launch_template_version

import (
	"context"
	"strconv"

	ackerr "github.com/aws-controllers-k8s/runtime/pkg/errors"
	svcsdk "github.com/aws/aws-sdk-go/service/ec2"
)

// This is to find current version number of launch template and increment it by 1 as new version number
// and pass as input to SDKFIND
func (rm *resourceManager) customSdkfind(ctx context.Context,
	r *resource,
	input *svcsdk.DescribeLaunchTemplateVersionsInput) error {

	res_launch_template := &svcsdk.DescribeLaunchTemplatesInput{}

	if r.ko.Spec.LaunchTemplateName != nil {
		f2 := []*string{}
		f2 = append(f2, r.ko.Spec.LaunchTemplateName)
		res_launch_template.SetLaunchTemplateNames(f2)
	}

	if r.ko.Spec.LaunchTemplateID != nil {
		f2 := []*string{}
		f2 = append(f2, r.ko.Spec.LaunchTemplateID)
		res_launch_template.SetLaunchTemplateIds(f2)
	}

	if res_launch_template.LaunchTemplateIds != nil && res_launch_template.LaunchTemplateNames != nil {
		res_launch_template.LaunchTemplateIds = nil
	}

	var resp_launch_template *svcsdk.DescribeLaunchTemplatesOutput
	resp_launch_template, err := rm.sdkapi.DescribeLaunchTemplatesWithContext(ctx, res_launch_template)
	rm.metrics.RecordAPICall("READ_MANY", "DescribeLaunchTemplates", err)
	if err != nil {
		if awsErr, ok := ackerr.AWSError(err); ok && awsErr.Code() == "InvalidLaunchTemplateName.NotFoundException" {
			return ackerr.NotFound
		}
		return err
	}

	for _, item := range resp_launch_template.LaunchTemplates {
		latest_version := item.LatestVersionNumber
		if r.ko.Status.VersionNumber != nil {
			version_number_str := strconv.Itoa(int(*r.ko.Status.VersionNumber))
			input.SetVersions([]*string{&version_number_str})
		} else {
			*latest_version++
			new_version_str := strconv.Itoa(int(*latest_version))
			input.SetVersions([]*string{&new_version_str})
		}
	}

	if input.LaunchTemplateId != nil && input.LaunchTemplateName != nil {
		input.LaunchTemplateId = nil
	}

	return nil
}

// This is to set launchtemplateid as nil in input becasue deletelaunchtemplateversions cannot accept name and id together.
// Also setting up version to delete in input
func (rm *resourceManager) customSdkdelete(r *resource, input *svcsdk.DeleteLaunchTemplateVersionsInput) error {

	if input.LaunchTemplateId != nil && input.LaunchTemplateName != nil {
		input.LaunchTemplateId = nil
	}

	if r.ko.Status.VersionNumber != nil {
		var versionnumber string
		versionnumber = strconv.Itoa(int(*r.ko.Status.VersionNumber))
		input.SetVersions([]*string{&versionnumber})
	}

	return nil
}

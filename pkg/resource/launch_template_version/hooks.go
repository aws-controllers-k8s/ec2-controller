package launch_template_version

import (
	"context"
	"fmt"
	"strconv"

	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	svcsdk "github.com/aws/aws-sdk-go/service/ec2"
)

// customDeleteApi deletes the supplied resource in the backend AWS service API
func (rm *resourceManager) customDeleteLaunchTemplateVersions(
	ctx context.Context,
	r *resource,
) (latest *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.sdkDelete")
	defer func() {
		exit(err)
	}()

	fmt.Println("I am here")
	// TODO(jaypipes): Figure this out...
	input, err := rm.newDeleteRequestPayload(r)
	if err != nil {
		return nil, err
	}
	var resp *svcsdk.DeleteLaunchTemplateVersionsOutput
	_ = resp
	resp, err = rm.sdkapi.DeleteLaunchTemplateVersionsWithContext(ctx, input)
	rm.metrics.RecordAPICall("DELETE", "DeleteLaunchTemplateVersions", err)
	return nil, err
}

// newDeleteRequestPayload returns an SDK-specific struct for the HTTP request
// payload of the Delete API call for the resource
func (rm *resourceManager) newDeleteRequestPayload(
	r *resource,
) (*svcsdk.DeleteLaunchTemplateVersionsInput, error) {
	res := &svcsdk.DeleteLaunchTemplateVersionsInput{}
	if r.ko.Spec.DryRun != nil {
		res.SetDryRun(*r.ko.Spec.DryRun)
	}
	if r.ko.Spec.LaunchTemplateName != nil {
		res.SetLaunchTemplateName(*r.ko.Spec.LaunchTemplateName)
	}
	// if r.ko.Spec.LaunchTemplateID != nil {
	// 	res.SetLaunchTemplateId(*r.ko.Spec.LaunchTemplateID)
	// }
	if r.ko.Status.VersionNumber != nil {
		var versionnumber string
		versionnumber = strconv.Itoa(int(*r.ko.Status.VersionNumber))
		res.SetVersions([]*string{&versionnumber})
	}
	return res, nil
}

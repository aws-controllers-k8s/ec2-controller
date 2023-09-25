package launch_template

import (
	"strconv"

	svcsdk "github.com/aws/aws-sdk-go/service/ec2"
)

func (rm *resourceManager) setDefaultTemplateVersion(r *resource, input *svcsdk.ModifyLaunchTemplateInput) error {

	if r.ko.Spec.DefaultVersionNumber != nil {
		input.SetDefaultVersion(strconv.Itoa(int(*r.ko.Spec.DefaultVersionNumber)))
	}

	return nil
}

package main

import (
	svcsapitypes "github.com/aws-controllers-k8s/ec2-controller/apis/v1alpha1"
	svcsdk "github.com/aws/aws-sdk-go-v2/service/ec2"
	svcsdktypes "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// updateTagSpecificationsInCreateRequest adds
// Tags defined in the Spec to CreateVpcEndpointInput.TagSpecification
// and ensures the ResourceType is always set to 'vpc-endpoint'
func updateTagSpecificationsInCreateRequest(tags []*svcsapitypes.Tag, input *svcsdk.CreateVpcEndpointInput) {
	input.TagSpecifications = nil
	desiredTagSpecs := svcsdktypes.TagSpecification{}
	if tags != nil {
		requestedTags := []svcsdktypes.Tag{}
		for _, desiredTag := range tags {
			// Add in tags defined in the Spec
			tag := svcsdktypes.Tag{}
			if desiredTag.Key != nil && desiredTag.Value != nil {
				tag.Key = desiredTag.Key
				tag.Value = desiredTag.Value
			}
			requestedTags = append(requestedTags, tag)
		}
		desiredTagSpecs.ResourceType = "vpc-endpoint"
		desiredTagSpecs.Tags = requestedTags
		input.TagSpecifications = []svcsdktypes.TagSpecification{desiredTagSpecs}
	}
}

func main() {

}

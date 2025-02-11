package capacity_reservation

import (
	"github.com/aws-controllers-k8s/ec2-controller/pkg/tags"
	svcsdk "github.com/aws/aws-sdk-go-v2/service/ec2"
	svcsdktypes "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

var syncTags = tags.Sync

// updateTagSpecificationsInCreateRequest adds
// Tags defined in the Spec to CreateCapacityReservationInput.TagSpecifications
// and ensures the ResourceType is always set to 'capacity-reservation'
func updateTagSpecificationsInCreateRequest(r *resource,
	input *svcsdk.CreateCapacityReservationInput) {
	input.TagSpecifications = nil
	desiredTagSpecs := svcsdktypes.TagSpecification{}
	if r.ko.Spec.Tags != nil {
		requestedTags := []svcsdktypes.Tag{}
		for _, desiredTag := range r.ko.Spec.Tags {
			// Add in tags defined in the Spec
			tag := svcsdktypes.Tag{}
			if desiredTag.Key != nil && desiredTag.Value != nil {
				tag.Key = desiredTag.Key
				tag.Value = desiredTag.Value
			}
			requestedTags = append(requestedTags, tag)
		}
		desiredTagSpecs.ResourceType = "capacity-reservation"
		desiredTagSpecs.Tags = requestedTags
		input.TagSpecifications = []svcsdktypes.TagSpecification{desiredTagSpecs}
	}
}

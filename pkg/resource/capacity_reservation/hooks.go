package capacity_reservation

import (
	"github.com/aws-controllers-k8s/ec2-controller/pkg/tags"
	svcsdk "github.com/aws/aws-sdk-go/service/ec2"
)

var syncTags = tags.Sync

// updateTagSpecificationsInCreateRequest adds
// Tags defined in the Spec to CreateCapacityReservationInput.TagSpecifications
// and ensures the ResourceType is always set to 'capacity-reservation'
func updateTagSpecificationsInCreateRequest(r *resource,
	input *svcsdk.CreateCapacityReservationInput) {
	input.TagSpecifications = nil
	desiredTagSpecs := svcsdk.TagSpecification{}
	if r.ko.Spec.Tags != nil {
		requestedTags := []*svcsdk.Tag{}
		for _, desiredTag := range r.ko.Spec.Tags {
			// Add in tags defined in the Spec
			tag := &svcsdk.Tag{}
			if desiredTag.Key != nil && desiredTag.Value != nil {
				tag.SetKey(*desiredTag.Key)
				tag.SetValue(*desiredTag.Value)
			}
			requestedTags = append(requestedTags, tag)
		}
		desiredTagSpecs.SetResourceType("capacity-reservation")
		desiredTagSpecs.SetTags(requestedTags)
		input.TagSpecifications = []*svcsdk.TagSpecification{&desiredTagSpecs}
	}
}

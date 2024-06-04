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

package nat_gateway

import (
	"context"

	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	svcsdk "github.com/aws/aws-sdk-go/service/ec2"

	"github.com/aws-controllers-k8s/ec2-controller/pkg/tags"
)

func (rm *resourceManager) customUpdateNATGateway(
	ctx context.Context,
	desired *resource,
	latest *resource,
	delta *ackcompare.Delta,
) (updated *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.customUpdateNATGateway")
	defer exit(err)

	// Default `updated` to `desired` because it is likely
	// EC2 `modify` APIs do NOT return output, only errors.
	// If the `modify` calls (i.e. `sync`) do NOT return
	// an error, then the update was successful and desired.Spec
	// (now updated.Spec) reflects the latest resource state.
	updated = rm.concreteResource(desired.DeepCopy())

	if delta.DifferentAt("Spec.Tags") {
		if err := tags.Sync(
			ctx, rm.sdkapi, rm.metrics, *latest.ko.Status.NATGatewayID,
			desired.ko.Spec.Tags, latest.ko.Spec.Tags,
		); err != nil {
			return nil, err
		}
	}

	return updated, nil
}

// updateTagSpecificationsInCreateRequest adds
// Tags defined in the Spec to CreateNatGatewayInput.TagSpecification
// and ensures the ResourceType is always set to 'natgateway'
func updateTagSpecificationsInCreateRequest(r *resource,
	input *svcsdk.CreateNatGatewayInput) {
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
		desiredTagSpecs.SetResourceType("natgateway")
		desiredTagSpecs.SetTags(requestedTags)
		input.TagSpecifications = []*svcsdk.TagSpecification{&desiredTagSpecs}
	}
}

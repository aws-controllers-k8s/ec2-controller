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

package key_pair

import (
	"context"

	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	svcsdk "github.com/aws/aws-sdk-go-v2/service/ec2"
	svcsdktypes "github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/aws-controllers-k8s/ec2-controller/pkg/tags"
)

// checkForMissingRequiredFields returns true when the key pair cannot be
// looked up yet. DescribeKeyPairs filters by name, so the name is all the
// read needs; without this the generated check would wait for Status.KeyPairID
// and a key pair that exists but has no status yet would read as not found.
func (rm *resourceManager) checkForMissingRequiredFields(r *resource) bool {
	return r.ko.Spec.KeyName == nil
}

// customUpdateKeyPair syncs the tags, the only mutable part of a key pair.
func (rm *resourceManager) customUpdateKeyPair(
	ctx context.Context,
	desired *resource,
	latest *resource,
	delta *ackcompare.Delta,
) (updated *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.customUpdateKeyPair")
	defer exit(err)

	updated = rm.concreteResource(desired.DeepCopy())

	if delta.DifferentAt("Spec.Tags") {
		if err := tags.Sync(
			ctx, rm.sdkapi, rm.metrics, *latest.ko.Status.KeyPairID,
			desired.ko.Spec.Tags, latest.ko.Spec.Tags,
		); err != nil {
			return nil, err
		}
	}

	return updated, nil
}

// updateTagSpecificationsInCreateRequest adds the tags defined in the Spec to
// ImportKeyPairInput.TagSpecifications with the resource type set to
// 'key-pair'.
func updateTagSpecificationsInCreateRequest(r *resource,
	input *svcsdk.ImportKeyPairInput) {
	input.TagSpecifications = nil
	if r.ko.Spec.Tags == nil {
		return
	}
	requestedTags := []svcsdktypes.Tag{}
	for _, desiredTag := range r.ko.Spec.Tags {
		tag := svcsdktypes.Tag{}
		if desiredTag.Key != nil && desiredTag.Value != nil {
			tag.Key = desiredTag.Key
			tag.Value = desiredTag.Value
		}
		requestedTags = append(requestedTags, tag)
	}
	input.TagSpecifications = []svcsdktypes.TagSpecification{{
		ResourceType: svcsdktypes.ResourceTypeKeyPair,
		Tags:         requestedTags,
	}}
}

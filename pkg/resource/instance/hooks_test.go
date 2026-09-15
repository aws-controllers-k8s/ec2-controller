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

package instance

import (
	"testing"

	svcapitypes "github.com/aws-controllers-k8s/ec2-controller/apis/v1alpha1"
	"github.com/aws/aws-sdk-go-v2/aws"
	svcsdk "github.com/aws/aws-sdk-go-v2/service/ec2"
	svcsdktypes "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/stretchr/testify/assert"
)

func instanceWithTags(tags []*svcapitypes.Tag) *resource {
	return &resource{
		ko: &svcapitypes.Instance{
			Spec: svcapitypes.InstanceSpec{
				Tags: tags,
			},
		},
	}
}

func instanceWithNetworkInterfaces(
	tags []*svcapitypes.Tag,
	networkInterfaces []*svcapitypes.InstanceNetworkInterfaceSpecification,
) *resource {
	return &resource{
		ko: &svcapitypes.Instance{
			Spec: svcapitypes.InstanceSpec{
				Tags:              tags,
				NetworkInterfaces: networkInterfaces,
			},
		},
	}
}

func resourceTypesOf(specs []svcsdktypes.TagSpecification) []svcsdktypes.ResourceType {
	out := make([]svcsdktypes.ResourceType, 0, len(specs))
	for _, s := range specs {
		out = append(out, s.ResourceType)
	}
	return out
}

func TestUpdateTagSpecificationsInCreateRequest(t *testing.T) {
	tag := func(k, v string) *svcapitypes.Tag {
		return &svcapitypes.Tag{Key: aws.String(k), Value: aws.String(v)}
	}

	// The tags in Spec.Tags must be applied to every resource type created during
	// instance launch so that tag-based SCPs (aws:RequestTag) are satisfied on the
	// sub-resources, not only on the instance.
	wantResourceTypes := []svcsdktypes.ResourceType{
		svcsdktypes.ResourceTypeInstance,
		svcsdktypes.ResourceTypeVolume,
		svcsdktypes.ResourceTypeNetworkInterface,
	}

	t.Run("nil spec tags leaves TagSpecifications empty", func(t *testing.T) {
		input := &svcsdk.RunInstancesInput{}
		updateTagSpecificationsInCreateRequest(instanceWithTags(nil), input)
		assert.Empty(t, input.TagSpecifications)
	})

	t.Run("spec tags applied to instance, volume and network interface", func(t *testing.T) {
		input := &svcsdk.RunInstancesInput{}
		desired := instanceWithTags([]*svcapitypes.Tag{
			tag("env", "prod"),
			tag("team", "ack"),
		})

		updateTagSpecificationsInCreateRequest(desired, input)

		assert.Equal(t, wantResourceTypes, resourceTypesOf(input.TagSpecifications))

		wantTags := []svcsdktypes.Tag{
			{Key: aws.String("env"), Value: aws.String("prod")},
			{Key: aws.String("team"), Value: aws.String("ack")},
		}
		for _, ts := range input.TagSpecifications {
			assert.Equal(t, wantTags, ts.Tags, "resource type %s", ts.ResourceType)
		}
	})

	t.Run("any pre-existing TagSpecifications are replaced", func(t *testing.T) {
		input := &svcsdk.RunInstancesInput{
			TagSpecifications: []svcsdktypes.TagSpecification{
				{ResourceType: svcsdktypes.ResourceTypeElasticGpu},
			},
		}
		updateTagSpecificationsInCreateRequest(
			instanceWithTags([]*svcapitypes.Tag{tag("k", "v")}), input)
		assert.Equal(t, wantResourceTypes, resourceTypesOf(input.TagSpecifications))
	})

	t.Run("tag with a nil value keeps its key", func(t *testing.T) {
		input := &svcsdk.RunInstancesInput{}
		desired := instanceWithTags([]*svcapitypes.Tag{
			{Key: aws.String("novalue")},
		})

		updateTagSpecificationsInCreateRequest(desired, input)

		assert.Equal(t, wantResourceTypes, resourceTypesOf(input.TagSpecifications))
		for _, ts := range input.TagSpecifications {
			assert.Equal(t,
				[]svcsdktypes.Tag{{Key: aws.String("novalue")}}, ts.Tags,
				"resource type %s", ts.ResourceType)
		}
	})

	t.Run("tag without a key is skipped", func(t *testing.T) {
		input := &svcsdk.RunInstancesInput{}
		desired := instanceWithTags([]*svcapitypes.Tag{
			{Value: aws.String("orphan")},
			tag("env", "prod"),
		})

		updateTagSpecificationsInCreateRequest(desired, input)

		for _, ts := range input.TagSpecifications {
			assert.Equal(t,
				[]svcsdktypes.Tag{{Key: aws.String("env"), Value: aws.String("prod")}},
				ts.Tags, "resource type %s", ts.ResourceType)
		}
	})

	t.Run("no usable tags leaves TagSpecifications empty", func(t *testing.T) {
		input := &svcsdk.RunInstancesInput{}
		desired := instanceWithTags([]*svcapitypes.Tag{
			{Value: aws.String("orphan")},
		})

		updateTagSpecificationsInCreateRequest(desired, input)

		// An entry with an empty Tags list is rejected by RunInstances, so no
		// tag specification must be emitted at all.
		assert.Empty(t, input.TagSpecifications)
	})

	t.Run("attaching only existing ENIs omits network-interface", func(t *testing.T) {
		input := &svcsdk.RunInstancesInput{}
		desired := instanceWithNetworkInterfaces(
			[]*svcapitypes.Tag{tag("env", "prod")},
			[]*svcapitypes.InstanceNetworkInterfaceSpecification{
				{NetworkInterfaceID: aws.String("eni-1111111111111111")},
			},
		)

		updateTagSpecificationsInCreateRequest(desired, input)

		assert.Equal(t, []svcsdktypes.ResourceType{
			svcsdktypes.ResourceTypeInstance,
			svcsdktypes.ResourceTypeVolume,
		}, resourceTypesOf(input.TagSpecifications))
	})

	t.Run("a mixed network interface list still creates one", func(t *testing.T) {
		input := &svcsdk.RunInstancesInput{}
		desired := instanceWithNetworkInterfaces(
			[]*svcapitypes.Tag{tag("env", "prod")},
			[]*svcapitypes.InstanceNetworkInterfaceSpecification{
				{NetworkInterfaceID: aws.String("eni-1111111111111111")},
				{DeviceIndex: aws.Int64(1)},
			},
		)

		updateTagSpecificationsInCreateRequest(desired, input)

		assert.Equal(t, wantResourceTypes, resourceTypesOf(input.TagSpecifications))
	})

	t.Run("an empty network interface list creates the primary ENI", func(t *testing.T) {
		input := &svcsdk.RunInstancesInput{}
		desired := instanceWithNetworkInterfaces(
			[]*svcapitypes.Tag{tag("env", "prod")},
			[]*svcapitypes.InstanceNetworkInterfaceSpecification{},
		)

		updateTagSpecificationsInCreateRequest(desired, input)

		assert.Equal(t, wantResourceTypes, resourceTypesOf(input.TagSpecifications))
	})
}

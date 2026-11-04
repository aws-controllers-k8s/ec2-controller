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

package tags

import (
	"context"
	"testing"

	svcapitypes "github.com/aws-controllers-k8s/ec2-controller/apis/v1alpha1"
	"github.com/aws/aws-sdk-go-v2/aws"
	svcsdk "github.com/aws/aws-sdk-go-v2/service/ec2"
	svcsdktypes "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/stretchr/testify/assert"
)

type fakeTagsClient struct {
	createInputs []*svcsdk.CreateTagsInput
	deleteInputs []*svcsdk.DeleteTagsInput
}

func (f *fakeTagsClient) CreateTags(
	_ context.Context,
	in *svcsdk.CreateTagsInput,
	_ ...func(*svcsdk.Options),
) (*svcsdk.CreateTagsOutput, error) {
	f.createInputs = append(f.createInputs, in)
	return &svcsdk.CreateTagsOutput{}, nil
}

func (f *fakeTagsClient) DescribeTags(
	_ context.Context,
	_ *svcsdk.DescribeTagsInput,
	_ ...func(*svcsdk.Options),
) (*svcsdk.DescribeTagsOutput, error) {
	return &svcsdk.DescribeTagsOutput{}, nil
}

func (f *fakeTagsClient) DeleteTags(
	_ context.Context,
	in *svcsdk.DeleteTagsInput,
	_ ...func(*svcsdk.Options),
) (*svcsdk.DeleteTagsOutput, error) {
	f.deleteInputs = append(f.deleteInputs, in)
	return &svcsdk.DeleteTagsOutput{}, nil
}

type fakeMetricsRecorder struct{}

func (fakeMetricsRecorder) RecordAPICall(string, string, error) {}

func createdTags(client *fakeTagsClient) []svcsdktypes.Tag {
	tags := []svcsdktypes.Tag{}
	for _, in := range client.createInputs {
		tags = append(tags, in.Tags...)
	}
	return tags
}

func deletedTags(client *fakeTagsClient) []svcsdktypes.Tag {
	tags := []svcsdktypes.Tag{}
	for _, in := range client.deleteInputs {
		tags = append(tags, in.Tags...)
	}
	return tags
}

func tag(key, value string) *svcapitypes.Tag {
	return &svcapitypes.Tag{Key: &key, Value: &value}
}

func sdkTag(key, value string) svcsdktypes.Tag {
	return svcsdktypes.Tag{Key: &key, Value: &value}
}

func sdkKey(key string) svcsdktypes.Tag {
	return svcsdktypes.Tag{Key: &key}
}

func TestSync(t *testing.T) {
	tests := []struct {
		name        string
		desired     []*svcapitypes.Tag
		latest      []*svcapitypes.Tag
		wantCreated []svcsdktypes.Tag
		wantDeleted []svcsdktypes.Tag
	}{
		{
			name:        "no-op when desired and latest are equal",
			desired:     []*svcapitypes.Tag{tag("a", "1")},
			latest:      []*svcapitypes.Tag{tag("a", "1")},
			wantCreated: []svcsdktypes.Tag{},
			wantDeleted: []svcsdktypes.Tag{},
		},
		{
			name:        "add a tag missing from latest",
			desired:     []*svcapitypes.Tag{tag("a", "1"), tag("b", "2")},
			latest:      []*svcapitypes.Tag{tag("a", "1")},
			wantCreated: []svcsdktypes.Tag{sdkTag("b", "2")},
			wantDeleted: []svcsdktypes.Tag{},
		},
		{
			name:        "remove a tag missing from desired",
			desired:     []*svcapitypes.Tag{tag("a", "1")},
			latest:      []*svcapitypes.Tag{tag("a", "1"), tag("b", "2")},
			wantCreated: []svcsdktypes.Tag{},
			wantDeleted: []svcsdktypes.Tag{sdkKey("b")},
		},
		{
			name:        "value change only adds the new value",
			desired:     []*svcapitypes.Tag{tag("a", "2")},
			latest:      []*svcapitypes.Tag{tag("a", "1")},
			wantCreated: []svcsdktypes.Tag{sdkTag("a", "2")},
			wantDeleted: []svcsdktypes.Tag{},
		},
		{
			name: "nil tags and nil keys are skipped without panicking",
			desired: []*svcapitypes.Tag{
				nil,
				{Value: nil},
				tag("a", "1"),
			},
			latest: []*svcapitypes.Tag{
				nil,
				{Value: nil},
			},
			wantCreated: []svcsdktypes.Tag{sdkTag("a", "1")},
			wantDeleted: []svcsdktypes.Tag{},
		},
		{
			name:        "no-op when desired and latest are both nil",
			desired:     nil,
			latest:      nil,
			wantCreated: []svcsdktypes.Tag{},
			wantDeleted: []svcsdktypes.Tag{},
		},
		{
			name:        "nil desired removes all latest tags",
			desired:     nil,
			latest:      []*svcapitypes.Tag{tag("k", "v")},
			wantCreated: []svcsdktypes.Tag{},
			wantDeleted: []svcsdktypes.Tag{sdkKey("k")},
		},
		{
			name:        "simultaneous add and remove",
			desired:     []*svcapitypes.Tag{tag("env", "prod"), tag("new", "y")},
			latest:      []*svcapitypes.Tag{tag("env", "prod"), tag("old", "x")},
			wantCreated: []svcsdktypes.Tag{sdkTag("new", "y")},
			wantDeleted: []svcsdktypes.Tag{sdkKey("old")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &fakeTagsClient{}
			err := Sync(
				context.Background(), client, fakeMetricsRecorder{},
				"i-0123456789abcdef0", tt.desired, tt.latest,
			)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantCreated, createdTags(client))
			assert.Equal(t, tt.wantDeleted, deletedTags(client))
		})
	}
}

func TestEqual(t *testing.T) {
	t.Run("same key, different value is not equal", func(t *testing.T) {
		a := []*svcapitypes.Tag{{Key: aws.String("k"), Value: aws.String("old")}}
		b := []*svcapitypes.Tag{{Key: aws.String("k"), Value: aws.String("new")}}
		assert.False(t, Equal(a, b))
	})
	t.Run("identical tags are equal", func(t *testing.T) {
		a := []*svcapitypes.Tag{{Key: aws.String("k"), Value: aws.String("v")}}
		b := []*svcapitypes.Tag{{Key: aws.String("k"), Value: aws.String("v")}}
		assert.True(t, Equal(a, b))
	})
	t.Run("nil entries are ignored and do not panic", func(t *testing.T) {
		a := []*svcapitypes.Tag{nil, {Key: aws.String("k"), Value: aws.String("v")}}
		b := []*svcapitypes.Tag{{Key: aws.String("k"), Value: aws.String("v")}}
		assert.NotPanics(t, func() {
			assert.True(t, Equal(a, b))
		})
	})
	t.Run("both nil is equal", func(t *testing.T) {
		assert.True(t, Equal(nil, nil))
	})
	t.Run("empty vs non-empty is not equal", func(t *testing.T) {
		b := []*svcapitypes.Tag{{Key: aws.String("k"), Value: aws.String("v")}}
		assert.False(t, Equal(nil, b))
		assert.False(t, Equal(b, nil))
	})
	t.Run("same tags different order is equal", func(t *testing.T) {
		a := []*svcapitypes.Tag{
			{Key: aws.String("k1"), Value: aws.String("a")},
			{Key: aws.String("k2"), Value: aws.String("b")},
		}
		b := []*svcapitypes.Tag{
			{Key: aws.String("k2"), Value: aws.String("b")},
			{Key: aws.String("k1"), Value: aws.String("a")},
		}
		assert.True(t, Equal(a, b))
	})
	t.Run("entry with nil key is ignored", func(t *testing.T) {
		a := []*svcapitypes.Tag{{Value: aws.String("orphan")}, {Key: aws.String("k"), Value: aws.String("v")}}
		b := []*svcapitypes.Tag{{Key: aws.String("k"), Value: aws.String("v")}}
		assert.True(t, Equal(a, b))
	})
	t.Run("nil value equals empty string", func(t *testing.T) {
		a := []*svcapitypes.Tag{{Key: aws.String("k")}}
		b := []*svcapitypes.Tag{{Key: aws.String("k"), Value: aws.String("")}}
		assert.True(t, Equal(a, b))
		assert.False(t, Equal(a, []*svcapitypes.Tag{{Key: aws.String("k"), Value: aws.String("x")}}))
	})
}

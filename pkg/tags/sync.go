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

	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	svcsdk "github.com/aws/aws-sdk-go-v2/service/ec2"
	svcsdktypes "github.com/aws/aws-sdk-go-v2/service/ec2/types"

	svcapitypes "github.com/aws-controllers-k8s/ec2-controller/apis/v1alpha1"
)

// TODO(a-hilaly) most of the utility in this package should ideally go to
// ack runtime repository.

type metricsRecorder interface {
	RecordAPICall(opType string, opID string, err error)
}

type tagsClient interface {
	CreateTags(context.Context, *svcsdk.CreateTagsInput, ...func(*svcsdk.Options)) (*svcsdk.CreateTagsOutput, error)
	DescribeTags(context.Context, *svcsdk.DescribeTagsInput, ...func(*svcsdk.Options)) (*svcsdk.DescribeTagsOutput, error)
	DeleteTags(context.Context, *svcsdk.DeleteTagsInput, ...func(*svcsdk.Options)) (*svcsdk.DeleteTagsOutput, error)
}

// Sync is responsible of taking two arrays of tags (desired and latest), comparing
// them, then making the appropriate APIs calls to up change the latest state into
// the desired state.
func Sync(
	ctx context.Context,
	client tagsClient,
	mr metricsRecorder,
	resourceID string,
	latestTags []*svcapitypes.Tag,
	desiredTags []*svcapitypes.Tag,
) error {
	var err error
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("common.Sync")
	defer func() { exit(err) }()

	addedOrUpdated, removed := ComputeTagsDelta(latestTags, desiredTags)

	if len(removed) > 0 {
		_, err = client.DeleteTags(
			ctx,
			&svcsdk.DeleteTagsInput{
				Resources: []string{resourceID},
				Tags:      sdkTagsFromResourceTags(removed),
			},
		)
		mr.RecordAPICall("UPDATE", "DeleteTags", err)
		if err != nil {
			return err
		}
	}

	if len(addedOrUpdated) > 0 {
		_, err = client.CreateTags(
			ctx,
			&svcsdk.CreateTagsInput{
				Resources: []string{resourceID},
				Tags:      sdkTagsFromResourceTags(addedOrUpdated),
			},
		)
		mr.RecordAPICall("UPDATE", "CreateTags", err)
		if err != nil {
			return err
		}
	}
	return nil
}

// computeTagsDelta compares two Tag arrays and return two different list
// containing the addedOrupdated and removed tags. The removed tags array
// only contains the tags Keys.
func ComputeTagsDelta(
	desired []*svcapitypes.Tag,
	latest []*svcapitypes.Tag,
) (toAdd []*svcapitypes.Tag, toDelete []*svcapitypes.Tag) {
	desiredTags := map[string]string{}
	for _, tag := range desired {
		desiredTags[safeString(tag.Key)] = safeString(tag.Value)
	}

	latestTags := map[string]string{}
	for _, tag := range latest {
		latestTags[safeString(tag.Key)] = safeString(tag.Value)
	}

	for _, tag := range desired {
		val, ok := latestTags[safeString(tag.Key)]
		if !ok || val != safeString(tag.Value) {
			toAdd = append(toAdd, tag)
		}
	}

	for _, tag := range latest {
		_, ok := desiredTags[safeString(tag.Key)]
		if !ok {
			toDelete = append(toDelete, tag)
		}
	}

	return toAdd, toDelete
}

// svcTagsFromResourceTags transforms a *svcapitypes.Tag array to a *svcsdk.Tag array.
func sdkTagsFromResourceTags(rTags []*svcapitypes.Tag) []svcsdktypes.Tag {
	tags := make([]svcsdktypes.Tag, len(rTags))
	for i := range rTags {
		if rTags[i] != nil {
			tags[i] = svcsdktypes.Tag{
				Key:   rTags[i].Key,
				Value: rTags[i].Value,
			}
		}
	}
	return tags
}

func equalStrings(a, b *string) bool {
	if a == nil {
		return b == nil || *b == ""
	}

	if a != nil && b == nil {
		return false
	}

	return (*a == "" && b == nil) || *a == *b
}

func safeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

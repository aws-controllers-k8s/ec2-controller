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
	"maps"
	"slices"

	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	acktags "github.com/aws-controllers-k8s/runtime/pkg/tags"
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

// Sync compares two arrays of tags, then makes the appropriate API calls to
// update the latest state into the desired state.
//
// desiredTags is the target state (typically taken from the resource spec).
// latestTags is the current observed state (typically read from AWS).
//
// Argument order matters: because both parameters share the same type,
// swapping them compiles cleanly but silently reverses the sync direction
// (desired tags get deleted and stale tags get re-created).
//
// latestTags is passed as from and desiredTags as to to GetTagsDifference;
// swapping them would reverse added and removed.
func Sync(
	ctx context.Context,
	client tagsClient,
	mr metricsRecorder,
	resourceID string,
	desiredTags []*svcapitypes.Tag,
	latestTags []*svcapitypes.Tag,
) error {
	var err error
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("common.Sync")
	defer func() { exit(err) }()

	from := toACKTags(latestTags)
	to := toACKTags(desiredTags)
	// from=latest, to=desired. Per ackcompare.GetTagsDifference
	// (github.com/aws-controllers-k8s/runtime/pkg/compare): keys only in from
	// are returned in removed; keys only in to, and keys whose value differs,
	// are returned in added only, never in removed (upsert semantics, added
	// in runtime v0.58.0 via aws-controllers-k8s/runtime#227; go.mod pins v0.67.0).
	// The middle unchanged result is not needed. CreateTags overwrites existing
	// values, so a changed value never needs a DeleteTags call.
	added, _, removed := ackcompare.GetTagsDifference(from, to)

	if len(removed) > 0 {
		_, err = client.DeleteTags(
			ctx,
			&svcsdk.DeleteTagsInput{
				Resources: []string{resourceID},
				Tags:      sdkTagsKeysFromACKTags(removed),
			},
		)
		mr.RecordAPICall("UPDATE", "DeleteTags", err)
		if err != nil {
			return err
		}
	}

	if len(added) > 0 {
		_, err = client.CreateTags(
			ctx,
			&svcsdk.CreateTagsInput{
				Resources: []string{resourceID},
				Tags:      sdkTagsFromACKTags(added),
			},
		)
		mr.RecordAPICall("UPDATE", "CreateTags", err)
		if err != nil {
			return err
		}
	}
	return nil
}

// Equal reports whether a and b contain the same tag keys and values.
// Slice order is ignored. Nil entries and entries with a nil Key are skipped,
// and a nil Value is treated as "" (see toACKTags). The comparison uses
// ackcompare.GetTagsDifference; with its upsert semantics (runtime >= v0.58.0,
// aws-controllers-k8s/runtime#227) a value change for an existing key shows up
// in added, so checking added and removed is enough.
func Equal(a, b []*svcapitypes.Tag) bool {
	added, _, removed := ackcompare.GetTagsDifference(toACKTags(a), toACKTags(b))
	return len(added) == 0 && len(removed) == 0
}

// toACKTags transforms a *svcapitypes.Tag array into an acktags.Tags map.
// Tags with a nil key are skipped.
func toACKTags(tags []*svcapitypes.Tag) acktags.Tags {
	m := make(acktags.Tags)
	for _, t := range tags {
		if t == nil || t.Key == nil {
			continue
		}
		v := ""
		if t.Value != nil {
			v = *t.Value
		}
		m[*t.Key] = v
	}
	return m
}

// sdkTagsFromACKTags transforms an acktags.Tags map into a sorted *svcsdk.Tag array.
func sdkTagsFromACKTags(tags acktags.Tags) []svcsdktypes.Tag {
	result := make([]svcsdktypes.Tag, 0, len(tags))
	for _, k := range slices.Sorted(maps.Keys(tags)) {
		v := tags[k]
		result = append(result, svcsdktypes.Tag{Key: &k, Value: &v})
	}
	return result
}

// sdkTagsKeysFromACKTags transforms the keys of an acktags.Tags map into a sorted
// *svcsdk.Tag array. Only the keys are set, as DeleteTags does not need values.
func sdkTagsKeysFromACKTags(tags acktags.Tags) []svcsdktypes.Tag {
	result := make([]svcsdktypes.Tag, 0, len(tags))
	for _, k := range slices.Sorted(maps.Keys(tags)) {
		result = append(result, svcsdktypes.Tag{Key: &k})
	}
	return result
}

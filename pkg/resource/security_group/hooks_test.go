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

package security_group

import (
	"testing"

	svcapitypes "github.com/aws-controllers-k8s/ec2-controller/apis/v1alpha1"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/assert"
)

const (
	testSelfID    = "sg-self"
	testOtherID   = "sg-other"
	testAccountID = "111122223333"
	testSelfName  = "self-sg-name"
)

func pair(groupID, userID, groupName *string) *svcapitypes.UserIDGroupPair {
	return &svcapitypes.UserIDGroupPair{
		GroupID:   groupID,
		UserID:    userID,
		GroupName: groupName,
	}
}

func ruleWithPairs(pairs ...*svcapitypes.UserIDGroupPair) *svcapitypes.IPPermission {
	return &svcapitypes.IPPermission{UserIDGroupPairs: pairs}
}

func mkResource(ingress, egress []*svcapitypes.IPPermission) *resource {
	return &resource{
		ko: &svcapitypes.SecurityGroup{
			Spec: svcapitypes.SecurityGroupSpec{
				IngressRules: ingress,
				EgressRules:  egress,
			},
			Status: svcapitypes.SecurityGroupStatus{
				ID: aws.String(testSelfID),
			},
		},
	}
}

func TestNormalizeSelfRefRules(t *testing.T) {
	t.Run("nil resource returns nil", func(t *testing.T) {
		assert.Nil(t, normalizeSelfRefRules(nil))
	})

	t.Run("nil status ID returns a copy untouched", func(t *testing.T) {
		r := mkResource(
			[]*svcapitypes.IPPermission{ruleWithPairs(pair(aws.String(testSelfID), nil, nil))},
			nil,
		)
		r.ko.Status.ID = nil
		got := normalizeSelfRefRules(r)
		assert.NotSame(t, r, got, "must return a distinct *resource")
		assert.Equal(t, testSelfID, *got.ko.Spec.IngressRules[0].UserIDGroupPairs[0].GroupID,
			"GroupID must not be touched when Status.ID is nil")
	})

	t.Run("does not mutate the input resource (deep-copy guarantee)", func(t *testing.T) {
		in := mkResource(
			[]*svcapitypes.IPPermission{ruleWithPairs(
				pair(aws.String(testSelfID), aws.String(testAccountID), aws.String(testSelfName)),
			)},
			nil,
		)
		_ = normalizeSelfRefRules(in)
		got := in.ko.Spec.IngressRules[0].UserIDGroupPairs[0]
		assert.Equal(t, testSelfID, *got.GroupID, "input GroupID must be untouched")
		assert.Equal(t, testAccountID, *got.UserID, "input UserID must be untouched")
		assert.Equal(t, testSelfName, *got.GroupName, "input GroupName must be untouched")
	})

	t.Run("strips server-fillable fields on self-ref ingress pair", func(t *testing.T) {
		r := mkResource(
			[]*svcapitypes.IPPermission{ruleWithPairs(
				pair(aws.String(testSelfID), aws.String(testAccountID), aws.String(testSelfName)),
			)},
			nil,
		)
		got := normalizeSelfRefRules(r).ko.Spec.IngressRules[0].UserIDGroupPairs[0]
		assert.Nil(t, got.GroupID)
		assert.Nil(t, got.UserID)
		assert.Nil(t, got.GroupName)
	})

	t.Run("strips server-fillable fields on self-ref egress pair", func(t *testing.T) {
		r := mkResource(
			nil,
			[]*svcapitypes.IPPermission{ruleWithPairs(
				pair(aws.String(testSelfID), aws.String(testAccountID), aws.String(testSelfName)),
			)},
		)
		got := normalizeSelfRefRules(r).ko.Spec.EgressRules[0].UserIDGroupPairs[0]
		assert.Nil(t, got.GroupID)
		assert.Nil(t, got.UserID)
		assert.Nil(t, got.GroupName)
	})

	t.Run("preserves cross-SG pair untouched", func(t *testing.T) {
		r := mkResource(
			[]*svcapitypes.IPPermission{ruleWithPairs(
				pair(aws.String(testOtherID), aws.String(testAccountID), nil),
			)},
			nil,
		)
		got := normalizeSelfRefRules(r).ko.Spec.IngressRules[0].UserIDGroupPairs[0]
		assert.Equal(t, testOtherID, *got.GroupID)
		assert.Equal(t, testAccountID, *got.UserID)
	})

	t.Run("nil GroupID counts as self-ref and clears UserID/GroupName", func(t *testing.T) {
		r := mkResource(
			[]*svcapitypes.IPPermission{ruleWithPairs(
				pair(nil, aws.String(testAccountID), aws.String(testSelfName)),
			)},
			nil,
		)
		got := normalizeSelfRefRules(r).ko.Spec.IngressRules[0].UserIDGroupPairs[0]
		assert.Nil(t, got.GroupID)
		assert.Nil(t, got.UserID)
		assert.Nil(t, got.GroupName)
	})

	t.Run("mixed pair: only self stripped, cross-SG kept", func(t *testing.T) {
		r := mkResource(
			[]*svcapitypes.IPPermission{ruleWithPairs(
				pair(aws.String(testSelfID), aws.String(testAccountID), aws.String(testSelfName)),
				pair(aws.String(testOtherID), aws.String(testAccountID), nil),
			)},
			nil,
		)
		pairs := normalizeSelfRefRules(r).ko.Spec.IngressRules[0].UserIDGroupPairs
		assert.Nil(t, pairs[0].GroupID)
		assert.Nil(t, pairs[0].UserID)
		assert.Equal(t, testOtherID, *pairs[1].GroupID)
		assert.Equal(t, testAccountID, *pairs[1].UserID)
	})

	t.Run("handles nil rule and nil pair entries", func(t *testing.T) {
		r := mkResource(
			[]*svcapitypes.IPPermission{
				nil,
				{UserIDGroupPairs: []*svcapitypes.UserIDGroupPair{
					nil,
					pair(aws.String(testSelfID), aws.String(testAccountID), nil),
				}},
			},
			nil,
		)
		var got *resource
		assert.NotPanics(t, func() { got = normalizeSelfRefRules(r) })
		p := got.ko.Spec.IngressRules[1].UserIDGroupPairs[1]
		assert.Nil(t, p.GroupID)
		assert.Nil(t, p.UserID)
	})

	t.Run("preserves description on self-ref pair", func(t *testing.T) {
		desc := aws.String("self-ref, description present")
		r := mkResource(
			[]*svcapitypes.IPPermission{ruleWithPairs(&svcapitypes.UserIDGroupPair{
				Description: desc,
				GroupID:     aws.String(testSelfID),
				UserID:      aws.String(testAccountID),
			})},
			nil,
		)
		got := normalizeSelfRefRules(r).ko.Spec.IngressRules[0].UserIDGroupPairs[0]
		assert.Nil(t, got.GroupID)
		assert.Nil(t, got.UserID)
		assert.NotNil(t, got.Description)
		assert.Equal(t, *desc, *got.Description)
	})
}

// TestContainsRule_SelfRef_AfterNormalisation locks in the actual fix:
// after normalisation, a desired rule expressed in canonical "GroupID
// omitted" form must be recognised as already present in the latest set
// when AWS auto-filled GroupID = self-SG-id and UserID = account-id on
// read-back. Without normalisation, containsRule returns false and
// syncSGRules schedules a Revoke + Authorize on every reconcile.
func TestContainsRule_SelfRef_AfterNormalisation(t *testing.T) {
	desc := aws.String("self-ref, description present, no userID, no groupID")
	desiredRule := ruleWithPairs(&svcapitypes.UserIDGroupPair{Description: desc})
	latestRule := ruleWithPairs(&svcapitypes.UserIDGroupPair{
		Description: desc,
		GroupID:     aws.String(testSelfID),
		UserID:      aws.String(testAccountID),
	})

	// Pre-condition: without normalisation the two are NOT equal.
	assert.False(t, containsRule([]*svcapitypes.IPPermission{latestRule}, desiredRule),
		"pre-condition: raw self-ref desired must not match AWS-filled latest")

	desired := normalizeSelfRefRules(mkResource(
		[]*svcapitypes.IPPermission{desiredRule}, nil,
	))
	latest := normalizeSelfRefRules(mkResource(
		[]*svcapitypes.IPPermission{latestRule}, nil,
	))

	assert.True(t,
		containsRule(latest.ko.Spec.IngressRules, desired.ko.Spec.IngressRules[0]),
		"after normalisation, desired self-ref must match latest self-ref",
	)
	assert.True(t,
		containsRule(desired.ko.Spec.IngressRules, latest.ko.Spec.IngressRules[0]),
		"match must be symmetric",
	)
}

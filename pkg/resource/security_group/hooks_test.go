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
	t.Run("nil resource does not panic", func(t *testing.T) {
		assert.NotPanics(t, func() { normalizeSelfRefRules(nil) })
	})

	t.Run("nil status ID is a no-op", func(t *testing.T) {
		r := mkResource(
			[]*svcapitypes.IPPermission{ruleWithPairs(pair(aws.String(testSelfID), nil, nil))},
			nil,
		)
		r.ko.Status.ID = nil
		normalizeSelfRefRules(r)
		assert.Equal(t, testSelfID, *r.ko.Spec.IngressRules[0].UserIDGroupPairs[0].GroupID,
			"GroupID must not be touched when Status.ID is nil")
	})

	t.Run("clears UserID/GroupName but preserves GroupID on self-ref ingress pair", func(t *testing.T) {
		r := mkResource(
			[]*svcapitypes.IPPermission{ruleWithPairs(
				pair(aws.String(testSelfID), aws.String(testAccountID), aws.String(testSelfName)),
			)},
			nil,
		)
		normalizeSelfRefRules(r)
		got := r.ko.Spec.IngressRules[0].UserIDGroupPairs[0]
		assert.Equal(t, testSelfID, *got.GroupID,
			"GroupID must be preserved so referencesResolved still gates syncSGRules open")
		assert.Nil(t, got.UserID)
		assert.Nil(t, got.GroupName)
	})

	t.Run("clears UserID/GroupName but preserves GroupID on self-ref egress pair", func(t *testing.T) {
		r := mkResource(
			nil,
			[]*svcapitypes.IPPermission{ruleWithPairs(
				pair(aws.String(testSelfID), aws.String(testAccountID), aws.String(testSelfName)),
			)},
		)
		normalizeSelfRefRules(r)
		got := r.ko.Spec.EgressRules[0].UserIDGroupPairs[0]
		assert.Equal(t, testSelfID, *got.GroupID)
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
		normalizeSelfRefRules(r)
		got := r.ko.Spec.IngressRules[0].UserIDGroupPairs[0]
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
		normalizeSelfRefRules(r)
		got := r.ko.Spec.IngressRules[0].UserIDGroupPairs[0]
		assert.Nil(t, got.GroupID, "a nil GroupID stays nil (nothing to preserve)")
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
		normalizeSelfRefRules(r)
		pairs := r.ko.Spec.IngressRules[0].UserIDGroupPairs
		assert.Equal(t, testSelfID, *pairs[0].GroupID)
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
		assert.NotPanics(t, func() { normalizeSelfRefRules(r) })
		p := r.ko.Spec.IngressRules[1].UserIDGroupPairs[1]
		assert.Equal(t, testSelfID, *p.GroupID)
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
		normalizeSelfRefRules(r)
		got := r.ko.Spec.IngressRules[0].UserIDGroupPairs[0]
		assert.Equal(t, testSelfID, *got.GroupID)
		assert.Nil(t, got.UserID)
		assert.NotNil(t, got.Description)
		assert.Equal(t, *desc, *got.Description)
	})
}

// TestCustomPreCompare_SelfRef_SuppressesDelta locks in the runtime-level
// fix: newResourceDelta must NOT flag Spec.IngressRules / Spec.EgressRules
// as different when the only divergence is server-fill on self-referencing
// pairs. customPreCompare (wired in via the delta_pre_compare hook in
// generator.yaml) normalises both sides before the generated DeepEqual.
//
// The desired pairs carry GroupID=testSelfID to mirror the real reconcile:
// ACK ResolveReferences populates GroupID from GroupRef before the delta is
// computed (this is also what referencesResolved relies on). AWS additionally
// fills UserID/GroupName on read-back (latest); those are the only fields
// that must be normalised away.
func TestCustomPreCompare_SelfRef_SuppressesDelta(t *testing.T) {
	desc := aws.String("self-ref TCP/53")
	desired := mkResource(
		[]*svcapitypes.IPPermission{ruleWithPairs(&svcapitypes.UserIDGroupPair{
			Description: desc,
			GroupID:     aws.String(testSelfID),
		})},
		[]*svcapitypes.IPPermission{ruleWithPairs(&svcapitypes.UserIDGroupPair{
			Description: desc,
			GroupID:     aws.String(testSelfID),
		})},
	)
	latest := mkResource(
		[]*svcapitypes.IPPermission{ruleWithPairs(&svcapitypes.UserIDGroupPair{
			Description: desc,
			GroupID:     aws.String(testSelfID),
			UserID:      aws.String(testAccountID),
			GroupName:   aws.String(testSelfName),
		})},
		[]*svcapitypes.IPPermission{ruleWithPairs(&svcapitypes.UserIDGroupPair{
			Description: desc,
			GroupID:     aws.String(testSelfID),
			UserID:      aws.String(testAccountID),
			GroupName:   aws.String(testSelfName),
		})},
	)

	delta := newResourceDelta(desired, latest)

	assert.False(t, delta.DifferentAt("Spec.IngressRules"),
		"self-ref ingress must not appear in delta after normalisation")
	assert.False(t, delta.DifferentAt("Spec.EgressRules"),
		"self-ref egress must not appear in delta after normalisation")
	assert.Empty(t, delta.Differences,
		"no other field changed; delta must be empty")
}

// TestCustomPreCompare_RealDiff_StillFires confirms that legitimate rule
// changes (e.g. port edits) are still surfaced after normalisation.
func TestCustomPreCompare_RealDiff_StillFires(t *testing.T) {
	desc := aws.String("self-ref")
	desired := mkResource(
		[]*svcapitypes.IPPermission{{
			FromPort:   aws.Int64(53),
			ToPort:     aws.Int64(53),
			IPProtocol: aws.String("tcp"),
			UserIDGroupPairs: []*svcapitypes.UserIDGroupPair{{
				Description: desc,
				GroupID:     aws.String(testSelfID),
			}},
		}},
		nil,
	)
	latest := mkResource(
		[]*svcapitypes.IPPermission{{
			FromPort:   aws.Int64(80), // <- changed
			ToPort:     aws.Int64(80),
			IPProtocol: aws.String("tcp"),
			UserIDGroupPairs: []*svcapitypes.UserIDGroupPair{{
				Description: desc,
				GroupID:     aws.String(testSelfID),
				UserID:      aws.String(testAccountID),
			}},
		}},
		nil,
	)

	delta := newResourceDelta(desired, latest)

	assert.True(t, delta.DifferentAt("Spec.IngressRules"),
		"a real port change must still produce a Spec.IngressRules diff")
}

// TestCustomPreCompare_CrossSGRef_NotSuppressed confirms scope: the fix is
// limited to self-references. A pair pointing at a *different* SG with the
// AWS-filled UserID missing on desired must still produce a delta, since
// cross-SG normalisation is explicitly out of scope.
func TestCustomPreCompare_CrossSGRef_NotSuppressed(t *testing.T) {
	desired := mkResource(
		[]*svcapitypes.IPPermission{ruleWithPairs(
			pair(aws.String(testOtherID), nil, nil),
		)},
		nil,
	)
	latest := mkResource(
		[]*svcapitypes.IPPermission{ruleWithPairs(
			pair(aws.String(testOtherID), aws.String(testAccountID), nil),
		)},
		nil,
	)

	delta := newResourceDelta(desired, latest)

	assert.True(t, delta.DifferentAt("Spec.IngressRules"),
		"cross-SG ref with server-filled UserID must still produce a diff (out of scope for this fix)")
}

// TestCustomPreCompare_Mutates_a_and_b documents that customPreCompare
// is intentionally allowed to mutate its inputs in place, matching the
// convention used by RouteTable, NetworkAcl, and VPC.
//
// Only UserID and GroupName are cleared. GroupID is preserved: the same
// desired object flows into customUpdateSecurityGroup, whose referencesResolved
// gate treats a pair with GroupRef set but GroupID nil as unresolved and skips
// syncSGRules. Clearing GroupID here would therefore permanently block creation
// of the self-ref rule (regression caught by the e2e perpetual-diff test).
func TestCustomPreCompare_Mutates_a_and_b(t *testing.T) {
	a := mkResource(
		[]*svcapitypes.IPPermission{ruleWithPairs(
			pair(aws.String(testSelfID), aws.String(testAccountID), aws.String(testSelfName)),
		)},
		nil,
	)
	b := mkResource(
		[]*svcapitypes.IPPermission{ruleWithPairs(
			pair(aws.String(testSelfID), aws.String(testAccountID), aws.String(testSelfName)),
		)},
		nil,
	)

	_ = newResourceDelta(a, b)

	aPair := a.ko.Spec.IngressRules[0].UserIDGroupPairs[0]
	bPair := b.ko.Spec.IngressRules[0].UserIDGroupPairs[0]
	assert.Equal(t, testSelfID, *aPair.GroupID, "a self-ref GroupID must be preserved in place")
	assert.Nil(t, aPair.UserID, "a self-ref UserID must be cleared in place")
	assert.Nil(t, aPair.GroupName, "a self-ref GroupName must be cleared in place")
	assert.Equal(t, testSelfID, *bPair.GroupID, "b self-ref GroupID must be preserved in place")
	assert.Nil(t, bPair.UserID, "b self-ref UserID must be cleared in place")
	assert.Nil(t, bPair.GroupName, "b self-ref GroupName must be cleared in place")
}

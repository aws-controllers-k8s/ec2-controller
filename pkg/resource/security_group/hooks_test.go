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

	ackv1alpha1 "github.com/aws-controllers-k8s/runtime/apis/core/v1alpha1"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/assert"

	svcapitypes "github.com/aws-controllers-k8s/ec2-controller/apis/v1alpha1"
)

const (
	testSelfID      = "sg-self"
	testOtherID     = "sg-other"
	testOwnerAcctID = "111122223333"
	testPeerAcctID  = "999988887777"
	testSelfName    = "self-sg-name"
)

func groupRef(name string) *ackv1alpha1.AWSResourceReferenceWrapper {
	return &ackv1alpha1.AWSResourceReferenceWrapper{
		From: &ackv1alpha1.AWSResourceReference{Name: aws.String(name)},
	}
}

func ruleWithPairs(pairs ...*svcapitypes.UserIDGroupPair) *svcapitypes.IPPermission {
	return &svcapitypes.IPPermission{UserIDGroupPairs: pairs}
}

// mkResource builds a resource with Status.ID = testSelfID and
// Status.ACKResourceMetadata.OwnerAccountID = testOwnerAcctID, mirroring the
// values the runtime populates on both desired and latest.
func mkResource(ingress, egress []*svcapitypes.IPPermission) *resource {
	return &resource{
		ko: &svcapitypes.SecurityGroup{
			Spec: svcapitypes.SecurityGroupSpec{
				IngressRules: ingress,
				EgressRules:  egress,
			},
			Status: svcapitypes.SecurityGroupStatus{
				ID: aws.String(testSelfID),
				ACKResourceMetadata: &ackv1alpha1.ResourceMetadata{
					OwnerAccountID: (*ackv1alpha1.AWSAccountID)(aws.String(testOwnerAcctID)),
				},
			},
		},
	}
}

// -----------------------------------------------------------------------------
// canonicalizeGroupPair
// -----------------------------------------------------------------------------

func TestCanonicalizeGroupPair(t *testing.T) {
	t.Run("nil pair does not panic", func(t *testing.T) {
		assert.NotPanics(t, func() { canonicalizeGroupPair(nil, testSelfID, testOwnerAcctID) })
	})

	t.Run("self-ref by explicit self GroupID: keep selfID, clear the rest", func(t *testing.T) {
		p := &svcapitypes.UserIDGroupPair{
			GroupID:   aws.String(testSelfID),
			UserID:    aws.String(testOwnerAcctID),
			GroupName: aws.String(testSelfName),
		}
		canonicalizeGroupPair(p, testSelfID, testOwnerAcctID)
		assert.Equal(t, testSelfID, *p.GroupID)
		assert.Nil(t, p.UserID)
		assert.Nil(t, p.GroupName)
		assert.Nil(t, p.GroupRef)
	})

	t.Run("self-ref by omission: GroupID canonicalised up to selfID", func(t *testing.T) {
		p := &svcapitypes.UserIDGroupPair{
			Description: aws.String("coredns"),
			UserID:      aws.String(testOwnerAcctID),
		}
		canonicalizeGroupPair(p, testSelfID, testOwnerAcctID)
		// This is the #2822 fix: the omitted GroupID must become selfID so
		// it matches the AWS-filled latest side.
		assert.NotNil(t, p.GroupID)
		assert.Equal(t, testSelfID, *p.GroupID)
		assert.Nil(t, p.UserID)
		assert.NotNil(t, p.Description, "description is preserved")
	})

	t.Run("self-ref by groupRef: clear GroupRef, keep selfID", func(t *testing.T) {
		p := &svcapitypes.UserIDGroupPair{
			GroupID:  aws.String(testSelfID),
			GroupRef: groupRef("myself"),
		}
		canonicalizeGroupPair(p, testSelfID, testOwnerAcctID)
		assert.Equal(t, testSelfID, *p.GroupID)
		assert.Nil(t, p.GroupRef, "GroupRef must be cleared to match AWS read-back form")
	})

	t.Run("GroupRef is cleared even on a resolved cross-SG pair", func(t *testing.T) {
		// A groupRef pointing at another SG resolves to that SG's GroupID.
		// The wrapper is stripped; the resolved GroupID is authoritative.
		p := &svcapitypes.UserIDGroupPair{
			GroupID:  aws.String(testOtherID),
			GroupRef: groupRef("other"),
		}
		canonicalizeGroupPair(p, testSelfID, testOwnerAcctID)
		assert.Nil(t, p.GroupRef, "GroupRef must never survive into the delta")
		assert.Equal(t, testOtherID, *p.GroupID, "resolved GroupID is preserved")
	})

	t.Run("groupRef with an unresolved (nil) GroupID canonicalises to self", func(t *testing.T) {
		// Accepted tradeoff: the delta assumes GroupID is resolved. A pair
		// that still carries a groupRef with no resolved GroupID is stripped
		// of the wrapper and, being otherwise identifier-less, treated as a
		// self-reference. The referencesResolved guard is what actually keeps
		// an unresolved reference from being authorised prematurely; this
		// function only shapes the delta.
		p := &svcapitypes.UserIDGroupPair{GroupRef: groupRef("not-yet-created")}
		canonicalizeGroupPair(p, testSelfID, testOwnerAcctID)
		assert.Nil(t, p.GroupRef)
		assert.Equal(t, testSelfID, *p.GroupID)
	})

	t.Run("cross-SG same-account: clear owner UserID, keep GroupID", func(t *testing.T) {
		p := &svcapitypes.UserIDGroupPair{
			GroupID: aws.String(testOtherID),
			UserID:  aws.String(testOwnerAcctID),
		}
		canonicalizeGroupPair(p, testSelfID, testOwnerAcctID)
		assert.Equal(t, testOtherID, *p.GroupID)
		assert.Nil(t, p.UserID, "server-filled owner account cleared")
	})

	t.Run("cross-account: preserve UserID and GroupID", func(t *testing.T) {
		p := &svcapitypes.UserIDGroupPair{
			GroupID: aws.String(testOtherID),
			UserID:  aws.String(testPeerAcctID),
		}
		canonicalizeGroupPair(p, testSelfID, testOwnerAcctID)
		assert.Equal(t, testOtherID, *p.GroupID)
		assert.Equal(t, testPeerAcctID, *p.UserID, "cross-account UserID must be preserved")
	})

	t.Run("cross-SG by ref: drop GroupRef once GroupID resolved", func(t *testing.T) {
		p := &svcapitypes.UserIDGroupPair{
			GroupID:  aws.String(testOtherID),
			GroupRef: groupRef("other"),
		}
		canonicalizeGroupPair(p, testSelfID, testOwnerAcctID)
		assert.Equal(t, testOtherID, *p.GroupID)
		assert.Nil(t, p.GroupRef)
	})

	t.Run("VPCRef cleared, VPCID preserved on a cross-SG pair", func(t *testing.T) {
		// VPCRef is a spec-only reference wrapper AWS never returns; VPCID is
		// the concrete value AWS does return on read-back and must survive.
		p := &svcapitypes.UserIDGroupPair{
			GroupID: aws.String(testOtherID),
			VPCID:   aws.String("vpc-123"),
			VPCRef:  groupRef("my-vpc"),
		}
		canonicalizeGroupPair(p, testSelfID, testOwnerAcctID)
		assert.Nil(t, p.VPCRef, "VPCRef must be cleared to match AWS read-back form")
		assert.Equal(t, "vpc-123", *p.VPCID, "concrete VPCID must be preserved")
	})

	t.Run("VPCRef cleared on a self-ref pair", func(t *testing.T) {
		p := &svcapitypes.UserIDGroupPair{
			GroupID: aws.String(testSelfID),
			VPCRef:  groupRef("my-vpc"),
		}
		canonicalizeGroupPair(p, testSelfID, testOwnerAcctID)
		assert.Equal(t, testSelfID, *p.GroupID)
		assert.Nil(t, p.VPCRef, "VPCRef must be cleared on a self-reference too")
	})

	t.Run("by-name reference is not treated as self-ref", func(t *testing.T) {
		p := &svcapitypes.UserIDGroupPair{GroupName: aws.String("some-named-sg")}
		canonicalizeGroupPair(p, testSelfID, testOwnerAcctID)
		assert.Nil(t, p.GroupID, "a by-name pair must not be canonicalised to selfID")
		assert.Equal(t, "some-named-sg", *p.GroupName)
	})

	t.Run("empty selfID (nil Status.ID): omitted pair is not stamped with a group id", func(t *testing.T) {
		// Pre-create / adoption: Status.ID is not yet known. An omitted
		// self-ref pair must not gain a bogus (empty) GroupID and must not
		// panic; the server-fillable fields are still cleared.
		p := &svcapitypes.UserIDGroupPair{
			Description: aws.String("coredns"),
			UserID:      aws.String(testOwnerAcctID),
		}
		canonicalizeGroupPair(p, "", testOwnerAcctID)
		assert.Nil(t, p.GroupID, "must not set a GroupID when selfID is unknown")
		assert.Nil(t, p.UserID)
	})

	t.Run("empty selfID: explicit GroupID pair is left intact (self unknown)", func(t *testing.T) {
		// With no known selfID we cannot classify an explicit-id pair as a
		// self-reference, so it is treated as a cross-SG pair: GroupID kept,
		// only an owner-account UserID stripped.
		p := &svcapitypes.UserIDGroupPair{
			GroupID: aws.String(testSelfID),
			UserID:  aws.String(testOwnerAcctID),
		}
		canonicalizeGroupPair(p, "", testOwnerAcctID)
		assert.Equal(t, testSelfID, *p.GroupID)
		assert.Nil(t, p.UserID)
	})
}

// TestCanonicalizeGroupPair_InconsistentInputs documents what happens to the
// UserID/GroupName fields that the self-ref branch drops when the input is
// inconsistent. Ground truth, verified directly against the EC2 API:
//
//   - {GroupId=self}              -> accepted; AWS auto-fills UserId=owner
//   - {GroupId=self, UserId=owner} -> accepted; identical rule to the above
//   - {GroupId=self, UserId=foreign} -> REJECTED (InvalidGroup.NotFound): a
//     foreign UserId scopes the group lookup to that account, where the SG's
//     own id does not exist
//   - {UserId=foreign} (no group) -> REJECTED (MissingParameter)
//
// So dropping the owner UserId/GroupName on a genuine self-reference is
// lossless, and the only case a foreign UserId is dropped is when
// GroupId==selfID -- an impossible input AWS rejects anyway. A legitimate
// cross-account reference always uses GroupId=peer (!= selfID) and is
// preserved (see TestCustomPostCompare_CrossAccount_NotSuppressed).
func TestCanonicalizeGroupPair_InconsistentInputs(t *testing.T) {
	t.Run("self GroupID + foreign UserID: classified self, foreign UserID dropped", func(t *testing.T) {
		p := &svcapitypes.UserIDGroupPair{
			GroupID: aws.String(testSelfID),
			UserID:  aws.String(testPeerAcctID),
		}
		canonicalizeGroupPair(p, testSelfID, testOwnerAcctID)
		assert.Equal(t, testSelfID, *p.GroupID)
		assert.Nil(t, p.UserID,
			"a foreign UserID paired with the SG's own GroupID is nonsensical "+
				"(AWS rejects it as InvalidGroup.NotFound); dropping it is safe")
	})

	t.Run("omitted group + foreign UserID: classified self, UserID dropped", func(t *testing.T) {
		// {UserID: foreign} with no group is invalid to AWS (MissingParameter).
		p := &svcapitypes.UserIDGroupPair{UserID: aws.String(testPeerAcctID)}
		canonicalizeGroupPair(p, testSelfID, testOwnerAcctID)
		assert.Equal(t, testSelfID, *p.GroupID)
		assert.Nil(t, p.UserID)
	})

	t.Run("peer GroupID + foreign UserID: NOT self, cross-account preserved", func(t *testing.T) {
		p := &svcapitypes.UserIDGroupPair{
			GroupID: aws.String(testOtherID),
			UserID:  aws.String(testPeerAcctID),
		}
		canonicalizeGroupPair(p, testSelfID, testOwnerAcctID)
		assert.Equal(t, testOtherID, *p.GroupID)
		assert.Equal(t, testPeerAcctID, *p.UserID,
			"a legitimate cross-account reference (GroupId=peer) must be preserved")
	})
}

// -----------------------------------------------------------------------------
// canonicalizeRuleList
// -----------------------------------------------------------------------------

func TestCanonicalizeRuleList(t *testing.T) {
	t.Run("nil stays nil (absent == empty)", func(t *testing.T) {
		assert.Nil(t, canonicalizeRuleList(nil, testSelfID, testOwnerAcctID))
	})

	t.Run("drops port range for all-protocol -1 rules", func(t *testing.T) {
		out := canonicalizeRuleList([]*svcapitypes.IPPermission{{
			IPProtocol: aws.String("-1"),
			FromPort:   aws.Int64(0),
			ToPort:     aws.Int64(0),
			IPRanges:   []*svcapitypes.IPRange{{CIDRIP: aws.String("0.0.0.0/0")}},
		}}, testSelfID, testOwnerAcctID)
		assert.Len(t, out, 1)
		assert.Nil(t, out[0].FromPort)
		assert.Nil(t, out[0].ToPort)
	})

	t.Run("maps well-known numeric protocols to names, leaves others as-is", func(t *testing.T) {
		out := canonicalizeRuleList([]*svcapitypes.IPPermission{
			{IPProtocol: aws.String("6"), FromPort: aws.Int64(22), ToPort: aws.Int64(22)},
			{IPProtocol: aws.String("17"), FromPort: aws.Int64(53), ToPort: aws.Int64(53)},
			{IPProtocol: aws.String("1"), FromPort: aws.Int64(-1), ToPort: aws.Int64(-1)},
			{IPProtocol: aws.String("58")},
			{IPProtocol: aws.String("tcp"), FromPort: aws.Int64(80), ToPort: aws.Int64(80)},
			{IPProtocol: aws.String("47")}, // GRE: not well-known, kept as number
		}, testSelfID, testOwnerAcctID)
		got := map[string]bool{}
		for _, r := range out {
			got[*r.IPProtocol] = true
		}
		assert.True(t, got["tcp"], "6 -> tcp")
		assert.True(t, got["udp"], "17 -> udp")
		assert.True(t, got["icmp"], "1 -> icmp")
		assert.True(t, got["icmpv6"], "58 -> icmpv6")
		assert.True(t, got["47"], "unknown protocol number kept as-is")
		assert.False(t, got["6"], "numeric 6 must not survive")
	})

	t.Run("numeric and name forms aggregate together", func(t *testing.T) {
		// "6" and "tcp" on the same port must collapse to one rule.
		out := canonicalizeRuleList([]*svcapitypes.IPPermission{
			{IPProtocol: aws.String("6"), FromPort: aws.Int64(443), ToPort: aws.Int64(443),
				IPRanges: []*svcapitypes.IPRange{{CIDRIP: aws.String("10.0.0.0/20")}}},
			{IPProtocol: aws.String("tcp"), FromPort: aws.Int64(443), ToPort: aws.Int64(443),
				IPRanges: []*svcapitypes.IPRange{{CIDRIP: aws.String("192.168.0.0/16")}}},
		}, testSelfID, testOwnerAcctID)
		assert.Len(t, out, 1, "6 and tcp on the same port are the same protocol")
	})

	t.Run("canonicalizes IPv4 and IPv6 CIDRs to network form", func(t *testing.T) {
		out := canonicalizeRuleList([]*svcapitypes.IPPermission{{
			IPProtocol: aws.String("tcp"), FromPort: aws.Int64(443), ToPort: aws.Int64(443),
			IPRanges:   []*svcapitypes.IPRange{{CIDRIP: aws.String("100.68.0.18/18")}},
			IPv6Ranges: []*svcapitypes.IPv6Range{{CIDRIPv6: aws.String("2001:DB8:abcd:0012::1/64")}},
		}}, testSelfID, testOwnerAcctID)
		assert.Equal(t, "100.68.0.0/18", *out[0].IPRanges[0].CIDRIP)
		assert.Equal(t, "2001:db8:abcd:12::/64", *out[0].IPv6Ranges[0].CIDRIPv6)
	})

	t.Run("non-canonical and canonical CIDR aggregate to one grant shape", func(t *testing.T) {
		// A spec written as "100.68.0.18/18" must match the AWS read-back
		// "100.68.0.0/18" (same rule, same port).
		a := canonicalizeRuleList([]*svcapitypes.IPPermission{{
			IPProtocol: aws.String("tcp"), FromPort: aws.Int64(443), ToPort: aws.Int64(443),
			IPRanges: []*svcapitypes.IPRange{{CIDRIP: aws.String("100.68.0.18/18")}},
		}}, testSelfID, testOwnerAcctID)
		b := canonicalizeRuleList([]*svcapitypes.IPPermission{{
			IPProtocol: aws.String("tcp"), FromPort: aws.Int64(443), ToPort: aws.Int64(443),
			IPRanges: []*svcapitypes.IPRange{{CIDRIP: aws.String("100.68.0.0/18")}},
		}}, testSelfID, testOwnerAcctID)
		assert.Equal(t, *a[0].IPRanges[0].CIDRIP, *b[0].IPRanges[0].CIDRIP)
	})

	t.Run("leaves a malformed CIDR untouched", func(t *testing.T) {
		out := canonicalizeRuleList([]*svcapitypes.IPPermission{{
			IPProtocol: aws.String("tcp"), FromPort: aws.Int64(443), ToPort: aws.Int64(443),
			IPRanges: []*svcapitypes.IPRange{{CIDRIP: aws.String("not-a-cidr")}},
		}}, testSelfID, testOwnerAcctID)
		assert.Equal(t, "not-a-cidr", *out[0].IPRanges[0].CIDRIP)
	})

	t.Run("does not drop ports for icmp (-1 type is meaningful)", func(t *testing.T) {
		out := canonicalizeRuleList([]*svcapitypes.IPPermission{{
			IPProtocol: aws.String("icmp"),
			FromPort:   aws.Int64(-1),
			ToPort:     aws.Int64(-1),
		}}, testSelfID, testOwnerAcctID)
		assert.Len(t, out, 1)
		assert.NotNil(t, out[0].FromPort)
		assert.NotNil(t, out[0].ToPort)
	})

	t.Run("aggregates rules sharing (proto, fromPort, toPort)", func(t *testing.T) {
		out := canonicalizeRuleList([]*svcapitypes.IPPermission{
			{IPProtocol: aws.String("tcp"), FromPort: aws.Int64(443), ToPort: aws.Int64(443),
				IPRanges: []*svcapitypes.IPRange{{CIDRIP: aws.String("10.0.0.0/20")}}},
			{IPProtocol: aws.String("tcp"), FromPort: aws.Int64(443), ToPort: aws.Int64(443),
				IPRanges: []*svcapitypes.IPRange{{CIDRIP: aws.String("192.168.0.0/16")}}},
		}, testSelfID, testOwnerAcctID)
		assert.Len(t, out, 1, "the two rules must merge into one")
		assert.Len(t, out[0].IPRanges, 2)
	})

	t.Run("does not aggregate rules with different ports", func(t *testing.T) {
		out := canonicalizeRuleList([]*svcapitypes.IPPermission{
			{IPProtocol: aws.String("tcp"), FromPort: aws.Int64(80), ToPort: aws.Int64(80)},
			{IPProtocol: aws.String("tcp"), FromPort: aws.Int64(443), ToPort: aws.Int64(443)},
		}, testSelfID, testOwnerAcctID)
		assert.Len(t, out, 2)
	})

	t.Run("produces a deterministic order regardless of input order", func(t *testing.T) {
		in1 := []*svcapitypes.IPPermission{
			{IPProtocol: aws.String("tcp"), FromPort: aws.Int64(443), ToPort: aws.Int64(443)},
			{IPProtocol: aws.String("tcp"), FromPort: aws.Int64(80), ToPort: aws.Int64(80)},
		}
		in2 := []*svcapitypes.IPPermission{
			{IPProtocol: aws.String("tcp"), FromPort: aws.Int64(80), ToPort: aws.Int64(80)},
			{IPProtocol: aws.String("tcp"), FromPort: aws.Int64(443), ToPort: aws.Int64(443)},
		}
		out1 := canonicalizeRuleList(in1, testSelfID, testOwnerAcctID)
		out2 := canonicalizeRuleList(in2, testSelfID, testOwnerAcctID)
		assert.Equal(t, ruleAggregationKey(out1[0]), ruleAggregationKey(out2[0]))
		assert.Equal(t, ruleAggregationKey(out1[1]), ruleAggregationKey(out2[1]))
	})

	t.Run("does not alias caller grant slices when aggregating", func(t *testing.T) {
		orig := &svcapitypes.IPPermission{
			IPProtocol: aws.String("tcp"), FromPort: aws.Int64(443), ToPort: aws.Int64(443),
			IPRanges: []*svcapitypes.IPRange{{CIDRIP: aws.String("10.0.0.0/20")}},
		}
		_ = canonicalizeRuleList([]*svcapitypes.IPPermission{
			orig,
			{IPProtocol: aws.String("tcp"), FromPort: aws.Int64(443), ToPort: aws.Int64(443),
				IPRanges: []*svcapitypes.IPRange{{CIDRIP: aws.String("192.168.0.0/16")}}},
		}, testSelfID, testOwnerAcctID)
		assert.Len(t, orig.IPRanges, 1, "the caller's original rule must not be mutated by aggregation")
	})

	t.Run("aggregates and sorts IPv6Ranges and PrefixListIDs", func(t *testing.T) {
		out := canonicalizeRuleList([]*svcapitypes.IPPermission{
			{IPProtocol: aws.String("tcp"), FromPort: aws.Int64(443), ToPort: aws.Int64(443),
				IPv6Ranges:    []*svcapitypes.IPv6Range{{CIDRIPv6: aws.String("2001:db8::/48")}},
				PrefixListIDs: []*svcapitypes.PrefixListID{{PrefixListID: aws.String("pl-22")}}},
			{IPProtocol: aws.String("tcp"), FromPort: aws.Int64(443), ToPort: aws.Int64(443),
				IPv6Ranges:    []*svcapitypes.IPv6Range{{CIDRIPv6: aws.String("2001:db8:1::/48")}},
				PrefixListIDs: []*svcapitypes.PrefixListID{{PrefixListID: aws.String("pl-11")}}},
		}, testSelfID, testOwnerAcctID)
		assert.Len(t, out, 1, "same (proto,from,to) rules merge")
		// merged and sorted ascending (lexicographic: '1' < ':')
		assert.Equal(t, []string{"2001:db8:1::/48", "2001:db8::/48"},
			[]string{*out[0].IPv6Ranges[0].CIDRIPv6, *out[0].IPv6Ranges[1].CIDRIPv6})
		assert.Equal(t, []string{"pl-11", "pl-22"},
			[]string{*out[0].PrefixListIDs[0].PrefixListID, *out[0].PrefixListIDs[1].PrefixListID})
	})

	t.Run("sorts UserIDGroupPairs so grant order does not matter", func(t *testing.T) {
		mk := func(gids ...string) []*svcapitypes.IPPermission {
			pairs := make([]*svcapitypes.UserIDGroupPair, 0, len(gids))
			for _, g := range gids {
				pairs = append(pairs, &svcapitypes.UserIDGroupPair{GroupID: aws.String(g), UserID: aws.String(testPeerAcctID)})
			}
			return []*svcapitypes.IPPermission{{
				IPProtocol: aws.String("tcp"), FromPort: aws.Int64(443), ToPort: aws.Int64(443),
				UserIDGroupPairs: pairs,
			}}
		}
		a := canonicalizeRuleList(mk("sg-aaa", "sg-bbb"), testSelfID, testOwnerAcctID)
		b := canonicalizeRuleList(mk("sg-bbb", "sg-aaa"), testSelfID, testOwnerAcctID)
		assert.Equal(t,
			[]string{*a[0].UserIDGroupPairs[0].GroupID, *a[0].UserIDGroupPairs[1].GroupID},
			[]string{*b[0].UserIDGroupPairs[0].GroupID, *b[0].UserIDGroupPairs[1].GroupID},
			"reordered group pairs must canonicalise to the same order")
	})

	t.Run("handles nil rule and nil pair entries without panic", func(t *testing.T) {
		var out []*svcapitypes.IPPermission
		assert.NotPanics(t, func() {
			out = canonicalizeRuleList([]*svcapitypes.IPPermission{
				nil,
				{IPProtocol: aws.String("tcp"), FromPort: aws.Int64(53), ToPort: aws.Int64(53),
					UserIDGroupPairs: []*svcapitypes.UserIDGroupPair{
						nil,
						{GroupID: aws.String(testSelfID)},
					}},
			}, testSelfID, testOwnerAcctID)
		})
		assert.Len(t, out, 1, "the nil rule is skipped, the real one is kept")
	})

	t.Run("handles nil CIDR/prefix grant elements without panic", func(t *testing.T) {
		// Defensive: sortGrants and the per-element loops must tolerate a nil
		// element in IPRanges/IPv6Ranges/PrefixListIDs. This cannot occur via
		// a CR (schema rejects null list entries) or on read-back (the setter
		// always allocates), but the guards keep the comparators panic-free.
		var out []*svcapitypes.IPPermission
		assert.NotPanics(t, func() {
			out = canonicalizeRuleList([]*svcapitypes.IPPermission{{
				IPProtocol: aws.String("tcp"), FromPort: aws.Int64(80), ToPort: aws.Int64(80),
				IPRanges: []*svcapitypes.IPRange{
					nil,
					{CIDRIP: aws.String("10.0.0.0/16")},
				},
				IPv6Ranges: []*svcapitypes.IPv6Range{
					nil,
					{CIDRIPv6: aws.String("2001:db8::/48")},
				},
				PrefixListIDs: []*svcapitypes.PrefixListID{
					nil,
					{PrefixListID: aws.String("pl-11")},
				},
			}}, testSelfID, testOwnerAcctID)
		})
		assert.Len(t, out, 1)
	})

	t.Run("mixed self + cross-SG pairs in one rule are handled independently", func(t *testing.T) {
		out := canonicalizeRuleList([]*svcapitypes.IPPermission{{
			IPProtocol: aws.String("tcp"), FromPort: aws.Int64(443), ToPort: aws.Int64(443),
			UserIDGroupPairs: []*svcapitypes.UserIDGroupPair{
				{GroupName: nil, UserID: aws.String(testOwnerAcctID)},                  // omitted self-ref
				{GroupID: aws.String(testOtherID), UserID: aws.String(testPeerAcctID)}, // cross-account
			},
		}}, testSelfID, testOwnerAcctID)
		pairs := out[0].UserIDGroupPairs
		// find each by GroupID
		var self, cross *svcapitypes.UserIDGroupPair
		for _, p := range pairs {
			if p.GroupID != nil && *p.GroupID == testSelfID {
				self = p
			}
			if p.GroupID != nil && *p.GroupID == testOtherID {
				cross = p
			}
		}
		assert.NotNil(t, self, "omitted pair canonicalised to selfID")
		assert.Nil(t, self.UserID, "self-ref owner UserID cleared")
		assert.NotNil(t, cross, "cross-SG pair kept")
		assert.Equal(t, testPeerAcctID, *cross.UserID, "cross-account UserID preserved")
	})
}

// -----------------------------------------------------------------------------
// canonicalizeCopiedRuleLists (resource-level guards, non-mutating)
// -----------------------------------------------------------------------------

func TestCanonicalizeCopiedRuleLists_Guards(t *testing.T) {
	t.Run("nil resource does not panic", func(t *testing.T) {
		assert.NotPanics(t, func() { canonicalizeCopiedRuleLists(nil) })
	})

	t.Run("nil ko does not panic", func(t *testing.T) {
		assert.NotPanics(t, func() { canonicalizeCopiedRuleLists(&resource{}) })
	})

	t.Run("nil Status.ID: omitted self-ref pair is left group-id-less", func(t *testing.T) {
		r := mkResource([]*svcapitypes.IPPermission{{
			FromPort: aws.Int64(53), ToPort: aws.Int64(53), IPProtocol: aws.String("tcp"),
			UserIDGroupPairs: []*svcapitypes.UserIDGroupPair{{Description: aws.String("coredns")}},
		}}, nil)
		r.ko.Status.ID = nil
		var ingress []*svcapitypes.IPPermission
		assert.NotPanics(t, func() { ingress, _ = canonicalizeCopiedRuleLists(r) })
		assert.Nil(t, ingress[0].UserIDGroupPairs[0].GroupID)
	})

	t.Run("does not mutate the source resource", func(t *testing.T) {
		// The whole point of the copy-based approach: canonicalisation must
		// never touch the caller's rule lists (they become the merge-patch
		// base in patchResourceMetadataAndSpec).
		r := mkResource([]*svcapitypes.IPPermission{{
			FromPort: aws.Int64(53), ToPort: aws.Int64(53), IPProtocol: aws.String("6"),
			UserIDGroupPairs: []*svcapitypes.UserIDGroupPair{{GroupRef: groupRef("myself")}},
		}}, nil)
		ingress, _ := canonicalizeCopiedRuleLists(r)
		// Source untouched: numeric protocol and groupRef preserved, no groupID injected.
		assert.Equal(t, "6", *r.ko.Spec.IngressRules[0].IPProtocol)
		assert.NotNil(t, r.ko.Spec.IngressRules[0].UserIDGroupPairs[0].GroupRef)
		assert.Nil(t, r.ko.Spec.IngressRules[0].UserIDGroupPairs[0].GroupID)
		// Copy canonicalised: protocol mapped to name, self-ref resolved to selfID.
		assert.Equal(t, "tcp", *ingress[0].IPProtocol)
		assert.Equal(t, testSelfID, *ingress[0].UserIDGroupPairs[0].GroupID)
	})
}

// -----------------------------------------------------------------------------
// customPostCompare via newResourceDelta (end-to-end delta suppression)
// -----------------------------------------------------------------------------

// desiredLatestDelta normalises both sides through newResourceDelta (which
// invokes customPostCompare) and returns the resulting delta.
func desiredLatestDelta(desired, latest *resource) *struct{ ing, egr bool } {
	d := newResourceDelta(desired, latest)
	return &struct{ ing, egr bool }{
		d.DifferentAt("Spec.IngressRules"),
		d.DifferentAt("Spec.EgressRules"),
	}
}

func TestCustomPostCompare_SelfRef_GroupRef_NoDiff(t *testing.T) {
	// desired mirrors the post-ResolveReferences state: GroupRef set AND
	// GroupID filled with selfID. latest is the AWS read-back: GroupID set,
	// UserID/GroupName filled, no GroupRef.
	desired := mkResource([]*svcapitypes.IPPermission{{
		FromPort: aws.Int64(53), ToPort: aws.Int64(53), IPProtocol: aws.String("tcp"),
		UserIDGroupPairs: []*svcapitypes.UserIDGroupPair{{
			GroupID: aws.String(testSelfID), GroupRef: groupRef("myself"),
		}},
	}}, nil)
	latest := mkResource([]*svcapitypes.IPPermission{{
		FromPort: aws.Int64(53), ToPort: aws.Int64(53), IPProtocol: aws.String("tcp"),
		UserIDGroupPairs: []*svcapitypes.UserIDGroupPair{{
			GroupID: aws.String(testSelfID), UserID: aws.String(testOwnerAcctID),
			GroupName: aws.String(testSelfName),
		}},
	}}, nil)

	got := desiredLatestDelta(desired, latest)
	assert.False(t, got.ing, "self-ref via groupRef must not produce a diff")
}

func TestCustomPostCompare_SelfRef_Omitted_NoDiff(t *testing.T) {
	// The exact #2822 reproducer: userIDGroupPair with only description +
	// userID, no groupID and no groupRef. latest has GroupID filled by AWS.
	desired := mkResource([]*svcapitypes.IPPermission{{
		FromPort: aws.Int64(53), ToPort: aws.Int64(53), IPProtocol: aws.String("tcp"),
		UserIDGroupPairs: []*svcapitypes.UserIDGroupPair{{
			Description: aws.String("coredns"), UserID: aws.String(testOwnerAcctID),
		}},
	}}, nil)
	latest := mkResource([]*svcapitypes.IPPermission{{
		FromPort: aws.Int64(53), ToPort: aws.Int64(53), IPProtocol: aws.String("tcp"),
		UserIDGroupPairs: []*svcapitypes.UserIDGroupPair{{
			Description: aws.String("coredns"), GroupID: aws.String(testSelfID),
			UserID: aws.String(testOwnerAcctID),
		}},
	}}, nil)

	got := desiredLatestDelta(desired, latest)
	assert.False(t, got.ing, "omitted-groupID self-ref (#2822) must not produce a diff")
}

func TestCustomPostCompare_GroupRefOnly_NoDiff(t *testing.T) {
	// desired mirrors the post-ResolveReferences state for a cross-SG rule
	// written with a groupRef: the wrapper is still set AND GroupID is
	// resolved. latest is the AWS read-back: GroupID only, no wrapper. The
	// spec-only groupRef must not drive a diff.
	desired := mkResource([]*svcapitypes.IPPermission{{
		FromPort: aws.Int64(443), ToPort: aws.Int64(443), IPProtocol: aws.String("tcp"),
		UserIDGroupPairs: []*svcapitypes.UserIDGroupPair{{
			GroupID:  aws.String(testOtherID),
			GroupRef: groupRef("other-sg"),
		}},
	}}, nil)
	latest := mkResource([]*svcapitypes.IPPermission{{
		FromPort: aws.Int64(443), ToPort: aws.Int64(443), IPProtocol: aws.String("tcp"),
		UserIDGroupPairs: []*svcapitypes.UserIDGroupPair{{
			GroupID: aws.String(testOtherID),
		}},
	}}, nil)

	got := desiredLatestDelta(desired, latest)
	assert.False(t, got.ing, "a spec-only groupRef must never produce a diff")
}

func TestCustomPostCompare_AllProtocolPorts_NoDiff(t *testing.T) {
	desired := mkResource(nil, []*svcapitypes.IPPermission{{
		IPProtocol: aws.String("-1"), FromPort: aws.Int64(0), ToPort: aws.Int64(0),
		IPRanges: []*svcapitypes.IPRange{{CIDRIP: aws.String("0.0.0.0/0")}},
	}})
	latest := mkResource(nil, []*svcapitypes.IPPermission{{
		IPProtocol: aws.String("-1"),
		IPRanges:   []*svcapitypes.IPRange{{CIDRIP: aws.String("0.0.0.0/0")}},
	}})

	got := desiredLatestDelta(desired, latest)
	assert.False(t, got.egr, "-1 rule with spec ports vs AWS-dropped ports must not diff")
}

func TestCustomPostCompare_GrantAggregation_NoDiff(t *testing.T) {
	// desired: two separate rules; latest: AWS-aggregated single rule, and
	// grants in reversed order to also exercise sorting.
	desired := mkResource([]*svcapitypes.IPPermission{
		{IPProtocol: aws.String("tcp"), FromPort: aws.Int64(443), ToPort: aws.Int64(443),
			IPRanges: []*svcapitypes.IPRange{{CIDRIP: aws.String("10.0.0.0/20")}}},
		{IPProtocol: aws.String("tcp"), FromPort: aws.Int64(443), ToPort: aws.Int64(443),
			IPRanges: []*svcapitypes.IPRange{{CIDRIP: aws.String("192.168.0.0/16")}}},
	}, nil)
	latest := mkResource([]*svcapitypes.IPPermission{
		{IPProtocol: aws.String("tcp"), FromPort: aws.Int64(443), ToPort: aws.Int64(443),
			IPRanges: []*svcapitypes.IPRange{
				{CIDRIP: aws.String("192.168.0.0/16")},
				{CIDRIP: aws.String("10.0.0.0/20")},
			}},
	}, nil)

	got := desiredLatestDelta(desired, latest)
	assert.False(t, got.ing, "split desired rules vs AWS-aggregated latest must not diff")
}

func TestCustomPostCompare_SelfRef_Egress_NoDiff(t *testing.T) {
	// Same self-ref suppression must hold on the egress side (symmetric code
	// path through canonicalizeRuleList).
	desired := mkResource(nil, []*svcapitypes.IPPermission{{
		FromPort: aws.Int64(53), ToPort: aws.Int64(53), IPProtocol: aws.String("tcp"),
		UserIDGroupPairs: []*svcapitypes.UserIDGroupPair{{Description: aws.String("egress self")}},
	}})
	latest := mkResource(nil, []*svcapitypes.IPPermission{{
		FromPort: aws.Int64(53), ToPort: aws.Int64(53), IPProtocol: aws.String("tcp"),
		UserIDGroupPairs: []*svcapitypes.UserIDGroupPair{{
			Description: aws.String("egress self"), GroupID: aws.String(testSelfID),
			UserID: aws.String(testOwnerAcctID),
		}},
	}})

	got := desiredLatestDelta(desired, latest)
	assert.False(t, got.egr, "self-ref egress must not produce a diff")
}

func TestCustomPostCompare_DescriptionChange_StillFires(t *testing.T) {
	// Description participates in comparison and must NOT be normalised away:
	// a description-only change on an otherwise-identical rule must diff.
	desired := mkResource([]*svcapitypes.IPPermission{{
		FromPort: aws.Int64(443), ToPort: aws.Int64(443), IPProtocol: aws.String("tcp"),
		IPRanges: []*svcapitypes.IPRange{{CIDRIP: aws.String("10.0.0.0/20"), Description: aws.String("before")}},
	}}, nil)
	latest := mkResource([]*svcapitypes.IPPermission{{
		FromPort: aws.Int64(443), ToPort: aws.Int64(443), IPProtocol: aws.String("tcp"),
		IPRanges: []*svcapitypes.IPRange{{CIDRIP: aws.String("10.0.0.0/20"), Description: aws.String("after")}},
	}}, nil)

	got := desiredLatestDelta(desired, latest)
	assert.True(t, got.ing, "a description-only change must still produce a diff")
}

func TestCustomPostCompare_NumericProtocol_NoDiff(t *testing.T) {
	// Verified against AWS: submitting IpProtocol "6" is stored and returned
	// as "tcp". The spec numeric form must not perpetually diff from the
	// name AWS returns.
	desired := mkResource([]*svcapitypes.IPPermission{{
		FromPort: aws.Int64(22), ToPort: aws.Int64(22), IPProtocol: aws.String("6"),
		IPRanges: []*svcapitypes.IPRange{{CIDRIP: aws.String("10.0.0.0/16")}},
	}}, nil)
	latest := mkResource([]*svcapitypes.IPPermission{{
		FromPort: aws.Int64(22), ToPort: aws.Int64(22), IPProtocol: aws.String("tcp"),
		IPRanges: []*svcapitypes.IPRange{{CIDRIP: aws.String("10.0.0.0/16")}},
	}}, nil)

	got := desiredLatestDelta(desired, latest)
	assert.False(t, got.ing, "numeric protocol (6) vs name (tcp) must not diff")
}

func TestCustomPostCompare_DifferentProtocol_StillFires(t *testing.T) {
	// A genuine protocol change (tcp -> udp) must still be detected.
	desired := mkResource([]*svcapitypes.IPPermission{{
		FromPort: aws.Int64(53), ToPort: aws.Int64(53), IPProtocol: aws.String("6"),
		IPRanges: []*svcapitypes.IPRange{{CIDRIP: aws.String("10.0.0.0/16")}},
	}}, nil)
	latest := mkResource([]*svcapitypes.IPPermission{{
		FromPort: aws.Int64(53), ToPort: aws.Int64(53), IPProtocol: aws.String("udp"),
		IPRanges: []*svcapitypes.IPRange{{CIDRIP: aws.String("10.0.0.0/16")}},
	}}, nil)

	got := desiredLatestDelta(desired, latest)
	assert.True(t, got.ing, "tcp vs udp must still produce a diff")
}

func TestCustomPostCompare_NonCanonicalCIDR_NoDiff(t *testing.T) {
	// Verified against AWS: submitting "100.68.0.18/18" is stored/returned as
	// "100.68.0.0/18"; IPv6 is also masked and text-normalised. The spec's
	// non-canonical form must not perpetually diff from the read-back form.
	desired := mkResource([]*svcapitypes.IPPermission{{
		FromPort: aws.Int64(443), ToPort: aws.Int64(443), IPProtocol: aws.String("tcp"),
		IPRanges:   []*svcapitypes.IPRange{{CIDRIP: aws.String("100.68.0.18/18")}},
		IPv6Ranges: []*svcapitypes.IPv6Range{{CIDRIPv6: aws.String("2001:DB8:abcd:0012::1/64")}},
	}}, nil)
	latest := mkResource([]*svcapitypes.IPPermission{{
		FromPort: aws.Int64(443), ToPort: aws.Int64(443), IPProtocol: aws.String("tcp"),
		IPRanges:   []*svcapitypes.IPRange{{CIDRIP: aws.String("100.68.0.0/18")}},
		IPv6Ranges: []*svcapitypes.IPv6Range{{CIDRIPv6: aws.String("2001:db8:abcd:12::/64")}},
	}}, nil)

	got := desiredLatestDelta(desired, latest)
	assert.False(t, got.ing, "non-canonical CIDR vs AWS-canonicalized form must not diff")
}

func TestCustomPostCompare_DifferentCIDR_StillFires(t *testing.T) {
	// A genuinely different network must still diff.
	desired := mkResource([]*svcapitypes.IPPermission{{
		FromPort: aws.Int64(443), ToPort: aws.Int64(443), IPProtocol: aws.String("tcp"),
		IPRanges: []*svcapitypes.IPRange{{CIDRIP: aws.String("10.0.0.0/16")}},
	}}, nil)
	latest := mkResource([]*svcapitypes.IPPermission{{
		FromPort: aws.Int64(443), ToPort: aws.Int64(443), IPProtocol: aws.String("tcp"),
		IPRanges: []*svcapitypes.IPRange{{CIDRIP: aws.String("10.1.0.0/16")}},
	}}, nil)

	got := desiredLatestDelta(desired, latest)
	assert.True(t, got.ing, "a different CIDR network must still produce a diff")
}

func TestCustomPostCompare_RealDiff_StillFires(t *testing.T) {
	desired := mkResource([]*svcapitypes.IPPermission{{
		FromPort: aws.Int64(53), ToPort: aws.Int64(53), IPProtocol: aws.String("tcp"),
		UserIDGroupPairs: []*svcapitypes.UserIDGroupPair{{GroupID: aws.String(testSelfID)}},
	}}, nil)
	latest := mkResource([]*svcapitypes.IPPermission{{
		FromPort: aws.Int64(80), ToPort: aws.Int64(80), IPProtocol: aws.String("tcp"),
		UserIDGroupPairs: []*svcapitypes.UserIDGroupPair{{
			GroupID: aws.String(testSelfID), UserID: aws.String(testOwnerAcctID),
		}},
	}}, nil)

	got := desiredLatestDelta(desired, latest)
	assert.True(t, got.ing, "a genuine port change must still produce a diff")
}

func TestCustomPostCompare_CrossAccount_NotSuppressed(t *testing.T) {
	// desired omits the cross-account UserID; latest carries the peer
	// account. Because the peer account differs from the owner account it is
	// NOT stripped, so the missing UserID on desired remains a real diff.
	desired := mkResource([]*svcapitypes.IPPermission{ruleWithPairs(
		&svcapitypes.UserIDGroupPair{GroupID: aws.String(testOtherID)},
	)}, nil)
	latest := mkResource([]*svcapitypes.IPPermission{ruleWithPairs(
		&svcapitypes.UserIDGroupPair{GroupID: aws.String(testOtherID), UserID: aws.String(testPeerAcctID)},
	)}, nil)

	got := desiredLatestDelta(desired, latest)
	assert.True(t, got.ing, "cross-account UserID divergence must remain a diff (not silently dropped)")
}

func TestCustomPostCompare_SelfRef_StaysResolved(t *testing.T) {
	// customUpdateSecurityGroup relies on GroupID being non-nil after
	// normalisation so that referencesResolved keeps syncSGRules open for a
	// self-reference expressed via groupRef.
	desired := mkResource([]*svcapitypes.IPPermission{{
		FromPort: aws.Int64(53), ToPort: aws.Int64(53), IPProtocol: aws.String("tcp"),
		UserIDGroupPairs: []*svcapitypes.UserIDGroupPair{{
			GroupID: aws.String(testSelfID), GroupRef: groupRef("myself"),
		}},
	}}, nil)
	latest := mkResource(nil, nil)

	_ = newResourceDelta(desired, latest)

	rm := &resourceManager{}
	assert.True(t, rm.referencesResolved(desired),
		"after normalisation a self-ref must still be seen as resolved")
}

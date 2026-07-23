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
	"k8s.io/apimachinery/pkg/api/equality"

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
		// The delta assumes GroupID is resolved: a wrapper-only pair is
		// stripped and, being identifier-less, treated as self. Guarding an
		// unresolved reference is referencesResolved's job, not this function's.
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

	t.Run("read-only reference metadata cleared on a cross-SG pair", func(t *testing.T) {
		// VPCID, PeeringStatus and VPCPeeringConnectionID are read-only values
		// EC2 derives from the referenced group (VPCID is filled only for a
		// cross-VPC peer). The customer cannot set them meaningfully, so they
		// must never drive a delta -- clear them alongside the VPCRef wrapper.
		p := &svcapitypes.UserIDGroupPair{
			GroupID:                aws.String(testOtherID),
			VPCID:                  aws.String("vpc-123"),
			VPCRef:                 groupRef("my-vpc"),
			PeeringStatus:          aws.String("active"),
			VPCPeeringConnectionID: aws.String("pcx-123"),
		}
		canonicalizeGroupPair(p, testSelfID, testOwnerAcctID)
		assert.Equal(t, testOtherID, *p.GroupID, "resolved GroupID is preserved")
		assert.Nil(t, p.VPCRef, "VPCRef must be cleared to match AWS read-back form")
		assert.Nil(t, p.VPCID, "server-derived VPCID must be cleared")
		assert.Nil(t, p.PeeringStatus, "read-only PeeringStatus must be cleared")
		assert.Nil(t, p.VPCPeeringConnectionID, "read-only VPCPeeringConnectionID must be cleared")
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

// TestCanonicalizeGroupPair_InconsistentInputs documents, against real EC2
// behavior, that dropping the owner UserID on a self-ref is lossless:
// {GroupId=self} and {GroupId=self, UserId=owner} produce the identical rule
// (AWS auto-fills the owner). A foreign UserID is only ever dropped when paired
// with the SG's own GroupID -- an input AWS rejects anyway; a legitimate
// cross-account ref uses GroupId=peer and is preserved (see
// TestCustomPostCompare_CrossAccount_NotSuppressed).
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

	t.Run("drops ports for protocols outside tcp/udp/icmp/icmpv6", func(t *testing.T) {
		// AWS omits the port range on read-back for any protocol not in
		// {tcp, udp, icmp, icmpv6}, e.g. 50 (ESP) or 47 (GRE). A spec that
		// carries ports for such a protocol must be canonicalised to no ports.
		out := canonicalizeRuleList([]*svcapitypes.IPPermission{
			{IPProtocol: aws.String("50"), FromPort: aws.Int64(-1), ToPort: aws.Int64(-1),
				IPRanges: []*svcapitypes.IPRange{{CIDRIP: aws.String("10.0.0.0/8")}}},
			{IPProtocol: aws.String("47"), FromPort: aws.Int64(0), ToPort: aws.Int64(0),
				IPRanges: []*svcapitypes.IPRange{{CIDRIP: aws.String("192.168.0.0/16")}}},
		}, testSelfID, testOwnerAcctID)
		assert.Len(t, out, 2)
		for _, r := range out {
			assert.Nil(t, r.FromPort, "protocol %s must drop fromPort", *r.IPProtocol)
			assert.Nil(t, r.ToPort, "protocol %s must drop toPort", *r.IPProtocol)
		}
	})

	t.Run("non-standard protocol: ported spec matches portless read-back", func(t *testing.T) {
		// The motivating diff: a spec written with -1/-1 ports on an ESP rule
		// must compare equal to the AWS read-back, which drops the ports
		// entirely. After canonicalisation both sides are portless and identical.
		spec := canonicalizeRuleList([]*svcapitypes.IPPermission{{
			IPProtocol: aws.String("50"), FromPort: aws.Int64(-1), ToPort: aws.Int64(-1),
			IPRanges: []*svcapitypes.IPRange{{CIDRIP: aws.String("10.0.0.0/8")}},
		}}, testSelfID, testOwnerAcctID)
		readBack := canonicalizeRuleList([]*svcapitypes.IPPermission{{
			IPProtocol: aws.String("50"),
			IPRanges:   []*svcapitypes.IPRange{{CIDRIP: aws.String("10.0.0.0/8")}},
		}}, testSelfID, testOwnerAcctID)
		assert.Equal(t, readBack, spec, "ported spec and portless read-back must canonicalise equal")
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

	t.Run("collapses the icmpv6 -1 wildcard type/code to nil", func(t *testing.T) {
		// icmpv6 read back from AWS as the "all types and codes" wildcard
		// carries FromPort/ToPort = -1. Canonicalise it to nil so it matches a
		// spec that simply omits the type/code.
		out := canonicalizeRuleList([]*svcapitypes.IPPermission{{
			IPProtocol: aws.String("icmpv6"),
			FromPort:   aws.Int64(-1),
			ToPort:     aws.Int64(-1),
			IPv6Ranges: []*svcapitypes.IPv6Range{{CIDRIPv6: aws.String("::/0")}},
		}}, testSelfID, testOwnerAcctID)
		assert.Len(t, out, 1)
		assert.Nil(t, out[0].FromPort)
		assert.Nil(t, out[0].ToPort)
	})

	t.Run("numeric protocol 58 also collapses the -1 wildcard", func(t *testing.T) {
		// "58" canonicalises to "icmpv6", so the wildcard collapse must apply to
		// the numeric spelling too.
		out := canonicalizeRuleList([]*svcapitypes.IPPermission{{
			IPProtocol: aws.String("58"),
			FromPort:   aws.Int64(-1),
			ToPort:     aws.Int64(-1),
		}}, testSelfID, testOwnerAcctID)
		assert.Len(t, out, 1)
		assert.Equal(t, "icmpv6", *out[0].IPProtocol)
		assert.Nil(t, out[0].FromPort)
		assert.Nil(t, out[0].ToPort)
	})

	t.Run("omitted and explicit-wildcard icmpv6 canonicalise equal", func(t *testing.T) {
		// The read-back form (-1/-1) and the spec shorthand (omitted) are the
		// same rule; both must collapse to the identical canonical shape so no
		// perpetual diff arises (aws-controllers-k8s/community#2822).
		omitted := canonicalizeRuleList([]*svcapitypes.IPPermission{{
			IPProtocol: aws.String("icmpv6"),
			IPv6Ranges: []*svcapitypes.IPv6Range{{CIDRIPv6: aws.String("::/0")}},
		}}, testSelfID, testOwnerAcctID)
		wildcard := canonicalizeRuleList([]*svcapitypes.IPPermission{{
			IPProtocol: aws.String("icmpv6"),
			FromPort:   aws.Int64(-1),
			ToPort:     aws.Int64(-1),
			IPv6Ranges: []*svcapitypes.IPv6Range{{CIDRIPv6: aws.String("::/0")}},
		}}, testSelfID, testOwnerAcctID)
		assert.True(t, equality.Semantic.DeepEqual(omitted, wildcard),
			"omitted and -1/-1 icmpv6 must be indistinguishable after canonicalisation")
	})

	t.Run("preserves a real icmpv6 type/code", func(t *testing.T) {
		// A concrete type/code (e.g. 128/0, echo request) is not the wildcard
		// and must survive so a genuine change is not swallowed.
		out := canonicalizeRuleList([]*svcapitypes.IPPermission{{
			IPProtocol: aws.String("icmpv6"),
			FromPort:   aws.Int64(128),
			ToPort:     aws.Int64(0),
		}}, testSelfID, testOwnerAcctID)
		assert.Len(t, out, 1)
		assert.NotNil(t, out[0].FromPort)
		assert.EqualValues(t, 128, *out[0].FromPort)
		assert.NotNil(t, out[0].ToPort)
		assert.EqualValues(t, 0, *out[0].ToPort)
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
		// Defensive only: a nil grant element can't occur via a CR or read-back,
		// but sortGrants and the per-element loops must tolerate it.
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
// canonicalizeCIDR
// -----------------------------------------------------------------------------

func TestCanonicalizeCIDR(t *testing.T) {
	cases := []struct {
		name string
		in   *string
		want *string
	}{
		{"nil passes through", nil, nil},
		{"IPv4 host bits masked", aws.String("100.68.0.18/18"), aws.String("100.68.0.0/18")},
		{"IPv4 already canonical", aws.String("10.0.0.0/24"), aws.String("10.0.0.0/24")},
		{"IPv4 /32 host preserved", aws.String("10.0.0.5/32"), aws.String("10.0.0.5/32")},
		{"IPv4 /0 collapses to zero network", aws.String("10.10.0.0/0"), aws.String("0.0.0.0/0")},
		// IPv6 is lowercased, zero-compressed, and leading zeros in a hextet
		// dropped -- the RFC 5952 form AWS returns on read-back.
		{"IPv6 host bits masked and normalized", aws.String("2001:DB8:abcd:0012::1/64"), aws.String("2001:db8:abcd:12::/64")},
		{"IPv6 already canonical", aws.String("2600:1f16::/48"), aws.String("2600:1f16::/48")},
		// AWS rejects a prefix longer than the address; we leave it
		// untouched rather than emit a bogus network.
		{"IPv4 prefix too long passes through", aws.String("10.0.0.0/40"), aws.String("10.0.0.0/40")},
		{"IPv6 prefix too long passes through", aws.String("2001:db8::/129"), aws.String("2001:db8::/129")},
		{"negative prefix passes through", aws.String("10.0.0.0/-1"), aws.String("10.0.0.0/-1")},
		{"missing prefix passes through", aws.String("10.0.0.0"), aws.String("10.0.0.0")},
		{"malformed address passes through", aws.String("not-a-cidr"), aws.String("not-a-cidr")},
		{"non-numeric prefix passes through", aws.String("10.0.0.0/abc"), aws.String("10.0.0.0/abc")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := canonicalizeCIDR(tc.in)
			if tc.want == nil {
				assert.Nil(t, got)
				return
			}
			if assert.NotNil(t, got) {
				assert.Equal(t, *tc.want, *got)
			}
		})
	}

	t.Run("does not mutate the caller's string", func(t *testing.T) {
		orig := "100.68.0.18/18"
		in := orig
		_ = canonicalizeCIDR(&in)
		assert.Equal(t, orig, in, "input pointer target must be unchanged")
	})
}

// TestCanonicalizeCIDR_BoundaryVectors exercises the partial-byte network mask
// at the /7, /8, /9, /31 and /127 prefix boundaries -- the part of the
// canonicalizeCIDR mask loop most prone to off-by-one bugs. Each case pairs a
// non-canonical input (one that carries host bits, which AWS rejects on input)
// with the canonical network canonicalizeCIDR rewrites it to, alongside the
// already-canonical form that must pass through unchanged.
func TestCanonicalizeCIDR_BoundaryVectors(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		// IPv6 /8 boundary (first byte only).
		{"ff10::/8", "ff00::/8"}, // AWS rejects as non-canonical
		{"ff00::/8", "ff00::/8"}, // AWS accepts as canonical
		// IPv6 /7 boundary (first byte mask 0xfe).
		{"ff00::/7", "fe00::/7"}, // AWS rejects
		{"fe00::/7", "fe00::/7"}, // AWS accepts
		// IPv4 /9 boundary (second byte mask 0x80).
		{"10.64.0.0/9", "10.0.0.0/9"},    // AWS rejects
		{"10.128.0.0/9", "10.128.0.0/9"}, // AWS accepts
		// IPv6 /8 with a host bit in the last hextet.
		{"::1/8", "::/8"}, // AWS rejects
		{"::/8", "::/8"},  // AWS accepts
		// IPv4 /31 boundary (last byte mask 0xfe).
		{"255.255.255.255/31", "255.255.255.254/31"}, // AWS rejects
		{"0.0.0.0/0", "0.0.0.0/0"},                   // AWS accepts
		{"255.255.255.255/32", "255.255.255.255/32"}, // AWS accepts
		// IPv6 /127 boundary and a canonical /64.
		{"ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff/127", "ffff:ffff:ffff:ffff:ffff:ffff:ffff:fffe/127"}, // AWS rejects
		{"ffff:ffff:ffff:ffff::/64", "ffff:ffff:ffff:ffff::/64"},                                       // AWS accepts
		{"6be8:7177:fd47:df14:f49b:a890:3a67:8000/113", "6be8:7177:fd47:df14:f49b:a890:3a67:8000/113"}, // AWS accepts (/113 boundary)
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			in := tc.in
			got := canonicalizeCIDR(&in)
			if assert.NotNil(t, got) {
				assert.Equal(t, tc.want, *got)
			}
		})
	}
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

// TestCustomPostCompare_ReadOnlyPeerMetadata_NoDiff proves that VPCID,
// PeeringStatus and VPCPeeringConnectionID never drive a delta. These are
// read-only values EC2 derives from the referenced group (populated on a
// cross-VPC peer read-back, absent on a same-VPC one) -- the authorize path
// accepts and discards any client-supplied value. Neither an omitted spec
// value nor a stale/mismatched one may churn against the AWS-filled read-back.
func TestCustomPostCompare_ReadOnlyPeerMetadata_NoDiff(t *testing.T) {
	// latest mirrors a cross-VPC peer reference as AWS returns it: GroupID +
	// owner UserID plus the three derived read-only fields.
	awsReadBack := func() []*svcapitypes.IPPermission {
		return []*svcapitypes.IPPermission{{
			FromPort: aws.Int64(443), ToPort: aws.Int64(443), IPProtocol: aws.String("tcp"),
			UserIDGroupPairs: []*svcapitypes.UserIDGroupPair{{
				GroupID:                aws.String(testOtherID),
				UserID:                 aws.String(testOwnerAcctID),
				VPCID:                  aws.String("vpc-peer-b"),
				PeeringStatus:          aws.String("active"),
				VPCPeeringConnectionID: aws.String("pcx-realvalue"),
			}},
		}}
	}

	t.Run("spec omits the read-only fields", func(t *testing.T) {
		desired := mkResource([]*svcapitypes.IPPermission{{
			FromPort: aws.Int64(443), ToPort: aws.Int64(443), IPProtocol: aws.String("tcp"),
			UserIDGroupPairs: []*svcapitypes.UserIDGroupPair{{
				GroupID: aws.String(testOtherID),
			}},
		}}, nil)
		latest := mkResource(awsReadBack(), nil)

		got := desiredLatestDelta(desired, latest)
		assert.False(t, got.ing, "AWS-derived peer metadata absent from spec must not diff")
	})

	t.Run("spec carries stale/bogus values for all three", func(t *testing.T) {
		// Even a spec value that disagrees with the AWS read-back must not
		// churn: EC2 ignores these on authorize, so they carry no real intent.
		desired := mkResource([]*svcapitypes.IPPermission{{
			FromPort: aws.Int64(443), ToPort: aws.Int64(443), IPProtocol: aws.String("tcp"),
			UserIDGroupPairs: []*svcapitypes.UserIDGroupPair{{
				GroupID:                aws.String(testOtherID),
				VPCID:                  aws.String("vpc-STALEbogus"),
				PeeringStatus:          aws.String("pending-acceptance"),
				VPCPeeringConnectionID: aws.String("pcx-STALEbogus"),
			}},
		}}, nil)
		latest := mkResource(awsReadBack(), nil)

		got := desiredLatestDelta(desired, latest)
		assert.False(t, got.ing, "stale spec peer metadata must not diff against AWS-derived values")
	})
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
	// customPostCompare works on copies and must not mutate desired, so
	// referencesResolved still sees the resolved GroupID and keeps syncSGRules
	// open for a groupRef self-reference.
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
		"delta must not nil the resolved GroupID of a self-ref")
}

// -----------------------------------------------------------------------------
// Full delta path: composite ingress + egress
//
// The tests above each isolate a single normalisation on one rule list. These
// exercise the whole delta path (newResourceDelta -> customPostCompare) with a
// realistic resource that stacks many normalisations across BOTH ingress and
// egress in one pass: self-reference by omission, grant aggregation, CIDR
// canonicalisation, read-only peer metadata, all-protocol port dropping and
// numeric-protocol naming -- desired in raw spec form, latest in AWS read-back
// form.
// -----------------------------------------------------------------------------

// compositeSpec builds a desired (raw spec) resource that combines several
// normalisations on both rule lists.
func compositeSpec() *resource {
	ingress := []*svcapitypes.IPPermission{
		// self-reference by omission (#2822): no groupID, no groupRef
		{FromPort: aws.Int64(53), ToPort: aws.Int64(53), IPProtocol: aws.String("tcp"),
			UserIDGroupPairs: []*svcapitypes.UserIDGroupPair{{Description: aws.String("coredns")}}},
		// two grants AWS aggregates under 443/tcp, one CIDR non-canonical
		{FromPort: aws.Int64(443), ToPort: aws.Int64(443), IPProtocol: aws.String("tcp"),
			IPRanges: []*svcapitypes.IPRange{{CIDRIP: aws.String("10.0.0.0/20")}}},
		{FromPort: aws.Int64(443), ToPort: aws.Int64(443), IPProtocol: aws.String("tcp"),
			IPRanges: []*svcapitypes.IPRange{{CIDRIP: aws.String("100.68.0.18/18")}}},
		// cross-SG reference; read-only peer metadata omitted in spec
		{FromPort: aws.Int64(8443), ToPort: aws.Int64(8443), IPProtocol: aws.String("tcp"),
			UserIDGroupPairs: []*svcapitypes.UserIDGroupPair{{GroupID: aws.String(testOtherID)}}},
	}
	egress := []*svcapitypes.IPPermission{
		// all-protocol rule with explicit ports AWS drops
		{IPProtocol: aws.String("-1"), FromPort: aws.Int64(0), ToPort: aws.Int64(0),
			IPRanges: []*svcapitypes.IPRange{{CIDRIP: aws.String("0.0.0.0/0")}}},
		// numeric protocol AWS returns by name
		{FromPort: aws.Int64(22), ToPort: aws.Int64(22), IPProtocol: aws.String("6"),
			IPRanges: []*svcapitypes.IPRange{{CIDRIP: aws.String("10.0.0.0/16")}}},
	}
	return mkResource(ingress, egress)
}

// compositeReadBack builds the latest resource as AWS returns it: rules
// aggregated, CIDRs canonicalised, grants reordered, self/owner and peer
// metadata filled, ports dropped, protocols named. Callers may tweak the
// returned lists to introduce a genuine change.
func compositeReadBack() *resource {
	ingress := []*svcapitypes.IPPermission{
		// self-reference: AWS filled groupID (self) + owner userID
		{FromPort: aws.Int64(53), ToPort: aws.Int64(53), IPProtocol: aws.String("tcp"),
			UserIDGroupPairs: []*svcapitypes.UserIDGroupPair{{
				Description: aws.String("coredns"), GroupID: aws.String(testSelfID),
				UserID: aws.String(testOwnerAcctID)}}},
		// aggregated 443/tcp, CIDRs canonical and in reversed order
		{FromPort: aws.Int64(443), ToPort: aws.Int64(443), IPProtocol: aws.String("tcp"),
			IPRanges: []*svcapitypes.IPRange{
				{CIDRIP: aws.String("100.68.0.0/18")},
				{CIDRIP: aws.String("10.0.0.0/20")}}},
		// cross-SG reference: owner userID + derived read-only peer metadata
		{FromPort: aws.Int64(8443), ToPort: aws.Int64(8443), IPProtocol: aws.String("tcp"),
			UserIDGroupPairs: []*svcapitypes.UserIDGroupPair{{
				GroupID: aws.String(testOtherID), UserID: aws.String(testOwnerAcctID),
				VPCID: aws.String("vpc-peer-b"), PeeringStatus: aws.String("active"),
				VPCPeeringConnectionID: aws.String("pcx-realvalue")}}},
	}
	egress := []*svcapitypes.IPPermission{
		// all-protocol rule: ports dropped
		{IPProtocol: aws.String("-1"),
			IPRanges: []*svcapitypes.IPRange{{CIDRIP: aws.String("0.0.0.0/0")}}},
		// protocol returned as name
		{FromPort: aws.Int64(22), ToPort: aws.Int64(22), IPProtocol: aws.String("tcp"),
			IPRanges: []*svcapitypes.IPRange{{CIDRIP: aws.String("10.0.0.0/16")}}},
	}
	return mkResource(ingress, egress)
}

func TestCustomPostCompare_Composite_IngressAndEgress_NoDiff(t *testing.T) {
	// A realistic resource stacking many normalisations on both lists must
	// converge with zero delta on ingress AND egress in a single pass.
	got := desiredLatestDelta(compositeSpec(), compositeReadBack())
	assert.False(t, got.ing, "composite ingress normalisations must not diff")
	assert.False(t, got.egr, "composite egress normalisations must not diff")
}

func TestCustomPostCompare_Composite_RealEgressChange_FiresEgressOnly(t *testing.T) {
	// Same fully-normalised resource, but with one genuine change buried in
	// the egress list (the 22/tcp rule's CIDR). The full path must still
	// suppress the (unchanged) ingress and surface the real egress diff --
	// the two lists are evaluated independently.
	desired := compositeSpec()
	latest := compositeReadBack()
	latest.ko.Spec.EgressRules[1].IPRanges[0].CIDRIP = aws.String("10.1.0.0/16") // was 10.0.0.0/16

	got := desiredLatestDelta(desired, latest)
	assert.False(t, got.ing, "unchanged ingress must stay suppressed")
	assert.True(t, got.egr, "a genuine egress CIDR change must produce a diff")
}

func TestCustomPostCompare_Composite_RealIngressChange_FiresIngressOnly(t *testing.T) {
	// Symmetric to the above: a genuine change buried in the ingress list
	// (the cross-SG reference now points at a different group) must surface on
	// ingress while the fully-normalised egress stays suppressed.
	desired := compositeSpec()
	latest := compositeReadBack()
	latest.ko.Spec.IngressRules[2].UserIDGroupPairs[0].GroupID = aws.String("sg-different0000")

	got := desiredLatestDelta(desired, latest)
	assert.True(t, got.ing, "a genuine ingress group reference change must produce a diff")
	assert.False(t, got.egr, "unchanged egress must stay suppressed")
}

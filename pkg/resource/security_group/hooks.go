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
	"context"
	"net"
	"sort"
	"strconv"

	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackcondition "github.com/aws-controllers-k8s/runtime/pkg/condition"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	svcsdk "github.com/aws/aws-sdk-go-v2/service/ec2"
	svcsdktypes "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go/aws"
	awserr "github.com/aws/aws-sdk-go/aws/awserr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"

	svcapitypes "github.com/aws-controllers-k8s/ec2-controller/apis/v1alpha1"
	"github.com/aws-controllers-k8s/ec2-controller/pkg/tags"
)

// addRulesToSpec updates a resource's Spec EgressRules and IngressRules
// using data from a DescribeSecurityGroups response
func (rm *resourceManager) addRulesToSpec(
	ko *svcapitypes.SecurityGroup,
	resp svcsdktypes.SecurityGroup,
) {
	// if there are no rules to add to Spec, then
	// set Spec rules to nil to align with latest state;
	// otherwise, data from an older version of the
	// resource will persist.
	ko.Spec.IngressRules = nil
	ko.Spec.EgressRules = nil

	if resp.IpPermissions != nil {
		specIngress := []*svcapitypes.IPPermission{}
		for _, ip := range resp.IpPermissions {
			specIngress = append(specIngress, rm.setResourceIPPermission(&ip))
		}
		ko.Spec.IngressRules = specIngress
	}
	if resp.IpPermissionsEgress != nil {
		specEgress := []*svcapitypes.IPPermission{}
		for _, ep := range resp.IpPermissionsEgress {
			specEgress = append(specEgress, rm.setResourceIPPermission(&ep))
		}
		ko.Spec.EgressRules = specEgress
	}
}

// getRules calls DescribeSecurityGroupRules
// and returns the  response data to populate a Security Group's
// Status.Rules
func (rm *resourceManager) getRules(
	ctx context.Context,
	res *resource,
) (rules []*svcapitypes.SecurityGroupRule, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.getRules")
	defer exit(err)

	groupIDFilter := "group-id"
	input := &svcsdk.DescribeSecurityGroupRulesInput{
		Filters: []svcsdktypes.Filter{
			{
				Name:   &groupIDFilter,
				Values: []string{*res.ko.Status.ID},
			},
		},
	}

	for {
		resp, err := rm.sdkapi.DescribeSecurityGroupRules(ctx, input)
		rm.metrics.RecordAPICall("READ_MANY", "DescribeSecurityGroupRules", err)
		if err != nil || resp == nil {
			break
		}
		for _, sgRule := range resp.SecurityGroupRules {
			rules = append(rules, rm.setResourceSecurityGroupRule(&sgRule))
		}
		if resp.NextToken == nil || *resp.NextToken == "" {
			break
		}
		input.NextToken = resp.NextToken
	}
	if err != nil {
		return nil, err
	}

	return rules, nil
}

func (rm *resourceManager) requiredFieldsMissingForSGRule(
	r *resource,
) bool {
	return r.ko.Status.ID == nil
}

// referencesResolved checks that any referenced security group actually exists in AWS, before proceeding with syncSGRules.
// This is required because Rules.UserIDGroupPairs.GroupID.skip_resource_state_validations is set to true,
// meaning that any state validations performed at runtime, during ResolveReferences step, are being skipped.
func (rm *resourceManager) referencesResolved(
	r *resource,
) bool {
	for _, rule := range r.ko.Spec.IngressRules {
		for _, groupPair := range rule.UserIDGroupPairs {
			if groupPair.GroupRef != nil && groupPair.GroupID == nil {
				return false
			}
		}
	}
	for _, rule := range r.ko.Spec.EgressRules {
		for _, groupPair := range rule.UserIDGroupPairs {
			if groupPair.GroupRef != nil && groupPair.GroupID == nil {
				return false
			}
		}
	}
	return true
}

// allProtocols is the IpProtocol value EC2 uses to mean "all protocols".
// For such rules AWS ignores the port range and DescribeSecurityGroups
// returns the rule with FromPort/ToPort unset.
const allProtocols = "-1"

// wellKnownProtocols maps the IANA protocol numbers that EC2 canonicalises to
// names on DescribeSecurityGroups read-back. A spec that uses the numeric form
// (e.g. "6") would otherwise perpetually differ from the name AWS returns
// ("tcp"). Protocols outside this set are returned by AWS as-is (their number),
// so they need no mapping; "-1" (all protocols) is already canonical.
var wellKnownProtocols = map[string]string{
	"1":  "icmp",
	"6":  "tcp",
	"17": "udp",
	"58": "icmpv6",
}

// canonicalizeProtocol maps a numeric IpProtocol to the name EC2 returns on
// read-back so that numeric and name spellings of the same protocol compare
// equal. Unknown protocols and names are returned unchanged.
func canonicalizeProtocol(proto *string) *string {
	if proto == nil {
		return nil
	}
	if name, ok := wellKnownProtocols[*proto]; ok {
		return &name
	}
	return proto
}

// canonicalizeCIDR rewrites a CIDR to the network form EC2 returns on
// read-back: host bits are masked off and (for IPv6) the text is normalised
// to lowercase, zero-compressed RFC 5952 form -- e.g. "100.68.0.18/18" ->
// "100.68.0.0/18" and "2001:DB8:abcd:0012::1/64" -> "2001:db8:abcd:12::/64".
// A value that does not parse as a CIDR is returned unchanged (AWS will
// reject a truly malformed CIDR on the API call).
func canonicalizeCIDR(cidr *string) *string {
	if cidr == nil {
		return nil
	}
	_, ipNet, err := net.ParseCIDR(*cidr)
	if err != nil {
		return cidr
	}
	canon := ipNet.String()
	return &canon
}

// customPostCompare is injected at the end of the generated newResourceDelta
// (see delta.go) via the `delta_post_compare` hook in generator.yaml. The
// generated field-by-field comparison skips Spec.IngressRules and
// Spec.EgressRules entirely (they are marked `compare.is_ignored: true`);
// this hook compares them instead.
//
// Unlike a `delta_pre_compare` hook, it MUST NOT mutate a or b: the same
// objects are used downstream as the merge-patch base in
// patchResourceMetadataAndSpec, so any mutation here would leak into the
// persisted Spec. Instead it compares canonical *copies* and records a delta
// against the original (un-normalised) values, so genuine changes are
// reported while server-side normalisation never produces a spurious diff.
//
// canonicalizeCopiedRuleLists closes the perpetual-diff sources from
// aws-controllers-k8s/community#2822:
//
//  1. Reference wrappers and self-references. GroupRef/VPCRef are spec-only
//     wrappers AWS never returns; they are cleared in the copy. A pair is a
//     self-reference when GroupID equals the SG's own ID, or when the group
//     identifiers are all omitted (the spec shorthand). We canonicalise to
//     {GroupID: selfID} and clear GroupName/UserID.
//  2. All-protocol ("-1") port ranges. AWS drops FromPort/ToPort for these.
//  3. Server-filled owner account. AWS populates UserID with the owning
//     account for same-account grants; we clear it. Cross-account grants
//     (UserID != owner) are preserved so the reference stays intact.
//  4. Grant aggregation. AWS merges grants sharing (protocol, fromPort,
//     toPort) into one IpPermission; we aggregate both sides by that key and
//     sort deterministically so ordering never drives a diff.
//  5. Protocol notation. AWS canonicalises well-known IANA protocol numbers
//     to names on read-back ("6" -> "tcp").
//  6. CIDR canonicalisation. AWS masks host bits and normalises IPv6 text
//     ("100.68.0.18/18" -> "100.68.0.0/18").
func customPostCompare(
	delta *ackcompare.Delta,
	a *resource,
	b *resource,
) {
	if a == nil || b == nil || a.ko == nil || b.ko == nil {
		return
	}
	aIngress, aEgress := canonicalizeCopiedRuleLists(a)
	bIngress, bEgress := canonicalizeCopiedRuleLists(b)
	if !equality.Semantic.DeepEqual(aIngress, bIngress) {
		delta.Add("Spec.IngressRules", a.ko.Spec.IngressRules, b.ko.Spec.IngressRules)
	}
	if !equality.Semantic.DeepEqual(aEgress, bEgress) {
		delta.Add("Spec.EgressRules", a.ko.Spec.EgressRules, b.ko.Spec.EgressRules)
	}
}

// canonicalizeCopiedRuleLists returns canonical *copies* of r's Ingress and
// Egress rule lists. r is never mutated, so callers can use the result purely
// for comparison (customPostCompare) or to build AWS Authorize/Revoke inputs
// (syncSGRules) without affecting the persisted Spec.
func canonicalizeCopiedRuleLists(
	r *resource,
) (ingress []*svcapitypes.IPPermission, egress []*svcapitypes.IPPermission) {
	if r == nil || r.ko == nil {
		return nil, nil
	}
	var selfID string
	if r.ko.Status.ID != nil {
		selfID = *r.ko.Status.ID
	}
	var ownerAccountID string
	if r.ko.Status.ACKResourceMetadata != nil &&
		r.ko.Status.ACKResourceMetadata.OwnerAccountID != nil {
		ownerAccountID = string(*r.ko.Status.ACKResourceMetadata.OwnerAccountID)
	}
	ingress = canonicalizeRuleList(deepCopyRuleList(r.ko.Spec.IngressRules), selfID, ownerAccountID)
	egress = canonicalizeRuleList(deepCopyRuleList(r.ko.Spec.EgressRules), selfID, ownerAccountID)
	return ingress, egress
}

// deepCopyRuleList returns a deep copy of a rule list so that in-place
// canonicalisation never touches the caller's backing data. A nil input
// returns nil so an empty and an absent rule list keep comparing equal.
func deepCopyRuleList(rules []*svcapitypes.IPPermission) []*svcapitypes.IPPermission {
	if rules == nil {
		return nil
	}
	out := make([]*svcapitypes.IPPermission, len(rules))
	for i, r := range rules {
		out[i] = r.DeepCopy()
	}
	return out
}

// canonicalizeRuleList normalises each rule (ports + grants), aggregates
// rules sharing the same (protocol, fromPort, toPort) key, and returns the
// result sorted deterministically. A nil input returns nil so an empty and
// an absent rule list keep comparing equal.
func canonicalizeRuleList(
	rules []*svcapitypes.IPPermission,
	selfID string,
	ownerAccountID string,
) []*svcapitypes.IPPermission {
	if rules == nil {
		return nil
	}

	// Per-rule field normalisation.
	for _, rule := range rules {
		if rule == nil {
			continue
		}
		// Gap 5: AWS returns well-known protocols by name ("tcp"), so map a
		// numeric spec value ("6") to the same name before comparing.
		rule.IPProtocol = canonicalizeProtocol(rule.IPProtocol)
		// Gap 1: AWS ignores and drops the port range for "-1" rules.
		if rule.IPProtocol != nil && *rule.IPProtocol == allProtocols {
			rule.FromPort = nil
			rule.ToPort = nil
		}
		// Gap 6: AWS canonicalises CIDRs (masks host bits; lowercases and
		// zero-compresses IPv6). Match that form on both sides.
		for _, r := range rule.IPRanges {
			if r != nil {
				r.CIDRIP = canonicalizeCIDR(r.CIDRIP)
			}
		}
		for _, r := range rule.IPv6Ranges {
			if r != nil {
				r.CIDRIPv6 = canonicalizeCIDR(r.CIDRIPv6)
			}
		}
		for _, pair := range rule.UserIDGroupPairs {
			canonicalizeGroupPair(pair, selfID, ownerAccountID)
		}
	}

	// Gap 3: aggregate rules that share the same (protocol, fromPort,
	// toPort) key, preserving first-seen order for stability before the
	// final sort.
	byKey := map[string]*svcapitypes.IPPermission{}
	order := make([]string, 0, len(rules))
	for _, rule := range rules {
		if rule == nil {
			continue
		}
		key := ruleAggregationKey(rule)
		if existing, ok := byKey[key]; ok {
			existing.IPRanges = append(existing.IPRanges, rule.IPRanges...)
			existing.IPv6Ranges = append(existing.IPv6Ranges, rule.IPv6Ranges...)
			existing.PrefixListIDs = append(existing.PrefixListIDs, rule.PrefixListIDs...)
			existing.UserIDGroupPairs = append(existing.UserIDGroupPairs, rule.UserIDGroupPairs...)
			continue
		}
		// Copy the rule and its grant slices so aggregation never aliases
		// or mutates the caller's backing arrays.
		merged := &svcapitypes.IPPermission{
			FromPort:         rule.FromPort,
			ToPort:           rule.ToPort,
			IPProtocol:       rule.IPProtocol,
			IPRanges:         append([]*svcapitypes.IPRange(nil), rule.IPRanges...),
			IPv6Ranges:       append([]*svcapitypes.IPv6Range(nil), rule.IPv6Ranges...),
			PrefixListIDs:    append([]*svcapitypes.PrefixListID(nil), rule.PrefixListIDs...),
			UserIDGroupPairs: append([]*svcapitypes.UserIDGroupPair(nil), rule.UserIDGroupPairs...),
		}
		byKey[key] = merged
		order = append(order, key)
	}

	out := make([]*svcapitypes.IPPermission, 0, len(order))
	for _, key := range order {
		rule := byKey[key]
		sortGrants(rule)
		out = append(out, rule)
	}
	// Sort rules by their aggregation key so read-back ordering never drives
	// a diff. Keys are unique after aggregation, so the order is total.
	sort.Slice(out, func(i, j int) bool {
		return ruleAggregationKey(out[i]) < ruleAggregationKey(out[j])
	})
	return out
}

// canonicalizeGroupPair normalises a single UserIDGroupPair in place. Its
// first act is to strip the spec-only reference wrappers (GroupRef/VPCRef) so
// they play no part whatsoever in the delta; it then covers the
// self-reference (gap driving #2822) and server-filled owner account (gap 2)
// cases against the resolved concrete IDs.
func canonicalizeGroupPair(
	pair *svcapitypes.UserIDGroupPair,
	selfID string,
	ownerAccountID string,
) {
	if pair == nil {
		return
	}
	// Step 1, before anything else: drop the spec-only reference wrappers.
	// GroupRef/VPCRef are resolved into their concrete IDs (GroupID/VPCID) by
	// ResolveReferences before the delta ever runs, and AWS never returns them
	// on read-back. By the time we compare, the resolved ID is authoritative
	// and the wrapper is dead weight, so it must not influence the delta in
	// any way. A groupRef that has not resolved yet (GroupID still nil) is not
	// this code's concern: the referencesResolved guard defers rule sync until
	// the referenced ID is populated, so we assume GroupID is resolved here.
	pair.GroupRef = nil
	pair.VPCRef = nil

	// A self-reference is an explicit reference to the SG's own (resolved) ID,
	// or the spec shorthand that omits every group identifier ("this SG",
	// since the ID is unknown until AWS assigns it). GroupRef is intentionally
	// absent from this test: it has already been cleared above.
	isSelf := (pair.GroupID != nil && selfID != "" && *pair.GroupID == selfID) ||
		(pair.GroupID == nil && pair.GroupName == nil)
	if isSelf {
		if selfID != "" {
			id := selfID
			pair.GroupID = &id
		}
		pair.GroupName = nil
		pair.UserID = nil
		return
	}
	// Non-self pairs: AWS auto-fills UserID with the owning account for
	// same-account grants. Clear it so its absence in the spec is not a
	// diff. Cross-account grants (UserID != owner) are preserved -- the
	// account is required to identify the referenced group.
	if pair.UserID != nil && ownerAccountID != "" && *pair.UserID == ownerAccountID {
		pair.UserID = nil
	}
}

// ruleAggregationKey returns the key AWS aggregates IpPermissions by:
// protocol plus the (from, to) port range.
func ruleAggregationKey(rule *svcapitypes.IPPermission) string {
	proto := ""
	if rule.IPProtocol != nil {
		proto = *rule.IPProtocol
	}
	from := "nil"
	if rule.FromPort != nil {
		from = strconv.FormatInt(*rule.FromPort, 10)
	}
	to := "nil"
	if rule.ToPort != nil {
		to = strconv.FormatInt(*rule.ToPort, 10)
	}
	return proto + "/" + from + "/" + to
}

// sortGrants sorts every grant collection of a rule into a deterministic
// order so that read-back ordering within a rule never drives a diff.
//
// The per-element sort keys are nil-safe (a nil element sorts as ""), matching
// the nil check in canonicalizeRuleList. Neither the CRD schema (which rejects
// null list entries) nor the read-back setter (which always allocates each
// element) can produce a nil element, so this is defensive only -- but it keeps
// the comparators panic-free and consistent with the rest of the file.
func sortGrants(rule *svcapitypes.IPPermission) {
	if rule == nil {
		return
	}
	sort.Slice(rule.IPRanges, func(i, j int) bool {
		return ipRangeSortKey(rule.IPRanges[i]) < ipRangeSortKey(rule.IPRanges[j])
	})
	sort.Slice(rule.IPv6Ranges, func(i, j int) bool {
		return ipv6RangeSortKey(rule.IPv6Ranges[i]) < ipv6RangeSortKey(rule.IPv6Ranges[j])
	})
	sort.Slice(rule.PrefixListIDs, func(i, j int) bool {
		return prefixListIDSortKey(rule.PrefixListIDs[i]) < prefixListIDSortKey(rule.PrefixListIDs[j])
	})
	sort.Slice(rule.UserIDGroupPairs, func(i, j int) bool {
		return userIDGroupPairSortKey(rule.UserIDGroupPairs[i]) <
			userIDGroupPairSortKey(rule.UserIDGroupPairs[j])
	})
}

// ipRangeSortKey builds a nil-safe stable sort key for an IPv4 CIDR grant.
func ipRangeSortKey(r *svcapitypes.IPRange) string {
	if r == nil {
		return ""
	}
	return derefStr(r.CIDRIP)
}

// ipv6RangeSortKey builds a nil-safe stable sort key for an IPv6 CIDR grant.
func ipv6RangeSortKey(r *svcapitypes.IPv6Range) string {
	if r == nil {
		return ""
	}
	return derefStr(r.CIDRIPv6)
}

// prefixListIDSortKey builds a nil-safe stable sort key for a prefix-list grant.
func prefixListIDSortKey(p *svcapitypes.PrefixListID) string {
	if p == nil {
		return ""
	}
	return derefStr(p.PrefixListID)
}

// userIDGroupPairSortKey builds a stable sort key for a group pair from the
// fields that identify it.
func userIDGroupPairSortKey(pair *svcapitypes.UserIDGroupPair) string {
	if pair == nil {
		return ""
	}
	return derefStr(pair.GroupID) + "/" + derefStr(pair.UserID) + "/" +
		derefStr(pair.GroupName)
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// syncSGRules analyzes desired and latest (if any)
// resources and executes API calls to Create/Delete
// rules in order to achieve desired state.
func (rm *resourceManager) syncSGRules(
	ctx context.Context,
	desired *resource,
	latest *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.syncSGRules")
	defer func() { exit(err) }()

	// Compare and authorise the canonical form of both sides so that
	// server-side normalisation (see customPostCompare) never revokes and
	// re-authorises an equivalent rule. Copies are used so neither desired
	// nor latest -- and therefore neither the persisted Spec nor the AWS
	// read-back reflected into it -- is mutated.
	desiredIngressRules, desiredEgressRules := canonicalizeCopiedRuleLists(desired)
	var latestIngressRules, latestEgressRules []*svcapitypes.IPPermission
	if latest != nil {
		latestIngressRules, latestEgressRules = canonicalizeCopiedRuleLists(latest)
	}

	toAddIngress := []*svcapitypes.IPPermission{}
	toAddEgress := []*svcapitypes.IPPermission{}
	toDeleteIngress := []*svcapitypes.IPPermission{}
	toDeleteEgress := []*svcapitypes.IPPermission{}

	for _, desiredIngress := range desiredIngressRules {
		if latest == nil || !containsRule(latestIngressRules, desiredIngress) {
			// a desired rule is not in the latest resource; therefore, create
			toAddIngress = append(toAddIngress, desiredIngress)
		}
	}
	for _, desiredEgress := range desiredEgressRules {
		if latest == nil || !containsRule(latestEgressRules, desiredEgress) {
			toAddEgress = append(toAddEgress, desiredEgress)
		}
	}
	if latest != nil {
		for _, latestIngress := range latestIngressRules {
			if !containsRule(desiredIngressRules, latestIngress) {
				// a rule is in latest resource, but not in desired resource; therefore, delete
				toDeleteIngress = append(toDeleteIngress, latestIngress)
			}
		}
		for _, latestEgress := range latestEgressRules {
			if !containsRule(desiredEgressRules, latestEgress) {
				toDeleteEgress = append(toDeleteEgress, latestEgress)
			}
		}
	}

	// Delete before create for the following reasons:
	// - Updating a rule requires that it be removed before the updated version be added.
	// - If there is an error with adding new rules, it occurs after deletion of old ones;
	//   This is safer and closer to achieving desired resource state.
	if err = rm.deleteSecurityGroupRules(ctx, latest, toDeleteIngress, toDeleteEgress); err != nil {
		return err
	}
	if err = rm.createSecurityGroupRules(ctx, desired, toAddIngress, toAddEgress); err != nil {
		return err
	}

	return nil
}

// updateTagSpecificationsInCreateRequest adds
// Tags defined in the Spec to CreateSecurityGroupInput.TagSpecification
// and ensures the ResourceType is always set to 'security-group'
func updateTagSpecificationsInCreateRequest(r *resource,
	input *svcsdk.CreateSecurityGroupInput) {
	input.TagSpecifications = nil
	desiredTagSpecs := svcsdktypes.TagSpecification{}
	if r.ko.Spec.Tags != nil {
		requestedTags := []svcsdktypes.Tag{}
		for _, desiredTag := range r.ko.Spec.Tags {
			// Add in tags defined in the Spec
			tag := svcsdktypes.Tag{}
			if desiredTag.Key != nil && desiredTag.Value != nil {
				tag.Key = desiredTag.Key
				tag.Value = desiredTag.Value
			}
			requestedTags = append(requestedTags, tag)
		}
		desiredTagSpecs.ResourceType = "security-group"
		desiredTagSpecs.Tags = requestedTags
		input.TagSpecifications = []svcsdktypes.TagSpecification{desiredTagSpecs}
	}
}

// createSecurityGroupRules takes a list of ingress and egress
// rules and attaches them to a SecurityGroup resource via
// AuthorizeSecurityGroup API calls
func (rm *resourceManager) createSecurityGroupRules(
	ctx context.Context,
	r *resource,
	ingress []*svcapitypes.IPPermission,
	egress []*svcapitypes.IPPermission,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.createSecurityGroupRules")
	defer func() { exit(err) }()

	ingressRules := []svcsdktypes.IpPermission{}

	// Authorize ingress rules
	for _, i := range ingress {
		ipInput, err := rm.newIPPermission(*i)
		if err != nil {
			return err
		}
		for j := range ipInput.UserIdGroupPairs {
			// If not provided, we need to default the security group ID and vpc ID.
			//
			// The newIPPermission function doesn't return nil UserIdGroupPair items. It is safe to
			// access them here.
			if ipInput.UserIdGroupPairs[j].GroupId == nil && ipInput.UserIdGroupPairs[j].GroupName == nil {
				ipInput.UserIdGroupPairs[j].GroupId = r.ko.Status.ID
			}
			if ipInput.UserIdGroupPairs[j].VpcId == nil {
				ipInput.UserIdGroupPairs[j].VpcId = r.ko.Spec.VPCID
			}
		}
		ingressRules = append(ingressRules, *ipInput)
	}

	// API can only handle 1000 rules at a time. Send in batches of 1000.
	for i := 0; i < len(ingressRules); i += 1000 {
		end := i + 1000
		if end > len(ingressRules) {
			end = len(ingressRules)
		}
		req := &svcsdk.AuthorizeSecurityGroupIngressInput{
			GroupId:       r.ko.Status.ID,
			IpPermissions: ingressRules[i:end],
		}
		_, err = rm.sdkapi.AuthorizeSecurityGroupIngress(ctx, req)
		rm.metrics.RecordAPICall("CREATE", "AuthorizeSecurityGroupIngress", err)
		if err != nil {
			return err
		}
	}

	egressRules := []svcsdktypes.IpPermission{}
	// Authorize egress rules
	for _, e := range egress {
		ipInput, err := rm.newIPPermission(*e)
		if err != nil {
			return err
		}
		for j := range ipInput.UserIdGroupPairs {
			// If not provided, we need to default the security group ID and vpc ID.
			//
			// The newIPPermission function doesn't return nil UserIdGroupPair items. It is safe to
			// access them here.
			if ipInput.UserIdGroupPairs[j].GroupId == nil && ipInput.UserIdGroupPairs[j].GroupName == nil {
				ipInput.UserIdGroupPairs[j].GroupId = r.ko.Status.ID
			}
			if ipInput.UserIdGroupPairs[j].VpcId == nil {
				ipInput.UserIdGroupPairs[j].VpcId = r.ko.Spec.VPCID
			}
		}
		egressRules = append(egressRules, *ipInput)
	}

	// API can only handle 1000 rules at a time. Send in batches of 1000.
	for i := 0; i < len(egressRules); i += 1000 {
		end := i + 1000
		if end > len(egressRules) {
			end = len(egressRules)
		}
		req := &svcsdk.AuthorizeSecurityGroupEgressInput{
			GroupId:       r.ko.Status.ID,
			IpPermissions: egressRules[i:end],
		}
		_, err = rm.sdkapi.AuthorizeSecurityGroupEgress(ctx, req)
		rm.metrics.RecordAPICall("CREATE", "AuthorizeSecurityGroupEgress", err)
		if err != nil {
			return err
		}
	}

	return err
}

// deleteDefaultSecurityGroupRule deletes the default
// egress rule that is attached to a SecurityGroup upon creation.
// The rule is set explicitly (without helpers); otherwise, the quotes
// around IpProtocol value will not be set properly
func (rm *resourceManager) deleteDefaultSecurityGroupRule(
	ctx context.Context,
	r *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.deleteDefaultSecurityGroupRule")
	defer func() { exit(err) }()

	ipRange := svcsdktypes.IpRange{
		CidrIp: toStrPtr("0.0.0.0/0"),
	}
	input := svcsdktypes.IpPermission{
		FromPort:   aws.Int32(-1),
		ToPort:     aws.Int32(-1),
		IpProtocol: toStrPtr("-1"),
		IpRanges:   []svcsdktypes.IpRange{ipRange},
	}
	req := &svcsdk.RevokeSecurityGroupEgressInput{
		GroupId:       r.ko.Status.ID,
		IpPermissions: []svcsdktypes.IpPermission{input},
	}
	_, err = rm.sdkapi.RevokeSecurityGroupEgress(ctx, req)
	rm.metrics.RecordAPICall("DELETE", "RevokeSecurityGroupEgress", err)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case "InvalidPermission.NotFound":
				return
			}
		}
		return err
	}

	return err
}

// deleteSecurityGroupRules takes a list of ingress and egress
// rules and removes them from a SecurityGroup resource via
// RevokeSecurityGroup API calls
func (rm *resourceManager) deleteSecurityGroupRules(
	ctx context.Context,
	r *resource,
	ingress []*svcapitypes.IPPermission,
	egress []*svcapitypes.IPPermission,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.deleteSecurityGroupRules")
	defer func() { exit(err) }()

	// Revoke ingress rules
	ingressRules := []svcsdktypes.IpPermission{}
	for _, i := range ingress {
		ipInput, err := rm.newIPPermission(*i)
		if err != nil {
			return err
		}
		for j := range ipInput.UserIdGroupPairs {
			// If not provided, we need to default the security group ID and vpc ID.
			//
			// The newIPPermission function doesn't return nil UserIdGroupPair items. It is safe to
			// access them here.
			if ipInput.UserIdGroupPairs[j].GroupId == nil && ipInput.UserIdGroupPairs[j].GroupName == nil {
				ipInput.UserIdGroupPairs[j].GroupId = r.ko.Status.ID
			}
			if ipInput.UserIdGroupPairs[j].VpcId == nil {
				ipInput.UserIdGroupPairs[j].VpcId = r.ko.Spec.VPCID
			}
		}
		ingressRules = append(ingressRules, *ipInput)
	}

	// API can only handle 1000 rules at a time. Send in batches of 1000.
	for i := 0; i < len(ingressRules); i += 1000 {
		end := i + 1000
		if end > len(ingressRules) {
			end = len(ingressRules)
		}
		req := &svcsdk.RevokeSecurityGroupIngressInput{
			GroupId:       r.ko.Status.ID,
			IpPermissions: ingressRules[i:end],
		}
		_, err = rm.sdkapi.RevokeSecurityGroupIngress(ctx, req)
		rm.metrics.RecordAPICall("DELETE", "RevokeSecurityGroupIngress", err)
		if err != nil {
			return err
		}
	}

	// Revoke egress rules
	egressRules := []svcsdktypes.IpPermission{}
	for _, e := range egress {
		ipInput, err := rm.newIPPermission(*e)
		if err != nil {
			return err
		}
		for j := range ipInput.UserIdGroupPairs {
			// If not provided, we need to default the security group ID and vpc ID.
			//
			// The newIPPermission function doesn't return nil UserIdGroupPair items. It is safe to
			// access them here.
			if ipInput.UserIdGroupPairs[j].GroupId == nil && ipInput.UserIdGroupPairs[j].GroupName == nil {
				ipInput.UserIdGroupPairs[j].GroupId = r.ko.Status.ID
			}
			if ipInput.UserIdGroupPairs[j].VpcId == nil {
				ipInput.UserIdGroupPairs[j].VpcId = r.ko.Spec.VPCID
			}
		}
		egressRules = append(egressRules, *ipInput)
	}

	// API can only handle 1000 rules at a time. Send in batches of 1000.
	for i := 0; i < len(egressRules); i += 1000 {
		end := i + 1000
		if end > len(egressRules) {
			end = len(egressRules)
		}
		req := &svcsdk.RevokeSecurityGroupEgressInput{
			GroupId:       r.ko.Status.ID,
			IpPermissions: egressRules[i:end],
		}
		_, err = rm.sdkapi.RevokeSecurityGroupEgress(ctx, req)
		rm.metrics.RecordAPICall("DELETE", "RevokeSecurityGroupEgress", err)
		if err != nil {
			return err
		}
	}

	return err
}

// customUpdateSecurityGroup updates IngressRules and/or
// EgressRules, if a delta be detected between resources.
func (rm *resourceManager) customUpdateSecurityGroup(
	ctx context.Context,
	desired *resource,
	latest *resource,
	delta *ackcompare.Delta,
) (updated *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.customUpdateSecurityGroup")
	defer exit(err)

	// Default `updated` to `desired` because it is likely
	// EC2 `modify` APIs do NOT return output, only errors.
	// If the `modify` calls (i.e. `sync`) do NOT return
	// an error, then the update was successful and desired.Spec
	// (now updated.Spec) reflects the latest resource state.
	updated = rm.concreteResource(desired.DeepCopy())

	if delta.DifferentAt("Spec.IngressRules") || delta.DifferentAt("Spec.EgressRules") {
		if !rm.referencesResolved(updated) {
			ackcondition.SetSynced(updated, corev1.ConditionFalse, nil, nil)
			return updated, nil
		}

		if err := rm.syncSGRules(ctx, desired, latest); err != nil {
			return nil, err
		}
		// A ReadOne call for SecurityGroup Rules (NOT SecurityGroups)
		// is made to refresh Status.Rules with the recently-updated
		// data from the above `sync` call
		if rules, err := rm.getRules(ctx, desired); err != nil {
			return nil, err
		} else {
			updated.ko.Status.Rules = rules
		}
	}

	if delta.DifferentAt("Spec.Tags") {
		if err := tags.Sync(
			ctx, rm.sdkapi, rm.metrics, *latest.ko.Status.ID,
			desired.ko.Spec.Tags, latest.ko.Spec.Tags,
		); err != nil {
			return nil, err
		}
	}

	return updated, nil
}

// containsRule returns true if security group rule
// is found in the rule collection (all fields must match);
// otherwise, return false.
func containsRule(
	ruleCollection []*svcapitypes.IPPermission,
	rule *svcapitypes.IPPermission,
) bool {
	if ruleCollection == nil || rule == nil {
		return false
	}

	for _, r := range ruleCollection {
		delta := compareIPPermission(r, rule)
		if len(delta.Differences) == 0 {
			return true
		}
	}
	return false
}

func toStrPtr(str string) *string {
	return &str
}

func toInt64Ptr(integer int64) *int64 {
	return &integer
}

func (rm *resourceManager) getSecurityGroupID(
	ctx context.Context,
	r *resource,
) (id *string, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.getSecurityGroupID")
	defer func() {
		exit(err)
	}()

	// Both name and VPC ID are required for safe lookup
	if r.ko.Spec.Name == nil || r.ko.Spec.VPCID == nil {
		return nil, nil
	}

	// Build filters for name and VPC ID
	filters := []svcsdktypes.Filter{
		{
			Name:   aws.String("group-name"),
			Values: []string{*r.ko.Spec.Name},
		},
		{
			Name:   aws.String("vpc-id"),
			Values: []string{*r.ko.Spec.VPCID},
		},
	}

	resp, err := rm.sdkapi.DescribeSecurityGroups(ctx, &svcsdk.DescribeSecurityGroupsInput{
		Filters: filters,
	})
	if err != nil {
		return nil, err
	}

	if resp == nil || len(resp.SecurityGroups) == 0 {
		return nil, nil
	}

	// Security group names are unique within a VPC, so there should be exactly one match
	return resp.SecurityGroups[0].GroupId, nil
}

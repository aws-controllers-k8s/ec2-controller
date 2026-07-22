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
	"strings"

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

// allProtocols is the IpProtocol value for "all protocols"; AWS drops the port
// range for such rules on read-back.
const allProtocols = "-1"

// wellKnownProtocols are the IANA protocol numbers EC2 returns by name on
// read-back, so a numeric spec value ("6") would otherwise differ forever from
// the name ("tcp"). Other protocols (and "-1") are already canonical.
var wellKnownProtocols = map[string]string{
	"1":  "icmp",
	"6":  "tcp",
	"17": "udp",
	"58": "icmpv6",
}

// canonicalizeProtocol maps a numeric IpProtocol to the name EC2 returns;
// unknown protocols and names pass through unchanged.
func canonicalizeProtocol(proto *string) *string {
	if proto == nil {
		return nil
	}
	if name, ok := wellKnownProtocols[*proto]; ok {
		return &name
	}
	return proto
}

// canonicalizeCIDR rewrites a CIDR to the exact network form the EC2 backend
// stores. It is a direct port of com.amazon.ec2.nm.CidrBlock.canonicalizeAddress
// (package EC2-NM-Common, the class the EC2 control plane uses): a network mask
// is built one byte at a time from the prefix length and ANDed with the address,
// masking off every host bit. The masked address is rendered in the same text
// form AWS returns on read-back -- dotted-quad for IPv4, RFC 5952 lowercase
// zero-compressed for IPv6 (matching the backend's Guava
// InetAddresses.toAddrString) -- so "100.68.0.18/18" -> "100.68.0.0/18" and
// "2001:DB8:abcd:0012::1/64" -> "2001:db8:abcd:12::/64".
//
// Applying the backend's own conversion here, before both the delta comparison
// and the Authorize/Revoke request, is what stops the spurious diff: the desired
// spec form and the AWS read-back form collapse to the identical network.
//
// A value that is not a well-formed CIDR -- or whose prefix is longer than the
// address, which the backend rejects -- passes through unchanged.
func canonicalizeCIDR(cidr *string) *string {
	if cidr == nil {
		return nil
	}
	slash := strings.LastIndex(*cidr, "/")
	if slash < 0 {
		return cidr
	}
	ip := net.ParseIP((*cidr)[:slash])
	prefixLength, err := strconv.Atoi((*cidr)[slash+1:])
	if ip == nil || err != nil || prefixLength < 0 {
		return cidr
	}

	// Match the backend's byte layout: 4 bytes for IPv4, 16 for IPv6.
	addr := ip.To4()
	if addr == nil {
		addr = ip.To16()
	}
	if addr == nil {
		return cidr
	}

	// Build the network mask exactly as CidrBlock.canonicalizeAddress does --
	// one byte at a time -- and AND it into the address. Bytes past the prefix
	// stay zero, which is the host-bit masking.
	remaining := prefixLength
	network := make(net.IP, len(addr))
	for i := 0; i < len(addr) && remaining > 0; i++ {
		var maskByte byte
		if remaining > 8 {
			maskByte = 0xff
			remaining -= 8
		} else {
			maskByte = byte(((1 << remaining) - 1) << (8 - remaining))
			remaining = 0
		}
		network[i] = addr[i] & maskByte
	}
	// remaining != 0 means the prefix is longer than the address (e.g. /40 on an
	// IPv4 address); the backend treats this as malformed, so leave it untouched.
	if remaining != 0 {
		return cidr
	}

	canon := network.String() + "/" + strconv.Itoa(prefixLength)
	return &canon
}

// customPostCompare compares Spec.IngressRules/EgressRules, which the generated
// delta skips (they are marked compare.is_ignored). It normalises canonical
// *copies* of both sides -- never a or b, which become the merge-patch base in
// patchResourceMetadataAndSpec -- so the server-side rule normalisation behind
// aws-controllers-k8s/community#2822 never drives a spurious diff, while genuine
// changes still register. See canonicalizeRuleList and canonicalizeGroupPair for
// the specific normalisations.
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

// canonicalizeCopiedRuleLists returns canonical *copies* of r's rule lists; r is
// never mutated. Used for comparison (customPostCompare) and to build AWS
// Authorize/Revoke inputs (syncSGRules).
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

// deepCopyRuleList deep-copies a rule list so canonicalisation never touches the
// caller's data. nil in -> nil out (absent and empty stay equal).
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

// canonicalizeRuleList normalises each rule and its grants, aggregates rules
// sharing the same (protocol, fromPort, toPort) key, and returns them
// deterministically sorted. nil in -> nil out.
func canonicalizeRuleList(
	rules []*svcapitypes.IPPermission,
	selfID string,
	ownerAccountID string,
) []*svcapitypes.IPPermission {
	if rules == nil {
		return nil
	}

	// Per-rule field normalisation to match the AWS read-back form.
	for _, rule := range rules {
		if rule == nil {
			continue
		}
		rule.IPProtocol = canonicalizeProtocol(rule.IPProtocol)
		if rule.IPProtocol != nil && *rule.IPProtocol == allProtocols {
			rule.FromPort = nil // AWS drops the port range for "-1" rules.
			rule.ToPort = nil
		}
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

	// AWS merges grants sharing a (protocol, fromPort, toPort) key into one
	// IpPermission; aggregate the same way, keeping first-seen order.
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
		// Copy so aggregation never aliases the caller's backing arrays.
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
	// Sort by aggregation key (unique after aggregation) so read-back ordering
	// never drives a diff.
	sort.Slice(out, func(i, j int) bool {
		return ruleAggregationKey(out[i]) < ruleAggregationKey(out[j])
	})
	return out
}

// canonicalizeGroupPair normalises a single UserIDGroupPair (on a copy).
func canonicalizeGroupPair(
	pair *svcapitypes.UserIDGroupPair,
	selfID string,
	ownerAccountID string,
) {
	if pair == nil {
		return
	}
	// GroupRef/VPCRef are spec-only wrappers ResolveReferences turns into
	// concrete IDs before the delta and AWS never returns; drop them so they
	// never drive a diff. GroupID is assumed resolved (referencesResolved
	// defers sync until it is).
	pair.GroupRef = nil
	pair.VPCRef = nil

	// Self-reference: GroupID == selfID, or all group identifiers omitted (the
	// spec shorthand, since the ID is unknown until AWS assigns it). AWS fills
	// GroupID/UserID/GroupName on read-back, so canonicalise to {GroupID: selfID}.
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
	// AWS fills UserID with the owner account for same-account grants; clear it.
	// Cross-account grants (UserID != owner) are kept -- the account identifies
	// the referenced group.
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

// sortGrants deterministically orders a rule's grant collections so read-back
// ordering never drives a diff. Sort keys are nil-safe (defensive; the schema
// and setters never emit nil elements).
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

	// Compare and authorise canonical copies so server-side normalisation never
	// re-authorises an equivalent rule, without mutating desired/latest (and so
	// the persisted Spec).
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

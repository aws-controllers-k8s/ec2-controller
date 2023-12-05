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

// Code generated by ack-generate. DO NOT EDIT.

package vpc_peering_connection

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	ackv1alpha1 "github.com/aws-controllers-k8s/runtime/apis/core/v1alpha1"
	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackcondition "github.com/aws-controllers-k8s/runtime/pkg/condition"
	ackerr "github.com/aws-controllers-k8s/runtime/pkg/errors"
	ackrequeue "github.com/aws-controllers-k8s/runtime/pkg/requeue"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	"github.com/aws/aws-sdk-go/aws"
	svcsdk "github.com/aws/aws-sdk-go/service/ec2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	svcapitypes "github.com/aws-controllers-k8s/ec2-controller/apis/v1alpha1"
)

// Hack to avoid import errors during build...
var (
	_ = &metav1.Time{}
	_ = strings.ToLower("")
	_ = &aws.JSONValue{}
	_ = &svcsdk.EC2{}
	_ = &svcapitypes.VPCPeeringConnection{}
	_ = ackv1alpha1.AWSAccountID("")
	_ = &ackerr.NotFound
	_ = &ackcondition.NotManagedMessage
	_ = &reflect.Value{}
	_ = fmt.Sprintf("")
	_ = &ackrequeue.NoRequeue{}
)

// sdkFind returns SDK-specific information about a supplied resource
func (rm *resourceManager) sdkFind(
	ctx context.Context,
	r *resource,
) (latest *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.sdkFind")
	defer func() {
		exit(err)
	}()
	// If any required fields in the input shape are missing, AWS resource is
	// not created yet. Return NotFound here to indicate to callers that the
	// resource isn't yet created.
	if rm.requiredFieldsMissingFromReadManyInput(r) {
		return nil, ackerr.NotFound
	}

	input, err := rm.newListRequestPayload(r)
	if err != nil {
		return nil, err
	}
	var resp *svcsdk.DescribeVpcPeeringConnectionsOutput
	resp, err = rm.sdkapi.DescribeVpcPeeringConnectionsWithContext(ctx, input)
	rm.metrics.RecordAPICall("READ_MANY", "DescribeVpcPeeringConnections", err)
	if err != nil {
		if awsErr, ok := ackerr.AWSError(err); ok && awsErr.Code() == "UNKNOWN" {
			return nil, ackerr.NotFound
		}
		return nil, err
	}

	// Merge in the information we read from the API call above to the copy of
	// the original Kubernetes object we passed to the function
	ko := r.ko.DeepCopy()

	found := false
	for _, elem := range resp.VpcPeeringConnections {
		if elem.AccepterVpcInfo != nil {
			f0 := &svcapitypes.VPCPeeringConnectionVPCInfo{}
			if elem.AccepterVpcInfo.CidrBlock != nil {
				f0.CIDRBlock = elem.AccepterVpcInfo.CidrBlock
			}
			if elem.AccepterVpcInfo.CidrBlockSet != nil {
				f0f1 := []*svcapitypes.CIDRBlock{}
				for _, f0f1iter := range elem.AccepterVpcInfo.CidrBlockSet {
					f0f1elem := &svcapitypes.CIDRBlock{}
					if f0f1iter.CidrBlock != nil {
						f0f1elem.CIDRBlock = f0f1iter.CidrBlock
					}
					f0f1 = append(f0f1, f0f1elem)
				}
				f0.CIDRBlockSet = f0f1
			}
			if elem.AccepterVpcInfo.Ipv6CidrBlockSet != nil {
				f0f2 := []*svcapitypes.IPv6CIDRBlock{}
				for _, f0f2iter := range elem.AccepterVpcInfo.Ipv6CidrBlockSet {
					f0f2elem := &svcapitypes.IPv6CIDRBlock{}
					if f0f2iter.Ipv6CidrBlock != nil {
						f0f2elem.IPv6CIDRBlock = f0f2iter.Ipv6CidrBlock
					}
					f0f2 = append(f0f2, f0f2elem)
				}
				f0.IPv6CIDRBlockSet = f0f2
			}
			if elem.AccepterVpcInfo.OwnerId != nil {
				f0.OwnerID = elem.AccepterVpcInfo.OwnerId
			}
			if elem.AccepterVpcInfo.PeeringOptions != nil {
				f0f4 := &svcapitypes.VPCPeeringConnectionOptionsDescription{}
				if elem.AccepterVpcInfo.PeeringOptions.AllowDnsResolutionFromRemoteVpc != nil {
					f0f4.AllowDNSResolutionFromRemoteVPC = elem.AccepterVpcInfo.PeeringOptions.AllowDnsResolutionFromRemoteVpc
				}
				if elem.AccepterVpcInfo.PeeringOptions.AllowEgressFromLocalClassicLinkToRemoteVpc != nil {
					f0f4.AllowEgressFromLocalClassicLinkToRemoteVPC = elem.AccepterVpcInfo.PeeringOptions.AllowEgressFromLocalClassicLinkToRemoteVpc
				}
				if elem.AccepterVpcInfo.PeeringOptions.AllowEgressFromLocalVpcToRemoteClassicLink != nil {
					f0f4.AllowEgressFromLocalVPCToRemoteClassicLink = elem.AccepterVpcInfo.PeeringOptions.AllowEgressFromLocalVpcToRemoteClassicLink
				}
				f0.PeeringOptions = f0f4
			}
			if elem.AccepterVpcInfo.Region != nil {
				f0.Region = elem.AccepterVpcInfo.Region
			}
			if elem.AccepterVpcInfo.VpcId != nil {
				f0.VPCID = elem.AccepterVpcInfo.VpcId
			}
			ko.Status.AccepterVPCInfo = f0
		} else {
			ko.Status.AccepterVPCInfo = nil
		}
		if elem.ExpirationTime != nil {
			ko.Status.ExpirationTime = &metav1.Time{*elem.ExpirationTime}
		} else {
			ko.Status.ExpirationTime = nil
		}
		if elem.RequesterVpcInfo != nil {
			f2 := &svcapitypes.VPCPeeringConnectionVPCInfo{}
			if elem.RequesterVpcInfo.CidrBlock != nil {
				f2.CIDRBlock = elem.RequesterVpcInfo.CidrBlock
			}
			if elem.RequesterVpcInfo.CidrBlockSet != nil {
				f2f1 := []*svcapitypes.CIDRBlock{}
				for _, f2f1iter := range elem.RequesterVpcInfo.CidrBlockSet {
					f2f1elem := &svcapitypes.CIDRBlock{}
					if f2f1iter.CidrBlock != nil {
						f2f1elem.CIDRBlock = f2f1iter.CidrBlock
					}
					f2f1 = append(f2f1, f2f1elem)
				}
				f2.CIDRBlockSet = f2f1
			}
			if elem.RequesterVpcInfo.Ipv6CidrBlockSet != nil {
				f2f2 := []*svcapitypes.IPv6CIDRBlock{}
				for _, f2f2iter := range elem.RequesterVpcInfo.Ipv6CidrBlockSet {
					f2f2elem := &svcapitypes.IPv6CIDRBlock{}
					if f2f2iter.Ipv6CidrBlock != nil {
						f2f2elem.IPv6CIDRBlock = f2f2iter.Ipv6CidrBlock
					}
					f2f2 = append(f2f2, f2f2elem)
				}
				f2.IPv6CIDRBlockSet = f2f2
			}
			if elem.RequesterVpcInfo.OwnerId != nil {
				f2.OwnerID = elem.RequesterVpcInfo.OwnerId
			}
			if elem.RequesterVpcInfo.PeeringOptions != nil {
				f2f4 := &svcapitypes.VPCPeeringConnectionOptionsDescription{}
				if elem.RequesterVpcInfo.PeeringOptions.AllowDnsResolutionFromRemoteVpc != nil {
					f2f4.AllowDNSResolutionFromRemoteVPC = elem.RequesterVpcInfo.PeeringOptions.AllowDnsResolutionFromRemoteVpc
				}
				if elem.RequesterVpcInfo.PeeringOptions.AllowEgressFromLocalClassicLinkToRemoteVpc != nil {
					f2f4.AllowEgressFromLocalClassicLinkToRemoteVPC = elem.RequesterVpcInfo.PeeringOptions.AllowEgressFromLocalClassicLinkToRemoteVpc
				}
				if elem.RequesterVpcInfo.PeeringOptions.AllowEgressFromLocalVpcToRemoteClassicLink != nil {
					f2f4.AllowEgressFromLocalVPCToRemoteClassicLink = elem.RequesterVpcInfo.PeeringOptions.AllowEgressFromLocalVpcToRemoteClassicLink
				}
				f2.PeeringOptions = f2f4
			}
			if elem.RequesterVpcInfo.Region != nil {
				f2.Region = elem.RequesterVpcInfo.Region
			}
			if elem.RequesterVpcInfo.VpcId != nil {
				f2.VPCID = elem.RequesterVpcInfo.VpcId
			}
			ko.Status.RequesterVPCInfo = f2
		} else {
			ko.Status.RequesterVPCInfo = nil
		}
		if elem.Status != nil {
			f3 := &svcapitypes.VPCPeeringConnectionStateReason{}
			if elem.Status.Code != nil {
				f3.Code = elem.Status.Code
			}
			if elem.Status.Message != nil {
				f3.Message = elem.Status.Message
			}
			ko.Status.Status = f3
		} else {
			ko.Status.Status = nil
		}
		if elem.Tags != nil {
			f4 := []*svcapitypes.Tag{}
			for _, f4iter := range elem.Tags {
				f4elem := &svcapitypes.Tag{}
				if f4iter.Key != nil {
					f4elem.Key = f4iter.Key
				}
				if f4iter.Value != nil {
					f4elem.Value = f4iter.Value
				}
				f4 = append(f4, f4elem)
			}
			ko.Spec.Tags = f4
		} else {
			ko.Spec.Tags = nil
		}
		if elem.VpcPeeringConnectionId != nil {
			ko.Status.VPCPeeringConnectionID = elem.VpcPeeringConnectionId
		} else {
			ko.Status.VPCPeeringConnectionID = nil
		}
		found = true
		break
	}
	if !found {
		return nil, ackerr.NotFound
	}

	rm.setStatusDefaults(ko)
	return &resource{ko}, nil
}

// requiredFieldsMissingFromReadManyInput returns true if there are any fields
// for the ReadMany Input shape that are required but not present in the
// resource's Spec or Status
func (rm *resourceManager) requiredFieldsMissingFromReadManyInput(
	r *resource,
) bool {
	return r.ko.Status.VPCPeeringConnectionID == nil

}

// newListRequestPayload returns SDK-specific struct for the HTTP request
// payload of the List API call for the resource
func (rm *resourceManager) newListRequestPayload(
	r *resource,
) (*svcsdk.DescribeVpcPeeringConnectionsInput, error) {
	res := &svcsdk.DescribeVpcPeeringConnectionsInput{}

	if r.ko.Status.VPCPeeringConnectionID != nil {
		f4 := []*string{}
		f4 = append(f4, r.ko.Status.VPCPeeringConnectionID)
		res.SetVpcPeeringConnectionIds(f4)
	}

	return res, nil
}

// sdkCreate creates the supplied resource in the backend AWS service API and
// returns a copy of the resource with resource fields (in both Spec and
// Status) filled in with values from the CREATE API operation's Output shape.
func (rm *resourceManager) sdkCreate(
	ctx context.Context,
	desired *resource,
) (created *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.sdkCreate")
	defer func() {
		exit(err)
	}()
	input, err := rm.newCreateRequestPayload(ctx, desired)
	if err != nil {
		return nil, err
	}
	updateTagSpecificationsInCreateRequest(desired, input)

	var resp *svcsdk.CreateVpcPeeringConnectionOutput
	_ = resp
	resp, err = rm.sdkapi.CreateVpcPeeringConnectionWithContext(ctx, input)
	rm.metrics.RecordAPICall("CREATE", "CreateVpcPeeringConnection", err)
	if err != nil {
		return nil, err
	}
	// Merge in the information we read from the API call above to the copy of
	// the original Kubernetes object we passed to the function
	ko := desired.ko.DeepCopy()

	if resp.VpcPeeringConnection.AccepterVpcInfo != nil {
		f0 := &svcapitypes.VPCPeeringConnectionVPCInfo{}
		if resp.VpcPeeringConnection.AccepterVpcInfo.CidrBlock != nil {
			f0.CIDRBlock = resp.VpcPeeringConnection.AccepterVpcInfo.CidrBlock
		}
		if resp.VpcPeeringConnection.AccepterVpcInfo.CidrBlockSet != nil {
			f0f1 := []*svcapitypes.CIDRBlock{}
			for _, f0f1iter := range resp.VpcPeeringConnection.AccepterVpcInfo.CidrBlockSet {
				f0f1elem := &svcapitypes.CIDRBlock{}
				if f0f1iter.CidrBlock != nil {
					f0f1elem.CIDRBlock = f0f1iter.CidrBlock
				}
				f0f1 = append(f0f1, f0f1elem)
			}
			f0.CIDRBlockSet = f0f1
		}
		if resp.VpcPeeringConnection.AccepterVpcInfo.Ipv6CidrBlockSet != nil {
			f0f2 := []*svcapitypes.IPv6CIDRBlock{}
			for _, f0f2iter := range resp.VpcPeeringConnection.AccepterVpcInfo.Ipv6CidrBlockSet {
				f0f2elem := &svcapitypes.IPv6CIDRBlock{}
				if f0f2iter.Ipv6CidrBlock != nil {
					f0f2elem.IPv6CIDRBlock = f0f2iter.Ipv6CidrBlock
				}
				f0f2 = append(f0f2, f0f2elem)
			}
			f0.IPv6CIDRBlockSet = f0f2
		}
		if resp.VpcPeeringConnection.AccepterVpcInfo.OwnerId != nil {
			f0.OwnerID = resp.VpcPeeringConnection.AccepterVpcInfo.OwnerId
		}
		if resp.VpcPeeringConnection.AccepterVpcInfo.PeeringOptions != nil {
			f0f4 := &svcapitypes.VPCPeeringConnectionOptionsDescription{}
			if resp.VpcPeeringConnection.AccepterVpcInfo.PeeringOptions.AllowDnsResolutionFromRemoteVpc != nil {
				f0f4.AllowDNSResolutionFromRemoteVPC = resp.VpcPeeringConnection.AccepterVpcInfo.PeeringOptions.AllowDnsResolutionFromRemoteVpc
			}
			if resp.VpcPeeringConnection.AccepterVpcInfo.PeeringOptions.AllowEgressFromLocalClassicLinkToRemoteVpc != nil {
				f0f4.AllowEgressFromLocalClassicLinkToRemoteVPC = resp.VpcPeeringConnection.AccepterVpcInfo.PeeringOptions.AllowEgressFromLocalClassicLinkToRemoteVpc
			}
			if resp.VpcPeeringConnection.AccepterVpcInfo.PeeringOptions.AllowEgressFromLocalVpcToRemoteClassicLink != nil {
				f0f4.AllowEgressFromLocalVPCToRemoteClassicLink = resp.VpcPeeringConnection.AccepterVpcInfo.PeeringOptions.AllowEgressFromLocalVpcToRemoteClassicLink
			}
			f0.PeeringOptions = f0f4
		}
		if resp.VpcPeeringConnection.AccepterVpcInfo.Region != nil {
			f0.Region = resp.VpcPeeringConnection.AccepterVpcInfo.Region
		}
		if resp.VpcPeeringConnection.AccepterVpcInfo.VpcId != nil {
			f0.VPCID = resp.VpcPeeringConnection.AccepterVpcInfo.VpcId
		}
		ko.Status.AccepterVPCInfo = f0
	} else {
		ko.Status.AccepterVPCInfo = nil
	}
	if resp.VpcPeeringConnection.ExpirationTime != nil {
		ko.Status.ExpirationTime = &metav1.Time{*resp.VpcPeeringConnection.ExpirationTime}
	} else {
		ko.Status.ExpirationTime = nil
	}
	if resp.VpcPeeringConnection.RequesterVpcInfo != nil {
		f2 := &svcapitypes.VPCPeeringConnectionVPCInfo{}
		if resp.VpcPeeringConnection.RequesterVpcInfo.CidrBlock != nil {
			f2.CIDRBlock = resp.VpcPeeringConnection.RequesterVpcInfo.CidrBlock
		}
		if resp.VpcPeeringConnection.RequesterVpcInfo.CidrBlockSet != nil {
			f2f1 := []*svcapitypes.CIDRBlock{}
			for _, f2f1iter := range resp.VpcPeeringConnection.RequesterVpcInfo.CidrBlockSet {
				f2f1elem := &svcapitypes.CIDRBlock{}
				if f2f1iter.CidrBlock != nil {
					f2f1elem.CIDRBlock = f2f1iter.CidrBlock
				}
				f2f1 = append(f2f1, f2f1elem)
			}
			f2.CIDRBlockSet = f2f1
		}
		if resp.VpcPeeringConnection.RequesterVpcInfo.Ipv6CidrBlockSet != nil {
			f2f2 := []*svcapitypes.IPv6CIDRBlock{}
			for _, f2f2iter := range resp.VpcPeeringConnection.RequesterVpcInfo.Ipv6CidrBlockSet {
				f2f2elem := &svcapitypes.IPv6CIDRBlock{}
				if f2f2iter.Ipv6CidrBlock != nil {
					f2f2elem.IPv6CIDRBlock = f2f2iter.Ipv6CidrBlock
				}
				f2f2 = append(f2f2, f2f2elem)
			}
			f2.IPv6CIDRBlockSet = f2f2
		}
		if resp.VpcPeeringConnection.RequesterVpcInfo.OwnerId != nil {
			f2.OwnerID = resp.VpcPeeringConnection.RequesterVpcInfo.OwnerId
		}
		if resp.VpcPeeringConnection.RequesterVpcInfo.PeeringOptions != nil {
			f2f4 := &svcapitypes.VPCPeeringConnectionOptionsDescription{}
			if resp.VpcPeeringConnection.RequesterVpcInfo.PeeringOptions.AllowDnsResolutionFromRemoteVpc != nil {
				f2f4.AllowDNSResolutionFromRemoteVPC = resp.VpcPeeringConnection.RequesterVpcInfo.PeeringOptions.AllowDnsResolutionFromRemoteVpc
			}
			if resp.VpcPeeringConnection.RequesterVpcInfo.PeeringOptions.AllowEgressFromLocalClassicLinkToRemoteVpc != nil {
				f2f4.AllowEgressFromLocalClassicLinkToRemoteVPC = resp.VpcPeeringConnection.RequesterVpcInfo.PeeringOptions.AllowEgressFromLocalClassicLinkToRemoteVpc
			}
			if resp.VpcPeeringConnection.RequesterVpcInfo.PeeringOptions.AllowEgressFromLocalVpcToRemoteClassicLink != nil {
				f2f4.AllowEgressFromLocalVPCToRemoteClassicLink = resp.VpcPeeringConnection.RequesterVpcInfo.PeeringOptions.AllowEgressFromLocalVpcToRemoteClassicLink
			}
			f2.PeeringOptions = f2f4
		}
		if resp.VpcPeeringConnection.RequesterVpcInfo.Region != nil {
			f2.Region = resp.VpcPeeringConnection.RequesterVpcInfo.Region
		}
		if resp.VpcPeeringConnection.RequesterVpcInfo.VpcId != nil {
			f2.VPCID = resp.VpcPeeringConnection.RequesterVpcInfo.VpcId
		}
		ko.Status.RequesterVPCInfo = f2
	} else {
		ko.Status.RequesterVPCInfo = nil
	}
	if resp.VpcPeeringConnection.Status != nil {
		f3 := &svcapitypes.VPCPeeringConnectionStateReason{}
		if resp.VpcPeeringConnection.Status.Code != nil {
			f3.Code = resp.VpcPeeringConnection.Status.Code
		}
		if resp.VpcPeeringConnection.Status.Message != nil {
			f3.Message = resp.VpcPeeringConnection.Status.Message
		}
		ko.Status.Status = f3
	} else {
		ko.Status.Status = nil
	}
	if resp.VpcPeeringConnection.Tags != nil {
		f4 := []*svcapitypes.Tag{}
		for _, f4iter := range resp.VpcPeeringConnection.Tags {
			f4elem := &svcapitypes.Tag{}
			if f4iter.Key != nil {
				f4elem.Key = f4iter.Key
			}
			if f4iter.Value != nil {
				f4elem.Value = f4iter.Value
			}
			f4 = append(f4, f4elem)
		}
		ko.Spec.Tags = f4
	} else {
		ko.Spec.Tags = nil
	}
	if resp.VpcPeeringConnection.VpcPeeringConnectionId != nil {
		ko.Status.VPCPeeringConnectionID = resp.VpcPeeringConnection.VpcPeeringConnectionId
	} else {
		ko.Status.VPCPeeringConnectionID = nil
	}

	rm.setStatusDefaults(ko)
	return &resource{ko}, nil
}

// newCreateRequestPayload returns an SDK-specific struct for the HTTP request
// payload of the Create API call for the resource
func (rm *resourceManager) newCreateRequestPayload(
	ctx context.Context,
	r *resource,
) (*svcsdk.CreateVpcPeeringConnectionInput, error) {
	res := &svcsdk.CreateVpcPeeringConnectionInput{}

	if r.ko.Spec.PeerOwnerID != nil {
		res.SetPeerOwnerId(*r.ko.Spec.PeerOwnerID)
	}
	if r.ko.Spec.PeerRegion != nil {
		res.SetPeerRegion(*r.ko.Spec.PeerRegion)
	}
	if r.ko.Spec.PeerVPCID != nil {
		res.SetPeerVpcId(*r.ko.Spec.PeerVPCID)
	}
	if r.ko.Spec.VPCID != nil {
		res.SetVpcId(*r.ko.Spec.VPCID)
	}

	return res, nil
}

// sdkUpdate patches the supplied resource in the backend AWS service API and
// returns a new resource with updated fields.
func (rm *resourceManager) sdkUpdate(
	ctx context.Context,
	desired *resource,
	latest *resource,
	delta *ackcompare.Delta,
) (updated *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.sdkUpdate")
	defer func() {
		exit(err)
	}()

	if delta.DifferentAt("Spec.Tags") {
		if err := rm.syncTags(ctx, desired, latest); err != nil {
			return nil, err
		}
	}

	if delta.DifferentAt("Spec.AcceptRequest") {
		// Throw a Terminal Error, if the field was set to 'true' and is now set to 'false'
		if !*desired.ko.Spec.AcceptRequest {
			msg := fmt.Sprintf("You cannot set AcceptRequest to false after setting it to true")
			return nil, ackerr.NewTerminalError(fmt.Errorf(msg))
		}

		// Accept the VPC Peering Connection Request, if the field is set to 'true'
		acceptInput := &svcsdk.AcceptVpcPeeringConnectionInput{
			VpcPeeringConnectionId: latest.ko.Status.VPCPeeringConnectionID,
		}
		acceptResp, err = rm.sdkapi.AcceptVpcPeeringConnectionWithContext(ctx, acceptInput)
		if err != nil {
			return nil, err
		}
		rlog.Debug("VPC Peering Connection accepted", "VpcPeeringConnectionId", *acceptResp.VpcPeeringConnection.VpcPeeringConnectionId)
	}

	// Only continue if something other than Tags or certain fields has changed in the Spec
	if !delta.DifferentExcept("Spec.Tags", "Spec.AcceptRequest") {
		return desired, nil
	}
	input, err := rm.newUpdateRequestPayload(ctx, desired, delta)
	if err != nil {
		return nil, err
	}

	var resp *svcsdk.ModifyVpcPeeringConnectionOptionsOutput
	_ = resp
	resp, err = rm.sdkapi.ModifyVpcPeeringConnectionOptionsWithContext(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "ModifyVpcPeeringConnectionOptions", err)
	if err != nil {
		return nil, err
	}
	// Merge in the information we read from the API call above to the copy of
	// the original Kubernetes object we passed to the function
	ko := desired.ko.DeepCopy()

	if resp.AccepterPeeringConnectionOptions != nil {
		f0 := &svcapitypes.PeeringConnectionOptionsRequest{}
		if resp.AccepterPeeringConnectionOptions.AllowDnsResolutionFromRemoteVpc != nil {
			f0.AllowDNSResolutionFromRemoteVPC = resp.AccepterPeeringConnectionOptions.AllowDnsResolutionFromRemoteVpc
		}
		if resp.AccepterPeeringConnectionOptions.AllowEgressFromLocalClassicLinkToRemoteVpc != nil {
			f0.AllowEgressFromLocalClassicLinkToRemoteVPC = resp.AccepterPeeringConnectionOptions.AllowEgressFromLocalClassicLinkToRemoteVpc
		}
		if resp.AccepterPeeringConnectionOptions.AllowEgressFromLocalVpcToRemoteClassicLink != nil {
			f0.AllowEgressFromLocalVPCToRemoteClassicLink = resp.AccepterPeeringConnectionOptions.AllowEgressFromLocalVpcToRemoteClassicLink
		}
		ko.Spec.AccepterPeeringConnectionOptions = f0
	} else {
		ko.Spec.AccepterPeeringConnectionOptions = nil
	}
	if resp.RequesterPeeringConnectionOptions != nil {
		f1 := &svcapitypes.PeeringConnectionOptionsRequest{}
		if resp.RequesterPeeringConnectionOptions.AllowDnsResolutionFromRemoteVpc != nil {
			f1.AllowDNSResolutionFromRemoteVPC = resp.RequesterPeeringConnectionOptions.AllowDnsResolutionFromRemoteVpc
		}
		if resp.RequesterPeeringConnectionOptions.AllowEgressFromLocalClassicLinkToRemoteVpc != nil {
			f1.AllowEgressFromLocalClassicLinkToRemoteVPC = resp.RequesterPeeringConnectionOptions.AllowEgressFromLocalClassicLinkToRemoteVpc
		}
		if resp.RequesterPeeringConnectionOptions.AllowEgressFromLocalVpcToRemoteClassicLink != nil {
			f1.AllowEgressFromLocalVPCToRemoteClassicLink = resp.RequesterPeeringConnectionOptions.AllowEgressFromLocalVpcToRemoteClassicLink
		}
		ko.Spec.RequesterPeeringConnectionOptions = f1
	} else {
		ko.Spec.RequesterPeeringConnectionOptions = nil
	}

	rm.setStatusDefaults(ko)
	return &resource{ko}, nil
}

// newUpdateRequestPayload returns an SDK-specific struct for the HTTP request
// payload of the Update API call for the resource
func (rm *resourceManager) newUpdateRequestPayload(
	ctx context.Context,
	r *resource,
	delta *ackcompare.Delta,
) (*svcsdk.ModifyVpcPeeringConnectionOptionsInput, error) {
	res := &svcsdk.ModifyVpcPeeringConnectionOptionsInput{}

	if delta.DifferentAt("Spec.AccepterPeeringConnectionOptions") {
		if r.ko.Spec.AccepterPeeringConnectionOptions != nil {
			f0 := &svcsdk.PeeringConnectionOptionsRequest{}
			if r.ko.Spec.AccepterPeeringConnectionOptions.AllowDNSResolutionFromRemoteVPC != nil {
				f0.SetAllowDnsResolutionFromRemoteVpc(*r.ko.Spec.AccepterPeeringConnectionOptions.AllowDNSResolutionFromRemoteVPC)
			}
			if r.ko.Spec.AccepterPeeringConnectionOptions.AllowEgressFromLocalClassicLinkToRemoteVPC != nil {
				f0.SetAllowEgressFromLocalClassicLinkToRemoteVpc(*r.ko.Spec.AccepterPeeringConnectionOptions.AllowEgressFromLocalClassicLinkToRemoteVPC)
			}
			if r.ko.Spec.AccepterPeeringConnectionOptions.AllowEgressFromLocalVPCToRemoteClassicLink != nil {
				f0.SetAllowEgressFromLocalVpcToRemoteClassicLink(*r.ko.Spec.AccepterPeeringConnectionOptions.AllowEgressFromLocalVPCToRemoteClassicLink)
			}
			res.SetAccepterPeeringConnectionOptions(f0)
		}
	}
	if delta.DifferentAt("Spec.RequesterPeeringConnectionOptions") {
		if r.ko.Spec.RequesterPeeringConnectionOptions != nil {
			f2 := &svcsdk.PeeringConnectionOptionsRequest{}
			if r.ko.Spec.RequesterPeeringConnectionOptions.AllowDNSResolutionFromRemoteVPC != nil {
				f2.SetAllowDnsResolutionFromRemoteVpc(*r.ko.Spec.RequesterPeeringConnectionOptions.AllowDNSResolutionFromRemoteVPC)
			}
			if r.ko.Spec.RequesterPeeringConnectionOptions.AllowEgressFromLocalClassicLinkToRemoteVPC != nil {
				f2.SetAllowEgressFromLocalClassicLinkToRemoteVpc(*r.ko.Spec.RequesterPeeringConnectionOptions.AllowEgressFromLocalClassicLinkToRemoteVPC)
			}
			if r.ko.Spec.RequesterPeeringConnectionOptions.AllowEgressFromLocalVPCToRemoteClassicLink != nil {
				f2.SetAllowEgressFromLocalVpcToRemoteClassicLink(*r.ko.Spec.RequesterPeeringConnectionOptions.AllowEgressFromLocalVPCToRemoteClassicLink)
			}
			res.SetRequesterPeeringConnectionOptions(f2)
		}
	}
	if r.ko.Status.VPCPeeringConnectionID != nil {
		res.SetVpcPeeringConnectionId(*r.ko.Status.VPCPeeringConnectionID)
	}

	return res, nil
}

// sdkDelete deletes the supplied resource in the backend AWS service API
func (rm *resourceManager) sdkDelete(
	ctx context.Context,
	r *resource,
) (latest *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.sdkDelete")
	defer func() {
		exit(err)
	}()
	input, err := rm.newDeleteRequestPayload(r)
	if err != nil {
		return nil, err
	}
	var resp *svcsdk.DeleteVpcPeeringConnectionOutput
	_ = resp
	resp, err = rm.sdkapi.DeleteVpcPeeringConnectionWithContext(ctx, input)
	rm.metrics.RecordAPICall("DELETE", "DeleteVpcPeeringConnection", err)
	return nil, err
}

// newDeleteRequestPayload returns an SDK-specific struct for the HTTP request
// payload of the Delete API call for the resource
func (rm *resourceManager) newDeleteRequestPayload(
	r *resource,
) (*svcsdk.DeleteVpcPeeringConnectionInput, error) {
	res := &svcsdk.DeleteVpcPeeringConnectionInput{}

	if r.ko.Status.VPCPeeringConnectionID != nil {
		res.SetVpcPeeringConnectionId(*r.ko.Status.VPCPeeringConnectionID)
	}

	return res, nil
}

// setStatusDefaults sets default properties into supplied custom resource
func (rm *resourceManager) setStatusDefaults(
	ko *svcapitypes.VPCPeeringConnection,
) {
	if ko.Status.ACKResourceMetadata == nil {
		ko.Status.ACKResourceMetadata = &ackv1alpha1.ResourceMetadata{}
	}
	if ko.Status.ACKResourceMetadata.Region == nil {
		ko.Status.ACKResourceMetadata.Region = &rm.awsRegion
	}
	if ko.Status.ACKResourceMetadata.OwnerAccountID == nil {
		ko.Status.ACKResourceMetadata.OwnerAccountID = &rm.awsAccountID
	}
	if ko.Status.Conditions == nil {
		ko.Status.Conditions = []*ackv1alpha1.Condition{}
	}
}

// updateConditions returns updated resource, true; if conditions were updated
// else it returns nil, false
func (rm *resourceManager) updateConditions(
	r *resource,
	onSuccess bool,
	err error,
) (*resource, bool) {
	ko := r.ko.DeepCopy()
	rm.setStatusDefaults(ko)

	// Terminal condition
	var terminalCondition *ackv1alpha1.Condition = nil
	var recoverableCondition *ackv1alpha1.Condition = nil
	var syncCondition *ackv1alpha1.Condition = nil
	for _, condition := range ko.Status.Conditions {
		if condition.Type == ackv1alpha1.ConditionTypeTerminal {
			terminalCondition = condition
		}
		if condition.Type == ackv1alpha1.ConditionTypeRecoverable {
			recoverableCondition = condition
		}
		if condition.Type == ackv1alpha1.ConditionTypeResourceSynced {
			syncCondition = condition
		}
	}
	var termError *ackerr.TerminalError
	if rm.terminalAWSError(err) || err == ackerr.SecretTypeNotSupported || err == ackerr.SecretNotFound || errors.As(err, &termError) {
		if terminalCondition == nil {
			terminalCondition = &ackv1alpha1.Condition{
				Type: ackv1alpha1.ConditionTypeTerminal,
			}
			ko.Status.Conditions = append(ko.Status.Conditions, terminalCondition)
		}
		var errorMessage = ""
		if err == ackerr.SecretTypeNotSupported || err == ackerr.SecretNotFound || errors.As(err, &termError) {
			errorMessage = err.Error()
		} else {
			awsErr, _ := ackerr.AWSError(err)
			errorMessage = awsErr.Error()
		}
		terminalCondition.Status = corev1.ConditionTrue
		terminalCondition.Message = &errorMessage
	} else {
		// Clear the terminal condition if no longer present
		if terminalCondition != nil {
			terminalCondition.Status = corev1.ConditionFalse
			terminalCondition.Message = nil
		}
		// Handling Recoverable Conditions
		if err != nil {
			if recoverableCondition == nil {
				// Add a new Condition containing a non-terminal error
				recoverableCondition = &ackv1alpha1.Condition{
					Type: ackv1alpha1.ConditionTypeRecoverable,
				}
				ko.Status.Conditions = append(ko.Status.Conditions, recoverableCondition)
			}
			recoverableCondition.Status = corev1.ConditionTrue
			awsErr, _ := ackerr.AWSError(err)
			errorMessage := err.Error()
			if awsErr != nil {
				errorMessage = awsErr.Error()
			}
			recoverableCondition.Message = &errorMessage
		} else if recoverableCondition != nil {
			recoverableCondition.Status = corev1.ConditionFalse
			recoverableCondition.Message = nil
		}
	}
	// Required to avoid the "declared but not used" error in the default case
	_ = syncCondition
	if terminalCondition != nil || recoverableCondition != nil || syncCondition != nil {
		return &resource{ko}, true // updated
	}
	return nil, false // not updated
}

// terminalAWSError returns awserr, true; if the supplied error is an aws Error type
// and if the exception indicates that it is a Terminal exception
// 'Terminal' exception are specified in generator configuration
func (rm *resourceManager) terminalAWSError(err error) bool {
	// No terminal_errors specified for this resource in generator config
	return false
}

func (rm *resourceManager) newTag(
	c svcapitypes.Tag,
) *svcsdk.Tag {
	res := &svcsdk.Tag{}
	if c.Key != nil {
		res.SetKey(*c.Key)
	}
	if c.Value != nil {
		res.SetValue(*c.Value)
	}

	return res
}

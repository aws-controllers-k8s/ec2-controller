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

package route_table

import (
	"context"

	svcapitypes "github.com/aws-controllers-k8s/ec2-controller/apis/v1alpha1"
	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	svcsdk "github.com/aws/aws-sdk-go/service/ec2"
)

// RouteAction stores the possible actions that can be performed on
// any of a Route Table's Routes
type RouteAction int

const (
	RouteActionNone RouteAction = iota
	RouteActionCreate
)

func (rm *resourceManager) createRoutes(
	ctx context.Context,
	r *resource,
) error {
	if err := rm.syncRoutes(ctx, r, nil); err != nil {
		return err
	}
	return nil
}

func (rm *resourceManager) syncRoutes(
	ctx context.Context,
	desired *resource,
	latest *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.syncRoutes")
	defer exit(err)

	for _, rc := range desired.ko.Spec.Routes {
		action := getRouteAction(rc, latest)
		switch action {
		case RouteActionCreate:
			if err = rm.createRoute(ctx, desired, *rc); err != nil {
				return err
			}

		default:
		}
	}

	if latest != nil {
		for _, l := range latest.ko.Spec.Routes {
			desiredRoute := false
			for _, d := range desired.ko.Spec.Routes {
				delta := compareRoute(l, d)
				//if a Route matches identically, then it is desired
				if len(delta.Differences) == 0 {
					desiredRoute = true
					break
				}
			}
			if !desiredRoute {
				if err = rm.deleteRoute(ctx, latest, *l); err != nil {
					return err
				}
			}

		}
	}

	return nil
}

// getRouteAction returns the determined action for a given
// route object, depending on the latest and desired values
func getRouteAction(
	desired *svcapitypes.Route,
	latest *resource,
) RouteAction {
	//the default route created by RouteTable; no action needed
	if *desired.GatewayID == "local" {
		return RouteActionNone
	}

	action := RouteActionCreate
	if latest != nil {
		for _, l := range latest.ko.Spec.Routes {
			delta := compareRoute(l, desired)
			if len(delta.Differences) == 0 {
				return RouteActionNone
			}
		}
	}
	return action
}

func (rm *resourceManager) createRoute(
	ctx context.Context,
	r *resource,
	c svcapitypes.Route,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.createRoute")
	defer exit(err)

	input := rm.newCreateRoutePayload(r, c)
	_, err = rm.sdkapi.CreateRouteWithContext(ctx, input)
	rm.metrics.RecordAPICall("CREATE", "CreateRoute", err)
	return err
}

func (rm *resourceManager) deleteRoute(
	ctx context.Context,
	r *resource,
	c svcapitypes.Route,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.deleteRoute")
	defer exit(err)

	input := rm.newDeleteRoutePayload(r, c)
	_, err = rm.sdkapi.DeleteRouteWithContext(ctx, input)
	rm.metrics.RecordAPICall("DELETE", "DeleteRoute", err)
	return err
}

func (rm *resourceManager) newCreateRoutePayload(
	r *resource,
	c svcapitypes.Route,
) *svcsdk.CreateRouteInput {
	input := &svcsdk.CreateRouteInput{}
	if r.ko.Status.RouteTableID != nil {
		input.SetRouteTableId(*r.ko.Status.RouteTableID)
	}
	if c.CarrierGatewayID != nil {
		input.SetCarrierGatewayId(*c.CarrierGatewayID)
	}
	if c.DestinationCIDRBlock != nil {
		input.SetDestinationCidrBlock(*c.DestinationCIDRBlock)
	}
	if c.DestinationIPv6CIDRBlock != nil {
		input.SetDestinationIpv6CidrBlock(*c.DestinationIPv6CIDRBlock)
	}
	if c.DestinationPrefixListID != nil {
		input.SetDestinationPrefixListId(*c.DestinationPrefixListID)
	}
	if c.EgressOnlyInternetGatewayID != nil {
		input.SetEgressOnlyInternetGatewayId(*c.EgressOnlyInternetGatewayID)
	}
	if c.GatewayID != nil {
		input.SetGatewayId(*c.GatewayID)
	}
	if c.InstanceID != nil {
		input.SetInstanceId(*c.InstanceID)
	}
	if c.LocalGatewayID != nil {
		input.SetLocalGatewayId(*c.LocalGatewayID)
	}
	if c.NatGatewayID != nil {
		input.SetNatGatewayId(*c.NatGatewayID)
	}
	if c.NetworkInterfaceID != nil {
		input.SetNetworkInterfaceId(*c.NetworkInterfaceID)
	}
	if c.TransitGatewayID != nil {
		input.SetTransitGatewayId(*c.TransitGatewayID)
	}
	if c.VPCPeeringConnectionID != nil {
		input.SetVpcPeeringConnectionId(*c.VPCPeeringConnectionID)
	}
	input.SetDryRun(false)

	return input
}

func (rm *resourceManager) newDeleteRoutePayload(
	r *resource,
	c svcapitypes.Route,
) *svcsdk.DeleteRouteInput {
	input := &svcsdk.DeleteRouteInput{}
	if r.ko.Status.RouteTableID != nil {
		input.SetRouteTableId(*r.ko.Status.RouteTableID)
	}
	if c.DestinationCIDRBlock != nil {
		input.SetDestinationCidrBlock(*c.DestinationCIDRBlock)
	}
	if c.DestinationIPv6CIDRBlock != nil {
		input.SetDestinationIpv6CidrBlock(*c.DestinationIPv6CIDRBlock)
	}
	if c.DestinationPrefixListID != nil {
		input.SetDestinationPrefixListId(*c.DestinationPrefixListID)
	}

	return input
}

func (rm *resourceManager) customUpdateRouteTable(
	ctx context.Context,
	desired *resource,
	latest *resource,
	delta *ackcompare.Delta,
) (updated *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.customUpdateRouteTable")
	defer exit(err)

	ko := desired.ko.DeepCopy()
	rm.setStatusDefaults(ko)

	if delta.DifferentAt("Spec.Routes") {
		if err := rm.syncRoutes(ctx, desired, latest); err != nil {
			return nil, err
		}
		latest, err = rm.sdkFind(ctx, latest)
		if err != nil {
			return nil, err
		}
	}

	return latest, nil
}

func (rm *resourceManager) requiredFieldsMissingForCreateRoute(
	r *resource,
) bool {
	return r.ko.Status.RouteTableID == nil
}

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

func compareRoute(
	a *svcapitypes.Route,
	b *svcapitypes.Route,
) *ackcompare.Delta {
	delta := ackcompare.NewDelta()
	if ackcompare.HasNilDifference(a.CarrierGatewayID, b.CarrierGatewayID) {
		delta.Add("Route.CarrierGatewayID", a.CarrierGatewayID, b.CarrierGatewayID)
	} else if a.CarrierGatewayID != nil && b.CarrierGatewayID != nil {
		if *a.CarrierGatewayID != *b.CarrierGatewayID {
			delta.Add("Route.CarrierGatewayID", a.CarrierGatewayID, b.CarrierGatewayID)
		}
	}
	if ackcompare.HasNilDifference(a.DestinationCIDRBlock, b.DestinationCIDRBlock) {
		delta.Add("Route.DestinationCIDRBlock", a.DestinationCIDRBlock, b.DestinationCIDRBlock)
	} else if a.DestinationCIDRBlock != nil && b.DestinationCIDRBlock != nil {
		if *a.DestinationCIDRBlock != *b.DestinationCIDRBlock {
			delta.Add("Route.DestinationCIDRBlock", a.DestinationCIDRBlock, b.DestinationCIDRBlock)
		}
	}
	if ackcompare.HasNilDifference(a.DestinationIPv6CIDRBlock, b.DestinationIPv6CIDRBlock) {
		delta.Add("Route.DestinationIPv6CIDRBlock", a.DestinationIPv6CIDRBlock, b.DestinationIPv6CIDRBlock)
	} else if a.DestinationIPv6CIDRBlock != nil && b.DestinationIPv6CIDRBlock != nil {
		if *a.DestinationIPv6CIDRBlock != *b.DestinationIPv6CIDRBlock {
			delta.Add("Route.DestinationIPv6CIDRBlock", a.DestinationIPv6CIDRBlock, b.DestinationIPv6CIDRBlock)
		}
	}
	if ackcompare.HasNilDifference(a.DestinationPrefixListID, b.DestinationPrefixListID) {
		delta.Add("Route.DestinationPrefixListID", a.DestinationPrefixListID, b.DestinationPrefixListID)
	} else if a.DestinationPrefixListID != nil && b.DestinationPrefixListID != nil {
		if *a.DestinationPrefixListID != *b.DestinationPrefixListID {
			delta.Add("Route.DestinationPrefixListID", a.DestinationPrefixListID, b.DestinationPrefixListID)
		}
	}
	if ackcompare.HasNilDifference(a.EgressOnlyInternetGatewayID, b.EgressOnlyInternetGatewayID) {
		delta.Add("Route.EgressOnlyInternetGatewayID", a.EgressOnlyInternetGatewayID, b.EgressOnlyInternetGatewayID)
	} else if a.EgressOnlyInternetGatewayID != nil && b.EgressOnlyInternetGatewayID != nil {
		if *a.EgressOnlyInternetGatewayID != *b.EgressOnlyInternetGatewayID {
			delta.Add("Route.EgressOnlyInternetGatewayID", a.EgressOnlyInternetGatewayID, b.EgressOnlyInternetGatewayID)
		}
	}
	if ackcompare.HasNilDifference(a.GatewayID, b.GatewayID) {
		delta.Add("Route.GatewayID", a.GatewayID, b.GatewayID)
	} else if a.GatewayID != nil && b.GatewayID != nil {
		if *a.GatewayID != *b.GatewayID {
			delta.Add("Route.GatewayID", a.GatewayID, b.GatewayID)
		}
	}
	if ackcompare.HasNilDifference(a.InstanceID, b.InstanceID) {
		delta.Add("Route.InstanceID", a.InstanceID, b.InstanceID)
	} else if a.InstanceID != nil && b.InstanceID != nil {
		if *a.InstanceID != *b.InstanceID {
			delta.Add("Route.InstanceID", a.InstanceID, b.InstanceID)
		}
	}
	if ackcompare.HasNilDifference(a.InstanceOwnerID, b.InstanceOwnerID) {
		delta.Add("Route.InstanceOwnerID", a.InstanceOwnerID, b.InstanceOwnerID)
	} else if a.InstanceOwnerID != nil && b.InstanceOwnerID != nil {
		if *a.InstanceOwnerID != *b.InstanceOwnerID {
			delta.Add("Route.InstanceOwnerID", a.InstanceOwnerID, b.InstanceOwnerID)
		}
	}
	if ackcompare.HasNilDifference(a.LocalGatewayID, b.LocalGatewayID) {
		delta.Add("Route.LocalGatewayID", a.LocalGatewayID, b.LocalGatewayID)
	} else if a.LocalGatewayID != nil && b.LocalGatewayID != nil {
		if *a.LocalGatewayID != *b.LocalGatewayID {
			delta.Add("Route.LocalGatewayID", a.LocalGatewayID, b.LocalGatewayID)
		}
	}
	if ackcompare.HasNilDifference(a.NatGatewayID, b.NatGatewayID) {
		delta.Add("Route.NatGatewayID", a.NatGatewayID, b.NatGatewayID)
	} else if a.NatGatewayID != nil && b.NatGatewayID != nil {
		if *a.NatGatewayID != *b.NatGatewayID {
			delta.Add("Route.NatGatewayID", a.NatGatewayID, b.NatGatewayID)
		}
	}
	if ackcompare.HasNilDifference(a.NetworkInterfaceID, b.NetworkInterfaceID) {
		delta.Add("Route.NetworkInterfaceID", a.NetworkInterfaceID, b.NetworkInterfaceID)
	} else if a.NetworkInterfaceID != nil && b.NetworkInterfaceID != nil {
		if *a.NetworkInterfaceID != *b.NetworkInterfaceID {
			delta.Add("Route.NetworkInterfaceID", a.NetworkInterfaceID, b.NetworkInterfaceID)
		}
	}
	if ackcompare.HasNilDifference(a.TransitGatewayID, b.TransitGatewayID) {
		delta.Add("Route.TransitGatewayID", a.TransitGatewayID, b.TransitGatewayID)
	} else if a.TransitGatewayID != nil && b.TransitGatewayID != nil {
		if *a.TransitGatewayID != *b.TransitGatewayID {
			delta.Add("Route.TransitGatewayID", a.TransitGatewayID, b.TransitGatewayID)
		}
	}
	if ackcompare.HasNilDifference(a.VPCPeeringConnectionID, b.VPCPeeringConnectionID) {
		delta.Add("Route.VPCPeeringConnectionID", a.VPCPeeringConnectionID, b.VPCPeeringConnectionID)
	} else if a.VPCPeeringConnectionID != nil && b.VPCPeeringConnectionID != nil {
		if *a.VPCPeeringConnectionID != *b.VPCPeeringConnectionID {
			delta.Add("Route.VPCPeeringConnectionID", a.VPCPeeringConnectionID, b.VPCPeeringConnectionID)
		}
	}
	if ackcompare.HasNilDifference(a.Origin, b.Origin) {
		delta.Add("Route.Origin", a.Origin, b.Origin)
	} else if a.Origin != nil && b.Origin != nil {
		if *a.Origin != *b.Origin {
			delta.Add("Route.Origin", a.Origin, b.Origin)
		}
	}
	if ackcompare.HasNilDifference(a.State, b.State) {
		delta.Add("Route.State", a.State, b.State)
	} else if a.State != nil && b.State != nil {
		if *a.State != *b.State {
			delta.Add("Route.State", a.State, b.State)
		}
	}

	return delta
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

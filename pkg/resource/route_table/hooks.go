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

const LocalRouteGateway = "local"

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
	toAdd := []*svcapitypes.CreateRouteInput{}
	toDelete := []*svcapitypes.CreateRouteInput{}

	for _, desiredRoute := range desired.ko.Spec.Routes {
		if *desiredRoute.GatewayID == LocalRouteGateway {
			// no-op for default route
			continue
		}
		if latestRoute := getMatchingRoute(desiredRoute, latest); latestRoute != nil {
			delta := compareCreateRouteInput(desiredRoute, latestRoute)
			if len(delta.Differences) > 0 {
				// "update" route by deleting old route and adding the new route
				toDelete = append(toDelete, latestRoute)
				toAdd = append(toAdd, desiredRoute)
			}
		} else {
			// a desired route is not in latest; therefore, create
			toAdd = append(toAdd, desiredRoute)
		}
	}
	if latest != nil {
		for _, latestRoute := range latest.ko.Spec.Routes {
			if desiredRoute := getMatchingRoute(latestRoute, desired); desiredRoute == nil {
				// latest has a route that is not desired; therefore, delete
				toDelete = append(toDelete, latestRoute)
			}
		}
	}

	for _, route := range toAdd {
		rlog.Debug("adding route to route table")
		if err = rm.createRoute(ctx, desired, *route); err != nil {
			return err
		}
	}
	for _, route := range toDelete {
		rlog.Debug("deleting route from route table")
		if err = rm.deleteRoute(ctx, latest, *route); err != nil {
			return err
		}
	}

	return nil
}

func getMatchingRoute(
	routeToMatch *svcapitypes.CreateRouteInput,
	resource *resource,
) *svcapitypes.CreateRouteInput {
	if resource == nil {
		return nil
	}

	for _, route := range resource.ko.Spec.Routes {
		delta := compareCreateRouteInput(routeToMatch, route)
		if len(delta.Differences) == 0 {
			return route
		} else {
			if routeToMatch.CarrierGatewayID != nil {
				if !delta.DifferentAt("CreateRouteInput.CarrierGatewayID") {
					return route
				}
			}
			if routeToMatch.EgressOnlyInternetGatewayID != nil {
				if !delta.DifferentAt("CreateRouteInput.EgressOnlyInternetGatewayID") {
					return route
				}
			}
			if routeToMatch.GatewayID != nil {
				if !delta.DifferentAt("CreateRouteInput.GatewayID") {
					return route
				}
			}
			if routeToMatch.LocalGatewayID != nil {
				if !delta.DifferentAt("CreateRouteInput.LocalGatewayID") {
					return route
				}
			}
			if routeToMatch.NATGatewayID != nil {
				if !delta.DifferentAt("CreateRouteInput.NATGatewayID") {
					return route
				}
			}
			if routeToMatch.TransitGatewayID != nil {
				if !delta.DifferentAt("CreateRouteInput.TransitGatewayID") {
					return route
				}
			}
			if routeToMatch.VPCPeeringConnectionID != nil {
				if !delta.DifferentAt("CreateRouteInput.VPCPeeringConnectionID") {
					return route
				}
			}
		}
	}

	return nil
}

func (rm *resourceManager) createRoute(
	ctx context.Context,
	r *resource,
	c svcapitypes.CreateRouteInput,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.createRoute")
	defer exit(err)

	input := rm.newCreateRouteInput(c)
	// Routes should only be configurable for the
	// RouteTable in which they are defined
	input.RouteTableId = r.ko.Status.RouteTableID
	_, err = rm.sdkapi.CreateRouteWithContext(ctx, input)
	rm.metrics.RecordAPICall("CREATE", "CreateRoute", err)
	return err
}

func (rm *resourceManager) deleteRoute(
	ctx context.Context,
	r *resource,
	c svcapitypes.CreateRouteInput,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.deleteRoute")
	defer exit(err)

	input := rm.newDeleteRouteInput(c)
	// Routes should only be configurable for the
	// RouteTable in which they are defined
	input.RouteTableId = r.ko.Status.RouteTableID
	_, err = rm.sdkapi.DeleteRouteWithContext(ctx, input)
	rm.metrics.RecordAPICall("DELETE", "DeleteRoute", err)
	return err
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

// addRoutesToStatus takes a RouteTable from an aws-sdk output
// and adds its Routes to RouteTable.Status.RouteStatuses.
// This cannot be auto-generated because the code-generator already associates
// RouteTable's Routes from aws-sdk output with RouteTable.Spec.Routes.
func (rm *resourceManager) addRoutesToStatus(
	ko *svcapitypes.RouteTable,
	routeTable *svcsdk.RouteTable,
) {
	ko.Status.RouteStatuses = nil
	if routeTable.Routes != nil {
		routesInStatus := []*svcapitypes.Route{}
		for _, r := range routeTable.Routes {
			routesInStatus = append(routesInStatus, rm.setResourceRoute(r))
		}
		ko.Status.RouteStatuses = routesInStatus
	}
}

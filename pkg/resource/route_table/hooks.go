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
		if (*desiredRoute).GatewayID != nil && *desiredRoute.GatewayID == LocalRouteGateway {
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
			if (*latestRoute).GatewayID != nil && *latestRoute.GatewayID == LocalRouteGateway {
				// no-op for default route
				continue
			}
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
	}

	if delta.DifferentAt("Spec.Tags") {
		if err := rm.syncTags(ctx, desired, latest); err != nil {
			return nil, err
		}
	}

	latest, err = rm.sdkFind(ctx, latest)
	if err != nil {
		return nil, err
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

// customPreCompare ensures that default values of types are initialised and
// server side defaults are excluded from the delta.
func customPreCompare(
	a *resource,
	b *resource,
) {
	a.ko.Spec.Routes = removeLocalRoute(a.ko.Spec.Routes)
	b.ko.Spec.Routes = removeLocalRoute(b.ko.Spec.Routes)
}

// removeLocalRoute will filter out any routes that have a gateway ID that
// matches the local gateway. Every route table contains a local route for
// communication within the VPC, which cannot be deleted or modified, and should
// not be included as part of the spec.
func removeLocalRoute(
	routes []*svcapitypes.CreateRouteInput,
) (ret []*svcapitypes.CreateRouteInput) {
	ret = make([]*svcapitypes.CreateRouteInput, 0)

	for _, route := range routes {
		if route.GatewayID == nil || *route.GatewayID != LocalRouteGateway {
			ret = append(ret, route)
		}
	}

	return ret
}

// syncTags used to keep tags in sync by calling Create and Delete Tags API's
func (rm *resourceManager) syncTags(
	ctx context.Context,
	desired *resource,
	latest *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.syncTags")
	defer func(err error) {
		exit(err)
	}(err)

	resourceId := []*string{latest.ko.Status.RouteTableID}

	toAdd, toDelete := computeTagsDelta(
		desired.ko.Spec.Tags, latest.ko.Spec.Tags,
	)

	if len(toDelete) > 0 {
		rlog.Debug("removing tags from route table resource", "tags", toDelete)
		_, err = rm.sdkapi.DeleteTagsWithContext(
			ctx,
			&svcsdk.DeleteTagsInput{
				Resources: resourceId,
				Tags:      rm.sdkTags(toDelete),
			},
		)
		rm.metrics.RecordAPICall("UPDATE", "DeleteTags", err)
		if err != nil {
			return err
		}

	}

	if len(toAdd) > 0 {
		rlog.Debug("adding tags to route table resource", "tags", toAdd)
		_, err = rm.sdkapi.CreateTagsWithContext(
			ctx,
			&svcsdk.CreateTagsInput{
				Resources: resourceId,
				Tags:      rm.sdkTags(toAdd),
			},
		)
		rm.metrics.RecordAPICall("UPDATE", "CreateTags", err)
		if err != nil {
			return err
		}
	}

	return nil
}

// sdkTags converts *svcapitypes.Tag array to a *svcsdk.Tag array
func (rm *resourceManager) sdkTags(
	tags []*svcapitypes.Tag,
) (sdktags []*svcsdk.Tag) {

	for _, i := range tags {
		sdktag := rm.newTag(*i)
		sdktags = append(sdktags, sdktag)
	}

	return sdktags
}

// computeTagsDelta returns tags to be added and removed from the resource
func computeTagsDelta(
	desired []*svcapitypes.Tag,
	latest []*svcapitypes.Tag,
) (toAdd []*svcapitypes.Tag, toDelete []*svcapitypes.Tag) {

	desiredTags := map[string]string{}
	for _, tag := range desired {
		desiredTags[*tag.Key] = *tag.Value
	}

	latestTags := map[string]string{}
	for _, tag := range latest {
		latestTags[*tag.Key] = *tag.Value
	}

	for _, tag := range desired {
		val, ok := latestTags[*tag.Key]
		if !ok || val != *tag.Value {
			toAdd = append(toAdd, tag)
		}
	}

	for _, tag := range latest {
		_, ok := desiredTags[*tag.Key]
		if !ok {
			toDelete = append(toDelete, tag)
		}
	}

	return toAdd, toDelete

}

// updateTagSpecificationsInCreateRequest adds
// Tags defined in the Spec to CreateRouteTableInput.TagSpecification
// and ensures the ResourceType is always set to 'route-table'
func updateTagSpecificationsInCreateRequest(r *resource,
	input *svcsdk.CreateRouteTableInput) {
	desiredTagSpecs := svcsdk.TagSpecification{}
	if r.ko.Spec.Tags != nil {
		requestedTags := []*svcsdk.Tag{}
		for _, desiredTag := range r.ko.Spec.Tags {
			// Add in tags defined in the Spec
			tag := &svcsdk.Tag{}
			if desiredTag.Key != nil && desiredTag.Value != nil {
				tag.SetKey(*desiredTag.Key)
				tag.SetValue(*desiredTag.Value)
			}
			requestedTags = append(requestedTags, tag)
		}
		desiredTagSpecs.SetResourceType("route-table")
		desiredTagSpecs.SetTags(requestedTags)
	}
	input.TagSpecifications = []*svcsdk.TagSpecification{&desiredTagSpecs}
}

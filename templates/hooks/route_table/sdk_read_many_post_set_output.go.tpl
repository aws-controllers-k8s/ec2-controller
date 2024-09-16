	
    if found {
        rm.addRoutesToStatus(ko, resp.RouteTables[0])
    }
	toAdd, toDelete := computeTagsDelta(r.ko.Spec.Tags, ko.Spec.Tags)
	if len(toAdd) == 0 && len(toDelete) == 0 {
		// if resource's initial tags and response tags are equal,
		// then assign resource's tags to maintain tag order
		ko.Spec.Tags = r.ko.Spec.Tags
	}
    
	// Even if route is created with arguments as VPCEndpointID,
	// when aws api is called to describe the route (inside skdFind), it
	// returns VPCEndpointID as GatewayID. Due to this bug, spec section for
	// routes is populated incorrectly in above auto-gen code.
	// To solve this, if 'GatewayID' has prefix 'vpce-', then the entry is
	// moved from 'GatewayID' to 'VPCEndpointID'.
	for i, route := range ko.Spec.Routes {
		if route.GatewayID != nil && strings.HasPrefix(*route.GatewayID, "vpce-") {
			ko.Spec.Routes[i].VPCEndpointID = route.GatewayID
			ko.Spec.Routes[i].GatewayID = nil
		}
	}
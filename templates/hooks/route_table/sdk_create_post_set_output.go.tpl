	rm.addRoutesToStatus(ko, resp.RouteTable)
	
	if rm.requiredFieldsMissingForCreateRoute(&resource{ko}) {
		return nil, ackerr.NotFound
	}

	if len(desired.ko.Spec.Routes) > 0 {
		//desired routes are overwritten by RouteTable's default route
		ko.Spec.Routes = append(ko.Spec.Routes, desired.ko.Spec.Routes...)
		if err := rm.createRoutes(ctx, &resource{ko}); err != nil {
			return nil, err
		}
	}

	toAdd, toDelete := computeTagsDelta(desired.ko.Spec.Tags, ko.Spec.Tags)
	if len(toAdd) == 0 && len(toDelete) == 0 {
		// if desired tags and response tags are equal,
		// then assign desired tags to maintain tag order
		ko.Spec.Tags = desired.ko.Spec.Tags
	}
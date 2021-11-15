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
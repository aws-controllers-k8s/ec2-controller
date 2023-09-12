	if ko.Spec.VPC != nil {
		if err = rm.attachToVPC(ctx, &resource{ko}); err != nil {
			return nil, err
		}
	}

	if err = rm.createRouteTableAssociations(ctx, &resource{ko}); err != nil {
        return nil, err
    }

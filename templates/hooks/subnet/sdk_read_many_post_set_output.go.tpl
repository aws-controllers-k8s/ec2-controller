	assocs, err := rm.getRouteTableAssociations(ctx, &resource{ko})
	if err != nil {
		return nil, err
	} else {
		ko.Spec.RouteTables = make([]*string, len(assocs))
		for i, assoc := range assocs {
			ko.Spec.RouteTables[i] = assoc.RouteTableId
		}
	}
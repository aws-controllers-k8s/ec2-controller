	if ko.Status.PrivateDNSNameOptionsOnLaunch != nil {
		if ko.Status.PrivateDNSNameOptionsOnLaunch.EnableResourceNameDNSARecord != nil {
			ko.Spec.EnableResourceNameDNSARecord = ko.Status.PrivateDNSNameOptionsOnLaunch.EnableResourceNameDNSARecord
		}
		if ko.Status.PrivateDNSNameOptionsOnLaunch.EnableResourceNameDNSAAAARecord != nil {
			ko.Spec.EnableResourceNameDNSAAAARecord = ko.Status.PrivateDNSNameOptionsOnLaunch.EnableResourceNameDNSAAAARecord
		}
		if ko.Status.PrivateDNSNameOptionsOnLaunch.HostnameType != nil {
			ko.Spec.HostnameType = ko.Status.PrivateDNSNameOptionsOnLaunch.HostnameType
		}
	}

	assocs, err := rm.getRouteTableAssociations(ctx, &resource{ko})
	if err != nil {
		return nil, err
	} else {
		ko.Spec.RouteTables = make([]*string, len(assocs))
		for i, assoc := range assocs {
			ko.Spec.RouteTables[i] = assoc.RouteTableId
		}
	}
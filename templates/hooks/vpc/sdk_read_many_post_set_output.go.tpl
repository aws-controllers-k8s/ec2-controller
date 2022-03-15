	if dnsAttrs, err := rm.getDNSAttributes(ctx, *ko.Status.VPCID); err != nil {
		return nil, err
	} else {
		ko.Spec.EnableDNSSupport = dnsAttrs.EnableSupport
		ko.Spec.EnableDNSHostnames = dnsAttrs.EnableHostnames
	}
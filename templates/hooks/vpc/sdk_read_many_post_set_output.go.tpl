	rm.setSpecCIDRs(ko)
	if dnsAttrs, err := rm.getDNSAttributes(ctx, *ko.Status.VPCID); err != nil {
		return nil, err
	} else {
		ko.Spec.EnableDNSSupport = dnsAttrs.EnableSupport
		ko.Spec.EnableDNSHostnames = dnsAttrs.EnableHostnames
	}
	sgDefaultRulesExist, err := rm.hasSecurityGroupDefaultRules(ctx, &resource{ko})
	if err != nil {
		return nil, err
	}

	// If default security group rules exist, then set
	// DisallowSecurityGroupDefaultRules field in spec to false. This will
	// allow sdkUpdate to be invoked if 'desired' cr has
	// DisallowSecurityGroupDefaultRules field in spec set to true.
	disallowSGDefaultRules := !sgDefaultRulesExist
	ko.Spec.DisallowSecurityGroupDefaultRules = &disallowSGDefaultRules

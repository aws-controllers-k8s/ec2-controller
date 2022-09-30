	
	if rm.requiredFieldsMissingForSGRule(&resource{ko}) {
		return nil, ackerr.NotFound
	}
	if err = rm.syncSGRules(ctx, &resource{ko}, nil); err != nil {
		return nil, err
	}

	// A ReadOne call for SecurityGroup Rules (NOT SecurityGroups)
	// is made to refresh Status.Rules with the recently-updated
	// data from the above `sync` call
	if rules, err := rm.getRules(ctx, &resource{ko}); err != nil {
		return nil, err
	} else {
		ko.Status.Rules = rules
	}

	// if user defines any egress rule, then remove the default
	// egress rule; otherwise, add default rule Spec to align with
	// resource's server-side state (i.e. Status.Rules)
	if len(desired.ko.Spec.EgressRules) > 0 {
		if err = rm.deleteDefaultSecurityGroupRule(ctx, &resource{ko}); err != nil {
			return nil, err
		}
	} else {
		ko.Spec.EgressRules = append(ko.Spec.EgressRules, rm.defaultEgressRule())
	}

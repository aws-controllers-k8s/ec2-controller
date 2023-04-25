	
	if rm.requiredFieldsMissingForSGRule(&resource{ko}) {
		return nil, ackerr.NotFound
	}

	// Delete the default egress rule
	if err = rm.deleteDefaultSecurityGroupRule(ctx, &resource{ko}); err != nil {
		return nil, err
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

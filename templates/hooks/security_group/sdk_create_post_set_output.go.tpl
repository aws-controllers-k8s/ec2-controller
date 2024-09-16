	
	if rm.requiredFieldsMissingForSGRule(&resource{ko}) {
		return nil, ackerr.NotFound
	}

	// Delete the default egress rule
	if err = rm.deleteDefaultSecurityGroupRule(ctx, &resource{ko}); err != nil {
		return &resource{ko}, err
	}

	if !rm.referencesResolved(&resource{ko}) {
		ackcondition.SetSynced(&resource{ko}, corev1.ConditionFalse, nil, nil)
        return &resource{ko}, nil
	}

	if err = rm.syncSGRules(ctx, &resource{ko}, nil); err != nil {
		return &resource{ko}, err
	}

	// A ReadOne call for SecurityGroup Rules (NOT SecurityGroups)
	// is made to refresh Status.Rules with the recently-updated
	// data from the above `sync` call
	if rules, err := rm.getRules(ctx, &resource{ko}); err != nil {
		return &resource{ko}, err
	} else {
		ko.Status.Rules = rules
	}

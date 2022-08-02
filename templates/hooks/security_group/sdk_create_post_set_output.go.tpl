	
	if rm.requiredFieldsMissingForSGRule(&resource{ko}) {
		return nil, ackerr.NotFound
	}
	if err = rm.syncSGRules(ctx, &resource{ko}, nil); err != nil {
		return nil, err
	}
	rm.addRulesToStatus(ko, ctx)

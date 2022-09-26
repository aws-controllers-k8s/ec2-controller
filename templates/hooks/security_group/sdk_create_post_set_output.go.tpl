	
	if rm.requiredFieldsMissingForSGRule(&resource{ko}) {
		return nil, ackerr.NotFound
	}
	if err = rm.syncSGRules(ctx, &resource{ko}, nil); err != nil {
		return nil, err
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
	created, err = rm.sdkFindRules(ctx, &resource{ko})
	if err != nil {
		ko.Status.Rules = created.ko.Status.Rules
	}

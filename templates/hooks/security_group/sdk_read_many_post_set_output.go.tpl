	if found {
        rm.addRulesToSpec(ko, resp.SecurityGroups[0])
    	
        // A ReadOne call for SecurityGroup Rules (NOT SecurityGroups)
	    // is made to refresh Status.Rules
	    if rules, err := rm.getRules(ctx, &resource{ko}); err != nil {
		    return nil, err
	    } else {
		    ko.Status.Rules = rules
	    }
    }
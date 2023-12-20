	if found {

	// Needed because SecurityGroups Name are held in GroupName property of the AWS resource
        ko.Spec.Name = resp.SecurityGroups[0].GroupName

        rm.addRulesToSpec(ko, resp.SecurityGroups[0])
    	
        // A ReadOne call for SecurityGroup Rules (NOT SecurityGroups)
	    // is made to refresh Status.Rules
	    if rules, err := rm.getRules(ctx, &resource{ko}); err != nil {
		    return nil, err
	    } else {
		    ko.Status.Rules = rules
	    }
    }

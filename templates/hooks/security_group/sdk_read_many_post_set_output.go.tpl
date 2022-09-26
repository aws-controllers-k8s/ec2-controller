	
    if found {
        rm.addRulesToSpec(ko, resp.SecurityGroups[0])
        latest, err = rm.sdkFindRules(ctx, &resource{ko})
		if err != nil {
			ko.Status.Rules = latest.ko.Status.Rules
		}
    }
    

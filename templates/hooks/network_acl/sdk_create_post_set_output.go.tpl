	if rm.requiredFieldsMissingForCreateNetworkAcl(&resource{ko}) {
		return nil, ackerr.NotFound
	}

	if len(desired.ko.Spec.Associations) > 0 {
		ko.Spec.Associations = desired.ko.Spec.Associations
		copy := ko.DeepCopy()
		if err := rm.createAssociation(ctx, &resource{copy}); err != nil {
			rlog.Debug("Error while syncing Association", err)
		}
	}

	if len(desired.ko.Spec.Entries) > 0 {
		// Filter out default rules and only keep desired entries
		filteredEntries := []*svcapitypes.NetworkACLEntry{}
		for _, entry := range desired.ko.Spec.Entries {
			if entry.RuleNumber != nil && *entry.RuleNumber == int64(DefaultRuleNumber) {
				continue
			}
			filteredEntries = append(filteredEntries, entry)
		}
		ko.Spec.Entries = filteredEntries
		copy := ko.DeepCopy()
		if err := rm.createEntries(ctx, &resource{copy}); err != nil {
			rlog.Debug("Error while syncing entries", err)
		}
	}

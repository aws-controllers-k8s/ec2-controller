	if rm.requiredFieldsMissingForCreateNetworkAcl(&resource{ko}) {
		return nil, ackerr.NotFound
	}

	if len(desired.ko.Spec.Associations) > 0 {
		ko.Spec.Associations = desired.ko.Spec.Associations
		if err := rm.createAssociation(ctx, &resource{ko}); err != nil {
			rlog.Debug("Error while syncing Association", err)
		}
	}

	if len(desired.ko.Spec.Entries) > 0 {
		//desired rules are overwritten by NetworkACL's default rules
		ko.Spec.Entries = append(ko.Spec.Entries, desired.ko.Spec.Entries...)
		if err := rm.createEntries(ctx, &resource{ko}); err != nil {
			rlog.Debug("Error while syncing routes", err)
		}
	}

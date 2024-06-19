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
		//desired rules are overwritten by NetworkACL's default rules
		ko.Spec.Entries = append(ko.Spec.Entries, desired.ko.Spec.Entries...)
		copy := ko.DeepCopy()
		if err := rm.createEntries(ctx, &resource{copy}); err != nil {
			rlog.Debug("Error while syncing routes", err)
		}
	}

	if rm.requiredFieldsMissingForCreateNetworkAcl(&resource{ko}) {
		return nil, ackerr.NotFound
	}
	if len(desired.ko.Spec.Entries) > 0 {
		//desired rules are overwritten by NetworkACL's default rules
		ko.Spec.Entries = append(ko.Spec.Entries, desired.ko.Spec.Entries...)
		if err := rm.createRules(ctx, &resource{ko}); err != nil {
			rlog.Debug("Error while syncing routes", err)
		}
	}

	toAdd, toDelete := computeTagsDelta(desired.ko.Spec.Tags, ko.Spec.Tags)
	if len(toAdd) == 0 && len(toDelete) == 0 {
		// if desired tags and response tags are equal,
		// then assign desired tags to maintain tag order
		ko.Spec.Tags = desired.ko.Spec.Tags
	}
	f6, f6ok := identifier.AdditionalKeys["resourceID"]
	if f6ok {
		r.ko.Spec.ResourceID = &f6
	}
        f7, f7ok := identifier.AdditionalKeys["resourceType"]
        if f7ok {
                r.ko.Spec.ResourceType = &f7
        }

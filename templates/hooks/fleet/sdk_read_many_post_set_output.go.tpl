	// Here we want to check if the fleet is deleted
	// returning NotFound will trigger a create
	if fleetDeleted(r) {
		return nil, ackerr.NotFound
	}

	toAdd, toDelete := computeTagsDelta(r.ko.Spec.Tags, ko.Spec.Tags)
	if len(toAdd) == 0 && len(toDelete) == 0 {
		// if resource's initial tags and response tags are equal,
		// then assign resource's tags to maintain tag order
		ko.Spec.Tags = r.ko.Spec.Tags
	}

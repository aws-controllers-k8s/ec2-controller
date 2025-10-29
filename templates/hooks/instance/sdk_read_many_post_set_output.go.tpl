	// Here we want to check if the instance is terminated(deleted)
	// returning NotFound will trigger a create
	if needsRestart(ko) {
		return nil, ackerr.NotFound
	}

	setAdditionalFields(resp.Reservations[0].Instances[0], ko)

	if !isRunning(ko) {
		ackcondition.SetSynced(&resource{ko}, corev1.ConditionFalse, nil, aws.String("waiting for resource to be running"))
	}
	
	toAdd, toDelete := computeTagsDelta(r.ko.Spec.Tags, ko.Spec.Tags)
	if len(toAdd) == 0 && len(toDelete) == 0 {
		// if resource's initial tags and response tags are equal,
		// then assign resource's tags to maintain tag order
		ko.Spec.Tags = r.ko.Spec.Tags
	}

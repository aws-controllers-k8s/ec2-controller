
	setAdditionalFields(resp.Instances[0], ko)
	
	toAdd, toDelete := computeTagsDelta(desired.ko.Spec.Tags, ko.Spec.Tags)
	if len(toAdd) == 0 && len(toDelete) == 0 {
		// if desired tags and response tags are equal,
		// then assign desired tags to maintain tag order
		ko.Spec.Tags = desired.ko.Spec.Tags
	}

	// SourceDestCheck cannot be set at launch; apply it after create.
	if desired.ko.Spec.SourceDestCheckEnabled != nil {
		ko.Spec.SourceDestCheckEnabled = desired.ko.Spec.SourceDestCheckEnabled
		msg := "Instance is pending update for SourceDestCheckEnabled"
		ackcondition.SetSynced(&resource{ko}, corev1.ConditionFalse, &msg, nil)
	}
	setAdditionalFields(resp.Instances[0], ko)
	
	toAdd, toDelete := computeTagsDelta(desired.ko.Spec.Tags, ko.Spec.Tags)
	if len(toAdd) == 0 && len(toDelete) == 0 {
		// if desired tags and response tags are equal,
		// then assign desired tags to maintain tag order
		ko.Spec.Tags = desired.ko.Spec.Tags
	}

	// SourceDestCheck cannot be set at launch time. If user specified a value,
	// preserve their desired value and mark not synced so the reconciler will
	// call ModifyInstanceAttribute once the instance is running.
	if desired.ko.Spec.SourceDestCheckEnabled != nil {
		ko.Spec.SourceDestCheckEnabled = desired.ko.Spec.SourceDestCheckEnabled
		ackcondition.SetSynced(&resource{ko}, corev1.ConditionFalse, nil, nil)
	}
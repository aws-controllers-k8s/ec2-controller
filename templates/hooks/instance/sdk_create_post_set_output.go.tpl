	toAdd, toDelete := computeTagsDelta(desired.ko.Spec.Tags, ko.Spec.Tags)
	if len(toAdd) == 0 && len(toDelete) == 0 {
		// if desired tags and response tags are equal,
		// then assign desired tags to maintain tag order
		ko.Spec.Tags = desired.ko.Spec.Tags
	}
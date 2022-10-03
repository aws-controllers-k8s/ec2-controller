	
	toAdd, toDelete := computeTagsDelta(r.ko.Spec.Tags, ko.Spec.Tags)
	if len(toAdd) == 0 && len(toDelete) == 0 {
		// if resource's initial tags and response tags are equal,
		// then assign resource's tags to maintain tag order
		ko.Spec.Tags = r.ko.Spec.Tags
	}

	// A resource is synced when it has achieved the desired state. However, users
	// cannot desire a temporary state for an Instance (i.e. pending); therefore, 
	// if the resource is in a temporary state, then set synced to false and reconcile 
	// until a permanent state is achieved.
	if inTransitoryState(&resource{ko}) {
		msg := "Instance is in a transitory  state, current status=" + string(*ko.Status.State.Name)
		ackcondition.SetSynced(&resource{ko}, corev1.ConditionFalse, &msg, nil)
		return &resource{ko}, nil
	} else {
		ackcondition.SetSynced(&resource{ko}, corev1.ConditionTrue, nil, nil)
	}
    

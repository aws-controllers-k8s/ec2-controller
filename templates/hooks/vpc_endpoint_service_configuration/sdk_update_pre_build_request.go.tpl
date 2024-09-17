
	// Only continue if the VPC Endpoint Service is in 'Available' state
	if *latest.ko.Status.ServiceState != "Available" {
		return desired, requeueWaitNotAvailable
	}

	if delta.DifferentAt("Spec.Tags") {
		if err := syncTags(
			ctx, rm.sdkapi, rm.metrics, *latest.ko.Status.ServiceID,
			desired.ko.Spec.Tags, latest.ko.Spec.Tags,
		); err != nil {
			return nil, err
		}
	}

	if delta.DifferentAt("Spec.AllowedPrincipals") {
		if desired, err := rm.syncAllowedPrincipals(ctx, desired, latest); err != nil {
			// This causes a requeue and the rest of the fields will be synced on the next reconciliation loop
			ackcondition.SetSynced(desired, corev1.ConditionFalse, nil, nil)
			return desired, err
		}
	}

	// Only continue if something other than Tags or certain fields has changed in the Spec
	if !delta.DifferentExcept("Spec.Tags", "Spec.AllowedPrincipals") {
		return desired, nil
	}

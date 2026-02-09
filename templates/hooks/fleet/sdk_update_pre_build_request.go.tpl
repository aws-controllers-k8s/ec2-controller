	if delta.DifferentAt("Spec.Tags") {
		if err := syncTags(
			ctx, rm.sdkapi, rm.metrics, *latest.ko.Status.FleetID,
			desired.ko.Spec.Tags, latest.ko.Spec.Tags,
		); err != nil {
			return nil, err
		}
	}

	if !delta.DifferentExcept("Spec.Tags") {
		return desired, nil
	}

	immutable_fields := []string{"Spec.Type", "Spec.ReplaceUnhealthyInstances", "Spec.TerminateInstancesWithExpiration", "Spec.TargetCapacitySpecification.DefaultTargetCapacityType", "Spec.SpotOptions", "Spec.OnDemandOptions"}
	for _, field := range immutable_fields {
		if delta.DifferentAt(field) {
			// Throw a Terminal Error if immutable fields are modified
			if latest.ko.Spec.TargetCapacitySpecification.DefaultTargetCapacityType != desired.ko.Spec.TargetCapacitySpecification.DefaultTargetCapacityType {
				msg := "field " + field + " is not updatable after fleet creation"
				return nil, ackerr.NewTerminalError(fmt.Errorf("%s", msg))
			}
		}
	}

	// This value is automatically populated in TargetCapacitySpecification, but is not supported in ModifyFleetRequest
	desired.ko.Spec.TargetCapacitySpecification.DefaultTargetCapacityType = nil

	

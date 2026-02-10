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

	// Only tag updates are supported in Fleets of type instant
	if *latest.ko.Spec.Type == "instant" {
		msg := "api error Unsupported: Fleets of type 'instant' cannot be modified."
		return nil, ackerr.NewTerminalError(fmt.Errorf("%s", msg))
	}

	// Throw error if an immutable field is updated in the CRD
	// The immutableFieldChanges function mentioned in https://aws-controllers-k8s.github.io/community/docs/contributor-docs/code-generator-config/#is_immutable-mutable-vs-immutable-fields does not seem to be working
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

	// Ensure Launch Template Version is int and not $Latest/$Default
	// Preventing those values as as those strings are updated in the backend with the ints they represent
	// This confuses the reconciliation as the aws state falls out of sync with the CRD
	// If we stick to int strings, all works smoothly
	for _, config := range desired.ko.Spec.LaunchTemplateConfigs {
		if config.LaunchTemplateSpecification != nil {
			_, err := strconv.Atoi(*config.LaunchTemplateSpecification.Version)
			if err != nil {
				msg := "Only int values are supported for Launch Template Version in EC2 fleet spec"
				return nil, ackerr.NewTerminalError(fmt.Errorf("%s", msg))
			}
		}
	}
	

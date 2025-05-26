
	if instanceCount, ok := identifier.AdditionalKeys["instanceCount"]; ok {
			parsedInstanceCount, err := strconv.ParseInt(instanceCount, 10, 64)
			if err != nil {
				return fmt.Errorf("failed to parse instanceCount: %v", err)
			}
			r.ko.Spec.InstanceCount = &parsedInstanceCount
	}

	if instancePlatform, ok := identifier.AdditionalKeys["instancePlatform"]; ok {
		r.ko.Spec.InstancePlatform = &instancePlatform
	}

	if instanceType, ok := identifier.AdditionalKeys["instanceType"]; ok {
		r.ko.Spec.InstanceType = &instanceType
	}

	if availabilityZone, ok := identifier.AdditionalKeys["availabilityZone"]; ok {
		r.ko.Spec.AvailabilityZone = &availabilityZone
	}

	if availabilityZoneID, ok := identifier.AdditionalKeys["availabilityZoneID"]; ok {
		r.ko.Spec.AvailabilityZoneID = &availabilityZoneID
	}

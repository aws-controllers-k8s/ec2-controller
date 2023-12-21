
	// Only continue if the VPC Endpoint Service is in 'Available' state
	if *latest.ko.Status.ServiceState != "Available" {
		return desired, ackrequeue.NeededAfter(fmt.Errorf("VPCEndpointService is not in %v state yet, requeuing", "Available"), 5 * time.Second)
	}

	if delta.DifferentAt("Spec.Tags") {
		if err := rm.syncTags(ctx, desired, latest); err != nil {
			// This causes a requeue and the rest of the fields will be synced on the next reconciliation loop
			ackcondition.SetSynced(desired, corev1.ConditionFalse, nil, nil)
			return desired, err
		}
	}

	var listOfPrincipalsToAdd []*string
	if delta.DifferentAt("Spec.AllowedPrincipals") {
		for _, desiredPrincipal := range desired.ko.Spec.AllowedPrincipals {
			for _, latestPrincipal := range latest.ko.Spec.AllowedPrincipals {
				if *desiredPrincipal == *latestPrincipal {
					// Principal already in Allow List, skip
					continue
				}
				// Principal is not in the Allow List, add it
				listOfPrincipalsToAdd = append(listOfPrincipalsToAdd, desiredPrincipal)
			}
		}
		// Make the AWS API call to add the principals
		if len(listOfPrincipalsToAdd) > 0 {
			modifyPermissionsInput := &svcsdk.ModifyVpcEndpointServicePermissionsInput{
				ServiceId: latest.ko.Status.ServiceID,
				AddAllowedPrincipals: listOfPrincipalsToAdd,
			}
			_, err := rm.sdkapi.ModifyVpcEndpointServicePermissions(modifyPermissionsInput)
			rm.metrics.RecordAPICall("UPDATE", "ModifyVpcEndpointServicePermissions", err)
			if err != nil {
				return nil, err
			}
		}

		// TODO: Add Logic to remove any principal that is not on the allowed list anymore
	}

	// Only continue if something other than Tags or certain fields has changed in the Spec
	if !delta.DifferentExcept("Spec.Tags", "Spec.AllowedPrincipals") {
		return desired, nil
	}

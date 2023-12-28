
	// Only continue if the VPC Endpoint Service is in 'Available' state
	if *latest.ko.Status.ServiceState != "Available" {
		return desired, requeueWaitNotAvailable
	}

	if delta.DifferentAt("Spec.Tags") {
		if err := rm.syncTags(ctx, desired, latest); err != nil {
			// This causes a requeue and the rest of the fields will be synced on the next reconciliation loop
			ackcondition.SetSynced(desired, corev1.ConditionFalse, nil, nil)
			return desired, err
		}
	}

	if delta.DifferentAt("Spec.AllowedPrincipals") {
		var listOfPrincipalsToAdd []*string
		var listOfPrincipalsToRemove []*string

		// If the latest list of principals is empty, we want to add all principals
		if len(latest.ko.Spec.AllowedPrincipals) == 0 && len(desired.ko.Spec.AllowedPrincipals) > 0 {
			listOfPrincipalsToAdd = desired.ko.Spec.AllowedPrincipals

			// If the desired list of principals is empty, we want to remove all principals
		} else if len(desired.ko.Spec.AllowedPrincipals) == 0 && len(latest.ko.Spec.AllowedPrincipals) > 0 {
			listOfPrincipalsToRemove = latest.ko.Spec.AllowedPrincipals
			// Otherwise, we'll compare the two lists and add/remove principals as needed
		} else {
			for _, desiredPrincipal := range desired.ko.Spec.AllowedPrincipals {
				principalToAddAlreadyFound := false
				for _, latestPrincipal := range latest.ko.Spec.AllowedPrincipals {
					if *desiredPrincipal == *latestPrincipal {
						// Principal already in Allow List, skip
						principalToAddAlreadyFound = true
						break
					}
				}
				if !principalToAddAlreadyFound {
					// Desired Principal is not in the Allowed List, add it to the list of those to add
					listOfPrincipalsToAdd = append(listOfPrincipalsToAdd, desiredPrincipal)
				}
			}

			// Remove any principal that is not on the allowed list anymore
			for _, latestPrincipal := range latest.ko.Spec.AllowedPrincipals {
				principalToRemoveAlreadyFound := false
				for _, desiredPrincipal := range desired.ko.Spec.AllowedPrincipals {
					if *desiredPrincipal == *latestPrincipal {
						// Principal still in Allow List, skip
						principalToRemoveAlreadyFound = true
						break
					}
				}
				if !principalToRemoveAlreadyFound {
					// Latest Principal is not in the Allowed List, add it to the list of those to remove
					listOfPrincipalsToRemove = append(listOfPrincipalsToRemove, latestPrincipal)
				}
			}

		}

		// Make the AWS API call to add the principals
		if len(listOfPrincipalsToAdd) > 0 {
			modifyPermissionsInput := &svcsdk.ModifyVpcEndpointServicePermissionsInput{
				ServiceId:            latest.ko.Status.ServiceID,
				AddAllowedPrincipals: listOfPrincipalsToAdd,
			}
			_, err := rm.sdkapi.ModifyVpcEndpointServicePermissions(modifyPermissionsInput)
			rm.metrics.RecordAPICall("UPDATE", "ModifyVpcEndpointServicePermissions", err)
			if err != nil {
				return desired, err
			}
		}

		// Make the AWS API call to remove the principals
		if len(listOfPrincipalsToRemove) > 0 {
			modifyPermissionsInput := &svcsdk.ModifyVpcEndpointServicePermissionsInput{
				ServiceId:               latest.ko.Status.ServiceID,
				RemoveAllowedPrincipals: listOfPrincipalsToRemove,
			}
			_, err := rm.sdkapi.ModifyVpcEndpointServicePermissions(modifyPermissionsInput)
			rm.metrics.RecordAPICall("UPDATE", "ModifyVpcEndpointServicePermissions", err)
			if err != nil {
				return desired, err
			}
		}
	}

	// Only continue if something other than Tags or certain fields has changed in the Spec
	if !delta.DifferentExcept("Spec.Tags", "Spec.AllowedPrincipals") {
		return desired, nil
	}

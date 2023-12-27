
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

	rlog.Debug("AAAAAAAAAAAAAAAAAAA sdkUpdate", "deltaResult", delta.DifferentAt("Spec.AllowedPrincipals"))
	if delta.DifferentAt("Spec.AllowedPrincipals") {
		rlog.Debug("AAAAAAAAAAAAAAAAAAA sdkUpdate", "Found difference at Spec.AllowedPrincipals")
		var listOfPrincipalsToAdd []*string
		for _, desiredPrincipal := range desired.ko.Spec.AllowedPrincipals {
			for _, latestPrincipal := range latest.ko.Spec.AllowedPrincipals {
				if *desiredPrincipal == *latestPrincipal {
					// Principal already in Allow List, skip
					continue
				}
				// Principal is not in the Allow List, add it to the list of those to add
				listOfPrincipalsToAdd = append(listOfPrincipalsToAdd, desiredPrincipal)
			}
		}
		rlog.Debug("AAAAAAAAAAAAAAAAAAA sdkUpdate", "listOfPrincipalsToAdd", listOfPrincipalsToAdd)
		// Make the AWS API call to add the principals
		if len(listOfPrincipalsToAdd) > 0 {
			modifyPermissionsInput := &svcsdk.ModifyVpcEndpointServicePermissionsInput{
				ServiceId:            latest.ko.Status.ServiceID,
				AddAllowedPrincipals: listOfPrincipalsToAdd,
			}
			_, err := rm.sdkapi.ModifyVpcEndpointServicePermissions(modifyPermissionsInput)
			rm.metrics.RecordAPICall("UPDATE", "ModifyVpcEndpointServicePermissions", err)
			if err != nil {
				return nil, err
			}
		}

		// Remove any principal that is not on the allowed list anymore
		var listOfPrincipalsToRemove []*string
		for _, latestPrincipal := range latest.ko.Spec.AllowedPrincipals {
			for _, desiredPrincipal := range desired.ko.Spec.AllowedPrincipals {
				if *desiredPrincipal == *latestPrincipal {
					// Principal still in Allow List, skip
					continue
				}
				// Principal is not in the Allow List, add it to the list of those to remove
				listOfPrincipalsToRemove = append(listOfPrincipalsToRemove, latestPrincipal)
			}
		}
		rlog.Debug("AAAAAAAAAAAAAAAAAAA sdkUpdate", "listOfPrincipalsToRemove", listOfPrincipalsToRemove)
		// Make the AWS API call to remove the principals
		if len(listOfPrincipalsToRemove) > 0 {
			modifyPermissionsInput := &svcsdk.ModifyVpcEndpointServicePermissionsInput{
				ServiceId:               latest.ko.Status.ServiceID,
				RemoveAllowedPrincipals: listOfPrincipalsToRemove,
			}
			_, err := rm.sdkapi.ModifyVpcEndpointServicePermissions(modifyPermissionsInput)
			rm.metrics.RecordAPICall("UPDATE", "ModifyVpcEndpointServicePermissions", err)
			if err != nil {
				return nil, err
			}
		}
	}

	// Only continue if something other than Tags or certain fields has changed in the Spec
	if !delta.DifferentExcept("Spec.Tags", "Spec.AllowedPrincipals") {
		return desired, nil
	}

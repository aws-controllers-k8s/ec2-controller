  
  	if isVPCPeeringConnectionCreating(desired) {
		return desired, requeueWaitWhileCreating
	}
	if isVPCPeeringConnectionProvisioning(desired) {
		return desired, requeueWaitWhileProvisioning
	}
	if isVPCPeeringConnectionDeleting(desired) {
		return desired, requeueWaitWhileDeleting
	}
	// If the VPC Peering Connection is Pending Acceptance or Active, continue

	if delta.DifferentAt("Spec.Tags") {
			if err := rm.syncTags(ctx, desired, latest); err != nil {
				// This causes a requeue and the rest of the fields will be synced on the next reconciliation loop
				ackcondition.SetSynced(desired, corev1.ConditionFalse, nil, nil)
				return desired, err
			}
		}

	if delta.DifferentAt("Spec.AcceptRequest") {
		// Throw a Terminal Error, if the field was set to 'true' and is now set to 'false'
		if desired.ko.Spec.AcceptRequest == nil || !*desired.ko.Spec.AcceptRequest {
			msg := fmt.Sprintf("You cannot set AcceptRequest to false after setting it to true")
			return nil, ackerr.NewTerminalError(fmt.Errorf(msg))

			// Accept the VPC Peering Connection Request, if the field is set to 'true' and is still at status Pending Acceptance
		} else if *latest.ko.Status.Status.Code == "pending-acceptance" {
			acceptInput := &svcsdk.AcceptVpcPeeringConnectionInput{
				VpcPeeringConnectionId: latest.ko.Status.VPCPeeringConnectionID,
			}
			acceptResp, err := rm.sdkapi.AcceptVpcPeeringConnectionWithContext(ctx, acceptInput)
			if err != nil {
				return nil, err
			}
			rlog.Debug("VPC Peering Connection accepted", "apiResponse", acceptResp)
			readOneLatest, err := rm.ReadOne(ctx, desired)
			if err != nil {
				return nil, err
			}
			latest = rm.concreteResource(readOneLatest.DeepCopy())
			desired.ko.Status.Status =  latest.ko.Status.Status
			// This causes a requeue and the rest of the fields will be synced on the next reconciliation loop
			ackcondition.SetSynced(desired, corev1.ConditionFalse, nil, nil)
			return desired, nil
		}
	}

	// Only continue if something other than Tags or certain fields has changed in the Spec
	if !delta.DifferentExcept("Spec.Tags", "Spec.AcceptRequest") {
		return desired, nil
	}

  	if desired.ko.Spec.AccepterPeeringConnectionOptions != nil {
		f0 := &svcapitypes.PeeringConnectionOptionsRequest{}
		if desired.ko.Spec.AccepterPeeringConnectionOptions.AllowDNSResolutionFromRemoteVPC != nil {
			f0.AllowDNSResolutionFromRemoteVPC = desired.ko.Spec.AccepterPeeringConnectionOptions.AllowDNSResolutionFromRemoteVPC
		}
		if desired.ko.Spec.AccepterPeeringConnectionOptions.AllowEgressFromLocalClassicLinkToRemoteVPC != nil {
			f0.AllowEgressFromLocalClassicLinkToRemoteVPC = desired.ko.Spec.AccepterPeeringConnectionOptions.AllowEgressFromLocalClassicLinkToRemoteVPC
		}
		if desired.ko.Spec.AccepterPeeringConnectionOptions.AllowEgressFromLocalVPCToRemoteClassicLink != nil {
			f0.AllowEgressFromLocalVPCToRemoteClassicLink = desired.ko.Spec.AccepterPeeringConnectionOptions.AllowEgressFromLocalVPCToRemoteClassicLink
		}
		desired.ko.Spec.AccepterPeeringConnectionOptions = f0
	} else {
		desired.ko.Spec.AccepterPeeringConnectionOptions = nil
	}
	if desired.ko.Spec.RequesterPeeringConnectionOptions != nil {
		f1 := &svcapitypes.PeeringConnectionOptionsRequest{}
		if desired.ko.Spec.RequesterPeeringConnectionOptions.AllowDNSResolutionFromRemoteVPC != nil {
			f1.AllowDNSResolutionFromRemoteVPC = desired.ko.Spec.RequesterPeeringConnectionOptions.AllowDNSResolutionFromRemoteVPC
		}
		if desired.ko.Spec.RequesterPeeringConnectionOptions.AllowEgressFromLocalClassicLinkToRemoteVPC != nil {
			f1.AllowEgressFromLocalClassicLinkToRemoteVPC = desired.ko.Spec.RequesterPeeringConnectionOptions.AllowEgressFromLocalClassicLinkToRemoteVPC
		}
		if desired.ko.Spec.RequesterPeeringConnectionOptions.AllowEgressFromLocalVPCToRemoteClassicLink != nil {
			f1.AllowEgressFromLocalVPCToRemoteClassicLink = desired.ko.Spec.RequesterPeeringConnectionOptions.AllowEgressFromLocalVPCToRemoteClassicLink
		}
		desired.ko.Spec.RequesterPeeringConnectionOptions = f1
	} else {
		desired.ko.Spec.RequesterPeeringConnectionOptions = nil
	}
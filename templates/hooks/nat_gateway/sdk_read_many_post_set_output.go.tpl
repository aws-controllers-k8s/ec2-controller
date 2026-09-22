	// Populate Status.StatusVPCID from Spec.VPCID for backward compatibility.
	// VpcId moved from Status to Spec after the SDK bump added it to
	// CreateNatGatewayInput, but existing users may read it from Status.
	if ko.Spec.VPCID != nil {
		ko.Status.StatusVPCID = ko.Spec.VPCID
	}
	if isResourceDeleted(&resource{ko}) {
		return nil, ackerr.NotFound
	}
	// Return the observed resource so the runtime patches status while pending.
	if isResourcePending(&resource{ko}) {
		return &resource{ko}, requeueWaitWhilePending
	}

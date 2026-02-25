	// Populate Status.StatusVPCID from Spec.VPCID for backward compatibility.
	// VpcId moved from Status to Spec after the SDK bump added it to
	// CreateNatGatewayInput, but existing users may read it from Status.
	if ko.Spec.VPCID != nil {
		ko.Status.StatusVPCID = ko.Spec.VPCID
	}

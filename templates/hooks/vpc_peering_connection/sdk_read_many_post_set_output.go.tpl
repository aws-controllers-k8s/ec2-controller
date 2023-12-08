
	// Artificially trigger detection by delta.DifferentAt("Spec.AcceptRequest")
	res := &resource{ko}
	if isVPCPeeringConnectionPendingAcceptance(res) {
		res.ko.Spec.AcceptRequest = aws.Bool(false)
	} else if isVPCPeeringConnectionActive(res) || isVPCPeeringConnectionProvisioning(res) {
		res.ko.Spec.AcceptRequest = aws.Bool(true)
	} else if isVPCPeeringConnectionCreating(res) {
		return nil, requeueWaitWhileCreating
	}


	// if ko.Spec.AccepterPeeringConnectionOptions != nil {
	// 	f0 := &svcapitypes.PeeringConnectionOptionsRequest{}
	// 	if ko.Spec.AccepterPeeringConnectionOptions.AllowDNSResolutionFromRemoteVPC != nil {
	// 		f0.AllowDNSResolutionFromRemoteVPC = ko.Spec.AccepterPeeringConnectionOptions.AllowDNSResolutionFromRemoteVPC
	// 	}
	// 	if ko.Spec.AccepterPeeringConnectionOptions.AllowEgressFromLocalClassicLinkToRemoteVPC != nil {
	// 		f0.AllowEgressFromLocalClassicLinkToRemoteVPC = ko.Spec.AccepterPeeringConnectionOptions.AllowEgressFromLocalClassicLinkToRemoteVPC
	// 	}
	// 	if ko.Spec.AccepterPeeringConnectionOptions.AllowEgressFromLocalVPCToRemoteClassicLink != nil {
	// 		f0.AllowEgressFromLocalVPCToRemoteClassicLink = ko.Spec.AccepterPeeringConnectionOptions.AllowEgressFromLocalVPCToRemoteClassicLink
	// 	}
	// 	ko.Spec.AccepterPeeringConnectionOptions = f0
	// } else {
	// 	ko.Spec.AccepterPeeringConnectionOptions = nil
	// }
	// if ko.Spec.RequesterPeeringConnectionOptions != nil {
	// 	f1 := &svcapitypes.PeeringConnectionOptionsRequest{}
	// 	if ko.Spec.RequesterPeeringConnectionOptions.AllowDNSResolutionFromRemoteVPC != nil {
	// 		f1.AllowDNSResolutionFromRemoteVPC = ko.Spec.RequesterPeeringConnectionOptions.AllowDNSResolutionFromRemoteVPC
	// 	}
	// 	if ko.Spec.RequesterPeeringConnectionOptions.AllowEgressFromLocalClassicLinkToRemoteVPC != nil {
	// 		f1.AllowEgressFromLocalClassicLinkToRemoteVPC = ko.Spec.RequesterPeeringConnectionOptions.AllowEgressFromLocalClassicLinkToRemoteVPC
	// 	}
	// 	if ko.Spec.RequesterPeeringConnectionOptions.AllowEgressFromLocalVPCToRemoteClassicLink != nil {
	// 		f1.AllowEgressFromLocalVPCToRemoteClassicLink = ko.Spec.RequesterPeeringConnectionOptions.AllowEgressFromLocalVPCToRemoteClassicLink
	// 	}
	// 	ko.Spec.RequesterPeeringConnectionOptions = f1
	// } else {
	// 	ko.Spec.RequesterPeeringConnectionOptions = nil
	// }
	
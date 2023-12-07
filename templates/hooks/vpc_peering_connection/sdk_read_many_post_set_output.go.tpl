
	// Artificially trigger detection by delta.DifferentAt("Spec.AcceptRequest")
	res := &resource{ko}
	if isVPCPeeringConnectionPendingAcceptance(res) {
		res.ko.Spec.AcceptRequest = aws.Bool(false)
	} else if isVPCPeeringConnectionActive(res) || isVPCPeeringConnectionProvisioning(res) {
		res.ko.Spec.AcceptRequest = aws.Bool(true)
	} else if isVPCPeeringConnectionCreating(res) {
		return nil, requeueWaitWhileCreating
	}


	if res.ko.Spec.AccepterPeeringConnectionOptions != nil {
		f0 := &svcapitypes.PeeringConnectionOptionsRequest{}
		if res.ko.Spec.AccepterPeeringConnectionOptions.AllowDNSResolutionFromRemoteVPC != nil {
			f0.AllowDNSResolutionFromRemoteVPC = res.ko.Spec.AccepterPeeringConnectionOptions.AllowDNSResolutionFromRemoteVPC
		}
		if res.ko.Spec.AccepterPeeringConnectionOptions.AllowEgressFromLocalClassicLinkToRemoteVPC != nil {
			f0.AllowEgressFromLocalClassicLinkToRemoteVPC = res.ko.Spec.AccepterPeeringConnectionOptions.AllowEgressFromLocalClassicLinkToRemoteVPC
		}
		if res.ko.Spec.AccepterPeeringConnectionOptions.AllowEgressFromLocalVPCToRemoteClassicLink != nil {
			f0.AllowEgressFromLocalVPCToRemoteClassicLink = res.ko.Spec.AccepterPeeringConnectionOptions.AllowEgressFromLocalVPCToRemoteClassicLink
		}
		ko.Spec.AccepterPeeringConnectionOptions = f0
	} else {
		ko.Spec.AccepterPeeringConnectionOptions = nil
	}
	if res.ko.Spec.RequesterPeeringConnectionOptions != nil {
		f1 := &svcapitypes.PeeringConnectionOptionsRequest{}
		if res.ko.Spec.RequesterPeeringConnectionOptions.AllowDNSResolutionFromRemoteVPC != nil {
			f1.AllowDNSResolutionFromRemoteVPC = res.ko.Spec.RequesterPeeringConnectionOptions.AllowDNSResolutionFromRemoteVPC
		}
		if res.ko.Spec.RequesterPeeringConnectionOptions.AllowEgressFromLocalClassicLinkToRemoteVPC != nil {
			f1.AllowEgressFromLocalClassicLinkToRemoteVPC = res.ko.Spec.RequesterPeeringConnectionOptions.AllowEgressFromLocalClassicLinkToRemoteVPC
		}
		if res.ko.Spec.RequesterPeeringConnectionOptions.AllowEgressFromLocalVPCToRemoteClassicLink != nil {
			f1.AllowEgressFromLocalVPCToRemoteClassicLink = res.ko.Spec.RequesterPeeringConnectionOptions.AllowEgressFromLocalVPCToRemoteClassicLink
		}
		ko.Spec.RequesterPeeringConnectionOptions = f1
	} else {
		ko.Spec.RequesterPeeringConnectionOptions = nil
	}
	
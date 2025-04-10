
	// This prevents reference resolution errors when adopting existing resources where these fields are not provided in the manifest.
	if ko.Spec.VPCID == nil && ko.Status.RequesterVPCInfo != nil && ko.Status.RequesterVPCInfo.VPCID != nil {
		ko.Spec.VPCID = ko.Status.RequesterVPCInfo.VPCID
	}

	if r.ko.Spec.AccepterPeeringConnectionOptions != nil {
		f0 := &svcapitypes.PeeringConnectionOptionsRequest{}
		if r.ko.Spec.AccepterPeeringConnectionOptions.AllowDNSResolutionFromRemoteVPC != nil {
			f0.AllowEgressFromLocalClassicLinkToRemoteVPC = r.ko.Spec.AccepterPeeringConnectionOptions.AllowDNSResolutionFromRemoteVPC
		}
		if r.ko.Spec.AccepterPeeringConnectionOptions.AllowEgressFromLocalClassicLinkToRemoteVPC != nil {
			f0.AllowEgressFromLocalClassicLinkToRemoteVPC = r.ko.Spec.AccepterPeeringConnectionOptions.AllowEgressFromLocalClassicLinkToRemoteVPC
		}
		if r.ko.Spec.AccepterPeeringConnectionOptions.AllowEgressFromLocalVPCToRemoteClassicLink != nil {
			f0.AllowEgressFromLocalVPCToRemoteClassicLink = r.ko.Spec.AccepterPeeringConnectionOptions.AllowEgressFromLocalVPCToRemoteClassicLink
		}
		ko.Spec.AccepterPeeringConnectionOptions = f0
	} else {
		ko.Spec.AccepterPeeringConnectionOptions = nil
	}
	if r.ko.Spec.RequesterPeeringConnectionOptions != nil {
		f1 := &svcapitypes.PeeringConnectionOptionsRequest{}
		if r.ko.Spec.RequesterPeeringConnectionOptions.AllowDNSResolutionFromRemoteVPC != nil {
			f1.AllowDNSResolutionFromRemoteVPC = r.ko.Spec.RequesterPeeringConnectionOptions.AllowDNSResolutionFromRemoteVPC
		}
		if r.ko.Spec.RequesterPeeringConnectionOptions.AllowEgressFromLocalClassicLinkToRemoteVPC != nil {
			f1.AllowEgressFromLocalClassicLinkToRemoteVPC = r.ko.Spec.RequesterPeeringConnectionOptions.AllowEgressFromLocalClassicLinkToRemoteVPC
		}
		if r.ko.Spec.RequesterPeeringConnectionOptions.AllowEgressFromLocalVPCToRemoteClassicLink != nil {
			f1.AllowEgressFromLocalVPCToRemoteClassicLink = r.ko.Spec.RequesterPeeringConnectionOptions.AllowEgressFromLocalVPCToRemoteClassicLink
		}
		ko.Spec.RequesterPeeringConnectionOptions = f1
	} else {
		ko.Spec.RequesterPeeringConnectionOptions = nil
	}

	// Artificially trigger detection by delta.DifferentAt("Spec.AcceptRequest")
	res := &resource{ko}
	if isVPCPeeringConnectionPendingAcceptance(res) {
		res.ko.Spec.AcceptRequest = aws.Bool(false)
	} else if isVPCPeeringConnectionActive(res) || isVPCPeeringConnectionProvisioning(res) {
		res.ko.Spec.AcceptRequest = aws.Bool(true)
	} else if isVPCPeeringConnectionCreating(res) {
		return res, requeueWaitWhileCreating
	}

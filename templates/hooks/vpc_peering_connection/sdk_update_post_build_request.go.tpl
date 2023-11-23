	
	// Optionally include parameters in the payload before sending it, if they are in the resource's Spec
	trueCondition := true
	if desired.ko.Spec.AccepterPeeringConnectionOptions.AllowDNSResolutionFromRemoteVPC == &trueCondition {
		input.AccepterPeeringConnectionOptions.AllowDnsResolutionFromRemoteVpc = &trueCondition
	}
	if desired.ko.Spec.AccepterPeeringConnectionOptions.AllowEgressFromLocalClassicLinkToRemoteVPC == &trueCondition {
		input.AccepterPeeringConnectionOptions.AllowEgressFromLocalClassicLinkToRemoteVpc = &trueCondition
	}
	if desired.ko.Spec.AccepterPeeringConnectionOptions.AllowEgressFromLocalVPCToRemoteClassicLink == &trueCondition {
		input.AccepterPeeringConnectionOptions.AllowEgressFromLocalVpcToRemoteClassicLink = &trueCondition
	}
	if desired.ko.Spec.RequesterPeeringConnectionOptions.AllowDNSResolutionFromRemoteVPC == &trueCondition {
		input.RequesterPeeringConnectionOptions.AllowDnsResolutionFromRemoteVpc = &trueCondition
	}
	if desired.ko.Spec.RequesterPeeringConnectionOptions.AllowEgressFromLocalClassicLinkToRemoteVPC == &trueCondition {
		input.RequesterPeeringConnectionOptions.AllowEgressFromLocalClassicLinkToRemoteVpc = &trueCondition
	}
	if desired.ko.Spec.RequesterPeeringConnectionOptions.AllowEgressFromLocalVPCToRemoteClassicLink == &trueCondition {
		input.RequesterPeeringConnectionOptions.AllowEgressFromLocalVpcToRemoteClassicLink = &trueCondition
	}
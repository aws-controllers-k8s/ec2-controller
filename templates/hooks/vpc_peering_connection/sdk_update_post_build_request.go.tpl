	
	// Optionally include custom parameters in the payload before sending it
	trueCondition := true
	if desired.ko.Spec.AccepterPeeringConnectionOptions.AllowDNSResolutionFromRemoteVPC == &trueCondition {
		input.AccepterPeeringConnectionOptions.AllowDnsResolutionFromRemoteVpc = &trueCondition
	}
	if desired.ko.Spec.RequesterPeeringConnectionOptions.AllowDNSResolutionFromRemoteVPC == &trueCondition {
		input.RequesterPeeringConnectionOptions.AllowDnsResolutionFromRemoteVpc = &trueCondition
	}
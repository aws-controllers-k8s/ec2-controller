
	// Artificially trigger detection by delta.DifferentAt("Spec.AcceptRequest")
	res := &resource{ko}
	if isVPCPeeringConnectionPendingAcceptance(res) {
		res.ko.Spec.AcceptRequest = aws.Bool(false)
	} else if isVPCPeeringConnectionActive(res) || isVPCPeeringConnectionProvisioning(res) {
		res.ko.Spec.AcceptRequest = aws.Bool(true)
	} else if isVPCPeeringConnectionCreating(res) {
		return nil, requeueWaitWhileCreating
	}

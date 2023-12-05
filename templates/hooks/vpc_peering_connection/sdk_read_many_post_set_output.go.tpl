
   // Hack to artificially trigger detection by delta.DifferentAt("Spec.AcceptRequest")
	if isVPCPeeringConnectionPendingAcceptance(r) {
		r.ko.Spec.AcceptRequest = aws.Bool(true)
	} else if isVPCPeeringConnectionActive(r) || isVPCPeeringConnectionProvisioning(r) {
		r.ko.Spec.AcceptRequest = aws.Bool(false)
	}

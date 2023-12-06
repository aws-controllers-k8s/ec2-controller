
   // Hack to artificially trigger detection by delta.DifferentAt("Spec.AcceptRequest")
	if isVPCPeeringConnectionPendingAcceptance(r) {
		rlog.Debug("Setting VPC Peering Connection spec.acceptRequest to false", "Resource", r)
		r.ko.Spec.AcceptRequest = aws.Bool(false)
	} else if isVPCPeeringConnectionActive(r) || isVPCPeeringConnectionProvisioning(r) {
		rlog.Debug("Setting VPC Peering Connection spec.acceptRequest to true", "Resource", r)
		r.ko.Spec.AcceptRequest = aws.Bool(true)
	}

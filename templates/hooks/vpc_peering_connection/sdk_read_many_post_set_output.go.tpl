
	// Hack to artificially trigger detection by delta.DifferentAt("Spec.AcceptRequest")
	rlog.Debug("Hack to artificially trigger detection",
		"r.ko.Status.Status", r.ko.Status.Status,
		"r.ko.Spec.AcceptRequest", r.ko.Spec.AcceptRequest,
		"ko.Status.Status", ko.Status.Status,
		"ko.Spec.AcceptRequest", ko.Spec.AcceptRequest)
	if isVPCPeeringConnectionPendingAcceptance(r) {
		rlog.Debug("Setting VPC Peering Connection spec.acceptRequest to false")
		r.ko.Spec.AcceptRequest = aws.Bool(false)
	} else if isVPCPeeringConnectionActive(r) || isVPCPeeringConnectionProvisioning(r) {
		rlog.Debug("Setting VPC Peering Connection spec.acceptRequest to true")
		r.ko.Spec.AcceptRequest = aws.Bool(true)
	} else if isVPCPeeringConnectionCreating(r) {
		return nil, requeueWaitWhileCreating
	}

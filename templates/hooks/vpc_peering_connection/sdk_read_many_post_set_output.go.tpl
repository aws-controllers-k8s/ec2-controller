
	// Hack to artificially trigger detection by delta.DifferentAt("Spec.AcceptRequest")
	rlog.Debug("Hack to artificially trigger detection",
		"resource", r,
		"r.ko.Status.Status", r.ko.Status.Status,
		"r.ko.Status.Status.Code", r.ko.Status.Status.Code,
		"r.ko.Spec.AcceptRequest", r.ko.Spec.AcceptRequest,
		"ko", ko,
		"ko.Status.Status", ko.Status.Status,
		"ko.Status.Status.Code", ko.Status.Status.Code,
		"ko.Spec.AcceptRequest", ko.Spec.AcceptRequest)
	if isVPCPeeringConnectionPendingAcceptance(r) {
		rlog.Debug("Setting VPC Peering Connection spec.acceptRequest to false")
		ko.Spec.AcceptRequest = aws.Bool(false)
	} else if isVPCPeeringConnectionActive(r) || isVPCPeeringConnectionProvisioning(r) {
		rlog.Debug("Setting VPC Peering Connection spec.acceptRequest to true")
		ko.Spec.AcceptRequest = aws.Bool(true)
	}

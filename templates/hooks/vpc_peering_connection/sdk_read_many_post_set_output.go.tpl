
	// Hack to artificially trigger detection by delta.DifferentAt("Spec.AcceptRequest")
	rlog.Debug("Hack to artificially trigger detection",
		"r.ko.Status.Status", r.ko.Status.Status,
		"r.ko.Spec.AcceptRequest", r.ko.Spec.AcceptRequest,
		"ko.Status.Status", ko.Status.Status,
		"ko.Spec.AcceptRequest", ko.Spec.AcceptRequest)
	res := &resource{ko}
	if isVPCPeeringConnectionPendingAcceptance(res) {
		rlog.Debug("Setting VPC Peering Connection spec.acceptRequest to false")
		res.ko.Spec.AcceptRequest = aws.Bool(false)
	} else if isVPCPeeringConnectionActive(res) || isVPCPeeringConnectionProvisioning(res) {
		rlog.Debug("Setting VPC Peering Connection spec.acceptRequest to true")
		res.ko.Spec.AcceptRequest = aws.Bool(true)
	} else if isVPCPeeringConnectionCreating(res) {
		rlog.Debug("Requeuing until VPC Peering Connection is not Creating")
		return nil, requeueWaitWhileCreating
	}

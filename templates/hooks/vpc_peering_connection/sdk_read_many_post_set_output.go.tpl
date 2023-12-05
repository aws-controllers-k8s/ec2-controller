
    if isVPCPeeringConnectionPendingAcceptance(r) {
 		r.ko.Spec.AcceptRequest = aws.Bool(false)
	} else if isVPCPeeringConnectionActive(r) || isVPCPeeringConnectionProvisioning(r) {
		r.ko.Spec.AcceptRequest = aws.Bool(true)
 	}

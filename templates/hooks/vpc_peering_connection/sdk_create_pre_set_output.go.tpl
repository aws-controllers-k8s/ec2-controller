
    // Accept the VPC Peering Connection Request, if the field 'Spec.AcceptRequest' is set to true
    if *desired.ko.Spec.AcceptRequest {
        var acceptResp *svcsdk.AcceptVpcPeeringConnectionOutput
        _ = acceptResp
        acceptInput := &svcsdk.AcceptVpcPeeringConnectionInput{
            VpcPeeringConnectionId: ko.Status.VPCPeeringConnectionID,
        }
        acceptResp, err = rm.sdkapi.AcceptVpcPeeringConnectionWithContext(ctx, acceptInput)
        if err != nil {
            return nil, err
        }
        rlog.Debug("VPC Peering Connection accepted", "VpcPeeringConnectionId", *acceptResp.VpcPeeringConnection.VpcPeeringConnectionId)
	}

  
  if delta.DifferentAt("Spec.Tags") {
		if err := rm.syncTags(ctx, desired, latest); err != nil {
			return nil, err
		}
	}

  // Only continue if something other than Tags has changed in the Spec
  if !delta.DifferentExcept("Spec.Tags") {
      return desired, nil
  }

  // Accept the VPC Peering Connection Request, if the field 'Spec.AcceptRequest' is set to true
  if *desired.ko.Spec.AcceptRequest {
		if *latest.ko.Status.Status.Code == "pending-acceptance" {
			var acceptResp *svcsdk.AcceptVpcPeeringConnectionOutput
			_ = acceptResp
			acceptInput := &svcsdk.AcceptVpcPeeringConnectionInput{
				VpcPeeringConnectionId: latest.ko.Status.VPCPeeringConnectionID,
			}
			acceptResp, err = rm.sdkapi.AcceptVpcPeeringConnectionWithContext(ctx, acceptInput)
			if err != nil {
				return nil, err
			}
			rlog.Debug("VPC Peering Connection accepted", "VpcPeeringConnectionId", *acceptResp.VpcPeeringConnection.VpcPeeringConnectionId)
		}
	}

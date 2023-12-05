  
  if delta.DifferentAt("Spec.Tags") {
		if err := rm.syncTags(ctx, desired, latest); err != nil {
			return nil, err
		}
	}

	if delta.DifferentAt("Spec.AcceptRequest") {
         // Throw a Terminal Error, if the field was set to 'true' and is now set to 'false'
		 if !*desired.ko.Spec.AcceptRequest  {
         		msg := fmt.Sprintf("You cannot set AcceptRequest to false after setting it to true")
		        return nil, ackerr.NewTerminalError(fmt.Errorf(msg))
         }

         // Accept the VPC Peering Connection Request, if the field is set to 'true'
         acceptInput := &svcsdk.AcceptVpcPeeringConnectionInput{
             VpcPeeringConnectionId: latest.ko.Status.VPCPeeringConnectionID,
         }
         acceptResp, err := rm.sdkapi.AcceptVpcPeeringConnectionWithContext(ctx, acceptInput)
         if err != nil {
             return nil, err
         }
         rlog.Debug("VPC Peering Connection accepted", "VpcPeeringConnectionId", *acceptResp.VpcPeeringConnection.VpcPeeringConnectionId)
	}


  // Only continue if something other than Tags or certain fields has changed in the Spec
  if !delta.DifferentExcept("Spec.Tags", "Spec.AcceptRequest") {
      return desired, nil
  }
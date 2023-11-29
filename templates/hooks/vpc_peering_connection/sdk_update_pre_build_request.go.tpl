    // Custom update function to for Tags
    desired, err = rm.customUpdateVPCPeeringConnection(ctx, desired, latest, delta)
    if err != nil {
		return nil, err
	}
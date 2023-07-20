    if ko.Spec.VPC != nil {
		if err = rm.syncVPCs(ctx, &resource{ko},nil); err != nil {
			return nil, err
		}
	}
	vpcID, err := rm.getAttachedVPC(ctx, &resource{ko})
	if err != nil {
		return nil, err
	} else {
		ko.Spec.VPC = vpcID
	}
	ko.Spec.VPC, err = rm.getAttachedVPC(ctx, &resource{ko})
	if err != nil {
		return nil, err
	}
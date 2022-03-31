    if r.ko.Spec.VPC != nil && r.ko.Status.InternetGatewayID != nil {
		if err = rm.detachFromVPC(ctx, *r.ko.Spec.VPC, *r.ko.Status.InternetGatewayID); err != nil {
			return nil, err
		}
	}
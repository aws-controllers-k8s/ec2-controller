	if r.ko.Spec.VPC != nil && r.ko.Status.DHCPOptionsID != nil {
		desired := rm.concreteResource(r.DeepCopy())
		desired.ko.Spec.VPC = nil
		if err = rm.syncVPCs(ctx, desired,r); err != nil {
			return nil, err
		}
	}
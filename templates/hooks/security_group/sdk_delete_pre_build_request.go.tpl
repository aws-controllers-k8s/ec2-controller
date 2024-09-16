	sgCpy := r.ko.DeepCopy()
	sgCpy.Spec.IngressRules = nil
    sgCpy.Spec.EgressRules = nil
	if err := rm.syncSGRules(ctx, &resource{ko: sgCpy}, r); err != nil {
		return nil, err
	}
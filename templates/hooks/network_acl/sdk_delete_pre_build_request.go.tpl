
	if r.ko.Spec.Associations != nil {
		if err := rm.syncAssociation(ctx, nil, r); err != nil {
			return nil, err
		}
	}

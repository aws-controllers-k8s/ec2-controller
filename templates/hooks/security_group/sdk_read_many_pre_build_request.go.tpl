	if rm.requiredFieldsMissingFromReadManyInput(r) {
		id, err := rm.getSecurityGroupID(ctx, r)
		if err != nil {
			return nil, err
		}
		if id != nil {
			r.ko.Status.ID = id
		}
	}
	if r.ko.Spec.KeyName == nil {
		return nil, ackerr.NotFound
	}

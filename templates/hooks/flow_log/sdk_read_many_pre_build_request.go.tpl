    if r.ko.Status.FlowLogID == nil {
		return nil, ackerr.NotFound
	}
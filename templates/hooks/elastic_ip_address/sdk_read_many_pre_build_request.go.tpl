    if r.ko.Status.AllocationID == nil {
		return nil, ackerr.NotFound
	}
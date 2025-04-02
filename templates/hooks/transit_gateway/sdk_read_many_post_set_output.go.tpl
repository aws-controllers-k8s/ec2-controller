	if isResourceDeleted(&resource{ko}) {
		return nil, ackerr.NotFound
	}
	if isResourcePending(&resource{ko}) {
		return nil, ackrequeue.Needed(fmt.Errorf("resource is pending"))
	}

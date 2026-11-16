	if isResourceDeleted(&resource{ko}) {
		return nil, ackerr.NotFound
	}
	// Return the observed resource so the runtime patches status while pending.
	if isResourcePending(&resource{ko}) {
		return &resource{ko}, requeueWaitWhilePending
	}

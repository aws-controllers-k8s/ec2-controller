	if isResourceDeleted(&resource{ko}) {
		return nil, ackerr.NotFound
	}
	if isResourcePending(&resource{ko}) {
		return nil, ackrequeue.NeededAfter(
			fmt.Errorf("resource is pending"),
			ackrequeue.DefaultRequeueAfterDuration,
		)
	}

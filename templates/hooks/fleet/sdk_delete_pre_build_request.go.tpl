	if fleetDeleting(r) {
		return r, requeueWaitWhileDeleting
	}
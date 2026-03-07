    if err = addIDToDeleteRequest(r, input); err != nil {
        return nil, ackerr.NotFound
    }

    if r.ko.Spec.TerminateInstancesOnDeletion != nil {
        input.TerminateInstances = r.ko.Spec.TerminateInstancesOnDeletion
    } else {
        // Default TerminateInstances = true on DeleteFleets requests to prevent orphaned instances
	    input.TerminateInstances = aws.Bool(true)
    }
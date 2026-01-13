    if err = addIDToDeleteRequest(r, input); err != nil {
        return nil, ackerr.NotFound
    }
    
    // Always set TerminateInstances = true on DeleteFleets requests to prevent orphaned instances
	input.TerminateInstances = aws.Bool(true)
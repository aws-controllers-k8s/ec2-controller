    if err = addInstanceIDsToTerminateRequest(r, input); err != nil {
        return nil, ackerr.NotFound
    }
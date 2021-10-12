    if err = addIDToDeleteRequest(r, input); err != nil {
        return nil, ackerr.NotFound
    }
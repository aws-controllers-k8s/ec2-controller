	if err = addIdToListRequest(r, input); err != nil {
		return nil, ackerr.NotFound
	}
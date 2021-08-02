	err = addIdToListRequest(r, input)
	if err != nil {
		return nil, ackerr.NotFound
	}
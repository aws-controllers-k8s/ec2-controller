    if err = addReservationIDToListRequest(r, input); err != nil {
        return nil, ackerr.NotFound
    }
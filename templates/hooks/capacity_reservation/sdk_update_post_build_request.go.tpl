
    // If additionalInfo field is being updated, other fields cannot be modified simultaneously,
	// hence we update it separately or else we run into InvalidParameterCombination error
	if input.AdditionalInfo != nil {
		// update call with additional info only
		additionalInfoInput := &svcsdk.ModifyCapacityReservationInput{
			CapacityReservationId: input.CapacityReservationId,
			AdditionalInfo:        input.AdditionalInfo,
		}

		_, err = rm.sdkapi.ModifyCapacityReservation(ctx, additionalInfoInput)
		rm.metrics.RecordAPICall("UPDATE", "ModifyCapacityReservation", err)
		if err != nil {
			return nil, err
		}
	}
	
	// set additionalInfo to nil here because it has already been handled
	input.AdditionalInfo = nil

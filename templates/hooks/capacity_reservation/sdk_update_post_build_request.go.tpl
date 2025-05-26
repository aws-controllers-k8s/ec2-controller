
    // If additionalInfo field is being updated, other fields cannot be modified simultaneously,
	// hence we reset them or else we run into InvalidParameterCombination error
	if input.AdditionalInfo != nil {
		input.InstanceCount = nil
		input.EndDate = nil
		input.EndDateType = ""
		input.InstanceMatchCriteria = ""
	}

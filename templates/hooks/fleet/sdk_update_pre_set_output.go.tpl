
	// Modify Fleet doesn't return an updated Fleet object, so we need to set the state to "modifying" to reflect that the update is in progress
	ko.Status.FleetState = aws.String("modifying")

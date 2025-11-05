	// Prefix list names are not unique in AWS, so we must filter by ID
	// If we don't have a PrefixListID yet, the resource hasn't been created
	if r.ko.Status.PrefixListID != nil {
		input.PrefixListIds = []string{*r.ko.Status.PrefixListID}
	}


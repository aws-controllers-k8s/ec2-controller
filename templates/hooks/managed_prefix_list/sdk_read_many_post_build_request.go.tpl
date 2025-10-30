	// If we have a PrefixListID, use it to filter the describe call
	if r.ko.Status.PrefixListID != nil {
		input.PrefixListIds = []string{*r.ko.Status.PrefixListID}
	} else if r.ko.Spec.PrefixListName != nil {
		// If we don't have an ID yet, filter by both owner ID and prefix list name
		// This prevents matching against AWS-managed prefix lists
		// and other user-owned prefix lists with different names
		filters := []svcsdktypes.Filter{
			{
				Name:   aws.String("owner-id"),
				Values: []string{string(rm.awsAccountID)},
			},
			{
				Name:   aws.String("prefix-list-name"),
				Values: []string{*r.ko.Spec.PrefixListName},
			},
		}
		input.Filters = filters
	}


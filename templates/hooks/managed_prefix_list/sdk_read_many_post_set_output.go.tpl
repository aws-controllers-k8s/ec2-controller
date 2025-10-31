	// Get the entries separately
	if ko.Status.PrefixListID != nil {
		entriesResp, err := rm.sdkapi.GetManagedPrefixListEntries(
			ctx,
			&svcsdk.GetManagedPrefixListEntriesInput{
				PrefixListId: ko.Status.PrefixListID,
			},
		)
		rm.metrics.RecordAPICall("GET", "GetManagedPrefixListEntries", err)
		if err != nil {
			return nil, err
		}
		if entriesResp.Entries != nil {
			f0 := []*svcapitypes.AddPrefixListEntry{}
			for _, f0iter := range entriesResp.Entries {
				f0elem := &svcapitypes.AddPrefixListEntry{}
				if f0iter.Cidr != nil {
					f0elem.CIDR = f0iter.Cidr
				}
				if f0iter.Description != nil {
					f0elem.Description = f0iter.Description
				}
				f0 = append(f0, f0elem)
			}
			ko.Spec.Entries = f0
		} else {
			ko.Spec.Entries = nil
		}
	}



	if len(resp.PrefixLists) == 0 {
		return nil, ackerr.NotFound
	}
	found := false
	for _, elem := range resp.PrefixLists {
		if elem.PrefixListId != nil {
			if ko.Status.PrefixListID != nil {
				if *elem.PrefixListId == *ko.Status.PrefixListID {
					found = true
					if elem.PrefixListId != nil {
						ko.Status.PrefixListID = elem.PrefixListId
					} else {
						ko.Status.PrefixListID = nil
					}
					if elem.State != nil {
						ko.Status.State = elem.State
					} else {
						ko.Status.State = nil
					}
					if elem.Version != nil {
						ko.Status.Version = elem.Version
					} else {
						ko.Status.Version = nil
					}
					if elem.AddressFamily != nil {
						ko.Spec.AddressFamily = elem.AddressFamily
					} else {
						ko.Spec.AddressFamily = nil
					}
					if elem.MaxEntries != nil {
						ko.Spec.MaxEntries = elem.MaxEntries
					} else {
						ko.Spec.MaxEntries = nil
					}
					if elem.PrefixListName != nil {
						ko.Spec.PrefixListName = elem.PrefixListName
					} else {
						ko.Spec.PrefixListName = nil
					}
					if elem.Tags != nil {
						f6 := []*svcapitypes.Tag{}
						for _, f6iter := range elem.Tags {
							f6elem := &svcapitypes.Tag{}
							if f6iter.Key != nil {
								f6elem.Key = f6iter.Key
							}
							if f6iter.Value != nil {
								f6elem.Value = f6iter.Value
							}
							f6 = append(f6, f6elem)
						}
						ko.Spec.Tags = f6
					} else {
						ko.Spec.Tags = nil
					}
					break
				}
			}
		}
	}
	if !found {
		return nil, ackerr.NotFound
	}

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



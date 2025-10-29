	if resp.PrefixList != nil {
		if resp.PrefixList.PrefixListId != nil {
			ko.Status.PrefixListID = resp.PrefixList.PrefixListId
		} else {
			ko.Status.PrefixListID = nil
		}
		if resp.PrefixList.State != nil {
			ko.Status.State = resp.PrefixList.State
		} else {
			ko.Status.State = nil
		}
		if resp.PrefixList.Version != nil {
			ko.Status.Version = resp.PrefixList.Version
		} else {
			ko.Status.Version = nil
		}
	} else {
		ko.Status.PrefixListID = nil
		ko.Status.State = nil
		ko.Status.Version = nil
	}



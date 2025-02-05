 	if r.ko.Status.FlowLogID == nil {
		return nil, ackerr.NotFound
	}
    input.FlowLogIds = []string{*r.ko.Status.FlowLogID}
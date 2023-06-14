 	if r.ko.Status.FlowLogID == nil {
		return nil, ackerr.NotFound
	}
    input.SetFlowLogIds([]*string{r.ko.Status.FlowLogID})
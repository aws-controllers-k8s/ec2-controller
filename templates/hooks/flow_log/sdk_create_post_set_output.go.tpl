    if len(resp.FlowLogIds) > 0 && resp.FlowLogIds[0] != nil {
		ko.Status.FlowLogID = resp.FlowLogIds[0]
	}
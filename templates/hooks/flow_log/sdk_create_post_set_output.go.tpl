	if len(resp.FlowLogIds) > 0 && resp.FlowLogIds[0] != "" {
		ko.Status.FlowLogID = aws.String(resp.FlowLogIds[0])
	}
	// DeleteFlowLogs reports a refused delete in Unsuccessful, not as an error.
	// InvalidFlowLogId.NotFound means the flow log is already gone.
	if err == nil && resp != nil && len(resp.Unsuccessful) > 0 {
		failure := resp.Unsuccessful[0].Error
		if failure == nil {
			failure = &svcsdktypes.UnsuccessfulItemError{}
		}
		if aws.ToString(failure.Code) != "InvalidFlowLogId.NotFound" {
			err = &smithy.GenericAPIError{Code: aws.ToString(failure.Code), Message: aws.ToString(failure.Message)}
		}
	}

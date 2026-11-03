	// Defensive: DeleteFlowLogsOutput can report failures in Unsuccessful with
	// a nil error. EC2 currently returns these as top-level errors instead.
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

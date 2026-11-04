	// CreateFlowLogs reports a failed create in Unsuccessful, not as an error.
	if err == nil && resp != nil && len(resp.Unsuccessful) > 0 {
		failure := resp.Unsuccessful[0].Error
		if failure == nil {
			failure = &svcsdktypes.UnsuccessfulItemError{}
		}
		err = &smithy.GenericAPIError{Code: aws.ToString(failure.Code), Message: aws.ToString(failure.Message)}
	}

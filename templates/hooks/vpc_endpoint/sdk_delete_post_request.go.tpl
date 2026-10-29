	// DeleteVpcEndpoints returns no error for an endpoint it does not delete,
	// and lists the endpoint in Unsuccessful. Without this check, the runtime
	// deletes the custom resource and the endpoint stays in AWS. An entry with
	// code InvalidVpcEndpoint.NotFound means the endpoint is already gone.
	if err == nil && resp != nil && len(resp.Unsuccessful) > 0 {
		failure := resp.Unsuccessful[0].Error
		if failure == nil || aws.ToString(failure.Code) != "InvalidVpcEndpoint.NotFound" {
			errMsg := "failed to delete VPC Endpoint"
			if failure != nil && failure.Message != nil {
				errMsg = errMsg + ": " + *failure.Message
			}
			err = fmt.Errorf("%s", errMsg)
		}
	}

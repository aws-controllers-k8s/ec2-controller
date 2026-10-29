	// DeleteVpcEndpoints returns no error for an endpoint it does not delete,
	// and lists the endpoint in Unsuccessful. Without this check, the runtime
	// deletes the custom resource and the endpoint stays in AWS. An entry with
	// code InvalidVpcEndpoint.NotFound means the endpoint is already gone.
	// The error carries the entry's AWS code, so the ACK.Recoverable message
	// reads "api error <code>: <message>" like any other AWS error.
	if err == nil && resp != nil && len(resp.Unsuccessful) > 0 {
		failure := resp.Unsuccessful[0].Error
		if failure == nil {
			failure = &svcsdktypes.UnsuccessfulItemError{}
		}
		if aws.ToString(failure.Code) != "InvalidVpcEndpoint.NotFound" {
			err = &smithy.GenericAPIError{Code: aws.ToString(failure.Code), Message: aws.ToString(failure.Message)}
		}
	}

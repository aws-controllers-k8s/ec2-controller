	// DeleteVpcEndpointServiceConfigurations returns no error for a service
	// configuration it does not delete, and lists it in Unsuccessful. The
	// error carries the entry's AWS code, so the ACK.Recoverable message reads
	// "api error <code>: <message>" like any other AWS error.
	if err == nil && resp != nil && len(resp.Unsuccessful) > 0 {
		failure := resp.Unsuccessful[0].Error
		if failure == nil {
			failure = &svcsdktypes.UnsuccessfulItemError{}
		}
		err = &smithy.GenericAPIError{Code: aws.ToString(failure.Code), Message: aws.ToString(failure.Message)}
	}

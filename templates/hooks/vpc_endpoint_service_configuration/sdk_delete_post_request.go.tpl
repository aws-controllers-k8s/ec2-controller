	// DeleteVpcEndpointServiceConfigurations returns no error for a service
	// configuration it does not delete, and lists it in Unsuccessful. An entry
	// with code InvalidVpcEndpointService.NotFound means the service
	// configuration is already gone. The error carries the entry's AWS code,
	// so the ACK.Recoverable message reads "api error <code>: <message>" like
	// any other AWS error. An entry with no error gets code UnsuccessfulItem
	// and a message that names the service ID, so the message is not
	// "api error : ".
	if err == nil && resp != nil && len(resp.Unsuccessful) > 0 {
		failure := resp.Unsuccessful[0].Error
		if failure == nil {
			failure = &svcsdktypes.UnsuccessfulItemError{
				Code:    aws.String("UnsuccessfulItem"),
				Message: aws.String("EC2 did not delete " + aws.ToString(resp.Unsuccessful[0].ResourceId)),
			}
		}
		if aws.ToString(failure.Code) != "InvalidVpcEndpointService.NotFound" {
			err = &smithy.GenericAPIError{Code: aws.ToString(failure.Code), Message: aws.ToString(failure.Message)}
		}
	}

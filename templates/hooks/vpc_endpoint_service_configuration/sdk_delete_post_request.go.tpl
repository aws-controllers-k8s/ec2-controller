	if err == nil && resp != nil && len(resp.Unsuccessful) > 0 {
		errMsg := "failed to delete VPC Endpoint Service Configuration"
		if resp.Unsuccessful[0].Error != nil && resp.Unsuccessful[0].Error.Message != nil {
			errMsg = errMsg + ": " + *resp.Unsuccessful[0].Error.Message
		}
		err = fmt.Errorf("%s", errMsg)
	}

	if awsErr, ok := ackerr.AWSError(err); ok && awsErr.ErrorCode() == "InvalidLaunchTemplateName.NotFoundException" {
		err = ackerr.NotFound
	}

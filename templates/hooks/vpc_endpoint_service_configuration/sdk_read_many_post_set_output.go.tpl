
	permInput := &svcsdk.DescribeVpcEndpointServicePermissionsInput{
		ServiceId: ko.Status.ServiceID,
	}
	var permResp *svcsdk.DescribeVpcEndpointServicePermissionsOutput
	permResp, err = rm.sdkapi.DescribeVpcEndpointServicePermissionsWithContext(ctx, permInput)
	rm.metrics.RecordAPICall("READ_MANY", "DescribeVpcEndpointServicePermissions", err)
	if err != nil {
		if awsErr, ok := ackerr.AWSError(err); ok && awsErr.Code() == "UNKNOWN" {
			return nil, ackerr.NotFound
		}
		return nil, err
	}

	if permResp.AllowedPrincipals != nil {
		f0 := []*string{}
		for _, elem := range permResp.AllowedPrincipals {
			if elem.Principal != nil {
				f0 = append(f0, elem.Principal)
			}
		}
		ko.Spec.AllowedPrincipals = f0
	} else {
		ko.Spec.AllowedPrincipals = nil
	}

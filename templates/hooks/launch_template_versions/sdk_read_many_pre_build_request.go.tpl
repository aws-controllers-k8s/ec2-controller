	// sdk_read_many_pre_build starts here
	// This is to find latest version number of launch template and increment it by 1 as new version number as input to SDKFIND
	res_launch_template := &svcsdk.DescribeLaunchTemplatesInput{}

	if r.ko.Spec.DryRun != nil {
		res_launch_template.SetDryRun(*r.ko.Spec.DryRun)
	}
	//if r.ko.Spec.LaunchTemplateID != nil {
	//	f2 := []*string{}
	//	f2 = append(f2, r.ko.Spec.LaunchTemplateID)
	//	res_launch_template.SetLaunchTemplateIds(f2)
	//}

	if r.ko.Spec.LaunchTemplateName != nil {
		f2 := []*string{}
		f2 = append(f2, r.ko.Spec.LaunchTemplateName)
		res_launch_template.SetLaunchTemplateNames(f2)
	}

	var resp_launch_template *svcsdk.DescribeLaunchTemplatesOutput
	resp_launch_template, _ = rm.sdkapi.DescribeLaunchTemplatesWithContext(ctx, res_launch_template)
	rm.metrics.RecordAPICall("READ_MANY", "DescribeLaunchTemplates", err)
	if err != nil {
		if awsErr, ok := ackerr.AWSError(err); ok && awsErr.Code() == "InvalidLaunchTemplateName.NotFoundException" {
			return nil, ackerr.NotFound
		}
		return nil, err
	}

	for _, item := range resp_launch_template.LaunchTemplates {
		latest_version := item.LatestVersionNumber
		fmt.Println(" ======== PRINTING version number ===========")
		if r.ko.Status.VersionNumber != nil {
			fmt.Println(*r.ko.Status.VersionNumber)
			fmt.Println(*latest_version)
			//if *r.ko.Status.VersionNumber != *latest_version && len(r.ko.Status.Conditions) != 0 {
			//	fmt.Println(" ========  i am inside if ==========")
		    //		*latest_version++
		 	//	new_version_str := strconv.Itoa(int(*latest_version))
			//	input.SetVersions([]*string{&new_version_str})
			//} else {
				latest_version_str := strconv.Itoa(int(*latest_version))
				input.SetVersions([]*string{&latest_version_str})
			}else {
			*latest_version++
			new_version_str := strconv.Itoa(int(*latest_version))
			input.SetVersions([]*string{&new_version_str})
		}
	}

	// sdk_read_many_pre_build ends here
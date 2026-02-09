	for _, config := range input.LaunchTemplateConfigs {
		if config.LaunchTemplateSpecification != nil {

			// Ensure Launch Template Version is int and not $Latest/$Default
			// Preventing those values as as those strings are updated in the backend with the ints they represent
			// This confuses the reconciliation as the aws state falls out of sync with the CRD
			// If we stick to int strings, all works smoothly

			_, err := strconv.Atoi(*config.LaunchTemplateSpecification.Version)
			if err != nil {
				msg := "Only int values are supported for Launch Template Version in EC2 fleet spec"
				return nil, ackerr.NewTerminalError(fmt.Errorf("%s", msg))
			}
		}
	}
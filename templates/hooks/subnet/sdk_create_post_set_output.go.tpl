	// use desired resource data for fields that cannot be provided
	// in the create request, but are present in the create response;
	// otherwise, server-side data will incorrectly be treated as "desired"
	if desired.ko.Spec.AssignIPv6AddressOnCreation != nil {
		ko.Spec.AssignIPv6AddressOnCreation = desired.ko.Spec.AssignIPv6AddressOnCreation
	}
	if desired.ko.Spec.CustomerOwnedIPv4Pool != nil {
		ko.Spec.CustomerOwnedIPv4Pool = desired.ko.Spec.CustomerOwnedIPv4Pool
	}
	if desired.ko.Spec.EnableDNS64 != nil {
		ko.Spec.EnableDNS64 = desired.ko.Spec.EnableDNS64
	}
	if desired.ko.Spec.MapPublicIPOnLaunch != nil {
		ko.Spec.MapPublicIPOnLaunch = desired.ko.Spec.MapPublicIPOnLaunch
	}
    
	ackcondition.SetSynced(&resource{ko}, corev1.ConditionFalse, aws.String("subnet created, requeue for updates"), nil)
	err = ackrequeue.NeededAfter(fmt.Errorf("Reconciling to sync additional fields"), time.Second)
	return &resource{ko}, err

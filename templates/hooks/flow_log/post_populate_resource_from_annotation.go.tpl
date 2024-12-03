	if resourceID, ok := fields["resourceID"]; ok {
		r.ko.Spec.ResourceID = &resourceID
	} else {
		return ackerrors.MissingNameIdentifier
	}

	if resourceType, ok := fields["resourceType"]; ok {
			r.ko.Spec.ResourceType = &resourceType
	} else {
		return ackerrors.MissingNameIdentifier
	}

	identifierResourceID, identifierResourceIDOk := identifier.AdditionalKeys["resourceID"]
	if identifierResourceIDOk {
		r.ko.Spec.ResourceID = &identifierResourceID
	}
        identifierResourceType, identifierResourceTypeOk := identifier.AdditionalKeys["resourceType"]
        if identifierResourceTypeOk {
                r.ko.Spec.ResourceType = &identifierResourceType
        }

	if resourceID, ok := identifier.AdditionalKeys["resourceID"]; ok {
                r.ko.Spec.ResourceID = &resourceID
        }

        if resourceType, ok := identifier.AdditionalKeys["resourceType"]; ok {
                r.ko.Spec.ResourceType = &resourceType
        }

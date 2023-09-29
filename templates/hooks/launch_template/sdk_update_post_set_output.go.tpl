if desired.ko.Spec.Tags != nil {
		desiredTags := []*svcapitypes.Tag{}
		for _, tagitem := range desired.ko.Spec.Tags {
			elem := &svcapitypes.Tag{}
			if tagitem.Key != nil {
				elem.Key = tagitem.Key
			}
			if tagitem.Value != nil {
				elem.Value = tagitem.Value
			}
		
			desiredTags = append(desiredTags, elem)		
		}
		ko.Spec.Tags = desiredTags
	
	}else {
		ko.Spec.Tags = nil
	}

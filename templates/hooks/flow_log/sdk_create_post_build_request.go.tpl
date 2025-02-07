    updateTagSpecificationsInCreateRequest(desired, input)
    input.ResourceIds = []string{*desired.ko.Spec.ResourceID}
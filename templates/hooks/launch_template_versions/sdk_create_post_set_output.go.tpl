if ko.Status.VersionNumber != nil {
		newversionnumber := *ko.Status.VersionNumber + 1
		ko.Status.VersionNumber = &newversionnumber
	}

var t bool
defaultversion  := &t
*defaultversion = false

ko.Status.DefaultVersion = defaultversion
	// Regional NAT Gateways do not currently support private connectivity type.
	// The EC2 API silently overrides ConnectivityType to "public" instead of
	// returning an error, which would cause the CR spec to drift from the
	// actual resource state. Reject this combination upfront.
	// See: https://aws.amazon.com/blogs/networking-and-content-delivery/introducing-amazon-vpc-regional-nat-gateway/
	if desired.ko.Spec.AvailabilityMode != nil &&
		*desired.ko.Spec.AvailabilityMode == string(svcsdktypes.AvailabilityModeRegional) &&
		desired.ko.Spec.ConnectivityType != nil &&
		*desired.ko.Spec.ConnectivityType == string(svcsdktypes.ConnectivityTypePrivate) {
		return nil, ackerr.NewTerminalError(
			fmt.Errorf("regional NAT gateways do not currently support private connectivity type; use connectivityType \"public\" or use availabilityMode \"zonal\" for private connectivity"),
		)
	}

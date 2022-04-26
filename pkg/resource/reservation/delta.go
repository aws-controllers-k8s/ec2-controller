// Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

// Code generated by ack-generate. DO NOT EDIT.

package reservation

import (
	"bytes"
	"reflect"

	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
)

// Hack to avoid import errors during build...
var (
	_ = &bytes.Buffer{}
	_ = &reflect.Method{}
)

// newResourceDelta returns a new `ackcompare.Delta` used to compare two
// resources
func newResourceDelta(
	a *resource,
	b *resource,
) *ackcompare.Delta {
	delta := ackcompare.NewDelta()
	if (a == nil && b != nil) ||
		(a != nil && b == nil) {
		delta.Add("", a, b)
		return delta
	}

	if ackcompare.HasNilDifference(a.ko.Spec.AdditionalInfo, b.ko.Spec.AdditionalInfo) {
		delta.Add("Spec.AdditionalInfo", a.ko.Spec.AdditionalInfo, b.ko.Spec.AdditionalInfo)
	} else if a.ko.Spec.AdditionalInfo != nil && b.ko.Spec.AdditionalInfo != nil {
		if *a.ko.Spec.AdditionalInfo != *b.ko.Spec.AdditionalInfo {
			delta.Add("Spec.AdditionalInfo", a.ko.Spec.AdditionalInfo, b.ko.Spec.AdditionalInfo)
		}
	}
	if !reflect.DeepEqual(a.ko.Spec.BlockDeviceMappings, b.ko.Spec.BlockDeviceMappings) {
		delta.Add("Spec.BlockDeviceMappings", a.ko.Spec.BlockDeviceMappings, b.ko.Spec.BlockDeviceMappings)
	}
	if ackcompare.HasNilDifference(a.ko.Spec.CapacityReservationSpecification, b.ko.Spec.CapacityReservationSpecification) {
		delta.Add("Spec.CapacityReservationSpecification", a.ko.Spec.CapacityReservationSpecification, b.ko.Spec.CapacityReservationSpecification)
	} else if a.ko.Spec.CapacityReservationSpecification != nil && b.ko.Spec.CapacityReservationSpecification != nil {
		if ackcompare.HasNilDifference(a.ko.Spec.CapacityReservationSpecification.CapacityReservationPreference, b.ko.Spec.CapacityReservationSpecification.CapacityReservationPreference) {
			delta.Add("Spec.CapacityReservationSpecification.CapacityReservationPreference", a.ko.Spec.CapacityReservationSpecification.CapacityReservationPreference, b.ko.Spec.CapacityReservationSpecification.CapacityReservationPreference)
		} else if a.ko.Spec.CapacityReservationSpecification.CapacityReservationPreference != nil && b.ko.Spec.CapacityReservationSpecification.CapacityReservationPreference != nil {
			if *a.ko.Spec.CapacityReservationSpecification.CapacityReservationPreference != *b.ko.Spec.CapacityReservationSpecification.CapacityReservationPreference {
				delta.Add("Spec.CapacityReservationSpecification.CapacityReservationPreference", a.ko.Spec.CapacityReservationSpecification.CapacityReservationPreference, b.ko.Spec.CapacityReservationSpecification.CapacityReservationPreference)
			}
		}
		if ackcompare.HasNilDifference(a.ko.Spec.CapacityReservationSpecification.CapacityReservationTarget, b.ko.Spec.CapacityReservationSpecification.CapacityReservationTarget) {
			delta.Add("Spec.CapacityReservationSpecification.CapacityReservationTarget", a.ko.Spec.CapacityReservationSpecification.CapacityReservationTarget, b.ko.Spec.CapacityReservationSpecification.CapacityReservationTarget)
		} else if a.ko.Spec.CapacityReservationSpecification.CapacityReservationTarget != nil && b.ko.Spec.CapacityReservationSpecification.CapacityReservationTarget != nil {
			if ackcompare.HasNilDifference(a.ko.Spec.CapacityReservationSpecification.CapacityReservationTarget.CapacityReservationID, b.ko.Spec.CapacityReservationSpecification.CapacityReservationTarget.CapacityReservationID) {
				delta.Add("Spec.CapacityReservationSpecification.CapacityReservationTarget.CapacityReservationID", a.ko.Spec.CapacityReservationSpecification.CapacityReservationTarget.CapacityReservationID, b.ko.Spec.CapacityReservationSpecification.CapacityReservationTarget.CapacityReservationID)
			} else if a.ko.Spec.CapacityReservationSpecification.CapacityReservationTarget.CapacityReservationID != nil && b.ko.Spec.CapacityReservationSpecification.CapacityReservationTarget.CapacityReservationID != nil {
				if *a.ko.Spec.CapacityReservationSpecification.CapacityReservationTarget.CapacityReservationID != *b.ko.Spec.CapacityReservationSpecification.CapacityReservationTarget.CapacityReservationID {
					delta.Add("Spec.CapacityReservationSpecification.CapacityReservationTarget.CapacityReservationID", a.ko.Spec.CapacityReservationSpecification.CapacityReservationTarget.CapacityReservationID, b.ko.Spec.CapacityReservationSpecification.CapacityReservationTarget.CapacityReservationID)
				}
			}
			if ackcompare.HasNilDifference(a.ko.Spec.CapacityReservationSpecification.CapacityReservationTarget.CapacityReservationResourceGroupARN, b.ko.Spec.CapacityReservationSpecification.CapacityReservationTarget.CapacityReservationResourceGroupARN) {
				delta.Add("Spec.CapacityReservationSpecification.CapacityReservationTarget.CapacityReservationResourceGroupARN", a.ko.Spec.CapacityReservationSpecification.CapacityReservationTarget.CapacityReservationResourceGroupARN, b.ko.Spec.CapacityReservationSpecification.CapacityReservationTarget.CapacityReservationResourceGroupARN)
			} else if a.ko.Spec.CapacityReservationSpecification.CapacityReservationTarget.CapacityReservationResourceGroupARN != nil && b.ko.Spec.CapacityReservationSpecification.CapacityReservationTarget.CapacityReservationResourceGroupARN != nil {
				if *a.ko.Spec.CapacityReservationSpecification.CapacityReservationTarget.CapacityReservationResourceGroupARN != *b.ko.Spec.CapacityReservationSpecification.CapacityReservationTarget.CapacityReservationResourceGroupARN {
					delta.Add("Spec.CapacityReservationSpecification.CapacityReservationTarget.CapacityReservationResourceGroupARN", a.ko.Spec.CapacityReservationSpecification.CapacityReservationTarget.CapacityReservationResourceGroupARN, b.ko.Spec.CapacityReservationSpecification.CapacityReservationTarget.CapacityReservationResourceGroupARN)
				}
			}
		}
	}
	if ackcompare.HasNilDifference(a.ko.Spec.ClientToken, b.ko.Spec.ClientToken) {
		delta.Add("Spec.ClientToken", a.ko.Spec.ClientToken, b.ko.Spec.ClientToken)
	} else if a.ko.Spec.ClientToken != nil && b.ko.Spec.ClientToken != nil {
		if *a.ko.Spec.ClientToken != *b.ko.Spec.ClientToken {
			delta.Add("Spec.ClientToken", a.ko.Spec.ClientToken, b.ko.Spec.ClientToken)
		}
	}
	if ackcompare.HasNilDifference(a.ko.Spec.CPUOptions, b.ko.Spec.CPUOptions) {
		delta.Add("Spec.CPUOptions", a.ko.Spec.CPUOptions, b.ko.Spec.CPUOptions)
	} else if a.ko.Spec.CPUOptions != nil && b.ko.Spec.CPUOptions != nil {
		if ackcompare.HasNilDifference(a.ko.Spec.CPUOptions.CoreCount, b.ko.Spec.CPUOptions.CoreCount) {
			delta.Add("Spec.CPUOptions.CoreCount", a.ko.Spec.CPUOptions.CoreCount, b.ko.Spec.CPUOptions.CoreCount)
		} else if a.ko.Spec.CPUOptions.CoreCount != nil && b.ko.Spec.CPUOptions.CoreCount != nil {
			if *a.ko.Spec.CPUOptions.CoreCount != *b.ko.Spec.CPUOptions.CoreCount {
				delta.Add("Spec.CPUOptions.CoreCount", a.ko.Spec.CPUOptions.CoreCount, b.ko.Spec.CPUOptions.CoreCount)
			}
		}
		if ackcompare.HasNilDifference(a.ko.Spec.CPUOptions.ThreadsPerCore, b.ko.Spec.CPUOptions.ThreadsPerCore) {
			delta.Add("Spec.CPUOptions.ThreadsPerCore", a.ko.Spec.CPUOptions.ThreadsPerCore, b.ko.Spec.CPUOptions.ThreadsPerCore)
		} else if a.ko.Spec.CPUOptions.ThreadsPerCore != nil && b.ko.Spec.CPUOptions.ThreadsPerCore != nil {
			if *a.ko.Spec.CPUOptions.ThreadsPerCore != *b.ko.Spec.CPUOptions.ThreadsPerCore {
				delta.Add("Spec.CPUOptions.ThreadsPerCore", a.ko.Spec.CPUOptions.ThreadsPerCore, b.ko.Spec.CPUOptions.ThreadsPerCore)
			}
		}
	}
	if ackcompare.HasNilDifference(a.ko.Spec.CreditSpecification, b.ko.Spec.CreditSpecification) {
		delta.Add("Spec.CreditSpecification", a.ko.Spec.CreditSpecification, b.ko.Spec.CreditSpecification)
	} else if a.ko.Spec.CreditSpecification != nil && b.ko.Spec.CreditSpecification != nil {
		if ackcompare.HasNilDifference(a.ko.Spec.CreditSpecification.CPUCredits, b.ko.Spec.CreditSpecification.CPUCredits) {
			delta.Add("Spec.CreditSpecification.CPUCredits", a.ko.Spec.CreditSpecification.CPUCredits, b.ko.Spec.CreditSpecification.CPUCredits)
		} else if a.ko.Spec.CreditSpecification.CPUCredits != nil && b.ko.Spec.CreditSpecification.CPUCredits != nil {
			if *a.ko.Spec.CreditSpecification.CPUCredits != *b.ko.Spec.CreditSpecification.CPUCredits {
				delta.Add("Spec.CreditSpecification.CPUCredits", a.ko.Spec.CreditSpecification.CPUCredits, b.ko.Spec.CreditSpecification.CPUCredits)
			}
		}
	}
	if ackcompare.HasNilDifference(a.ko.Spec.DisableAPITermination, b.ko.Spec.DisableAPITermination) {
		delta.Add("Spec.DisableAPITermination", a.ko.Spec.DisableAPITermination, b.ko.Spec.DisableAPITermination)
	} else if a.ko.Spec.DisableAPITermination != nil && b.ko.Spec.DisableAPITermination != nil {
		if *a.ko.Spec.DisableAPITermination != *b.ko.Spec.DisableAPITermination {
			delta.Add("Spec.DisableAPITermination", a.ko.Spec.DisableAPITermination, b.ko.Spec.DisableAPITermination)
		}
	}
	if ackcompare.HasNilDifference(a.ko.Spec.EBSOptimized, b.ko.Spec.EBSOptimized) {
		delta.Add("Spec.EBSOptimized", a.ko.Spec.EBSOptimized, b.ko.Spec.EBSOptimized)
	} else if a.ko.Spec.EBSOptimized != nil && b.ko.Spec.EBSOptimized != nil {
		if *a.ko.Spec.EBSOptimized != *b.ko.Spec.EBSOptimized {
			delta.Add("Spec.EBSOptimized", a.ko.Spec.EBSOptimized, b.ko.Spec.EBSOptimized)
		}
	}
	if !reflect.DeepEqual(a.ko.Spec.ElasticGPUSpecification, b.ko.Spec.ElasticGPUSpecification) {
		delta.Add("Spec.ElasticGPUSpecification", a.ko.Spec.ElasticGPUSpecification, b.ko.Spec.ElasticGPUSpecification)
	}
	if !reflect.DeepEqual(a.ko.Spec.ElasticInferenceAccelerators, b.ko.Spec.ElasticInferenceAccelerators) {
		delta.Add("Spec.ElasticInferenceAccelerators", a.ko.Spec.ElasticInferenceAccelerators, b.ko.Spec.ElasticInferenceAccelerators)
	}
	if ackcompare.HasNilDifference(a.ko.Spec.EnclaveOptions, b.ko.Spec.EnclaveOptions) {
		delta.Add("Spec.EnclaveOptions", a.ko.Spec.EnclaveOptions, b.ko.Spec.EnclaveOptions)
	} else if a.ko.Spec.EnclaveOptions != nil && b.ko.Spec.EnclaveOptions != nil {
		if ackcompare.HasNilDifference(a.ko.Spec.EnclaveOptions.Enabled, b.ko.Spec.EnclaveOptions.Enabled) {
			delta.Add("Spec.EnclaveOptions.Enabled", a.ko.Spec.EnclaveOptions.Enabled, b.ko.Spec.EnclaveOptions.Enabled)
		} else if a.ko.Spec.EnclaveOptions.Enabled != nil && b.ko.Spec.EnclaveOptions.Enabled != nil {
			if *a.ko.Spec.EnclaveOptions.Enabled != *b.ko.Spec.EnclaveOptions.Enabled {
				delta.Add("Spec.EnclaveOptions.Enabled", a.ko.Spec.EnclaveOptions.Enabled, b.ko.Spec.EnclaveOptions.Enabled)
			}
		}
	}
	if ackcompare.HasNilDifference(a.ko.Spec.HibernationOptions, b.ko.Spec.HibernationOptions) {
		delta.Add("Spec.HibernationOptions", a.ko.Spec.HibernationOptions, b.ko.Spec.HibernationOptions)
	} else if a.ko.Spec.HibernationOptions != nil && b.ko.Spec.HibernationOptions != nil {
		if ackcompare.HasNilDifference(a.ko.Spec.HibernationOptions.Configured, b.ko.Spec.HibernationOptions.Configured) {
			delta.Add("Spec.HibernationOptions.Configured", a.ko.Spec.HibernationOptions.Configured, b.ko.Spec.HibernationOptions.Configured)
		} else if a.ko.Spec.HibernationOptions.Configured != nil && b.ko.Spec.HibernationOptions.Configured != nil {
			if *a.ko.Spec.HibernationOptions.Configured != *b.ko.Spec.HibernationOptions.Configured {
				delta.Add("Spec.HibernationOptions.Configured", a.ko.Spec.HibernationOptions.Configured, b.ko.Spec.HibernationOptions.Configured)
			}
		}
	}
	if ackcompare.HasNilDifference(a.ko.Spec.IAMInstanceProfile, b.ko.Spec.IAMInstanceProfile) {
		delta.Add("Spec.IAMInstanceProfile", a.ko.Spec.IAMInstanceProfile, b.ko.Spec.IAMInstanceProfile)
	} else if a.ko.Spec.IAMInstanceProfile != nil && b.ko.Spec.IAMInstanceProfile != nil {
		if ackcompare.HasNilDifference(a.ko.Spec.IAMInstanceProfile.ARN, b.ko.Spec.IAMInstanceProfile.ARN) {
			delta.Add("Spec.IAMInstanceProfile.ARN", a.ko.Spec.IAMInstanceProfile.ARN, b.ko.Spec.IAMInstanceProfile.ARN)
		} else if a.ko.Spec.IAMInstanceProfile.ARN != nil && b.ko.Spec.IAMInstanceProfile.ARN != nil {
			if *a.ko.Spec.IAMInstanceProfile.ARN != *b.ko.Spec.IAMInstanceProfile.ARN {
				delta.Add("Spec.IAMInstanceProfile.ARN", a.ko.Spec.IAMInstanceProfile.ARN, b.ko.Spec.IAMInstanceProfile.ARN)
			}
		}
		if ackcompare.HasNilDifference(a.ko.Spec.IAMInstanceProfile.Name, b.ko.Spec.IAMInstanceProfile.Name) {
			delta.Add("Spec.IAMInstanceProfile.Name", a.ko.Spec.IAMInstanceProfile.Name, b.ko.Spec.IAMInstanceProfile.Name)
		} else if a.ko.Spec.IAMInstanceProfile.Name != nil && b.ko.Spec.IAMInstanceProfile.Name != nil {
			if *a.ko.Spec.IAMInstanceProfile.Name != *b.ko.Spec.IAMInstanceProfile.Name {
				delta.Add("Spec.IAMInstanceProfile.Name", a.ko.Spec.IAMInstanceProfile.Name, b.ko.Spec.IAMInstanceProfile.Name)
			}
		}
	}
	if ackcompare.HasNilDifference(a.ko.Spec.ImageID, b.ko.Spec.ImageID) {
		delta.Add("Spec.ImageID", a.ko.Spec.ImageID, b.ko.Spec.ImageID)
	} else if a.ko.Spec.ImageID != nil && b.ko.Spec.ImageID != nil {
		if *a.ko.Spec.ImageID != *b.ko.Spec.ImageID {
			delta.Add("Spec.ImageID", a.ko.Spec.ImageID, b.ko.Spec.ImageID)
		}
	}
	if ackcompare.HasNilDifference(a.ko.Spec.InstanceInitiatedShutdownBehavior, b.ko.Spec.InstanceInitiatedShutdownBehavior) {
		delta.Add("Spec.InstanceInitiatedShutdownBehavior", a.ko.Spec.InstanceInitiatedShutdownBehavior, b.ko.Spec.InstanceInitiatedShutdownBehavior)
	} else if a.ko.Spec.InstanceInitiatedShutdownBehavior != nil && b.ko.Spec.InstanceInitiatedShutdownBehavior != nil {
		if *a.ko.Spec.InstanceInitiatedShutdownBehavior != *b.ko.Spec.InstanceInitiatedShutdownBehavior {
			delta.Add("Spec.InstanceInitiatedShutdownBehavior", a.ko.Spec.InstanceInitiatedShutdownBehavior, b.ko.Spec.InstanceInitiatedShutdownBehavior)
		}
	}
	if ackcompare.HasNilDifference(a.ko.Spec.InstanceMarketOptions, b.ko.Spec.InstanceMarketOptions) {
		delta.Add("Spec.InstanceMarketOptions", a.ko.Spec.InstanceMarketOptions, b.ko.Spec.InstanceMarketOptions)
	} else if a.ko.Spec.InstanceMarketOptions != nil && b.ko.Spec.InstanceMarketOptions != nil {
		if ackcompare.HasNilDifference(a.ko.Spec.InstanceMarketOptions.MarketType, b.ko.Spec.InstanceMarketOptions.MarketType) {
			delta.Add("Spec.InstanceMarketOptions.MarketType", a.ko.Spec.InstanceMarketOptions.MarketType, b.ko.Spec.InstanceMarketOptions.MarketType)
		} else if a.ko.Spec.InstanceMarketOptions.MarketType != nil && b.ko.Spec.InstanceMarketOptions.MarketType != nil {
			if *a.ko.Spec.InstanceMarketOptions.MarketType != *b.ko.Spec.InstanceMarketOptions.MarketType {
				delta.Add("Spec.InstanceMarketOptions.MarketType", a.ko.Spec.InstanceMarketOptions.MarketType, b.ko.Spec.InstanceMarketOptions.MarketType)
			}
		}
		if ackcompare.HasNilDifference(a.ko.Spec.InstanceMarketOptions.SpotOptions, b.ko.Spec.InstanceMarketOptions.SpotOptions) {
			delta.Add("Spec.InstanceMarketOptions.SpotOptions", a.ko.Spec.InstanceMarketOptions.SpotOptions, b.ko.Spec.InstanceMarketOptions.SpotOptions)
		} else if a.ko.Spec.InstanceMarketOptions.SpotOptions != nil && b.ko.Spec.InstanceMarketOptions.SpotOptions != nil {
			if ackcompare.HasNilDifference(a.ko.Spec.InstanceMarketOptions.SpotOptions.BlockDurationMinutes, b.ko.Spec.InstanceMarketOptions.SpotOptions.BlockDurationMinutes) {
				delta.Add("Spec.InstanceMarketOptions.SpotOptions.BlockDurationMinutes", a.ko.Spec.InstanceMarketOptions.SpotOptions.BlockDurationMinutes, b.ko.Spec.InstanceMarketOptions.SpotOptions.BlockDurationMinutes)
			} else if a.ko.Spec.InstanceMarketOptions.SpotOptions.BlockDurationMinutes != nil && b.ko.Spec.InstanceMarketOptions.SpotOptions.BlockDurationMinutes != nil {
				if *a.ko.Spec.InstanceMarketOptions.SpotOptions.BlockDurationMinutes != *b.ko.Spec.InstanceMarketOptions.SpotOptions.BlockDurationMinutes {
					delta.Add("Spec.InstanceMarketOptions.SpotOptions.BlockDurationMinutes", a.ko.Spec.InstanceMarketOptions.SpotOptions.BlockDurationMinutes, b.ko.Spec.InstanceMarketOptions.SpotOptions.BlockDurationMinutes)
				}
			}
			if ackcompare.HasNilDifference(a.ko.Spec.InstanceMarketOptions.SpotOptions.InstanceInterruptionBehavior, b.ko.Spec.InstanceMarketOptions.SpotOptions.InstanceInterruptionBehavior) {
				delta.Add("Spec.InstanceMarketOptions.SpotOptions.InstanceInterruptionBehavior", a.ko.Spec.InstanceMarketOptions.SpotOptions.InstanceInterruptionBehavior, b.ko.Spec.InstanceMarketOptions.SpotOptions.InstanceInterruptionBehavior)
			} else if a.ko.Spec.InstanceMarketOptions.SpotOptions.InstanceInterruptionBehavior != nil && b.ko.Spec.InstanceMarketOptions.SpotOptions.InstanceInterruptionBehavior != nil {
				if *a.ko.Spec.InstanceMarketOptions.SpotOptions.InstanceInterruptionBehavior != *b.ko.Spec.InstanceMarketOptions.SpotOptions.InstanceInterruptionBehavior {
					delta.Add("Spec.InstanceMarketOptions.SpotOptions.InstanceInterruptionBehavior", a.ko.Spec.InstanceMarketOptions.SpotOptions.InstanceInterruptionBehavior, b.ko.Spec.InstanceMarketOptions.SpotOptions.InstanceInterruptionBehavior)
				}
			}
			if ackcompare.HasNilDifference(a.ko.Spec.InstanceMarketOptions.SpotOptions.MaxPrice, b.ko.Spec.InstanceMarketOptions.SpotOptions.MaxPrice) {
				delta.Add("Spec.InstanceMarketOptions.SpotOptions.MaxPrice", a.ko.Spec.InstanceMarketOptions.SpotOptions.MaxPrice, b.ko.Spec.InstanceMarketOptions.SpotOptions.MaxPrice)
			} else if a.ko.Spec.InstanceMarketOptions.SpotOptions.MaxPrice != nil && b.ko.Spec.InstanceMarketOptions.SpotOptions.MaxPrice != nil {
				if *a.ko.Spec.InstanceMarketOptions.SpotOptions.MaxPrice != *b.ko.Spec.InstanceMarketOptions.SpotOptions.MaxPrice {
					delta.Add("Spec.InstanceMarketOptions.SpotOptions.MaxPrice", a.ko.Spec.InstanceMarketOptions.SpotOptions.MaxPrice, b.ko.Spec.InstanceMarketOptions.SpotOptions.MaxPrice)
				}
			}
			if ackcompare.HasNilDifference(a.ko.Spec.InstanceMarketOptions.SpotOptions.SpotInstanceType, b.ko.Spec.InstanceMarketOptions.SpotOptions.SpotInstanceType) {
				delta.Add("Spec.InstanceMarketOptions.SpotOptions.SpotInstanceType", a.ko.Spec.InstanceMarketOptions.SpotOptions.SpotInstanceType, b.ko.Spec.InstanceMarketOptions.SpotOptions.SpotInstanceType)
			} else if a.ko.Spec.InstanceMarketOptions.SpotOptions.SpotInstanceType != nil && b.ko.Spec.InstanceMarketOptions.SpotOptions.SpotInstanceType != nil {
				if *a.ko.Spec.InstanceMarketOptions.SpotOptions.SpotInstanceType != *b.ko.Spec.InstanceMarketOptions.SpotOptions.SpotInstanceType {
					delta.Add("Spec.InstanceMarketOptions.SpotOptions.SpotInstanceType", a.ko.Spec.InstanceMarketOptions.SpotOptions.SpotInstanceType, b.ko.Spec.InstanceMarketOptions.SpotOptions.SpotInstanceType)
				}
			}
			if ackcompare.HasNilDifference(a.ko.Spec.InstanceMarketOptions.SpotOptions.ValidUntil, b.ko.Spec.InstanceMarketOptions.SpotOptions.ValidUntil) {
				delta.Add("Spec.InstanceMarketOptions.SpotOptions.ValidUntil", a.ko.Spec.InstanceMarketOptions.SpotOptions.ValidUntil, b.ko.Spec.InstanceMarketOptions.SpotOptions.ValidUntil)
			} else if a.ko.Spec.InstanceMarketOptions.SpotOptions.ValidUntil != nil && b.ko.Spec.InstanceMarketOptions.SpotOptions.ValidUntil != nil {
				if !a.ko.Spec.InstanceMarketOptions.SpotOptions.ValidUntil.Equal(b.ko.Spec.InstanceMarketOptions.SpotOptions.ValidUntil) {
					delta.Add("Spec.InstanceMarketOptions.SpotOptions.ValidUntil", a.ko.Spec.InstanceMarketOptions.SpotOptions.ValidUntil, b.ko.Spec.InstanceMarketOptions.SpotOptions.ValidUntil)
				}
			}
		}
	}
	if ackcompare.HasNilDifference(a.ko.Spec.InstanceType, b.ko.Spec.InstanceType) {
		delta.Add("Spec.InstanceType", a.ko.Spec.InstanceType, b.ko.Spec.InstanceType)
	} else if a.ko.Spec.InstanceType != nil && b.ko.Spec.InstanceType != nil {
		if *a.ko.Spec.InstanceType != *b.ko.Spec.InstanceType {
			delta.Add("Spec.InstanceType", a.ko.Spec.InstanceType, b.ko.Spec.InstanceType)
		}
	}
	if ackcompare.HasNilDifference(a.ko.Spec.IPv6AddressCount, b.ko.Spec.IPv6AddressCount) {
		delta.Add("Spec.IPv6AddressCount", a.ko.Spec.IPv6AddressCount, b.ko.Spec.IPv6AddressCount)
	} else if a.ko.Spec.IPv6AddressCount != nil && b.ko.Spec.IPv6AddressCount != nil {
		if *a.ko.Spec.IPv6AddressCount != *b.ko.Spec.IPv6AddressCount {
			delta.Add("Spec.IPv6AddressCount", a.ko.Spec.IPv6AddressCount, b.ko.Spec.IPv6AddressCount)
		}
	}
	if !reflect.DeepEqual(a.ko.Spec.IPv6Addresses, b.ko.Spec.IPv6Addresses) {
		delta.Add("Spec.IPv6Addresses", a.ko.Spec.IPv6Addresses, b.ko.Spec.IPv6Addresses)
	}
	if ackcompare.HasNilDifference(a.ko.Spec.KernelID, b.ko.Spec.KernelID) {
		delta.Add("Spec.KernelID", a.ko.Spec.KernelID, b.ko.Spec.KernelID)
	} else if a.ko.Spec.KernelID != nil && b.ko.Spec.KernelID != nil {
		if *a.ko.Spec.KernelID != *b.ko.Spec.KernelID {
			delta.Add("Spec.KernelID", a.ko.Spec.KernelID, b.ko.Spec.KernelID)
		}
	}
	if ackcompare.HasNilDifference(a.ko.Spec.KeyName, b.ko.Spec.KeyName) {
		delta.Add("Spec.KeyName", a.ko.Spec.KeyName, b.ko.Spec.KeyName)
	} else if a.ko.Spec.KeyName != nil && b.ko.Spec.KeyName != nil {
		if *a.ko.Spec.KeyName != *b.ko.Spec.KeyName {
			delta.Add("Spec.KeyName", a.ko.Spec.KeyName, b.ko.Spec.KeyName)
		}
	}
	if ackcompare.HasNilDifference(a.ko.Spec.LaunchTemplate, b.ko.Spec.LaunchTemplate) {
		delta.Add("Spec.LaunchTemplate", a.ko.Spec.LaunchTemplate, b.ko.Spec.LaunchTemplate)
	} else if a.ko.Spec.LaunchTemplate != nil && b.ko.Spec.LaunchTemplate != nil {
		if ackcompare.HasNilDifference(a.ko.Spec.LaunchTemplate.LaunchTemplateID, b.ko.Spec.LaunchTemplate.LaunchTemplateID) {
			delta.Add("Spec.LaunchTemplate.LaunchTemplateID", a.ko.Spec.LaunchTemplate.LaunchTemplateID, b.ko.Spec.LaunchTemplate.LaunchTemplateID)
		} else if a.ko.Spec.LaunchTemplate.LaunchTemplateID != nil && b.ko.Spec.LaunchTemplate.LaunchTemplateID != nil {
			if *a.ko.Spec.LaunchTemplate.LaunchTemplateID != *b.ko.Spec.LaunchTemplate.LaunchTemplateID {
				delta.Add("Spec.LaunchTemplate.LaunchTemplateID", a.ko.Spec.LaunchTemplate.LaunchTemplateID, b.ko.Spec.LaunchTemplate.LaunchTemplateID)
			}
		}
		if ackcompare.HasNilDifference(a.ko.Spec.LaunchTemplate.LaunchTemplateName, b.ko.Spec.LaunchTemplate.LaunchTemplateName) {
			delta.Add("Spec.LaunchTemplate.LaunchTemplateName", a.ko.Spec.LaunchTemplate.LaunchTemplateName, b.ko.Spec.LaunchTemplate.LaunchTemplateName)
		} else if a.ko.Spec.LaunchTemplate.LaunchTemplateName != nil && b.ko.Spec.LaunchTemplate.LaunchTemplateName != nil {
			if *a.ko.Spec.LaunchTemplate.LaunchTemplateName != *b.ko.Spec.LaunchTemplate.LaunchTemplateName {
				delta.Add("Spec.LaunchTemplate.LaunchTemplateName", a.ko.Spec.LaunchTemplate.LaunchTemplateName, b.ko.Spec.LaunchTemplate.LaunchTemplateName)
			}
		}
		if ackcompare.HasNilDifference(a.ko.Spec.LaunchTemplate.Version, b.ko.Spec.LaunchTemplate.Version) {
			delta.Add("Spec.LaunchTemplate.Version", a.ko.Spec.LaunchTemplate.Version, b.ko.Spec.LaunchTemplate.Version)
		} else if a.ko.Spec.LaunchTemplate.Version != nil && b.ko.Spec.LaunchTemplate.Version != nil {
			if *a.ko.Spec.LaunchTemplate.Version != *b.ko.Spec.LaunchTemplate.Version {
				delta.Add("Spec.LaunchTemplate.Version", a.ko.Spec.LaunchTemplate.Version, b.ko.Spec.LaunchTemplate.Version)
			}
		}
	}
	if !reflect.DeepEqual(a.ko.Spec.LicenseSpecifications, b.ko.Spec.LicenseSpecifications) {
		delta.Add("Spec.LicenseSpecifications", a.ko.Spec.LicenseSpecifications, b.ko.Spec.LicenseSpecifications)
	}
	if ackcompare.HasNilDifference(a.ko.Spec.MaxCount, b.ko.Spec.MaxCount) {
		delta.Add("Spec.MaxCount", a.ko.Spec.MaxCount, b.ko.Spec.MaxCount)
	} else if a.ko.Spec.MaxCount != nil && b.ko.Spec.MaxCount != nil {
		if *a.ko.Spec.MaxCount != *b.ko.Spec.MaxCount {
			delta.Add("Spec.MaxCount", a.ko.Spec.MaxCount, b.ko.Spec.MaxCount)
		}
	}
	if ackcompare.HasNilDifference(a.ko.Spec.MetadataOptions, b.ko.Spec.MetadataOptions) {
		delta.Add("Spec.MetadataOptions", a.ko.Spec.MetadataOptions, b.ko.Spec.MetadataOptions)
	} else if a.ko.Spec.MetadataOptions != nil && b.ko.Spec.MetadataOptions != nil {
		if ackcompare.HasNilDifference(a.ko.Spec.MetadataOptions.HTTPEndpoint, b.ko.Spec.MetadataOptions.HTTPEndpoint) {
			delta.Add("Spec.MetadataOptions.HTTPEndpoint", a.ko.Spec.MetadataOptions.HTTPEndpoint, b.ko.Spec.MetadataOptions.HTTPEndpoint)
		} else if a.ko.Spec.MetadataOptions.HTTPEndpoint != nil && b.ko.Spec.MetadataOptions.HTTPEndpoint != nil {
			if *a.ko.Spec.MetadataOptions.HTTPEndpoint != *b.ko.Spec.MetadataOptions.HTTPEndpoint {
				delta.Add("Spec.MetadataOptions.HTTPEndpoint", a.ko.Spec.MetadataOptions.HTTPEndpoint, b.ko.Spec.MetadataOptions.HTTPEndpoint)
			}
		}
		if ackcompare.HasNilDifference(a.ko.Spec.MetadataOptions.HTTPProtocolIPv6, b.ko.Spec.MetadataOptions.HTTPProtocolIPv6) {
			delta.Add("Spec.MetadataOptions.HTTPProtocolIPv6", a.ko.Spec.MetadataOptions.HTTPProtocolIPv6, b.ko.Spec.MetadataOptions.HTTPProtocolIPv6)
		} else if a.ko.Spec.MetadataOptions.HTTPProtocolIPv6 != nil && b.ko.Spec.MetadataOptions.HTTPProtocolIPv6 != nil {
			if *a.ko.Spec.MetadataOptions.HTTPProtocolIPv6 != *b.ko.Spec.MetadataOptions.HTTPProtocolIPv6 {
				delta.Add("Spec.MetadataOptions.HTTPProtocolIPv6", a.ko.Spec.MetadataOptions.HTTPProtocolIPv6, b.ko.Spec.MetadataOptions.HTTPProtocolIPv6)
			}
		}
		if ackcompare.HasNilDifference(a.ko.Spec.MetadataOptions.HTTPPutResponseHopLimit, b.ko.Spec.MetadataOptions.HTTPPutResponseHopLimit) {
			delta.Add("Spec.MetadataOptions.HTTPPutResponseHopLimit", a.ko.Spec.MetadataOptions.HTTPPutResponseHopLimit, b.ko.Spec.MetadataOptions.HTTPPutResponseHopLimit)
		} else if a.ko.Spec.MetadataOptions.HTTPPutResponseHopLimit != nil && b.ko.Spec.MetadataOptions.HTTPPutResponseHopLimit != nil {
			if *a.ko.Spec.MetadataOptions.HTTPPutResponseHopLimit != *b.ko.Spec.MetadataOptions.HTTPPutResponseHopLimit {
				delta.Add("Spec.MetadataOptions.HTTPPutResponseHopLimit", a.ko.Spec.MetadataOptions.HTTPPutResponseHopLimit, b.ko.Spec.MetadataOptions.HTTPPutResponseHopLimit)
			}
		}
		if ackcompare.HasNilDifference(a.ko.Spec.MetadataOptions.HTTPTokens, b.ko.Spec.MetadataOptions.HTTPTokens) {
			delta.Add("Spec.MetadataOptions.HTTPTokens", a.ko.Spec.MetadataOptions.HTTPTokens, b.ko.Spec.MetadataOptions.HTTPTokens)
		} else if a.ko.Spec.MetadataOptions.HTTPTokens != nil && b.ko.Spec.MetadataOptions.HTTPTokens != nil {
			if *a.ko.Spec.MetadataOptions.HTTPTokens != *b.ko.Spec.MetadataOptions.HTTPTokens {
				delta.Add("Spec.MetadataOptions.HTTPTokens", a.ko.Spec.MetadataOptions.HTTPTokens, b.ko.Spec.MetadataOptions.HTTPTokens)
			}
		}
	}
	if ackcompare.HasNilDifference(a.ko.Spec.MinCount, b.ko.Spec.MinCount) {
		delta.Add("Spec.MinCount", a.ko.Spec.MinCount, b.ko.Spec.MinCount)
	} else if a.ko.Spec.MinCount != nil && b.ko.Spec.MinCount != nil {
		if *a.ko.Spec.MinCount != *b.ko.Spec.MinCount {
			delta.Add("Spec.MinCount", a.ko.Spec.MinCount, b.ko.Spec.MinCount)
		}
	}
	if ackcompare.HasNilDifference(a.ko.Spec.Monitoring, b.ko.Spec.Monitoring) {
		delta.Add("Spec.Monitoring", a.ko.Spec.Monitoring, b.ko.Spec.Monitoring)
	} else if a.ko.Spec.Monitoring != nil && b.ko.Spec.Monitoring != nil {
		if ackcompare.HasNilDifference(a.ko.Spec.Monitoring.Enabled, b.ko.Spec.Monitoring.Enabled) {
			delta.Add("Spec.Monitoring.Enabled", a.ko.Spec.Monitoring.Enabled, b.ko.Spec.Monitoring.Enabled)
		} else if a.ko.Spec.Monitoring.Enabled != nil && b.ko.Spec.Monitoring.Enabled != nil {
			if *a.ko.Spec.Monitoring.Enabled != *b.ko.Spec.Monitoring.Enabled {
				delta.Add("Spec.Monitoring.Enabled", a.ko.Spec.Monitoring.Enabled, b.ko.Spec.Monitoring.Enabled)
			}
		}
	}
	if !reflect.DeepEqual(a.ko.Spec.NetworkInterfaces, b.ko.Spec.NetworkInterfaces) {
		delta.Add("Spec.NetworkInterfaces", a.ko.Spec.NetworkInterfaces, b.ko.Spec.NetworkInterfaces)
	}
	if ackcompare.HasNilDifference(a.ko.Spec.Placement, b.ko.Spec.Placement) {
		delta.Add("Spec.Placement", a.ko.Spec.Placement, b.ko.Spec.Placement)
	} else if a.ko.Spec.Placement != nil && b.ko.Spec.Placement != nil {
		if ackcompare.HasNilDifference(a.ko.Spec.Placement.Affinity, b.ko.Spec.Placement.Affinity) {
			delta.Add("Spec.Placement.Affinity", a.ko.Spec.Placement.Affinity, b.ko.Spec.Placement.Affinity)
		} else if a.ko.Spec.Placement.Affinity != nil && b.ko.Spec.Placement.Affinity != nil {
			if *a.ko.Spec.Placement.Affinity != *b.ko.Spec.Placement.Affinity {
				delta.Add("Spec.Placement.Affinity", a.ko.Spec.Placement.Affinity, b.ko.Spec.Placement.Affinity)
			}
		}
		if ackcompare.HasNilDifference(a.ko.Spec.Placement.AvailabilityZone, b.ko.Spec.Placement.AvailabilityZone) {
			delta.Add("Spec.Placement.AvailabilityZone", a.ko.Spec.Placement.AvailabilityZone, b.ko.Spec.Placement.AvailabilityZone)
		} else if a.ko.Spec.Placement.AvailabilityZone != nil && b.ko.Spec.Placement.AvailabilityZone != nil {
			if *a.ko.Spec.Placement.AvailabilityZone != *b.ko.Spec.Placement.AvailabilityZone {
				delta.Add("Spec.Placement.AvailabilityZone", a.ko.Spec.Placement.AvailabilityZone, b.ko.Spec.Placement.AvailabilityZone)
			}
		}
		if ackcompare.HasNilDifference(a.ko.Spec.Placement.GroupName, b.ko.Spec.Placement.GroupName) {
			delta.Add("Spec.Placement.GroupName", a.ko.Spec.Placement.GroupName, b.ko.Spec.Placement.GroupName)
		} else if a.ko.Spec.Placement.GroupName != nil && b.ko.Spec.Placement.GroupName != nil {
			if *a.ko.Spec.Placement.GroupName != *b.ko.Spec.Placement.GroupName {
				delta.Add("Spec.Placement.GroupName", a.ko.Spec.Placement.GroupName, b.ko.Spec.Placement.GroupName)
			}
		}
		if ackcompare.HasNilDifference(a.ko.Spec.Placement.HostID, b.ko.Spec.Placement.HostID) {
			delta.Add("Spec.Placement.HostID", a.ko.Spec.Placement.HostID, b.ko.Spec.Placement.HostID)
		} else if a.ko.Spec.Placement.HostID != nil && b.ko.Spec.Placement.HostID != nil {
			if *a.ko.Spec.Placement.HostID != *b.ko.Spec.Placement.HostID {
				delta.Add("Spec.Placement.HostID", a.ko.Spec.Placement.HostID, b.ko.Spec.Placement.HostID)
			}
		}
		if ackcompare.HasNilDifference(a.ko.Spec.Placement.HostResourceGroupARN, b.ko.Spec.Placement.HostResourceGroupARN) {
			delta.Add("Spec.Placement.HostResourceGroupARN", a.ko.Spec.Placement.HostResourceGroupARN, b.ko.Spec.Placement.HostResourceGroupARN)
		} else if a.ko.Spec.Placement.HostResourceGroupARN != nil && b.ko.Spec.Placement.HostResourceGroupARN != nil {
			if *a.ko.Spec.Placement.HostResourceGroupARN != *b.ko.Spec.Placement.HostResourceGroupARN {
				delta.Add("Spec.Placement.HostResourceGroupARN", a.ko.Spec.Placement.HostResourceGroupARN, b.ko.Spec.Placement.HostResourceGroupARN)
			}
		}
		if ackcompare.HasNilDifference(a.ko.Spec.Placement.PartitionNumber, b.ko.Spec.Placement.PartitionNumber) {
			delta.Add("Spec.Placement.PartitionNumber", a.ko.Spec.Placement.PartitionNumber, b.ko.Spec.Placement.PartitionNumber)
		} else if a.ko.Spec.Placement.PartitionNumber != nil && b.ko.Spec.Placement.PartitionNumber != nil {
			if *a.ko.Spec.Placement.PartitionNumber != *b.ko.Spec.Placement.PartitionNumber {
				delta.Add("Spec.Placement.PartitionNumber", a.ko.Spec.Placement.PartitionNumber, b.ko.Spec.Placement.PartitionNumber)
			}
		}
		if ackcompare.HasNilDifference(a.ko.Spec.Placement.SpreadDomain, b.ko.Spec.Placement.SpreadDomain) {
			delta.Add("Spec.Placement.SpreadDomain", a.ko.Spec.Placement.SpreadDomain, b.ko.Spec.Placement.SpreadDomain)
		} else if a.ko.Spec.Placement.SpreadDomain != nil && b.ko.Spec.Placement.SpreadDomain != nil {
			if *a.ko.Spec.Placement.SpreadDomain != *b.ko.Spec.Placement.SpreadDomain {
				delta.Add("Spec.Placement.SpreadDomain", a.ko.Spec.Placement.SpreadDomain, b.ko.Spec.Placement.SpreadDomain)
			}
		}
		if ackcompare.HasNilDifference(a.ko.Spec.Placement.Tenancy, b.ko.Spec.Placement.Tenancy) {
			delta.Add("Spec.Placement.Tenancy", a.ko.Spec.Placement.Tenancy, b.ko.Spec.Placement.Tenancy)
		} else if a.ko.Spec.Placement.Tenancy != nil && b.ko.Spec.Placement.Tenancy != nil {
			if *a.ko.Spec.Placement.Tenancy != *b.ko.Spec.Placement.Tenancy {
				delta.Add("Spec.Placement.Tenancy", a.ko.Spec.Placement.Tenancy, b.ko.Spec.Placement.Tenancy)
			}
		}
	}
	if ackcompare.HasNilDifference(a.ko.Spec.PrivateIPAddress, b.ko.Spec.PrivateIPAddress) {
		delta.Add("Spec.PrivateIPAddress", a.ko.Spec.PrivateIPAddress, b.ko.Spec.PrivateIPAddress)
	} else if a.ko.Spec.PrivateIPAddress != nil && b.ko.Spec.PrivateIPAddress != nil {
		if *a.ko.Spec.PrivateIPAddress != *b.ko.Spec.PrivateIPAddress {
			delta.Add("Spec.PrivateIPAddress", a.ko.Spec.PrivateIPAddress, b.ko.Spec.PrivateIPAddress)
		}
	}
	if ackcompare.HasNilDifference(a.ko.Spec.RamdiskID, b.ko.Spec.RamdiskID) {
		delta.Add("Spec.RamdiskID", a.ko.Spec.RamdiskID, b.ko.Spec.RamdiskID)
	} else if a.ko.Spec.RamdiskID != nil && b.ko.Spec.RamdiskID != nil {
		if *a.ko.Spec.RamdiskID != *b.ko.Spec.RamdiskID {
			delta.Add("Spec.RamdiskID", a.ko.Spec.RamdiskID, b.ko.Spec.RamdiskID)
		}
	}
	if !ackcompare.SliceStringPEqual(a.ko.Spec.SecurityGroupIDs, b.ko.Spec.SecurityGroupIDs) {
		delta.Add("Spec.SecurityGroupIDs", a.ko.Spec.SecurityGroupIDs, b.ko.Spec.SecurityGroupIDs)
	}
	if !ackcompare.SliceStringPEqual(a.ko.Spec.SecurityGroups, b.ko.Spec.SecurityGroups) {
		delta.Add("Spec.SecurityGroups", a.ko.Spec.SecurityGroups, b.ko.Spec.SecurityGroups)
	}
	if ackcompare.HasNilDifference(a.ko.Spec.SubnetID, b.ko.Spec.SubnetID) {
		delta.Add("Spec.SubnetID", a.ko.Spec.SubnetID, b.ko.Spec.SubnetID)
	} else if a.ko.Spec.SubnetID != nil && b.ko.Spec.SubnetID != nil {
		if *a.ko.Spec.SubnetID != *b.ko.Spec.SubnetID {
			delta.Add("Spec.SubnetID", a.ko.Spec.SubnetID, b.ko.Spec.SubnetID)
		}
	}
	if !reflect.DeepEqual(a.ko.Spec.TagSpecifications, b.ko.Spec.TagSpecifications) {
		delta.Add("Spec.TagSpecifications", a.ko.Spec.TagSpecifications, b.ko.Spec.TagSpecifications)
	}
	if ackcompare.HasNilDifference(a.ko.Spec.UserData, b.ko.Spec.UserData) {
		delta.Add("Spec.UserData", a.ko.Spec.UserData, b.ko.Spec.UserData)
	} else if a.ko.Spec.UserData != nil && b.ko.Spec.UserData != nil {
		if *a.ko.Spec.UserData != *b.ko.Spec.UserData {
			delta.Add("Spec.UserData", a.ko.Spec.UserData, b.ko.Spec.UserData)
		}
	}

	return delta
}
# Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License"). You may
# not use this file except in compliance with the License. A copy of the
# License is located at
#
# 	 http://aws.amazon.com/apache2.0/
#
# or in the "license" file accompanying this file. This file is distributed
# on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
# express or implied. See the License for the specific language governing
# permissions and limitations under the License.

"""Integration tests for the DHCPOptions API.
"""

import pytest
import time
import logging

from acktest.resources import random_suffix_name
from acktest.k8s import resource as k8s
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_ec2_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e.tests.helper import EC2Validator

RESOURCE_PLURAL = "dhcpoptions"

DEFAULT_WAIT_AFTER_SECONDS = 5
CREATE_WAIT_AFTER_SECONDS = 10
DELETE_WAIT_AFTER_SECONDS = 10

@service_marker
@pytest.mark.canary
class TestDhcpOptions:
    def test_create_delete(self, ec2_client):
        test_resource_values = REPLACEMENT_VALUES.copy()
        resource_name = random_suffix_name("dhcp-opts-test", 24)

        test_resource_values["DHCP_OPTIONS_NAME"] = resource_name
        test_resource_values["DHCP_KEY_1"] = "domain-name"
        test_resource_values["DHCP_VAL_1"] = "ack-example.com"
        test_resource_values["DHCP_KEY_2"] = "domain-name-servers"
        test_resource_values["DHCP_VAL_2_1"] = "10.2.5.1"
        test_resource_values["DHCP_VAL_2_2"] = "10.2.5.2"

        # Load DHCP Options CR
        resource_data = load_ec2_resource(
            "dhcp_options",
            additional_replacements=test_resource_values,
        )
        logging.debug(resource_data)

        # Create k8s resource
        ref = k8s.CustomResourceReference(
            CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
            resource_name, namespace="default",
        )
        k8s.create_custom_resource(ref, resource_data)
        cr = k8s.wait_resource_consumed_by_controller(ref)

        assert cr is not None
        assert k8s.get_resource_exists(ref)

        resource = k8s.get_resource(ref)
        resource_id = resource["status"]["dhcpOptionsID"]

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # Check DHCP Options exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_dhcp_options(resource_id)

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check DHCP Options no longer exists in AWS
        ec2_validator.assert_dhcp_options(resource_id, exists=False)

    def test_terminal_condition_invalid_parameter_value(self):
        test_resource_values = REPLACEMENT_VALUES.copy()
        resource_name = random_suffix_name("dhcp-opts-fail", 24)

        test_resource_values["DHCP_OPTIONS_NAME"] = resource_name
        test_resource_values["DHCP_KEY_1"] = "InvalidValue"
        test_resource_values["DHCP_VAL_1"] = "ack-example.com"
        test_resource_values["DHCP_KEY_2"] = "domain-name-servers"
        test_resource_values["DHCP_VAL_2_1"] = "10.2.5.1"
        test_resource_values["DHCP_VAL_2_2"] = "10.2.5.2"

        # Load DHCP Options CR
        resource_data = load_ec2_resource(
            "dhcp_options",
            additional_replacements=test_resource_values,
        )
        logging.debug(resource_data)

        # Create k8s resource
        ref = k8s.CustomResourceReference(
            CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
            resource_name, namespace="default",
        )
        k8s.create_custom_resource(ref, resource_data)
        cr = k8s.wait_resource_consumed_by_controller(ref)

        assert cr is not None
        assert k8s.get_resource_exists(ref)

        expected_msg = "InvalidParameterValue: Value (InvalidValue) for parameter name is invalid. Unknown DHCP option"
        terminal_condition = k8s.get_resource_condition(ref, "ACK.Terminal")
        # Example condition message:
        # An error occurred (InvalidParameterValue) when calling the CreateDhcpOptions operation:
        # Value (InvalidValue) for parameter value is invalid.
        # Unknown DHCP option
        assert expected_msg in terminal_condition['message']
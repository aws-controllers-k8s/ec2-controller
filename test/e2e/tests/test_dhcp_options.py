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

from acktest import tags
from acktest.resources import random_suffix_name
from acktest.k8s import resource as k8s
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_ec2_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e.bootstrap_resources import get_bootstrap_resources
from e2e.tests.helper import EC2Validator

RESOURCE_PLURAL = "dhcpoptions"

DEFAULT_WAIT_AFTER_SECONDS = 5
CREATE_WAIT_AFTER_SECONDS = 10
DELETE_WAIT_AFTER_SECONDS = 10
MODIFY_WAIT_AFTER_SECONDS = 5

@pytest.fixture
def simple_dhcp_options(request,simple_vpc):
    replacements = REPLACEMENT_VALUES.copy()
    resource_name = random_suffix_name("dhcp-opts-test", 24)
    resource_file = "dhcp_options"

    replacements["DHCP_OPTIONS_NAME"] = resource_name
    replacements["DHCP_KEY_1"] = "domain-name"
    replacements["DHCP_VAL_1"] = "ack-example.com"
    replacements["DHCP_KEY_2"] = "domain-name-servers"
    replacements["DHCP_VAL_2_1"] = "10.2.5.1"
    replacements["DHCP_VAL_2_2"] = "10.2.5.2"

    marker = request.node.get_closest_marker("resource_data")
    if marker is not None:
        data = marker.args[0]
        if 'resource_file' in data:
            resource_file = data['resource_file']
        if 'create_vpc' in data and data['create_vpc'] is True:
            (_, vpc_cr) = simple_vpc
            vpc_id = vpc_cr["status"]["vpcID"]
            replacements["VPC_ID"] = vpc_id
        if 'dhcp_key_1' in data:
            replacements["DHCP_KEY_1"] = data['dhcp_key_1']
        if 'tag_key' in data:
            replacements["TAG_KEY"] = data['tag_key']
        if 'tag_value' in data:
            replacements["TAG_VALUE"] = data['tag_value']


    # Load DHCP Options CR
    resource_data = load_ec2_resource(
        resource_file,
        additional_replacements=replacements,
    )

    # Create k8s resource
    ref = k8s.CustomResourceReference(
        CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
        resource_name, namespace="default",
    )
    k8s.create_custom_resource(ref, resource_data)

    time.sleep(CREATE_WAIT_AFTER_SECONDS)

    # Get latest DHCP Options CR
    cr = k8s.wait_resource_consumed_by_controller(ref)

    assert cr is not None
    assert k8s.get_resource_exists(ref)

    yield (ref, cr)

    # Try to delete, if doesn't already exist
    try:
        _, deleted = k8s.delete_custom_resource(ref, 3, 10)
        assert deleted
    except:
        pass

@service_marker
@pytest.mark.canary
class TestDhcpOptions:
    def test_create_delete(self, ec2_client, simple_dhcp_options):
        (ref, cr) = simple_dhcp_options

        resource_id = cr["status"]["dhcpOptionsID"]

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

    @pytest.mark.resource_data({'dhcp_key_1': 'InvalidValue'})
    def test_terminal_condition_invalid_parameter_value(self, simple_dhcp_options):
        (ref, _) = simple_dhcp_options

        expected_msg = "InvalidParameterValue: Value (InvalidValue) for parameter name is invalid. Unknown DHCP option"
        terminal_condition = k8s.get_resource_condition(ref, "ACK.Terminal")
        # Example condition message:
        # An error occurred (InvalidParameterValue) when calling the CreateDhcpOptions operation:
        # Value (InvalidValue) for parameter value is invalid.
        # Unknown DHCP option
        assert expected_msg in terminal_condition['message']
    
    @pytest.mark.resource_data({'tag_key': 'initialtagkey', 'tag_value': 'initialtagvalue'})
    def test_crud_tags(self, ec2_client, simple_dhcp_options):
        (ref, cr) = simple_dhcp_options
        
        resource = k8s.get_resource(ref)
        resource_id = cr["status"]["dhcpOptionsID"]

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # Check dhcpOptions exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_dhcp_options(resource_id)

        # Check system and user tags exist for dhcpOptions resource
        dhcp_options = ec2_validator.get_dhcp_options(resource_id)
        user_tags = {
            "initialtagkey": "initialtagvalue"
        }
        tags.assert_ack_system_tags(
            tags=dhcp_options["Tags"],
        )
        tags.assert_equal_without_ack_tags(
            expected=user_tags,
            actual=dhcp_options["Tags"],
        )
        
        # Only user tags should be present in Spec
        assert len(resource["spec"]["tags"]) == 1
        assert resource["spec"]["tags"][0]["key"] == "initialtagkey"
        assert resource["spec"]["tags"][0]["value"] == "initialtagvalue"

        # Update tags
        update_tags = [
                {
                    "key": "updatedtagkey",
                    "value": "updatedtagvalue",
                }
            ]

        # Patch the dhcpOptions, updating the tags with new pair
        updates = {
            "spec": {"tags": update_tags},
        }

        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # Check resource synced successfully
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)
        
        # Check for updated user tags; system tags should persist
        dhcp_options = ec2_validator.get_dhcp_options(resource_id)
        updated_tags = {
            "updatedtagkey": "updatedtagvalue"
        }
        tags.assert_ack_system_tags(
            tags=dhcp_options["Tags"],
        )
        tags.assert_equal_without_ack_tags(
            expected=updated_tags,
            actual=dhcp_options["Tags"],
        )
               
        # Only user tags should be present in Spec
        resource = k8s.get_resource(ref)
        assert len(resource["spec"]["tags"]) == 1
        assert resource["spec"]["tags"][0]["key"] == "updatedtagkey"
        assert resource["spec"]["tags"][0]["value"] == "updatedtagvalue"

        # Patch the dhcpOptions resource, deleting the tags
        updates = {
                "spec": {"tags": []},
        }

        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # Check resource synced successfully
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)
        
        # Check for removed user tags; system tags should persist
        dhcp_options = ec2_validator.get_dhcp_options(resource_id)
        tags.assert_ack_system_tags(
            tags=dhcp_options["Tags"],
        )
        tags.assert_equal_without_ack_tags(
            expected=[],
            actual=dhcp_options["Tags"],
        )
        
        # Check user tags are removed from Spec
        resource = k8s.get_resource(ref)
        assert len(resource["spec"]["tags"]) == 0

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check dhcpOptions no longer exists in AWS
        ec2_validator.assert_dhcp_options(resource_id, exists=False)

    @pytest.mark.resource_data({'create_vpc': True, 'resource_file': 'dhcp_options_vpc_ref'})
    def test_dhcpoptions_creation_with_vpcref(self,ec2_client, simple_dhcp_options):
        (ref, cr) = simple_dhcp_options

        resource_id = cr["status"]["dhcpOptionsID"]
        vpc_id = cr["spec"]["vpc"][0]

        # Check DHCP Options exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_dhcp_options(resource_id)

        # Check if DHCP Options gets associated to VPC
        ec2_validator.assert_dhcp_vpc_association(resource_id,vpc_id)

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check DHCP Options no longer exists in AWS
        ec2_validator.assert_dhcp_options(resource_id, exists=False)


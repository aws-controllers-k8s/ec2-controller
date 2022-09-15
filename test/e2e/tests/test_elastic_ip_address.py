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

"""Integration tests for the Elastic IP Addresses API.
"""

import pytest
import time
import logging

from acktest.resources import random_suffix_name
from acktest.k8s import resource as k8s
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_ec2_resource
from e2e.replacement_values import REPLACEMENT_VALUES

RESOURCE_PLURAL = "elasticipaddresses"

CREATE_WAIT_AFTER_SECONDS = 10
DELETE_WAIT_AFTER_SECONDS = 10
MODIFY_WAIT_AFTER_SECONDS = 5


def get_address(ec2_client, allocation_id: str) -> dict:
    try:
        resp = ec2_client.describe_addresses(
            AllocationIds=[allocation_id]
        )
    except Exception as e:
        logging.debug(e)
        return None

    if len(resp["Addresses"]) == 0:
        return None
    return resp["Addresses"][0]


def address_exists(ec2_client, allocation_id: str) -> bool:
    return get_address(ec2_client, allocation_id) is not None

@pytest.fixture
def simple_elastic_ip_address(request):
    resource_name = random_suffix_name("elastic-ip-ack-test", 24)
    resource_file = "elastic_ip_address"

    replacements = REPLACEMENT_VALUES.copy()
    replacements["ADDRESS_NAME"] = resource_name
    replacements["PUBLIC_IPV4_POOL"] = "amazon"

    marker = request.node.get_closest_marker("resource_data")
    if marker is not None:
        data = marker.args[0]
        if 'resource_file' in data:
            resource_file = data['resource_file']
        if 'address' in data:
            replacements["ADDRESS"] = data['address']
        if 'public_ipv4_pool' in data:
            replacements["PUBLIC_IPV4_POOL"] = data['public_ipv4_pool']
        if 'tag_key' in data:
            replacements["TAG_KEY"] = data['tag_key']
        if 'tag_value' in data:
            replacements["TAG_VALUE"] = data['tag_value']

    # Load ElasticIPAddress CR
    resource_data = load_ec2_resource(
        resource_file,
        additional_replacements=replacements,
    )
    logging.debug(resource_data)

    # Create k8s resource
    ref = k8s.CustomResourceReference(
        CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
        resource_name, namespace="default",
    )
    k8s.create_custom_resource(ref, resource_data)
    time.sleep(CREATE_WAIT_AFTER_SECONDS)

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
class TestElasticIPAddress:
    def test_create_delete(self, ec2_client, simple_elastic_ip_address):
        (ref, cr) = simple_elastic_ip_address
        resource_id = cr["status"]["allocationID"]

        # Check Address exists
        exists = address_exists(ec2_client, resource_id)
        assert exists

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref, 2, 5)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check Address doesn't exist
        exists = address_exists(ec2_client, resource_id)
        assert not exists
    
    @pytest.mark.resource_data({'public_ipv4_pool': 'InvalidIpV4Address'})
    def test_terminal_condition_invalid_parameter_value(self, simple_elastic_ip_address):
        (ref, _) = simple_elastic_ip_address

        expected_msg = "InvalidParameterValue: invalid value for parameter pool: InvalidIpV4Address"
        terminal_condition = k8s.get_resource_condition(ref, "ACK.Terminal")
        # Example condition message:
        # An error occurred (InvalidParameterValue) when calling the AllocateAddress operation:
        # invalid value for parameter pool: InvalidIpV4Address
        assert expected_msg in terminal_condition['message']

    @pytest.mark.resource_data({'address': '52.27.68.220', 'resource_file': 'invalid/elastic_ip_invalid_combination'})
    def test_terminal_condition_invalid_parameter_combination(self, simple_elastic_ip_address):
        (ref, _) = simple_elastic_ip_address

        expected_msg = "InvalidParameterCombination: The parameter PublicIpv4Pool cannot be used with the parameter Address"
        terminal_condition = k8s.get_resource_condition(ref, "ACK.Terminal")
        # Example condition message:
        # An error occurred (InvalidParameterCombination) when calling the AllocateAddress operation:
        # The parameter PublicIpv4Pool cannot be used with the parameter Address
        assert expected_msg in terminal_condition['message']
    
    @pytest.mark.resource_data({'tag_key': 'initialtagkey', 'tag_value': 'initialtagvalue'})
    def test_crud_tags(self, ec2_client, simple_elastic_ip_address):
        (ref, cr) = simple_elastic_ip_address
        
        resource = k8s.get_resource(ref)
        resource_id = cr["status"]["allocationID"]

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # Check Address exists
        exists = address_exists(ec2_client, resource_id)
        assert exists
        
        # Check tags exist for elasticipaddress resource
        assert resource["spec"]["tags"][0]["key"] == "initialtagkey"
        assert resource["spec"]["tags"][0]["value"] == "initialtagvalue"

        # New pair of tags
        new_tags = [
                {
                    "key": "updatedtagkey",
                    "value": "updatedtagvalue",
                }
               
            ]

        # Patch the elasticipaddress, updating the tags with new pair
        updates = {
            "spec": {"tags": new_tags},
        }

        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # Check resource synced successfully
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)
        
        # Assert tags are updated for elasticipaddress resource
        resource = k8s.get_resource(ref)
        assert resource["spec"]["tags"][0]["key"] == "updatedtagkey"
        assert resource["spec"]["tags"][0]["value"] == "updatedtagvalue"

        # Patch the elasticipaddress resource, deleting the tags
        updates = {
                "spec": {"tags": []},
        }

        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # Check resource synced successfully
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)
        
        # Assert tags are deleted
        resource = k8s.get_resource(ref)
        assert len(resource['spec']['tags']) == 0

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check Address doesn't exists
        exists = address_exists(ec2_client, resource_id)
        assert not exists
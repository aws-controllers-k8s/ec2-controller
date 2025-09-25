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

"""Integration tests for the Subnet API.
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

from .test_route_table import RESOURCE_PLURAL as ROUTE_TABLE_PLURAL, CREATE_WAIT_AFTER_SECONDS as ROUTE_TABLE_CREATE_WAIT

RESOURCE_PLURAL = "subnets"

CREATE_WAIT_AFTER_SECONDS = 10
MODIFY_WAIT_AFTER_SECONDS = 10
DELETE_WAIT_AFTER_SECONDS = 10


def contains_tag(resource, tag):
    try:
        tag_key, tag_val = tag.popitem()
        for t in resource["spec"]["tags"]:
            if t["key"] == tag_key and t["value"] == tag_val:
                return True
    except:
        pass
    
    return False

def create_default_route_table(cidr_block: str):
    replacements = REPLACEMENT_VALUES.copy()
    resource_name = random_suffix_name("subnet-route-table", 24)
    test_vpc = get_bootstrap_resources().SharedTestVPC
    vpc_id = test_vpc.vpc_id
    igw_id = test_vpc.public_subnets.route_table.internet_gateway.internet_gateway_id

    replacements["ROUTE_TABLE_NAME"] = resource_name
    replacements["VPC_ID"] = vpc_id
    replacements["IGW_ID"] = igw_id
    replacements["DEST_CIDR_BLOCK"] = cidr_block

    resource_data = load_ec2_resource(
        "route_table",
        additional_replacements=replacements,
    )
    logging.debug(resource_data)

    # Create the k8s resource
    ref = k8s.CustomResourceReference(
        CRD_GROUP, CRD_VERSION, ROUTE_TABLE_PLURAL,
        resource_name, namespace="default",
    )
    k8s.create_custom_resource(ref, resource_data)
    cr = k8s.wait_resource_consumed_by_controller(ref)

    time.sleep(ROUTE_TABLE_CREATE_WAIT)

    assert cr is not None
    assert k8s.get_resource_exists(ref)

    return (ref, cr)

@pytest.fixture
def default_route_tables():
    rts = [
        create_default_route_table("192.168.0.0/24"),
        create_default_route_table("192.168.0.1/24")
    ]

    yield rts

    for rt in rts:
        (ref, _) = rt
        # Try to delete, if doesn't already exist
        try:
            _, deleted = k8s.delete_custom_resource(ref, 3, 10)
            assert deleted
        except:
            pass

@service_marker
@pytest.mark.canary
class TestSubnet:
    def test_crud(self, ec2_client):
        test_resource_values = REPLACEMENT_VALUES.copy()
        resource_name = random_suffix_name("subnet-test", 24)
        test_vpc = get_bootstrap_resources().SharedTestVPC
        vpc_id = test_vpc.vpc_id

        test_resource_values["SUBNET_NAME"] = resource_name
        test_resource_values["VPC_ID"] = vpc_id
        # CIDR needs to be within SharedTestVPC range and not overlap other subnets
        test_resource_values["CIDR_BLOCK"] = "10.0.255.0/24"

        # Load Subnet CR
        resource_data = load_ec2_resource(
            "subnet",
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

        # Check resource synced successfully
        assert k8s.wait_on_condition(ref, "Ready", "True", wait_periods=5)

        resource = k8s.get_resource(ref)
        resource_id = resource["status"]["subnetID"]

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # Check Subnet exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_subnet(resource_id)

        # Check Subnet data
        subnet = ec2_validator.get_subnet(resource_id)
        assert subnet['VpcId'] == vpc_id
        assert subnet['CidrBlock'] == "10.0.255.0/24"
        # MapPublicIpOnLaunch default value
        assert subnet['MapPublicIpOnLaunch'] == False

        # Patch the subnet
        updates = {
            "spec": {"mapPublicIPOnLaunch": True},
        }
        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # Check resource synced successfully
        assert k8s.wait_on_condition(ref, "Ready", "True", wait_periods=5)
        subnet = ec2_validator.get_subnet(resource_id)
        assert subnet['MapPublicIpOnLaunch'] == True

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check Subnet no longer exists in AWS
        ec2_validator.assert_subnet(resource_id, exists=False)

    def test_crud_tags(self, ec2_client):
        test_resource_values = REPLACEMENT_VALUES.copy()
        resource_name = random_suffix_name("subnet-test", 24)
        test_vpc = get_bootstrap_resources().SharedTestVPC
        vpc_id = test_vpc.vpc_id

        test_resource_values["SUBNET_NAME"] = resource_name
        test_resource_values["VPC_ID"] = vpc_id
        # CIDR needs to be within SharedTestVPC range and not overlap other subnets
        test_resource_values["CIDR_BLOCK"] = "10.0.255.0/24"
        test_resource_values["TAG_KEY"] = "initialtagkey"
        test_resource_values["TAG_VALUE"] = "initialtagvalue"

        # Load Subnet CR
        resource_data = load_ec2_resource(
            "subnet",
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

        # Check resource synced successfully
        assert k8s.wait_on_condition(ref, "Ready", "True", wait_periods=5)

        resource = k8s.get_resource(ref)
        resource_id = resource["status"]["subnetID"]
        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # Check Subnet exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_subnet(resource_id)

        # Check system and user tags exist for subnet resource
        subnet = ec2_validator.get_subnet(resource_id)
        user_tags = {
            "initialtagkey": "initialtagvalue"
        }
        tags.assert_ack_system_tags(
            tags=subnet["Tags"],
        )
        tags.assert_equal_without_ack_tags(
            expected=user_tags,
            actual=subnet["Tags"],
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

        # Patch the subnet, updating the tags with new pair
        updates = {
            "spec": {"tags": update_tags},
        }

        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # Check resource synced successfully
        assert k8s.wait_on_condition(ref, "Ready", "True", wait_periods=5)
        
        # Check for updated user tags; system tags should persist
        subnet = ec2_validator.get_subnet(resource_id)
        updated_tags = {
            "updatedtagkey": "updatedtagvalue"
        }
        tags.assert_ack_system_tags(
            tags=subnet["Tags"],
        )
        tags.assert_equal_without_ack_tags(
            expected=updated_tags,
            actual=subnet["Tags"],
        )
               
        # Only user tags should be present in Spec
        resource = k8s.get_resource(ref)
        assert len(resource["spec"]["tags"]) == 1
        assert resource["spec"]["tags"][0]["key"] == "updatedtagkey"
        assert resource["spec"]["tags"][0]["value"] == "updatedtagvalue"

        # Patch the subnet resource, deleting the tags
        new_tags = []
        updates = {
                "spec": {"tags": new_tags},
        }

        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # Check resource synced successfully
        assert k8s.wait_on_condition(ref, "Ready", "True", wait_periods=5)
        
        # Check for removed user tags; system tags should persist
        subnet = ec2_validator.get_subnet(resource_id)
        tags.assert_ack_system_tags(
            tags=subnet["Tags"],
        )
        tags.assert_equal_without_ack_tags(
            expected=[],
            actual=subnet["Tags"],
        )
        
        # Check user tags are removed from Spec
        resource = k8s.get_resource(ref)
        assert len(resource["spec"]["tags"]) == 0

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check Subnet no longer exists in AWS
        ec2_validator.assert_subnet(resource_id, exists=False)

    def test_route_table_associations(self, ec2_client, default_route_tables):
        test_resource_values = REPLACEMENT_VALUES.copy()
        resource_name = random_suffix_name("subnet-test", 24)
        test_vpc = get_bootstrap_resources().SharedTestVPC
        vpc_id = test_vpc.vpc_id

        (_, initial_rt_cr) = default_route_tables[0]
        test_resource_values["SUBNET_NAME"] = resource_name
        test_resource_values["VPC_ID"] = vpc_id
        # CIDR needs to be within SharedTestVPC range and not overlap other subnets
        test_resource_values["CIDR_BLOCK"] = "10.0.255.0/24"
        test_resource_values["ROUTE_TABLE_ID"] = initial_rt_cr["status"]["routeTableID"]

        # Load Subnet CR
        resource_data = load_ec2_resource(
            "subnet_route_table_assocations",
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

        # Check resource synced successfully
        assert k8s.wait_on_condition(ref, "Ready", "True", wait_periods=5)

        resource = k8s.get_resource(ref)
        resource_id = resource["status"]["subnetID"]

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # Check Subnet exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_subnet(resource_id)

        assert ec2_validator.get_route_table_association(initial_rt_cr["status"]["routeTableID"], resource_id) is not None

        # Patch the subnet, replacing the route tables
        updates = {
            "spec": {"routeTables": [
                default_route_tables[1][1]["status"]["routeTableID"]
            ]},
        }
        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # Check resource synced successfully
        assert k8s.wait_on_condition(ref, "Ready", "True", wait_periods=5)

        assert ec2_validator.get_route_table_association(initial_rt_cr["status"]["routeTableID"], resource_id) is None
        assert ec2_validator.get_route_table_association(default_route_tables[1][1]["status"]["routeTableID"], resource_id) is not None

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check Subnet no longer exists in AWS
        ec2_validator.assert_subnet(resource_id, exists=False)

    def test_terminal_condition_invalid_parameter_value(self):
        test_resource_values = REPLACEMENT_VALUES.copy()
        resource_name = random_suffix_name("subnet-fail", 24)
        test_vpc = get_bootstrap_resources().SharedTestVPC
        vpc_id = test_vpc.vpc_id

        test_resource_values["SUBNET_NAME"] = resource_name
        test_resource_values["CIDR_BLOCK"] = "InvalidCidrBlock"
        test_resource_values["VPC_ID"] = vpc_id

        # Load Subnet CR
        resource_data = load_ec2_resource(
            "subnet",
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

        expected_msg = "InvalidParameterValue: Value (InvalidCidrBlock) for parameter cidrBlock is invalid. This is not a valid CIDR block."
        terminal_condition = k8s.get_resource_condition(ref, "ACK.Terminal")
        # Example condition message:
        # An error occurred (InvalidParameterValue) when calling the CreateSubnet operation:
        # Value (InvalidCidrBlock) for parameter cidrBlock is invalid.
        # This is not a valid CIDR block.
        assert expected_msg in terminal_condition['message']
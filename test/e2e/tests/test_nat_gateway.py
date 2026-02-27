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

"""Integration tests for the NATGateway API.
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

RESOURCE_PLURAL = "natgateways"

CREATE_WAIT_AFTER_SECONDS = 10
DELETE_WAIT_AFTER_SECONDS = 10
MODIFY_WAIT_AFTER_SECONDS = 10

@pytest.fixture
def standard_elastic_address():
    cluster_name = random_suffix_name("nat-gateway-eip", 32)

    replacements = REPLACEMENT_VALUES.copy()
    replacements["ADDRESS_NAME"] = cluster_name
    replacements["PUBLIC_IPV4_POOL"] = "amazon"

    resource_data = load_ec2_resource(
        "elastic_ip_address",
        additional_replacements=replacements,
    )
    logging.debug(resource_data)

    # Create the k8s resource
    ref = k8s.CustomResourceReference(
        CRD_GROUP, CRD_VERSION, "elasticipaddresses",
        cluster_name, namespace="default",
    )
    k8s.create_custom_resource(ref, resource_data)
    cr = k8s.wait_resource_consumed_by_controller(ref)

    # ElasticIP are not usable immediately after they are created, so this will
    # buy us some time in case we try to mount it too early.
    time.sleep(CREATE_WAIT_AFTER_SECONDS)

    # Check resource synced successfully
    assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=10)

    assert cr is not None
    assert k8s.get_resource_exists(ref)

    yield (ref, cr)

    # Try to delete, if doesn't already exist
    try:
        _, deleted = k8s.delete_custom_resource(ref, 3, 10)
        time.sleep(DELETE_WAIT_AFTER_SECONDS)
        assert deleted
    except:
        pass

@pytest.fixture
def simple_nat_gateway(standard_elastic_address, request):
    test_resource_values = REPLACEMENT_VALUES.copy()
    resource_name = random_suffix_name("nat-gateway-test", 24)
    test_vpc = get_bootstrap_resources().SharedTestVPC
    subnet_id = test_vpc.public_subnets.subnet_ids[0]

    (_, eip) = standard_elastic_address

    test_resource_values["NAT_GATEWAY_NAME"] = resource_name
    test_resource_values["SUBNET_ID"] = subnet_id
    test_resource_values["ALLOCATION_ID"] = eip["status"]["allocationID"]

    marker = request.node.get_closest_marker("resource_data")
    if marker is not None:
        data = marker.args[0]
        if 'tag_key' in data:
            test_resource_values["TAG_KEY"] = data["tag_key"]
        if 'tag_value' in data:
            test_resource_values["TAG_VALUE"] = data["tag_value"]

    # Load NAT Gateway CR
    resource_data = load_ec2_resource(
        "nat_gateway",
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

    # NAT Gateways are not usable immediately after they are created, so this will
    # buy us some time in case we try to mount it too early.
    time.sleep(CREATE_WAIT_AFTER_SECONDS)

    # Check resource synced successfully
    assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)

    assert cr is not None
    assert k8s.get_resource_exists(ref)

    yield (ref, cr)

    # Try to delete, if doesn't already exist
    try:
        _, deleted = k8s.delete_custom_resource(ref, 3, 10)
        time.sleep(DELETE_WAIT_AFTER_SECONDS)
        assert deleted
    except:
        pass

@pytest.fixture
def regional_nat_gateway(request):
    test_resource_values = REPLACEMENT_VALUES.copy()
    resource_name = random_suffix_name("nat-gw-regional", 24)
    test_vpc = get_bootstrap_resources().SharedTestVPC
    vpc_id = test_vpc.vpc_id

    test_resource_values["NAT_GATEWAY_NAME"] = resource_name
    test_resource_values["VPC_ID"] = vpc_id

    marker = request.node.get_closest_marker("resource_data")
    if marker is not None:
        data = marker.args[0]
        if 'tag_key' in data:
            test_resource_values["TAG_KEY"] = data["tag_key"]
        if 'tag_value' in data:
            test_resource_values["TAG_VALUE"] = data["tag_value"]

    # Load Regional NAT Gateway CR
    resource_data = load_ec2_resource(
        "nat_gateway_regional",
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

    time.sleep(CREATE_WAIT_AFTER_SECONDS)

    # Check resource synced successfully
    assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)

    assert cr is not None
    assert k8s.get_resource_exists(ref)

    yield (ref, cr)

    # Try to delete, if doesn't already exist
    try:
        _, deleted = k8s.delete_custom_resource(ref, 3, 10)
        time.sleep(DELETE_WAIT_AFTER_SECONDS)
        assert deleted
    except:
        pass

@service_marker
@pytest.mark.canary
class TestNATGateway:
    def test_create_delete(self, simple_nat_gateway, ec2_client):
        (ref, cr) = simple_nat_gateway
        resource_id = cr["status"]["natGatewayID"]

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # Check NAT Gateway exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_nat_gateway(resource_id)

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check NAT Gateway no longer exists in AWS
        ec2_validator.assert_nat_gateway(resource_id, exists=False)
    
    @pytest.mark.resource_data({'tag_key': 'initialtagkey', 'tag_value': 'initialtagvalue'})
    def test_crud_tags(self, ec2_client, simple_nat_gateway):
        (ref, cr) = simple_nat_gateway
        
        resource = k8s.get_resource(ref)
        resource_id = cr["status"]["natGatewayID"]

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # Check natGateway exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_nat_gateway(resource_id)
        
        # Check system and user tags exist for natGateway resource
        nat_gateway = ec2_validator.get_nat_gateway(resource_id)
        user_tags = {
            "initialtagkey": "initialtagvalue"
        }
        tags.assert_ack_system_tags(
            tags=nat_gateway["Tags"],
        )
        tags.assert_equal_without_ack_tags(
            expected=user_tags,
            actual=nat_gateway["Tags"],
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

        # Patch the natGateway, updating the tags with new pair
        updates = {
            "spec": {"tags": update_tags},
        }

        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)
        
        # Check for updated user tags; system tags should persist
        nat_gateway = ec2_validator.get_nat_gateway(resource_id)
        updated_tags = {
            "updatedtagkey": "updatedtagvalue"
        }
        tags.assert_ack_system_tags(
            tags=nat_gateway["Tags"],
        )
        tags.assert_equal_without_ack_tags(
            expected=updated_tags,
            actual=nat_gateway["Tags"],
        )
               
        # Only user tags should be present in Spec
        resource = k8s.get_resource(ref)
        assert len(resource["spec"]["tags"]) == 1
        assert resource["spec"]["tags"][0]["key"] == "updatedtagkey"
        assert resource["spec"]["tags"][0]["value"] == "updatedtagvalue"

        # Patch the natGateway resource, deleting the tags
        updates = {
                "spec": {"tags": []},
        }

        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # Check resource synced successfully
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)
        
        # Check for removed user tags; system tags should persist
        nat_gateway = ec2_validator.get_nat_gateway(resource_id)
        tags.assert_ack_system_tags(
            tags=nat_gateway["Tags"],
        )
        tags.assert_equal_without_ack_tags(
            expected=[],
            actual=nat_gateway["Tags"],
        )
        
        # Check user tags are removed from Spec
        resource = k8s.get_resource(ref)
        assert len(resource["spec"]["tags"]) == 0

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check natGateway no longer exists in AWS
        ec2_validator.assert_nat_gateway(resource_id, exists=False)

    @pytest.mark.resource_data({'tag_key': 'regionaltag', 'tag_value': 'regionalvalue'})
    def test_regional_create_update_delete(self, ec2_client, regional_nat_gateway):
        (ref, cr) = regional_nat_gateway

        resource = k8s.get_resource(ref)
        resource_id = cr["status"]["natGatewayID"]

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # Check NAT Gateway exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_nat_gateway(resource_id)

        # Verify regional-specific fields
        nat_gateway = ec2_validator.get_nat_gateway(resource_id)
        assert nat_gateway["AvailabilityMode"] == "regional"

        # Verify spec fields on the CR
        assert resource["spec"].get("availabilityMode") == "regional"
        assert resource["spec"].get("vpcID") is not None

        # Verify status.vpcID is populated (backward compatibility)
        resource = k8s.get_resource(ref)
        assert resource["status"].get("vpcID") is not None

        # Update tags
        update_tags = [
            {
                "key": "updatedregionaltag",
                "value": "updatedregionalvalue",
            }
        ]
        updates = {
            "spec": {"tags": update_tags},
        }
        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # Check resource synced successfully after update
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)

        # Verify tags updated in AWS
        nat_gateway = ec2_validator.get_nat_gateway(resource_id)
        updated_tags = {
            "updatedregionaltag": "updatedregionalvalue"
        }
        tags.assert_ack_system_tags(
            tags=nat_gateway["Tags"],
        )
        tags.assert_equal_without_ack_tags(
            expected=updated_tags,
            actual=nat_gateway["Tags"],
        )

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check NAT Gateway no longer exists in AWS
        ec2_validator.assert_nat_gateway(resource_id, exists=False)


    def test_terminal_condition_invalid_subnet(self, standard_elastic_address):
        test_resource_values = REPLACEMENT_VALUES.copy()
        resource_name = random_suffix_name("nat-gateway-fail-1", 24)
        subnet_id = "InvalidSubnet"

        (_, eip) = standard_elastic_address

        test_resource_values["NAT_GATEWAY_NAME"] = resource_name
        test_resource_values["SUBNET_ID"] = subnet_id
        test_resource_values["ALLOCATION_ID"] = eip["status"]["allocationID"]

        # Load NAT Gateway CR
        resource_data = load_ec2_resource(
            "nat_gateway",
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

        expected_msg = "InvalidSubnet: The subnet ID 'InvalidSubnet' is malformed"
        terminal_condition = k8s.get_resource_condition(ref, "ACK.Terminal")
        # Example condition message:
        # An error occurred (InvalidSubnet) when calling the CreateNatGateway operation:
        # The subnet ID 'InvalidSubnet' is malformed
        assert expected_msg in terminal_condition['message']

    def test_terminal_condition_malformed_elastic_ip(self):
        test_resource_values = REPLACEMENT_VALUES.copy()
        resource_name = random_suffix_name("nat-gateway-fail-2", 24)
        test_vpc = get_bootstrap_resources().SharedTestVPC
        subnet_id = test_vpc.public_subnets.subnet_ids[0]

        test_resource_values["NAT_GATEWAY_NAME"] = resource_name
        test_resource_values["SUBNET_ID"] = subnet_id
        test_resource_values["ALLOCATION_ID"] = "MalformedElasticIpId"

        # Load NAT Gateway CR
        resource_data = load_ec2_resource(
            "nat_gateway",
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

        expected_msg = "InvalidElasticIpID.Malformed: The elastic-ip ID 'MalformedElasticIpId' is malformed"
        terminal_condition = k8s.get_resource_condition(ref, "ACK.Terminal")
        # Example condition message:
        # An error occurred (InvalidElasticIpID.Malformed) when calling the CreateNatGateway operation:
        # The elastic-ip ID 'MalformedElasticIpId' is malformed"
        assert expected_msg in terminal_condition['message']

    def test_terminal_condition_missing_parameter(self):
        test_resource_values = REPLACEMENT_VALUES.copy()
        resource_name = random_suffix_name("nat-gateway-fail-3", 24)
        test_vpc = get_bootstrap_resources().SharedTestVPC
        subnet_id = test_vpc.public_subnets.subnet_ids[0]

        test_resource_values["NAT_GATEWAY_NAME"] = resource_name
        test_resource_values["SUBNET_ID"] = subnet_id

        # ALLOCATION_ID is required for creating public nat gateways only
        test_resource_values["ALLOCATION_ID"] = ""

        # Load NAT Gateway CR
        resource_data = load_ec2_resource(
            "nat_gateway",
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

        expected_msg = "MissingParameter: The request must include the AllocationId parameter. Add the required parameter and retry the request."
        terminal_condition = k8s.get_resource_condition(ref, "ACK.Terminal")
        # Example condition message:
        # An error occurred (MissingParameter) when calling the CreateNatGateway operation:
        # The request must include the AllocationId parameter. 
        # Add the required parameter and retry the request.
        assert expected_msg in terminal_condition['message']
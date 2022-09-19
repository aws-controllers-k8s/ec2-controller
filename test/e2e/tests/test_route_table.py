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

"""Integration tests for the RouteTable API.
"""

import pytest
import time
import logging

from acktest.resources import random_suffix_name
from acktest.k8s import resource as k8s
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_ec2_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e.bootstrap_resources import get_bootstrap_resources
from e2e.tests.helper import EC2Validator

RESOURCE_PLURAL = "routetables"

DEFAULT_WAIT_AFTER_SECONDS = 5
CREATE_WAIT_AFTER_SECONDS = 10
DELETE_WAIT_AFTER_SECONDS = 10
MODIFY_WAIT_AFTER_SECONDS = 10

@pytest.fixture
def simple_route_table(request):
    replacements = REPLACEMENT_VALUES.copy()
    resource_name = random_suffix_name("route-table-test", 24)
    test_vpc = get_bootstrap_resources().SharedTestVPC
    vpc_id = test_vpc.vpc_id
    igw_id = test_vpc.public_subnets.route_table.internet_gateway.internet_gateway_id
    test_cidr_block = "192.168.0.0/24"

    replacements["ROUTE_TABLE_NAME"] = resource_name
    replacements["VPC_ID"] = vpc_id
    replacements["IGW_ID"] = igw_id
    replacements["DEST_CIDR_BLOCK"] = test_cidr_block

    marker = request.node.get_closest_marker("resource_data")
    if marker is not None:
        data = marker.args[0]
        if 'tag_key' in data:
            replacements["TAG_KEY"] = data["tag_key"]
        if 'tag_value' in data:
            replacements["TAG_VALUE"] = data["tag_value"]

    # Load RouteTable CR
    resource_data = load_ec2_resource(
        "route_table",
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
class TestRouteTable:
    def test_create_delete(self, ec2_client, simple_route_table):
        (ref, cr) = simple_route_table
        resource_id = cr["status"]["routeTableID"]

        # Check Route Table exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_route_table(resource_id)

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check Route Table no longer exists in AWS
        ec2_validator.assert_route_table(resource_id, exists=False)

    def test_crud_route(self, ec2_client, simple_route_table):
        (ref, cr) = simple_route_table
        resource_id = cr["status"]["routeTableID"]

        test_vpc = get_bootstrap_resources().SharedTestVPC
        igw_id = test_vpc.public_subnets.route_table.internet_gateway.internet_gateway_id

        # Check Route Table exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_route_table(resource_id)

        # Check Routes exist (default and desired) in AWS
        ec2_validator.assert_route(resource_id, "local", "CreateRouteTable")
        ec2_validator.assert_route(resource_id, igw_id, "CreateRoute")
        
        # Update the Route
        updated_cidr = "192.168.1.0/24"
        patch = {"spec": {"routes":[
                    {
                        "destinationCIDRBlock": updated_cidr,
                        "gatewayID": igw_id
                    }
        ]}}
        _ = k8s.patch_custom_resource(ref, patch)
        time.sleep(DEFAULT_WAIT_AFTER_SECONDS)

        # assert patched state
        resource = k8s.get_resource(ref)
        assert len(resource['status']['routeStatuses']) == 2
        
        # Delete the Route
        patch = {"spec": {"routes": []}}
        _ = k8s.patch_custom_resource(ref, patch)
        time.sleep(DEFAULT_WAIT_AFTER_SECONDS)

        resource = k8s.get_resource(ref)
        assert len(resource['spec']['routes']) == 0

        # Route should no longer exist in AWS (default will remain)
        ec2_validator.assert_route(resource_id, "local", "CreateRouteTable")
        ec2_validator.assert_route(resource_id, igw_id, "CreateRoute", exists=False)

        # Delete Route Table
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check Route Table no longer exists in AWS
        ec2_validator.assert_route_table(resource_id, exists=False)
    
    @pytest.mark.resource_data({'tag_key': 'initialtagkey', 'tag_value': 'initialtagvalue'})
    def test_crud_tags(self, ec2_client, simple_route_table):
        (ref, cr) = simple_route_table
        
        resource = k8s.get_resource(ref)
        resource_id = cr["status"]["routeTableID"]

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # Check Route Table exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_route_table(resource_id)
        
        # Check tags exist for Route Table resource
        assert resource["spec"]["tags"][0]["key"] == "initialtagkey"
        assert resource["spec"]["tags"][0]["value"] == "initialtagvalue"

        # New pair of tags
        new_tags = [
                {
                    "key": "updatedtagkey",
                    "value": "updatedtagvalue",
                }
               
            ]

        # Patch the Route Table, updating the tags with new pair
        updates = {
            "spec": {"tags": new_tags},
        }

        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # Check resource synced successfully
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)
        
        # Assert tags are updated for Route Table resource
        resource = k8s.get_resource(ref)
        assert resource["spec"]["tags"][0]["key"] == "updatedtagkey"
        assert resource["spec"]["tags"][0]["value"] == "updatedtagvalue"

        # Patch the Route Table resource, deleting the tags
        updates = {
                "spec": {"tags": []},
        }

        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)
        
        # Assert tags are deleted
        resource = k8s.get_resource(ref)
        assert len(resource['spec']['tags']) == 0

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check Route Table no longer exists in AWS
        ec2_validator.assert_route_table(resource_id, exists=False)
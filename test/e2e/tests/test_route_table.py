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

@service_marker
@pytest.mark.canary
class TestRouteTable:
    def test_create_delete(self, ec2_client):
        test_resource_values = REPLACEMENT_VALUES.copy()
        resource_name = random_suffix_name("route-table-test", 24)
        test_vpc = get_bootstrap_resources().SharedTestVPC
        vpc_id = test_vpc.vpc_id
        igw_id = test_vpc.public_subnets.route_table.internet_gateway.internet_gateway_id
        test_cidr_block = "192.168.0.0/24"

        test_resource_values["ROUTE_TABLE_NAME"] = resource_name
        test_resource_values["VPC_ID"] = vpc_id
        test_resource_values["IGW_ID"] = igw_id
        test_resource_values["DEST_CIDR_BLOCK"] = test_cidr_block

        # Load Route Table CR
        resource_data = load_ec2_resource(
            "route_table",
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
        resource_id = resource["status"]["routeTableID"]

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # Check Route Table exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_route_table(resource_id)

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check Route Table no longer exists in AWS
        ec2_validator.assert_route_table(resource_id, exists=False)

    def test_crud_route(self, ec2_client):
        test_resource_values = REPLACEMENT_VALUES.copy()
        resource_name = random_suffix_name("route-table-test", 24)
        test_vpc = get_bootstrap_resources().SharedTestVPC
        vpc_id = test_vpc.vpc_id
        igw_id = test_vpc.public_subnets.route_table.internet_gateway.internet_gateway_id
        test_cidr_block = "192.168.0.0/24"

        test_resource_values["ROUTE_TABLE_NAME"] = resource_name
        test_resource_values["VPC_ID"] = vpc_id
        test_resource_values["IGW_ID"] = igw_id
        test_resource_values["DEST_CIDR_BLOCK"] = test_cidr_block

        # Load Route Table CR
        resource_data = load_ec2_resource(
            "route_table",
            additional_replacements=test_resource_values,
        )
        logging.debug(resource_data)

        # Create Route Table
        ref = k8s.CustomResourceReference(
            CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
            resource_name, namespace="default",
        )
        k8s.create_custom_resource(ref, resource_data)
        cr = k8s.wait_resource_consumed_by_controller(ref)

        assert cr is not None
        assert k8s.get_resource_exists(ref)

        resource = k8s.get_resource(ref)
        resource_id = resource["status"]["routeTableID"]

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

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
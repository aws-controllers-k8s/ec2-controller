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

import boto3
import pytest
import time
import logging

from acktest.resources import random_suffix_name
from acktest.k8s import resource as k8s
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_ec2_resource
from e2e.replacement_values import REPLACEMENT_VALUES

RESOURCE_PLURAL = "subnets"

CREATE_WAIT_AFTER_SECONDS = 10
DELETE_WAIT_AFTER_SECONDS = 10

@pytest.fixture(scope="module")
def ec2_client():
    return boto3.client("ec2")

@service_marker
@pytest.mark.canary
class TestSubnet:
    def get_subnet(self, ec2_client, subnet_id: str) -> dict:
        try:
            resp = ec2_client.describe_subnets()
        except Exception as e:
            logging.debug(e)
            return None

        subnets = resp["Subnets"]
        for subnet in subnets:
            if subnet["SubnetId"] == subnet_id:
                return subnet

        return None

    def subnet_exists(self, ec2_client, subnet_id: str) -> bool:
        return self.get_subnet(ec2_client, subnet_id) is not None

    def test_create_delete(self, ec2_client, pytestconfig):
        test_resource_values = REPLACEMENT_VALUES.copy()
        resource_name = random_suffix_name("subnet-test", 24)
        vpc_id = pytestconfig.cache.get('vpc_id', None)
        vpc_cidr = pytestconfig.cache.get('vpc_cidr', None)

        test_resource_values["SUBNET_NAME"] = resource_name
        test_resource_values["VPC_ID"] = vpc_id
        test_resource_values["CIDR_BLOCK"] = vpc_cidr

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

        resource = k8s.get_resource(ref)
        resource_id = resource["status"]["subnetID"]

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # Check Subnet exists
        exists = self.subnet_exists(ec2_client, resource_id)
        assert exists

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check Subnet doesn't exist
        exists = self.subnet_exists(ec2_client, resource_id)
        assert not exists

    def test_terminal_condition(self, pytestconfig):
        test_resource_values = REPLACEMENT_VALUES.copy()
        resource_name = random_suffix_name("subnet-fail", 24)
        vpc_cidr = pytestconfig.cache.get('vpc_cidr', None)

        test_resource_values["SUBNET_NAME"] = resource_name
        test_resource_values["VPC_ID"] = "InvalidVpcId"
        test_resource_values["CIDR_BLOCK"] = vpc_cidr

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

        expected_msg = "InvalidVpcID.NotFound: The vpc ID 'InvalidVpcId' does not exist"
        terminal_condition = k8s.get_resource_condition(ref, "ACK.Terminal")
        # Example condition message:
        #   InvalidVpcID.NotFound: The vpc ID 'InvalidVpcId' does not exist
        #   status code: 400, request id: 5801fc80-67cf-465f-8b83-5e02d517d554
        # This check only verifies the error message; the request hash is irrelevant and therefore can be ignored.
        assert expected_msg in terminal_condition['message']


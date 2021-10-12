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

"""Integration tests for the Vpc Endpoint API.
"""

import boto3
import pytest
import time
import logging

from acktest.resources import random_suffix_name
from acktest.k8s import resource as k8s
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_ec2_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e.bootstrap_resources import get_bootstrap_resources

RESOURCE_PLURAL = "vpcendpoints"
ENDPOINT_SERVICE_NAME = "com.amazonaws.us-west-2.s3"

CREATE_WAIT_AFTER_SECONDS = 10
DELETE_WAIT_AFTER_SECONDS = 10

@pytest.fixture(scope="module")
def ec2_client():
    return boto3.client("ec2")


def get_vpc_endpoint(ec2_client, vpc_endpoint_id: str) -> dict:
    try:
        resp = ec2_client.describe_vpc_endpoints(
            Filters=[{"Name": "vpc-endpoint-id", "Values": [vpc_endpoint_id]}]
        )
    except Exception as e:
        logging.debug(e)
        return None

    if len(resp["VpcEndpoints"]) == 0:
        return None
    return resp["VpcEndpoints"][0]


def vpc_endpoint_exists(ec2_client, vpc_endpoint_id: str) -> bool:
    return get_vpc_endpoint(ec2_client, vpc_endpoint_id) is not None

@service_marker
@pytest.mark.canary
class TestVpcEndpoint:
    def test_create_delete(self, ec2_client):
        test_resource_values = REPLACEMENT_VALUES.copy()
        resource_name = random_suffix_name("vpc-endpoint-test", 24)
        test_vpc = get_bootstrap_resources().SharedTestVPC
        vpc_id = test_vpc.vpc_id

        test_resource_values["VPC_ENDPOINT_NAME"] = resource_name
        test_resource_values["SERVICE_NAME"] = ENDPOINT_SERVICE_NAME
        test_resource_values["VPC_ID"] = vpc_id

        # Load VPC Endpoint CR
        resource_data = load_ec2_resource(
            "vpc_endpoint",
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
        vpc_endpoint_services = ec2_client.describe_vpc_endpoint_services()
        resource_id = resource["status"]["vpcEndpointID"]

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # Check VPC Endpoint exists
        exists = vpc_endpoint_exists(ec2_client, resource_id)
        assert exists

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check VPC Endpoint doesn't exist
        exists = vpc_endpoint_exists(ec2_client, resource_id)
        assert not exists

    def test_terminal_condition_malformed_vpc(self):
        test_resource_values = REPLACEMENT_VALUES.copy()
        resource_name = random_suffix_name("vpc-endpoint-fail", 24)
        test_resource_values["VPC_ENDPOINT_NAME"] = resource_name
        test_resource_values["SERVICE_NAME"] = ENDPOINT_SERVICE_NAME
        test_resource_values["VPC_ID"] = "MalformedVpcId"

        # Load VPC Endpoint CR
        resource_data = load_ec2_resource(
            "vpc_endpoint",
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

        expected_msg = "InvalidVpcId.Malformed: Invalid Id: 'MalformedVpcId'"
        terminal_condition = k8s.get_resource_condition(ref, "ACK.Terminal")
        # Example condition message:
        # InvalidVpcId.Malformed: Invalid Id: 'MalformedVpcId'
        # (expecting 'vpc-...; the Id may only contain lowercase alphanumeric characters and a single dash')
        # status code: 400, request id: dc3595c5-4e6e-48db-abf7-9bdcc76ad2a8
        # This check only verifies the error message; the request hash is irrelevant and therefore can be ignored.
        assert expected_msg in terminal_condition['message']

    def test_terminal_condition_invalid_service(self):
        test_resource_values = REPLACEMENT_VALUES.copy()
        resource_name = random_suffix_name("vpc-endpoint-fail-2", 24)
        test_resource_values["VPC_ENDPOINT_NAME"] = resource_name
        test_resource_values["SERVICE_NAME"] = "InvalidService"

        test_vpc = get_bootstrap_resources().SharedTestVPC
        vpc_id = test_vpc.vpc_id
        test_resource_values["VPC_ID"] = vpc_id

        # Load VPC Endpoint CR
        resource_data = load_ec2_resource(
            "vpc_endpoint",
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

        expected_msg = "InvalidServiceName: The Vpc Endpoint Service 'InvalidService' does not exist"
        terminal_condition = k8s.get_resource_condition(ref, "ACK.Terminal")
        assert expected_msg in terminal_condition['message']
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

from acktest.resources import random_suffix_name
from acktest.k8s import resource as k8s
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_ec2_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e.bootstrap_resources import get_bootstrap_resources
from e2e.tests.helper import EC2Validator
from .test_elastic_ip_address import RESOURCE_PLURAL as ELASTIC_IP_PLURAL

RESOURCE_PLURAL = "natgateways"

CREATE_WAIT_AFTER_SECONDS = 10
DELETE_WAIT_AFTER_SECONDS = 10

@pytest.fixture
def standard_elastic_address():
    cluster_name = random_suffix_name("nat-gateway-eip", 32)

    replacements = REPLACEMENT_VALUES.copy()
    replacements["ADDRESS_NAME"] = cluster_name

    resource_data = load_ec2_resource(
        "elastic_ip_address",
        additional_replacements=replacements,
    )
    logging.debug(resource_data)

    # Create the k8s resource
    ref = k8s.CustomResourceReference(
        CRD_GROUP, CRD_VERSION, ELASTIC_IP_PLURAL,
        cluster_name, namespace="default",
    )
    k8s.create_custom_resource(ref, resource_data)
    cr = k8s.wait_resource_consumed_by_controller(ref)

    # ElasticIP are not usable immediately after they are created, so this will
    # buy us some time in case we try to mount it too early.
    time.sleep(CREATE_WAIT_AFTER_SECONDS)

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
class TestNATGateway:
    def test_create_delete(self, standard_elastic_address, ec2_client):
        test_resource_values = REPLACEMENT_VALUES.copy()
        resource_name = random_suffix_name("nat-gateway-test", 24)
        test_vpc = get_bootstrap_resources().SharedTestVPC
        subnet_id = test_vpc.public_subnets.subnet_ids[0]

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

        resource = k8s.get_resource(ref)
        resource_id = resource["status"]["natGatewayID"]

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
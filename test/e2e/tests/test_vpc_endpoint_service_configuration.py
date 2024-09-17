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

"""Integration tests for the Vpc Endpoint Service Configuraion API.
"""
from os import environ
import pytest
import time
import logging


from acktest.resources import random_suffix_name
from acktest.k8s import resource as k8s
from e2e.bootstrap_resources import get_bootstrap_resources
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_ec2_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e.tests.helper import EC2Validator

# Default to us-west-2 since that's where prow is deployed
REGION = "us-west-2" if environ.get('AWS_DEFAULT_REGION') is None else environ.get('AWS_DEFAULT_REGION')
RESOURCE_PLURAL = "vpcendpointserviceconfigurations"

CREATE_WAIT_AFTER_SECONDS = 10
DELETE_WAIT_AFTER_SECONDS = 10
MODIFY_WAIT_AFTER_SECONDS = 5

@pytest.fixture
def simple_vpc_endpoint_service_configuration(request):
    test_resource_values = REPLACEMENT_VALUES.copy()
    resources = get_bootstrap_resources()

    supported_ip_address_types = "ipv4"

    resource_name = random_suffix_name("vpc-ep-service", 24)

    test_resource_values["VPC_ENDPOINT_SERVICE_NAME"] = resource_name
    test_resource_values["ACCEPTANCE_REQUIRED"] = "False"
    test_resource_values["PRIVATE_DNS_NAME"] = ""
    test_resource_values["NETWORK_LOAD_BALANCER_ARN_SET"] = resources.NetworkLoadBalancer.arn
    test_resource_values["SUPPORTED_IP_ADDRESS_TYPE_SET"] = supported_ip_address_types
    test_resource_values["ALLOWED_PRINCIPAL"] = "arn:aws:iam::111111111111:root"


    marker = request.node.get_closest_marker("resource_data")
    if marker is not None:
        data = marker.args[0]
        if 'tag_key' in data:
            test_resource_values["TAG_KEY"] = data["tag_key"]
        if 'tag_value' in data:
            test_resource_values["TAG_VALUE"] = data["tag_value"]

    # Load VPC Endpoint Service Configuration CR
    resource_data = load_ec2_resource(
        "vpc_endpoint_service_configuration",
        additional_replacements=test_resource_values,
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
class TestVpcEndpointServiceConfiguration:
    def test_vpc_endpoint_service_configuration_create_delete(self, ec2_client, simple_vpc_endpoint_service_configuration):
        (ref, cr) = simple_vpc_endpoint_service_configuration

        resource_id = cr["status"]["serviceID"]

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # Check VPC Endpoint Service exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_vpc_endpoint_service_configuration(resource_id)

        # Check that the allowedPrincipal is properly set
        allowed_principals = ec2_validator.get_vpc_endpoint_service_permissions(resource_id)
        assert allowed_principals[0]["Principal"] == "arn:aws:iam::111111111111:root"

        # Payload used to remove the Principal
        update_allowed_principals_payload = {
            "spec": {
                "allowedPrincipals": []
            }
        }

        # Patch the VPCPeeringConnection with the payload
        k8s.patch_custom_resource(ref, update_allowed_principals_payload)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)
        
        # Check that the allowedPrincipal is no longer set
        allowed_principals = ec2_validator.get_vpc_endpoint_service_permissions(resource_id)
        assert allowed_principals is None

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check VPC Endpoint Service no longer exists in AWS
        ec2_validator.assert_vpc_endpoint_service_configuration(resource_id, exists=False)
    
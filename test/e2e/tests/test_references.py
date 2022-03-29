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

"""Integration tests for EC2 resource references
"""

from os import environ
import pytest
import time

from acktest.resources import random_suffix_name
from acktest.k8s import resource as k8s
from acktest.aws.identity import get_region
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_ec2_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e.tests.helper import EC2Validator

CREATE_WAIT_AFTER_SECONDS = 20
DELETE_WAIT_AFTER_SECONDS = 10

@service_marker
@pytest.mark.canary
class TestEC2References:
    def test_references(self, ec2_client):
        vpc_endpoint_name = random_suffix_name("vpc-endpoint-test", 24)
        vpc_name = random_suffix_name("vpc-ref-test", 24)
        subnet_name = random_suffix_name("subnet-ref-test", 24)
        security_group_name = random_suffix_name("sec-group-ref-test", 24)

        test_values = REPLACEMENT_VALUES.copy()
        test_values["VPC_ENDPOINT_REF_NAME"] = vpc_endpoint_name
        # Type 'Interface' allows the use of Security Groups and Subnet
        test_values["VPC_ENDPOINT_TYPE"] = "Interface"
        test_values["SERVICE_NAME"] = f'com.amazonaws.{get_region()}.s3'
        test_values["VPC_NAME"] = vpc_name
        test_values["CIDR_BLOCK"] = "10.0.0.0/16"
        test_values["SUBNET_CIDR_BLOCK"] = "10.0.255.0/24"
        test_values["SUBNET_REF_NAME"] = subnet_name
        test_values["SECURITY_GROUP_REF_NAME"] = security_group_name
        test_values["SECURITY_GROUP_DESCRIPTION"] = "TestingSecurityGroup-ack-ref"
        
        # Load CRs
        vpc_endpoint_resource_data = load_ec2_resource(
            "vpc_endpoint_ref",
            additional_replacements=test_values,
        )
        sg_resource_data = load_ec2_resource(
            "security_group_ref",
            additional_replacements=test_values,
        )
        subnet_resource_data = load_ec2_resource(
            "subnet_ref",
            additional_replacements=test_values,
        )
        vpc_resource_data = load_ec2_resource(
            "vpc",
            additional_replacements=test_values,
        )

        # This test creates resources in reverse order (VPC last) so that reference
        # resolution fails upon resource creation. Eventually, resources become synced
        # and references resolve.

        # Create VPC Endpoint. Requires: VPC, Subnet, and SecurityGroup
        vpc_endpoint_ref = k8s.CustomResourceReference(
            CRD_GROUP, CRD_VERSION, 'vpcendpoints',
            vpc_endpoint_name, namespace="default",
        )
        k8s.create_custom_resource(vpc_endpoint_ref, vpc_endpoint_resource_data)

        # Create Subnet. Requires: VPC
        subnet_ref = k8s.CustomResourceReference(
            CRD_GROUP, CRD_VERSION, 'subnets',
            subnet_name, namespace="default",
        )
        k8s.create_custom_resource(subnet_ref, subnet_resource_data)

        # Create SecurityGroups. Requires: VPC
        sg_ref = k8s.CustomResourceReference(
            CRD_GROUP, CRD_VERSION, 'securitygroups',
            security_group_name, namespace="default",
        )
        k8s.create_custom_resource(sg_ref, sg_resource_data)

        # Create VPC. Requires: None
        vpc_ref = k8s.CustomResourceReference(
            CRD_GROUP, CRD_VERSION, 'vpcs',
            vpc_name, namespace="default",
        )
        k8s.create_custom_resource(vpc_ref, vpc_resource_data)

        # Wait a few seconds so resources get persisted in etcd
        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # Check resources sync & resolve
        assert k8s.wait_on_condition(vpc_ref, "ACK.ResourceSynced", "True", wait_periods=5)
        assert k8s.wait_on_condition(sg_ref, "ACK.ResourceSynced", "True", wait_periods=5)
        assert k8s.wait_on_condition(subnet_ref, "ACK.ResourceSynced", "True", wait_periods=5)

        assert k8s.wait_on_condition(sg_ref, "ACK.ReferencesResolved", "True", wait_periods=5)
        assert k8s.wait_on_condition(subnet_ref, "ACK.ReferencesResolved", "True", wait_periods=5)
        assert k8s.wait_on_condition(vpc_endpoint_ref, "ACK.ReferencesResolved", "True", wait_periods=5)

        # Acquire resource IDs
        vpc_endpoint_cr = k8s.get_resource(vpc_endpoint_ref)
        vpc_endpoint_id = vpc_endpoint_cr["status"]["vpcEndpointID"]
        subnet_cr = k8s.get_resource(subnet_ref)
        subnet_id = subnet_cr["status"]["subnetID"]
        sg_cr = k8s.get_resource(sg_ref)
        sg_id = sg_cr["status"]["id"]
        vpc_cr = k8s.get_resource(vpc_ref)
        vpc_id = vpc_cr["status"]["vpcID"]

        # Check resources exist in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_vpc_endpoint(vpc_endpoint_id)
        ec2_validator.assert_subnet(subnet_id)
        ec2_validator.assert_security_group(sg_id)
        ec2_validator.assert_vpc(vpc_id)

        # Delete resources
        _, deleted = k8s.delete_custom_resource(vpc_endpoint_ref, 6, 5)
        assert deleted is True
        # Deleting an interface endpoint also deletes the endpoint network interfaces
        # and therefore requires more time to resolve server-side. Increasing the sleep
        # to a longer duration allows VPC Endpoint to be removed completely. Then, 
        # Subnet and other dependent resources can be deleted successfully.
        time.sleep(70)

        _, deleted = k8s.delete_custom_resource(subnet_ref, 6, 5)
        assert deleted is True
        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        _, deleted = k8s.delete_custom_resource(sg_ref, 6, 5)
        assert deleted is True
        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        _, deleted = k8s.delete_custom_resource(vpc_ref, 6, 5)
        assert deleted is True
        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check resources no longer exist in AWS
        ec2_validator.assert_vpc_endpoint(vpc_endpoint_id, exists=False)
        ec2_validator.assert_subnet(subnet_id, exists=False)
        ec2_validator.assert_security_group(sg_id, exists=False)
        ec2_validator.assert_vpc(vpc_id, exists=False)
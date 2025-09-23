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
import datetime
import pytest
import time

from acktest.resources import random_suffix_name
from acktest.k8s import resource as k8s
from acktest.aws.identity import get_region
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_ec2_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e.tests.helper import EC2Validator

CREATE_WAIT_AFTER_SECONDS = 20
MODIFY_WAIT_AFTER_SECONDS = 30
DELETE_WAIT_AFTER_SECONDS = 10
DELETE_TIMEOUT_SECONDS = 300

def wait_for_delete_or_die(ec2_client, vpc_endpoint_id, timeout):
    while True:
        if datetime.datetime.now() >= timeout:
            pytest.fail("Timed out waiting for VPC Endpoint to be deleted from EC2")
        time.sleep(DELETE_WAIT_AFTER_SECONDS)
        try:
            ec2_client.describe_vpc_endpoints(VpcEndpointIds=[vpc_endpoint_id])
        except ec2_client.exceptions.ClientError:
            break

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
        assert k8s.wait_on_condition(vpc_ref, "Ready", "True", wait_periods=5)
        assert k8s.wait_on_condition(sg_ref, "Ready", "True", wait_periods=5)
        assert k8s.wait_on_condition(subnet_ref, "Ready", "True", wait_periods=5)
        assert k8s.wait_on_condition(vpc_endpoint_ref, "Ready", "True", wait_periods=10)

        assert k8s.wait_on_condition(sg_ref, "ACK.ReferencesResolved", "True", wait_periods=5)
        assert k8s.wait_on_condition(subnet_ref, "ACK.ReferencesResolved", "True", wait_periods=5)
        assert k8s.wait_on_condition(vpc_endpoint_ref, "ACK.ReferencesResolved", "True", wait_periods=10)

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
        
        # If VPC Endpoint is not completely removed server-side, then remaining
        # resources will NOT delete successfully due to dependency exceptions
        now = datetime.datetime.now()
        timeout = now + datetime.timedelta(seconds=DELETE_TIMEOUT_SECONDS)
        wait_for_delete_or_die(ec2_client, vpc_endpoint_id, timeout)

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

    def test_array_references(self, ec2_client):
        route_table_name = random_suffix_name("route-table-test", 24)
        vpc_name = random_suffix_name("vpc-ref-test", 24)
        gateway_name = random_suffix_name("gateway-ref-test", 24)

        test_values = REPLACEMENT_VALUES.copy()
        test_values["ROUTE_TABLE_NAME"] = route_table_name
        test_values["DEST_CIDR_BLOCK"] = "0.0.0.0/0"
        test_values["INTERNET_GATEWAY_NAME"] = gateway_name
        test_values["VPC_NAME"] = vpc_name
        test_values["CIDR_BLOCK"] = "10.0.0.0/16"
        test_values["ENABLE_DNS_SUPPORT"] = "False"
        test_values["ENABLE_DNS_HOSTNAMES"] = "False"
        test_values["DISALLOW_DEFAULT_SECURITY_GROUP_RULE"] = "False"

        # Load CRs
        route_table_resource_data = load_ec2_resource(
            "route_table_ref",
            additional_replacements=test_values
        )
        vpc_resource_data = load_ec2_resource(
            "vpc",
            additional_replacements=test_values
        )
        gateway_resource_data = load_ec2_resource(
            "internet_gateway_ref",
            additional_replacements=test_values
        )

        # This test creates resources in order,
        
        # Create VPC
        vpc_ref = k8s.CustomResourceReference(
            CRD_GROUP, CRD_VERSION, 'vpcs',
            vpc_name, namespace="default",
        )
        k8s.create_custom_resource(vpc_ref, vpc_resource_data)

        # Create Internet Gateway
        gateway_ref = k8s.CustomResourceReference(
            CRD_GROUP, CRD_VERSION, 'internetgateways',
            gateway_name, namespace="default",
        )
        k8s.create_custom_resource(gateway_ref, gateway_resource_data)

        # Create route table
        route_table_ref = k8s.CustomResourceReference(
            CRD_GROUP, CRD_VERSION, 'routetables',
            route_table_name, namespace="default",
        )
        k8s.create_custom_resource(route_table_ref, route_table_resource_data)

        # Wait a few seconds so resources are synced
        time.sleep(CREATE_WAIT_AFTER_SECONDS)
        assert k8s.wait_on_condition(vpc_ref, "Ready", "True", wait_periods=5)
        assert k8s.wait_on_condition(gateway_ref, "Ready", "True", wait_periods=5)
        assert k8s.wait_on_condition(route_table_ref, "Ready", "True", wait_periods=10)

        assert k8s.wait_on_condition(gateway_ref, "ACK.ReferencesResolved", "True", wait_periods=5)
        assert k8s.wait_on_condition(route_table_ref, "ACK.ReferencesResolved", "True", wait_periods=10)

        # Acquire Internet Gateway ID
        gateway_cr = k8s.get_resource(gateway_ref)
        assert 'status' in gateway_cr
        gateway_id = gateway_cr["status"]["internetGatewayID"]

        # Ensure routetable contains reference in spec
        route_table_cr = k8s.get_resource(route_table_ref)
        assert 'spec' in route_table_cr
        assert 'vpcRef' in route_table_cr['spec']
        assert route_table_cr['spec']['vpcRef']['from']['name'] == vpc_name
        assert 'routes' in route_table_cr['spec']
        assert len(route_table_cr['spec']['routes']) == 1
        assert 'gatewayID' not in route_table_cr['spec']['routes'][0]
        assert 'gatewayRef' in route_table_cr['spec']['routes'][0]
        assert route_table_cr['spec']['routes'][0]['gatewayRef']['from']['name'] == gateway_name
        assert 'status' in route_table_cr
        assert 'routeStatuses' in route_table_cr['status']
        found_gateway_id = False
        for rs in route_table_cr['status']['routeStatuses']:
            if 'gatewayID' in rs and rs['gatewayID'] == gateway_id:
                found_gateway_id = True
        assert found_gateway_id

        user_tag = {
            "tag": "my_tag",
            "value": "my_val"
        }
        route_table_update = {
            'spec': {
                'tags': [user_tag]
            }
        }
        k8s.patch_custom_resource(route_table_ref, route_table_update)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)
        assert k8s.wait_on_condition(route_table_ref, "Ready", "True", wait_periods=5)
        assert k8s.wait_on_condition(route_table_ref, "ACK.ReferencesResolved", "True", wait_periods=5)
        
        # Ensure that the reference has not changed
        route_table_cr = k8s.get_resource(route_table_ref)
        assert 'spec' in route_table_cr
        assert 'routes' in route_table_cr['spec']
        assert len(route_table_cr['spec']['routes']) == 1
        assert 'gatewayID' not in route_table_cr['spec']['routes'][0]
        assert 'gatewayRef' in route_table_cr['spec']['routes'][0]
        assert route_table_cr['spec']['routes'][0]['gatewayRef']['from']['name'] == gateway_name

        # Delete All
        _, deleted = k8s.delete_custom_resource(route_table_ref)
        assert deleted
        _, deleted = k8s.delete_custom_resource(gateway_ref)
        assert deleted    
        _, deleted = k8s.delete_custom_resource(vpc_ref)
        assert deleted
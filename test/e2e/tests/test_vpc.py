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

"""Integration tests for the Vpc API.
"""

import pytest
import time
import logging

from acktest import tags
from acktest.resources import random_suffix_name
from acktest.k8s import resource as k8s
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_ec2_resource
from e2e.conftest import simple_vpc
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e.tests.helper import EC2Validator

RESOURCE_PLURAL = "vpcs"
PRIMARY_CIDR_DEFAULT = "10.0.0.0/16"

CREATE_WAIT_AFTER_SECONDS = 10
DELETE_WAIT_AFTER_SECONDS = 10
MODIFY_WAIT_AFTER_SECONDS = 15

def get_vpc_attribute(ec2_client, vpc_id: str, attribute_name: str) -> dict:
    return ec2_client.describe_vpc_attribute(Attribute=attribute_name, VpcId=vpc_id)
    
def get_dns_support(ec2_client, vpc_id: str) -> bool:
    attribute = get_vpc_attribute(ec2_client, vpc_id, 'enableDnsSupport')
    return attribute['EnableDnsSupport']['Value']

def get_dns_hostnames(ec2_client, vpc_id: str) -> bool:
    attribute = get_vpc_attribute(ec2_client, vpc_id, 'enableDnsHostnames')
    return attribute['EnableDnsHostnames']['Value']

def contains_default_sg_rule(ec2_client, vpc_id: str) -> bool:
    response = ec2_client.describe_security_groups(
            Filters=[
                {
                    'Name': 'vpc-id',
                    'Values': [vpc_id]
                },
                {
                    'Name': 'group-name',
                    'Values': ["default"]
                }
            ]
    )

    for sg in response['SecurityGroups']:
        sg_id = sg['GroupId']
        break

    resp = ec2_client.describe_security_group_rules(
            Filters=[
                {
                    'Name': 'group-id',
                    'Values': [sg_id]
                }
            ]
    )

    for rule in resp['SecurityGroupRules']:
        if is_default_sg_ingress_rule(rule):
            return True
        if is_default_sg_egress_rule(rule):
            return True
    
def is_default_sg_egress_rule(rule):
    return (
        rule.get('CidrIpv4') == "0.0.0.0/0" and
        rule.get('FromPort') == -1 and
        rule.get('ToPort') == -1 and
        rule.get('IpProtocol') == "-1" and
        rule.get('IsEgress') is True
    )

def is_default_sg_ingress_rule(rule):
    return (
        rule.get('FromPort') == -1 and
        rule.get('ToPort') == -1 and
        rule.get('IpProtocol') == "-1" and
        rule.get('IsEgress') is False and
        rule.get('ReferencedGroupInfo', {}).get('GroupId') == rule.get('GroupId')
    )

@service_marker
@pytest.mark.canary
class TestVpc:
    def test_crud(self, ec2_client, simple_vpc):
        (ref, cr) = simple_vpc

        resource_id = cr["status"]["vpcID"]

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # Check VPC exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_vpc(resource_id)

        # Validate CIDR Block
        vpc = ec2_validator.get_vpc(resource_id)
        assert len(vpc['CidrBlockAssociationSet']) == 1
        assert vpc['CidrBlockAssociationSet'][0]['CidrBlock'] == PRIMARY_CIDR_DEFAULT

        # Associate secondary CIDR
        secondary_cidr = "10.2.0.0/16"
        updates = {
            "spec": {"cidrBlocks": [PRIMARY_CIDR_DEFAULT, secondary_cidr]}
        }
        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        vpc = ec2_validator.get_vpc(resource_id)
        assert len(vpc['CidrBlockAssociationSet']) == 2
        assert vpc['CidrBlockAssociationSet'][0]['CidrBlock'] == PRIMARY_CIDR_DEFAULT
        assert vpc['CidrBlockAssociationSet'][1]['CidrBlock'] == secondary_cidr

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref, 2, 5)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check VPC no longer exists in AWS
        ec2_validator.assert_vpc(resource_id, exists=False)

    @pytest.mark.resource_data({'tag_key': 'initialtagkey', 'tag_value': 'initialtagvalue'})
    def test_crud_tags(self, ec2_client, simple_vpc):
        (ref, cr) = simple_vpc
        
        resource = k8s.get_resource(ref)
        resource_id = cr["status"]["vpcID"]

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # Check VPC exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_vpc(resource_id)
        
        # Check system and user tags exist for vpc resource
        vpc = ec2_validator.get_vpc(resource_id)
        user_tags = {
            "initialtagkey": "initialtagvalue"
        }
        tags.assert_ack_system_tags(
            tags=vpc["Tags"],
        )
        tags.assert_equal_without_ack_tags(
            expected=user_tags,
            actual=vpc["Tags"],
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

        # Patch the VPC, updating the tags with new pair
        updates = {
            "spec": {"tags": update_tags},
        }

        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # Check resource synced successfully
        assert k8s.wait_on_condition(ref, "Ready", "True", wait_periods=5)
        
        # Check for updated user tags; system tags should persist
        vpc = ec2_validator.get_vpc(resource_id)
        updated_tags = {
            "updatedtagkey": "updatedtagvalue"
        }
        tags.assert_ack_system_tags(
            tags=vpc["Tags"],
        )
        tags.assert_equal_without_ack_tags(
            expected=updated_tags,
            actual=vpc["Tags"],
        )
               
        # Only user tags should be present in Spec
        resource = k8s.get_resource(ref)
        assert len(resource["spec"]["tags"]) == 1
        assert resource["spec"]["tags"][0]["key"] == "updatedtagkey"
        assert resource["spec"]["tags"][0]["value"] == "updatedtagvalue"

        # Patch the VPC resource, deleting the tags
        updates = {
                "spec": {"tags": []},
        }

        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # Check resource synced successfully
        assert k8s.wait_on_condition(ref, "Ready", "True", wait_periods=5)
        
        # Check for removed user tags; system tags should persist
        vpc = ec2_validator.get_vpc(resource_id)
        tags.assert_ack_system_tags(
            tags=vpc["Tags"],
        )
        tags.assert_equal_without_ack_tags(
            expected=[],
            actual=vpc["Tags"],
        )
        
        # Check user tags are removed from Spec
        resource = k8s.get_resource(ref)
        assert len(resource["spec"]["tags"]) == 0

        k8s.wait_on_condition(ref, "Ready", "True", wait_periods=5)
        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check VPC no longer exists in AWS
        ec2_validator.assert_vpc(resource_id, exists=False)

    @pytest.mark.resource_data({'disallow_default_sg_rule': 'true'})
    def test_disallow_default_security_group_rule(self, ec2_client, simple_vpc):
        (ref, cr) = simple_vpc
        resource_id = cr["status"]["vpcID"]

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # Check VPC exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_vpc(resource_id)

        # Make sure default security group rule is deleted
        assert not contains_default_sg_rule(ec2_client, resource_id)

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref, 2, 5)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check VPC no longer exists in AWS
        ec2_validator.assert_vpc(resource_id, exists=False)

    def test_update_disallow_default_security_group_rule(self, ec2_client, simple_vpc):
        (ref, cr) = simple_vpc
        resource_id = cr["status"]["vpcID"]

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        k8s.wait_on_condition(ref, "Ready", "True", wait_periods=5)
        # Check VPC exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_vpc(resource_id)

        # Make sure default security group rule is not deleted
        assert contains_default_sg_rule(ec2_client, resource_id)

        # Set disallowSecurityGroupDefaultRules to delete default security
        # group rule
        updates = {
            "spec": {"disallowSecurityGroupDefaultRules": True}
        }
        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)
        k8s.wait_on_condition(ref, "Ready", "True", wait_periods=5)
        # Make sure default security group rule is deleted
        assert not contains_default_sg_rule(ec2_client, resource_id)

        # Reset disallowSecurityGroupDefaultRules to false.
        # This should be no-op since default security
        # group rule is previously deleted
        updates = {
            "spec": {"disallowSecurityGroupDefaultRules": False}
        }
        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # Make sure default security group rule is deleted
        assert not contains_default_sg_rule(ec2_client, resource_id)

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref, 2, 5)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check VPC no longer exists in AWS
        ec2_validator.assert_vpc(resource_id, exists=False)

    @pytest.mark.resource_data({'enable_dns_support': 'true', 'enable_dns_hostnames': 'true'})
    def test_enable_attributes(self, ec2_client, simple_vpc):
        (ref, cr) = simple_vpc
        resource_id = cr["status"]["vpcID"]

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # Check VPC exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_vpc(resource_id)

        # Assert the attributes are set correctly
        assert get_dns_support(ec2_client, resource_id)
        assert get_dns_hostnames(ec2_client, resource_id)

        # Disable the DNS support
        updates = {
            "spec": {"enableDNSSupport": False}
        }
        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # Assert DNS support has been updated
        assert not get_dns_support(ec2_client, resource_id)
        assert get_dns_hostnames(ec2_client, resource_id)

        # Disable the DNS hostname
        updates = {
            "spec": {"enableDNSHostnames": False}
        }
        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # Assert DNS hostname has been updated
        assert not get_dns_support(ec2_client, resource_id)
        assert not get_dns_hostnames(ec2_client, resource_id)

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref, 2, 5)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check VPC no longer exists in AWS
        ec2_validator.assert_vpc(resource_id, exists=False)

    def test_terminal_condition_invalid_parameter_value(self):
        resource_name = random_suffix_name("vpc-ack-fail", 24)
        replacements = REPLACEMENT_VALUES.copy()
        replacements["VPC_NAME"] = resource_name
        replacements["CIDR_BLOCK"] = "InvalidValue"

        # Load VPC CR
        resource_data = load_ec2_resource(
            "vpc",
            additional_replacements=replacements,
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

        expected_msg = "InvalidParameterValue: Value (InvalidValue) for parameter cidrBlock is invalid. This is not a valid CIDR block."
        terminal_condition = k8s.get_resource_condition(ref, "ACK.Terminal")
        # Example condition message:
        # An error occurred (InvalidParameterValue) when calling the CreateVpc operation:
        # Value (dsfre) for parameter cidrBlock is invalid.
        # This is not a valid CIDR block.
        assert expected_msg in terminal_condition['message']
    
    def test_vpc_creation_multiple_cidr(self,ec2_client):
        resource_name = random_suffix_name("vpc-ack-multicidr", 24)
        replacements = REPLACEMENT_VALUES.copy()
        replacements["VPC_NAME"] = resource_name
        replacements["PRIMARY_CIDR_BLOCK"] = PRIMARY_CIDR_DEFAULT
        replacements["SECONDARY_CIDR_BLOCK"] = "10.2.0.0/16"
        replacements["ENABLE_DNS_SUPPORT"] = "False"
        replacements["ENABLE_DNS_HOSTNAMES"] = "False"

        # Load VPC CR
        resource_data = load_ec2_resource(
            "vpc_multicidr",
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

        resource_id = cr["status"]["vpcID"]

        # Check VPC exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_vpc(resource_id)

        # Validate CIDR Block
        vpc = ec2_validator.get_vpc(resource_id)
        assert len(vpc['CidrBlockAssociationSet']) == 2
        assert vpc['CidrBlockAssociationSet'][0]['CidrBlock'] == PRIMARY_CIDR_DEFAULT
        assert vpc['CidrBlockAssociationSet'][1]['CidrBlock'] == "10.2.0.0/16"

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref, 3, 10)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check VPC no longer exists in AWS
        ec2_validator.assert_vpc(resource_id, exists=False)

    def test_vpc_updation_multiple_cidr(self,ec2_client):
        resource_name = random_suffix_name("vpc-ack-multicidr", 24)
        replacements = REPLACEMENT_VALUES.copy()
        replacements["VPC_NAME"] = resource_name
        replacements["PRIMARY_CIDR_BLOCK"] = PRIMARY_CIDR_DEFAULT
        replacements["SECONDARY_CIDR_BLOCK"] = "10.2.0.0/16"
        replacements["ENABLE_DNS_SUPPORT"] = "False"
        replacements["ENABLE_DNS_HOSTNAMES"] = "False"

        # Load VPC CR
        resource_data = load_ec2_resource(
            "vpc_multicidr",
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

        resource_id = cr["status"]["vpcID"]

        # Check VPC exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_vpc(resource_id)

        # Validate CIDR Block
        vpc = ec2_validator.get_vpc(resource_id)
        assert len(vpc['CidrBlockAssociationSet']) == 2
        assert vpc['CidrBlockAssociationSet'][0]['CidrBlock'] == PRIMARY_CIDR_DEFAULT
        assert vpc['CidrBlockAssociationSet'][1]['CidrBlock'] == "10.2.0.0/16"


        # Remove SECONDARY_CIDR_BLOCK
        updates = {
            "spec": {"cidrBlocks": [PRIMARY_CIDR_DEFAULT]}
        }
        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        k8s.wait_on_condition(ref, "Ready", "True", wait_periods=5)
        # Re-Validate CIDR Blocks State
        vpc = ec2_validator.get_vpc(resource_id)
        assert vpc['CidrBlockAssociationSet'][0]['CidrBlockState']['State'] == "associated"
        assert vpc['CidrBlockAssociationSet'][1]['CidrBlockState']['State'] == "disassociated"


        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref, 3, 10)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check VPC no longer exists in AWS
        ec2_validator.assert_vpc(resource_id, exists=False)

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

from os import environ
import pytest
import time
import logging

from acktest import tags
from acktest.resources import random_suffix_name
from acktest.k8s import resource as k8s
from acktest.k8s import condition
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_ec2_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e.bootstrap_resources import get_bootstrap_resources
from e2e.tests.helper import EC2Validator

# Default to us-west-2 since that's where prow is deployed
REGION = "us-west-2" if environ.get('AWS_DEFAULT_REGION') is None else environ.get('AWS_DEFAULT_REGION')
RESOURCE_PLURAL = "vpcendpoints"
ENDPOINT_SERVICE_NAME = "com.amazonaws.%s.s3" % REGION

CREATE_WAIT_AFTER_SECONDS = 10
MODIFY_WAIT_AFTER_SECONDS = 5

@pytest.fixture
def simple_vpc_endpoint(request):
    test_resource_values = REPLACEMENT_VALUES.copy()
    resource_name = random_suffix_name("vpc-endpoint-test", 24)
    test_vpc = get_bootstrap_resources().SharedTestVPC
    vpc_id = test_vpc.vpc_id

    test_resource_values["VPC_ENDPOINT_NAME"] = resource_name
    test_resource_values["SERVICE_NAME"] = ENDPOINT_SERVICE_NAME
    test_resource_values["VPC_ID"] = vpc_id

    marker = request.node.get_closest_marker("resource_data")
    if marker is not None:
        data = marker.args[0]
        if 'tag_key' in data:
            test_resource_values["TAG_KEY"] = data["tag_key"]
        if 'tag_value' in data:
            test_resource_values["TAG_VALUE"] = data["tag_value"]

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

@pytest.fixture
def modify_vpc_endpoint(request):
    test_resource_values = REPLACEMENT_VALUES.copy()
    resource_name = random_suffix_name("vpc-endpoint-test", 24)
    test_vpc = get_bootstrap_resources().SharedTestVPC
    vpc_id = test_vpc.vpc_id

    test_resource_values["VPC_ENDPOINT_NAME"] = resource_name
    test_resource_values["SERVICE_NAME"] = ENDPOINT_SERVICE_NAME
    test_resource_values["VPC_ID"] = vpc_id
    test_resource_values["SUBNET_ID"] = test_vpc.public_subnets.subnet_ids[0]


    marker = request.node.get_closest_marker("resource_data")
    if marker is not None:
        data = marker.args[0]
        if 'tag_key' in data:
            test_resource_values["TAG_KEY"] = data["tag_key"]
        if 'tag_value' in data:
            test_resource_values["TAG_VALUE"] = data["tag_value"]

    # Load VPC Endpoint CR
    resource_data = load_ec2_resource(
        "vpc_endpoint_modify",
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
class TestVpcEndpoint:
    def test_create_delete(self, ec2_client, simple_vpc_endpoint):
        (ref, cr) = simple_vpc_endpoint

        resource_id = cr["status"]["vpcEndpointID"]

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # Check VPC Endpoint exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_vpc_endpoint(resource_id)

        # Update policyDocument with a different policy to verify
        # updates still work with is_iam_policy comparison
        updates = {
            "spec": {
                "policyDocument": '{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":"*","Action":"s3:GetObject","Resource":"arn:aws:s3:::*"}]}'
            }
        }
        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        # Check VPC Endpoint no longer exists in AWS
        assert ec2_validator.wait_vpc_endpoint_deleted(resource_id)
    
    @pytest.mark.resource_data({'tag_key': 'initialtagkey', 'tag_value': 'initialtagvalue'})
    def test_crud_tags(self, ec2_client, simple_vpc_endpoint):
        (ref, cr) = simple_vpc_endpoint
        
        resource = k8s.get_resource(ref)
        resource_id = cr["status"]["vpcEndpointID"]

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # Check vpcEndpoint exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_vpc_endpoint(resource_id)
        
        # Check system and user tags exist for vpc endpoint resource
        vpc_endpoint = ec2_validator.get_vpc_endpoint(resource_id)
        user_tags = {
            "initialtagkey": "initialtagvalue"
        }
        tags.assert_ack_system_tags(
            tags=vpc_endpoint["Tags"],
        )
        tags.assert_equal_without_ack_tags(
            expected=user_tags,
            actual=vpc_endpoint["Tags"],
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

        # Patch the vpcEndpoint, updating the tags with new pair
        updates = {
            "spec": {"tags": update_tags},
        }

        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # Check resource synced successfully
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)
        
        # Check for updated user tags; system tags should persist
        vpc_endpoint = ec2_validator.get_vpc_endpoint(resource_id)
        updated_tags = {
            "updatedtagkey": "updatedtagvalue"
        }
        tags.assert_ack_system_tags(
            tags=vpc_endpoint["Tags"],
        )
        tags.assert_equal_without_ack_tags(
            expected=updated_tags,
            actual=vpc_endpoint["Tags"],
        )
               
        # Only user tags should be present in Spec
        resource = k8s.get_resource(ref)
        assert len(resource["spec"]["tags"]) == 1
        assert resource["spec"]["tags"][0]["key"] == "updatedtagkey"
        assert resource["spec"]["tags"][0]["value"] == "updatedtagvalue"

        # Patch the vpcEndpoint resource, deleting the tags
        updates = {
                "spec": {"tags": []},
        }

        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # Check resource synced successfully
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)
        
        # Check for removed user tags; system tags should persist
        vpc_endpoint = ec2_validator.get_vpc_endpoint(resource_id)
        tags.assert_ack_system_tags(
            tags=vpc_endpoint["Tags"],
        )
        tags.assert_equal_without_ack_tags(
            expected=[],
            actual=vpc_endpoint["Tags"],
        )
        
        # Check user tags are removed from Spec
        resource = k8s.get_resource(ref)
        assert len(resource["spec"]["tags"]) == 0

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        # Check vpcEndpoint no longer exists in AWS
        assert ec2_validator.wait_vpc_endpoint_deleted(resource_id)

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

    def test_update_subnets(self, ec2_client, modify_vpc_endpoint):
        (ref, cr) = modify_vpc_endpoint
        resource_id = cr["status"]["vpcEndpointID"]

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # Check VPC Endpoint exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_vpc_endpoint(resource_id)

        # Get initial state
        vpc_endpoint = ec2_validator.get_vpc_endpoint(resource_id)
        initial_subnets = vpc_endpoint.get("SubnetIds", [])

        # Get an additional subnet from the test VPC
        test_vpc = get_bootstrap_resources().SharedTestVPC
        available_subnets = test_vpc.public_subnets.subnet_ids
        new_subnet = next(subnet for subnet in available_subnets if subnet not in initial_subnets)

        # Update subnets
        updated_subnets = initial_subnets + [new_subnet]

        # Patch the VPC Endpoint
        updates = {
            "spec": {"subnetIDs": updated_subnets}
        }

        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # Check resource synced successfully
        assert k8s.wait_on_condition(
            ref, "ACK.ResourceSynced", "True", wait_periods=5)

        # Verify the update in AWS
        vpc_endpoint = ec2_validator.get_vpc_endpoint(resource_id)
        assert set(vpc_endpoint["SubnetIds"]) == set(updated_subnets)

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        # Check VPC Endpoint no longer exists in AWS
        assert ec2_validator.wait_vpc_endpoint_deleted(resource_id)

    def test_update_security_groups(self, ec2_client, modify_vpc_endpoint):
        (ref, cr) = modify_vpc_endpoint
        resource_id = cr["status"]["vpcEndpointID"]
        vpc_id = get_bootstrap_resources().SharedTestVPC.vpc_id

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # Check VPC Endpoint exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_vpc_endpoint(resource_id)

        # An S3 interface endpoint in us-west-2 stays pending for about 5.5
        # minutes after creation, and the controller keeps ACK.ResourceSynced
        # False until the endpoint is available. Without this wait, the wait
        # after the first patch also has to cover the creation time.
        assert k8s.wait_on_condition(
            ref, "ACK.ResourceSynced", "True", wait_periods=10)

        sg_ids = []
        try:
            for _ in range(2):
                resp = ec2_client.create_security_group(
                    GroupName=random_suffix_name("vpc-endpoint-sg", 24),
                    Description="ack-ec2-controller e2e VPC endpoint security group",
                    VpcId=vpc_id,
                )
                sg_ids.append(resp["GroupId"])
            (first_sg_id, second_sg_id) = sg_ids

            # The endpoint is created with no securityGroupIDs, so AWS gives it
            # the VPC's default security group. The first patch checks that the
            # controller also removes that group.
            for security_group_ids in (
                [first_sg_id],
                [first_sg_id, second_sg_id],
                [second_sg_id],
            ):
                updates = {
                    "spec": {"securityGroupIDs": security_group_ids}
                }

                # Until the controller handles the patch, ACK.ResourceSynced
                # still reads True from before it. The controller sets a new
                # lastTransitionTime each time it reconciles the resource, so
                # waiting for a newer one skips that old True.
                last_synced = condition.get_synced_last_transition_time(ref)
                k8s.patch_custom_resource(ref, updates)
                time.sleep(MODIFY_WAIT_AFTER_SECONDS)

                # Check resource synced successfully
                assert k8s.wait_on_condition_after(
                    ref, "ACK.ResourceSynced", "True",
                    last_transition_after=last_synced, wait_periods=10)

                # Verify the update in AWS
                vpc_endpoint = ec2_validator.get_vpc_endpoint(resource_id)
                aws_sg_ids = {group["GroupId"] for group in vpc_endpoint["Groups"]}
                assert aws_sg_ids == set(security_group_ids)
        finally:
            # A security group cannot be deleted while the endpoint uses it,
            # so delete the endpoint and wait for it to go first. The asserts
            # run last, so a failed one does not skip deleting the groups.
            _, deleted = k8s.delete_custom_resource(ref, 3, 10)
            endpoint_deleted = ec2_validator.wait_vpc_endpoint_deleted(resource_id)
            for sg_id in sg_ids:
                try:
                    ec2_validator.delete_security_group(sg_id)
                except Exception as e:
                    logging.warning(e)
            assert deleted is True
            assert endpoint_deleted

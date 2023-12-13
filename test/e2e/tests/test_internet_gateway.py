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

"""Integration tests for the InternetGateway API.
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
from e2e.bootstrap_resources import get_bootstrap_resources

from .test_route_table import (
    RESOURCE_PLURAL as ROUTE_TABLE_PLURAL,
    CREATE_WAIT_AFTER_SECONDS as ROUTE_TABLE_CREATE_WAIT,
)

RESOURCE_PLURAL = "internetgateways"

CREATE_WAIT_AFTER_SECONDS = 10
MODIFY_WAIT_AFTER_SECONDS = 10
DELETE_WAIT_AFTER_SECONDS = 10

@pytest.fixture
def simple_internet_gateway(request, simple_vpc):
    resource_name = random_suffix_name("ig-ack-test", 24)
    resource_file = "internet_gateway"

    replacements = REPLACEMENT_VALUES.copy()
    replacements["INTERNET_GATEWAY_NAME"] = resource_name

    marker = request.node.get_closest_marker("resource_data")
    if marker is not None:
        data = marker.args[0]
        if 'resource_file' in data:
            resource_file = data['resource_file']
        if 'create_vpc' in data and data['create_vpc'] is True:
            (_, vpc_cr) = simple_vpc
            vpc_id = vpc_cr["status"]["vpcID"]
            replacements["VPC_ID"] = vpc_id
        if 'tag_key' in data:
            replacements["TAG_KEY"] = data["tag_key"]
        if 'tag_value' in data:
            replacements["TAG_VALUE"] = data["tag_value"]

    # Load Internet Gateway CR
    resource_data = load_ec2_resource(
        resource_file,
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
class TestInternetGateway:
    def test_create_delete(self, ec2_client, simple_internet_gateway):
        (ref, cr) = simple_internet_gateway
        resource_id = cr["status"]["internetGatewayID"]

        # Check Internet Gateway exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_internet_gateway(resource_id)

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref, 2, 5)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check Internet Gateway no longer exists in AWS
        ec2_validator.assert_internet_gateway(resource_id, exists=False)

    @pytest.mark.resource_data({'create_vpc': True, 'resource_file': 'internet_gateway_vpc_attachment'})
    def test_vpc_association(self, ec2_client, simple_internet_gateway):
        (ref, cr) = simple_internet_gateway

        vpc_id = cr["spec"]["vpc"]
        resource_id = cr["status"]["internetGatewayID"]

        # Check Internet Gateway exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_internet_gateway(resource_id)

        # Check attachments appear on Internet Gateway
        igw = ec2_validator.get_internet_gateway(resource_id)
        assert len(igw["Attachments"]) == 1
        assert igw["Attachments"][0]["VpcId"] == vpc_id
        assert igw["Attachments"][0]["State"] == "available"

        rt_ref, rt_cr = simple_route_table(vpc_id)
        rt_id = rt_cr["status"]["routeTableID"]

        # Patch the IGW, adding route table association
        updates = {
            "spec": {"routeTables": [rt_id]},
        }
        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # Verify gateway is associated with route table
        ec2_validator.assert_route_table_association(
            rt_id, resource_id, "associated", True
        )

        # Patch the IGW, removing the route table association
        updates = {
            "spec": {"routeTables": None},
        }
        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # Verify gateway is disassociated with route table
        ec2_validator.assert_route_table_association(
            rt_id, resource_id, "disassociated", True
        )

        # Patch the IGW, removing the attachment
        updates = {
            "spec": {"vpc": None},
        }
        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # Check there are no attachments on Internet Gateway
        igw = ec2_validator.get_internet_gateway(resource_id)

        # In the case where it shows the attachment as being in detached state
        if len(igw["Attachments"]) == 1:
            assert igw["Attachments"][0]["VpcId"] == vpc_id
            assert igw["Attachments"][0]["State"] == "detached"
        else:
            # Otherwise there should be no attachment on the IGW
            assert len(igw["Attachments"]) == 0

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref, 2, 5)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check Internet Gateway no longer exists in AWS
        ec2_validator.assert_internet_gateway(resource_id, exists=False)

        # Delete route table
        _, deleted = k8s.delete_custom_resource(rt_ref, 2, 5)
        assert deleted is True

    @pytest.mark.resource_data({'tag_key': 'initialtagkey', 'tag_value': 'initialtagvalue'})
    def test_crud_tags(self, ec2_client, simple_internet_gateway):
        (ref, cr) = simple_internet_gateway
        
        resource = k8s.get_resource(ref)
        resource_id = cr["status"]["internetGatewayID"]

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # Check IGW exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_internet_gateway(resource_id)
        
        # Check system and user tags exist for igw resource
        internet_gateway = ec2_validator.get_internet_gateway(resource_id)
        user_tags = {
            "initialtagkey": "initialtagvalue"
        }
        tags.assert_ack_system_tags(
            tags=internet_gateway["Tags"],
        )
        tags.assert_equal_without_ack_tags(
            expected=user_tags,
            actual=internet_gateway["Tags"],
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

        # Patch the IGW, updating the tags with new pair
        updates = {
            "spec": {"tags": update_tags},
        }

        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # Check resource synced successfully
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)
        
        # Check for updated user tags; system tags should persist
        internet_gateway = ec2_validator.get_internet_gateway(resource_id)
        updated_tags = {
            "updatedtagkey": "updatedtagvalue"
        }
        tags.assert_ack_system_tags(
            tags=internet_gateway["Tags"],
        )
        tags.assert_equal_without_ack_tags(
            expected=updated_tags,
            actual=internet_gateway["Tags"],
        )
               
        # Only user tags should be present in Spec
        resource = k8s.get_resource(ref)
        assert len(resource["spec"]["tags"]) == 1
        assert resource["spec"]["tags"][0]["key"] == "updatedtagkey"
        assert resource["spec"]["tags"][0]["value"] == "updatedtagvalue"

        # Patch the IGW resource, deleting the tags
        updates = {
                "spec": {"tags": []},
        }

        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # Check resource synced successfully
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)

        # Check for removed user tags; system tags should persist
        internet_gateway = ec2_validator.get_internet_gateway(resource_id)
        tags.assert_ack_system_tags(
            tags=internet_gateway["Tags"],
        )
        tags.assert_equal_without_ack_tags(
            expected=[],
            actual=internet_gateway["Tags"],
        )
        
        # Check user tags are removed from Spec
        resource = k8s.get_resource(ref)
        assert len(resource["spec"]["tags"]) == 0

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check IGW no longer exists in AWS
        ec2_validator.assert_internet_gateway(resource_id, exists=False)

def simple_route_table(vpc_id: int):
    replacements = REPLACEMENT_VALUES.copy()
    resource_name = random_suffix_name("igw-route-table", 24)

    replacements["ROUTE_TABLE_NAME"] = resource_name
    replacements["VPC_ID"] = vpc_id

    resource_data = load_ec2_resource(
        "internet_gateway_route_table",
        additional_replacements=replacements,
    )
    logging.debug(resource_data)

    # Create the k8s resource
    ref = k8s.CustomResourceReference(
        CRD_GROUP,
        CRD_VERSION,
        ROUTE_TABLE_PLURAL,
        resource_name,
        namespace="default",
    )
    k8s.create_custom_resource(ref, resource_data)
    cr = k8s.wait_resource_consumed_by_controller(ref)

    time.sleep(ROUTE_TABLE_CREATE_WAIT)

    assert cr is not None
    assert k8s.get_resource_exists(ref)

    return ref, cr

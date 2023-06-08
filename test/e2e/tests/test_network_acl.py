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

"""Integration tests for the Network ACL API.
"""

import pytest
import time
import logging

from acktest import tags
from acktest.resources import random_suffix_name
from acktest.k8s import resource as k8s
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_ec2_resource
from e2e.bootstrap_resources import get_bootstrap_resources
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e.tests.helper import EC2Validator

RESOURCE_PLURAL = "networkacls"

CREATE_WAIT_AFTER_SECONDS = 10
DELETE_WAIT_AFTER_SECONDS = 10
MODIFY_WAIT_AFTER_SECONDS = 5

def get_network_acl_ids(ec2_client, network_acl_id: str) -> list:
    network_acl_ids = [network_acl_id]
    try:
        resp = ec2_client.describe_network_acls(
            NetworkAclIds=network_acl_ids
        )
    except Exception as e:
        logging.debug(e)
        return None

    network_acls = resp['NetworkAcls']

    if len(network_acls) == 0:
        return None
    return network_acls

def network_acl_exists(ec2_client, network_acl_id: str) -> bool:
    return get_network_acl_ids(ec2_client, network_acl_id) is not None

@pytest.fixture
def simple_network_acl(request):
    resource_name = random_suffix_name("network-acl-ack-test", 24)
    resource_file = "network_acl"
    resources = get_bootstrap_resources()

    replacements = REPLACEMENT_VALUES.copy()
    replacements["NETWORK_ACL_NAME"] = resource_name
    replacements["VPC_ID"] = resources.SharedTestVPC.vpc_id
    #replacements["TAG_KEY"] = "Name"
    #replacements["TAG_VALUE"] = resource_name

    marker = request.node.get_closest_marker("resource_data")
    if marker is not None:
        data = marker.args[0]
        if 'resource_file' in data:
            resource_file = data['resource_file']
        if 'vpc_id' in data:
            replacements["VPC_ID"] = data['vpc_id']
        if 'tag_key' in data:
            replacements["TAG_KEY"] = data['tag_key']
        if 'tag_value' in data:
            replacements["TAG_VALUE"] = data['tag_value']

    # Load NetworkACL CR
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
class TestNetworkACLs:
    def test_create_delete(self, ec2_client, simple_network_acl):
        (ref, cr) = simple_network_acl
        print(cr)
        network_acl_id = cr["status"]["networkACLID"]
        print(network_acl_id)

        # Check Network ACL exists
        exists = network_acl_exists(ec2_client, network_acl_id)
        assert exists

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref, 2, 5)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check Network ACL doesn't exist
        exists = network_acl_exists(ec2_client, simple_network_acl)
        assert not exists

    @pytest.mark.resource_data({'tag_key': 'initialtagkey', 'tag_value': 'initialtagvalue'})
    def test_crud_tags(self, ec2_client, simple_network_acl):
        (ref, cr) = simple_network_acl

        resource = k8s.get_resource(ref)
        resource_id = cr["status"]["networkACLID"]

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # Check NetworkAcl exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_network_acl(resource_id)

        # Check system and user tags exist for networkAcl resource
        network_acl = ec2_validator.get_network_acl(resource_id)
        user_tags = {
            "initialtagkey": "initialtagvalue"
        }
        tags.assert_ack_system_tags(
            tags=network_acl["Tags"],
        )
        tags.assert_equal_without_ack_tags(
            expected=user_tags,
            actual=network_acl["Tags"],
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

        # Patch the networkAcl, updating the tags with new pair
        updates = {
            "spec": {"tags": update_tags},
        }

        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # Check resource synced successfully
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)

        # Check for updated user tags; system tags should persist
        network_acl = ec2_validator.get_network_acl(resource_id)
        updated_tags = {
            "updatedtagkey": "updatedtagvalue"
        }
        tags.assert_ack_system_tags(
            tags=network_acl["Tags"],
        )
        tags.assert_equal_without_ack_tags(
            expected=updated_tags,
            actual=network_acl["Tags"],
        )

        # Only user tags should be present in Spec
        resource = k8s.get_resource(ref)
        assert len(resource["spec"]["tags"]) == 1
        assert resource["spec"]["tags"][0]["key"] == "updatedtagkey"
        assert resource["spec"]["tags"][0]["value"] == "updatedtagvalue"

        # Patch the networkAcl resource, deleting the tags
        updates = {
                "spec": {"tags": []},
        }

        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # Check resource synced successfully
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)

        # Check for removed user tags; system tags should persist
        network_acl = ec2_validator.get_network_acl(resource_id)
        tags.assert_ack_system_tags(
            tags=network_acl["Tags"],
        )
        tags.assert_equal_without_ack_tags(
            expected=[],
            actual=network_acl["Tags"],
        )

        # Check user tags are removed from Spec
        resource = k8s.get_resource(ref)
        assert len(resource["spec"]["tags"]) == 0

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check networkAcl no longer exists in AWS
        ec2_validator.assert_network_acl(resource_id, exists=False)


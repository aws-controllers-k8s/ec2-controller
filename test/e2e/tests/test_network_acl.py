
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
DEFAULT_WAIT_AFTER_SECONDS = 5

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
    resource_name = random_suffix_name("network-acl-test", 24)
    resource_file = "network_acl"
    resources = get_bootstrap_resources()

    replacements = REPLACEMENT_VALUES.copy()
    replacements["NETWORK_ACL_NAME"] = resource_name
    replacements["VPC_ID"] = resources.SharedTestVPC.vpc_id
    replacements["CIDR_BLOCK"] = "192.168.1.0/24"
    replacements["TAG_KEY"] = "initialtagkey"
    replacements["TAG_VALUE"] = "initialtagvalue"
    replacements["SUBNET_ID"] = resources.SharedTestVPC.public_subnets.subnet_ids[0]

    marker = request.node.get_closest_marker("resource_data")
    if marker is not None:
        data = marker.args[0]
        if 'vpc_id' in data:
            replacements["VPC_ID"] = data['vpc_id']
        if 'cidr_block' in data:
            replacements["CIDR_BLOCK"] = data['cidr_block']
        if 'tag_key' in data:
            replacements["TAG_KEY"] = data['tag_key']
        if 'tag_value' in data:
            replacements["TAG_VALUE"] = data['tag_value']
        if 'resource_file' in data:
            resource_file = data['resource_file']

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
        network_acl_id = cr["status"]["id"]

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

    def test_crud_entry(self, ec2_client, simple_network_acl):
        (ref, cr) = simple_network_acl
        network_acl_id = cr["status"]["id"]

        # Check Route Table exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_network_acl(network_acl_id)

        # Check Entries exist (default and desired) in AWS
        ec2_validator.assert_entry(network_acl_id, 32767, "True")
        ec2_validator.assert_entry(network_acl_id, 32767, "False")

        # Update the Entry

        updated_cidr = "192.168.1.0/24"
        patch = {"spec": {"entries":[
                    {
                        "cidrBlock": updated_cidr,
                        "egress": True,
                        "portRange": {
                            "from": 1025,
                            "to": 1026
                        },
                        "protocol": "6",
                        "ruleAction": "allow",
                        "ruleNumber": 100
                    },
                    {
                        "cidrBlock": updated_cidr,
                        "egress": False,
                        "portRange": {
                            "from": 1025,
                            "to": 1026
                        },
                        "protocol": "6",
                        "ruleAction": "allow",
                        "ruleNumber": 100
                    }
                ]}}
        _ = k8s.patch_custom_resource(ref, patch)
        time.sleep(DEFAULT_WAIT_AFTER_SECONDS)

        # assert patched state
        resource = k8s.get_resource(ref)
        # Check Entries exist (default and desired) in AWS
        ec2_validator.assert_entry(network_acl_id, 32767, "True")
        ec2_validator.assert_entry(network_acl_id, 32767, "False")
        ec2_validator.assert_entry(network_acl_id, 100, "True")
        ec2_validator.assert_entry(network_acl_id, 100, "False")


        test_vpc = get_bootstrap_resources().SharedTestVPC
        subnet_id = test_vpc.public_subnets.subnet_ids[0]


        # Update associations
        update_associations = [
                {
                "subnetID": subnet_id
            }
            ]

        # Patch the networkAcl, updating the tags with new pair
        updates = {
            "spec": {"associations": update_associations},
        }

        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # Check resource synced successfully
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)

        # assert patched state
        resource = k8s.get_resource(ref)
        # Check Association exist in AWS
        ec2_validator.assert_association(network_acl_id, subnet_id)

        # Removing association so that nacl can be deleted
        updates = {
            "spec": {"associations": []},
        }
        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # Check resource synced successfully
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)

        # Delete Network ACL 
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check Network ACL no longer exists in AWS
        ec2_validator.assert_network_acl(network_acl_id, exists=False)

    @pytest.mark.resource_data({'resource_file': 'network_acl_with_subnet_assoc'})
    def test_create_delete_with_subnet_assoc(self, ec2_client, simple_network_acl):
        (ref, cr) = simple_network_acl
        network_acl_id = cr["status"]["id"]

        # Check Route Table exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_network_acl(network_acl_id)

        assocs = cr["spec"]["associations"]
        subnet_id = assocs[0]["subnetID"]
        # Check Association exist in AWS
        ec2_validator.assert_association(network_acl_id, subnet_id)

        # Delete Network ACL
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check Network ACL no longer exists in AWS
        ec2_validator.assert_network_acl(network_acl_id, exists=False)

    def test_crud_tags(self, ec2_client, simple_network_acl):
        (ref, cr) = simple_network_acl

        resource = k8s.get_resource(ref)
        resource_id = cr["status"]["id"]

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

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

RESOURCE_PLURAL = "vpcpeeringconnections"
VPC_RESOURCE_PLURAL = "vpcs"

CREATE_WAIT_AFTER_SECONDS = 10
DELETE_WAIT_AFTER_SECONDS = 10
MODIFY_WAIT_AFTER_SECONDS = 5
DEFAULT_WAIT_AFTER_SECONDS = 5

def get_vpc_peering_connection_ids(ec2_client, vpc_peering_connection_id: str) -> list:
    vpc_peering_connection_ids = [vpc_peering_connection_id]
    try:
        resp = ec2_client.describe_vpc_peering_connections(
            VpcPeeringConnectionIds=vpc_peering_connection_ids
        )
    except Exception as e:
        logging.debug(e)
        return None

    vpc_peering_connections = resp['VpcPeeringConnections']

    if len(vpc_peering_connections) == 0:
        return None
    return vpc_peering_connections

def vpc_peering_connection_exists(ec2_client, vpc_peering_connection_id: str) -> bool:
    return get_vpc_peering_connection_ids(ec2_client, vpc_peering_connection_id) is not None

@pytest.fixture
def simple_vpc_peering_connection(request):
    resource_name = random_suffix_name("simple-vpc-peering-connection-test", 40)
    resources = get_bootstrap_resources()

    # Create an additional VPC to test Peering with the Shared Test VPC

    # Replacements for Test VPC
    replacements = REPLACEMENT_VALUES.copy()
    replacements["VPC_NAME"] = resource_name
    replacements["CIDR_BLOCK"] = "10.1.0.0/16"
    replacements["ENABLE_DNS_SUPPORT"] = "False"
    replacements["ENABLE_DNS_HOSTNAMES"] = "False"
    
    # Load VPC CR
    vpc_resource_data = load_ec2_resource(
        "vpc",
        additional_replacements=replacements,
    )
    logging.debug(vpc_resource_data)

    # Create k8s resource
    vpc_ref = k8s.CustomResourceReference(
        CRD_GROUP, CRD_VERSION, VPC_RESOURCE_PLURAL,
        resource_name, namespace="default",
    )
    k8s.create_custom_resource(vpc_ref, vpc_resource_data)
    time.sleep(CREATE_WAIT_AFTER_SECONDS)

    vpc_cr = k8s.wait_resource_consumed_by_controller(vpc_ref)
    assert vpc_cr is not None
    assert k8s.get_resource_exists(vpc_ref)

    # Create the VPC Peering Connection

    # Replacements for VPC Peering Connection
    replacements["VPC_PEERING_CONNECTION_NAME"] = resource_name
    replacements["VPC_ID"] = resources.SharedTestVPC.vpc_id
    replacements["PEER_VPC_ID"] = vpc_cr["status"]["vpcID"]

    # Load VPCPeeringConnection CR
    resource_data = load_ec2_resource(
        "vpc_peering_connection",
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
    # Can't uncomment this line until ACK VPCs support auto-accepting VPC Peering Requests
    # assert cr["status"]["code"] == "active" 

    yield (ref, cr)

    # Delete VPC Peering Connection k8s resource 
    _, deleted = k8s.delete_custom_resource(ref, 3, 10)
    assert deleted is True

    time.sleep(DELETE_WAIT_AFTER_SECONDS)

    # Delete VPC resource 
    _, vpc_deleted = k8s.delete_custom_resource(vpc_ref, 3, 10)
    assert vpc_deleted is True

@pytest.fixture
def ref_vpc_peering_connection(request):
    resource_name = random_suffix_name("ref-vpc-peering-connection-test", 40)

    # Create 2 VPCs with ACK to test Peering with and refer to them by their k8s resource name

    # Replacements for Test VPC 1
    replacements = REPLACEMENT_VALUES.copy()
    replacements["VPC_NAME"] = resource_name + "-1"
    replacements["CIDR_BLOCK"] = "10.0.0.0/16"
    replacements["ENABLE_DNS_SUPPORT"] = "False"
    replacements["ENABLE_DNS_HOSTNAMES"] = "False"
    
    # Load VPC CR
    vpc_1_resource_data = load_ec2_resource(
        "vpc",
        additional_replacements=replacements,
    )
    logging.debug(vpc_1_resource_data)

    # Create k8s resource
    vpc_1_ref = k8s.CustomResourceReference(
        CRD_GROUP, CRD_VERSION, VPC_RESOURCE_PLURAL,
        replacements["VPC_NAME"], namespace="default",
    )
    k8s.create_custom_resource(vpc_1_ref, vpc_1_resource_data)
    time.sleep(CREATE_WAIT_AFTER_SECONDS)

    vpc_1_cr = k8s.wait_resource_consumed_by_controller(vpc_1_ref)
    assert vpc_1_cr is not None
    assert k8s.get_resource_exists(vpc_1_ref)

    # Replacements for Test VPC 2
    replacements = REPLACEMENT_VALUES.copy()
    replacements["VPC_NAME"] = resource_name + "-2"
    replacements["CIDR_BLOCK"] = "10.1.0.0/16"
    replacements["ENABLE_DNS_SUPPORT"] = "False"
    replacements["ENABLE_DNS_HOSTNAMES"] = "False"
    
    # Load VPC CR
    vpc_2_resource_data = load_ec2_resource(
        "vpc",
        additional_replacements=replacements,
    )
    logging.debug(vpc_2_resource_data)

    # Create k8s resource
    vpc_2_ref = k8s.CustomResourceReference(
        CRD_GROUP, CRD_VERSION, VPC_RESOURCE_PLURAL,
        replacements["VPC_NAME"], namespace="default",
    )
    k8s.create_custom_resource(vpc_2_ref, vpc_2_resource_data)
    time.sleep(CREATE_WAIT_AFTER_SECONDS)

    vpc_2_cr = k8s.wait_resource_consumed_by_controller(vpc_2_ref)
    assert vpc_2_cr is not None
    assert k8s.get_resource_exists(vpc_2_ref)

    # Create the VPC Peering Connection

    # Replacements for VPC Peering Connection
    replacements["VPC_PEERING_CONNECTION_NAME"] = resource_name
    replacements["VPC_NAME_1"] = resource_name + "-1"
    replacements["VPC_NAME_2"] = resource_name + "-2"

    # Load VPCPeeringConnection CR
    resource_data = load_ec2_resource(
        "vpc_peering_connection_ref",
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
    # Can't uncomment this line until ACK VPCs support auto-accepting VPC Peering Requests
    # assert cr["status"]["code"] == "active" 

    yield (ref, cr)

    # Delete VPC Peering Connection k8s resource 
    _, deleted = k8s.delete_custom_resource(ref, 3, 10)
    assert deleted is True

    time.sleep(DELETE_WAIT_AFTER_SECONDS)

    # Delete 2 x VPC resources 
    _, vpc_1_deleted = k8s.delete_custom_resource(vpc_1_ref, 3, 10)
    _, vpc_2_deleted = k8s.delete_custom_resource(vpc_2_ref, 3, 10)
    assert vpc_1_deleted is True
    assert vpc_2_deleted is True

@service_marker
@pytest.mark.canary
class TestVPCPeeringConnections:
    def test_create_delete_ref(self, ec2_client, ref_vpc_peering_connection):
        (ref, cr) = ref_vpc_peering_connection
        vpc_peering_connection_id = cr["vpcPeeringConnectionID"]

        # Check VPC Peering Connection exists
        exists = vpc_peering_connection_exists(ec2_client, vpc_peering_connection_id)
        assert exists

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref, 2, 5)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check VPC Peering Connection doesn't exist
        exists = vpc_peering_connection_exists(ec2_client, vpc_peering_connection_id)
        assert not exists

    def test_create_delete(self, ec2_client, simple_vpc_peering_connection):
        (ref, cr) = simple_vpc_peering_connection
        vpc_peering_connection_id = cr["status"]["vpcPeeringConnectionID"]

        # Check VPC Peering Connection exists
        exists = vpc_peering_connection_exists(ec2_client, vpc_peering_connection_id)
        assert exists

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref, 2, 5)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check VPC Peering Connection doesn't exist
        exists = vpc_peering_connection_exists(ec2_client, vpc_peering_connection_id)
        assert not exists

    def test_crud_tags(self, ec2_client, simple_vpc_peering_connection):
        (ref, cr) = simple_vpc_peering_connection

        resource = k8s.get_resource(ref)
        resource_id = cr["vpcPeeringConnectionID"]

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # Check VPC Peering Connection exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_vpc_peering_connection(resource_id)

        # Check system and user tags exist for VPC Peering Connection resource
        vpc_peering_connection = ec2_validator.get_vpc_peering_connection(resource_id)
        user_tags = {
            "initialtagkey": "initialtagvalue"
        }
        tags.assert_ack_system_tags(
            tags=vpc_peering_connection["Tags"],
        )
        tags.assert_equal_without_ack_tags(
            expected=user_tags,
            actual=vpc_peering_connection["Tags"],
        )
        
        # Update tags
        update_tags = [
            {
                "key": "updatedtagkey",
                "value": "updatedtagvalue",
            }
        ]

        # Patch the VPCPeeringConnection, updating the tags with a new pair
        updates = {
            "spec": {"tags": update_tags},
        }

        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # Check resource synced successfully
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)

        # Check for updated user tags; system tags should persist
        vpc_peering_connection = ec2_validator.get_vpc_peering_connection(resource_id)
        updated_tags = {
            "updatedtagkey": "updatedtagvalue"
        }
        tags.assert_ack_system_tags(
            tags=vpc_peering_connection["Tags"],
        )
        tags.assert_equal_without_ack_tags(
            expected=updated_tags,
            actual=vpc_peering_connection["Tags"],
        )

        # Patch the VPCPeeringConnection resource, deleting the tags
        updates = {
            "spec": {"tags": []},
        }

        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # Check resource synced successfully
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)

        # Check for removed user tags; system tags should persist
        vpc_peering_connection = ec2_validator.get_vpc_peering_connection(resource_id)
        tags.assert_ack_system_tags(
            tags=vpc_peering_connection["Tags"],
        )
        tags.assert_equal_without_ack_tags(
            expected=[],
            actual=vpc_peering_connection["Tags"],
        )

        # Check user tags are removed from Spec
        resource = k8s.get_resource(ref)
        assert len(resource["spec"]["tags"]) == 0

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check VPC Peering Connection no longer exists in AWS
        ec2_validator.assert_vpc_peering_connection(resource_id, exists=False)


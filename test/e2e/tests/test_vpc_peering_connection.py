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
    replacements["TAG_KEY"] = "initialtagkey"
    replacements["TAG_VALUE"] = "initialtagvalue"
    
    marker = request.node.get_closest_marker("resource_data")
    if marker is not None:
        data = marker.args[0]
        if 'tag_key' in data:
            replacements["TAG_KEY"] = data['tag_key']
        if 'tag_value' in data:
            replacements["TAG_VALUE"] = data['tag_value']

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
    wait_for_vpc_peering_connection_status(ref)
    assert cr["status"]["status"]["code"] == "active" 

    yield (ref, cr)

    # Delete VPC Peering Connection k8s resource 
    try:
        _, deleted = k8s.delete_custom_resource(ref, 3, 10)
        assert deleted
    except:
        pass

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
    replacements["TAG_KEY"] = "initialtagkey"
    replacements["TAG_VALUE"] = "initialtagvalue"
    
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

    # Replacements for Test VPC 2 (squashes previous values used by VPC 1)
    replacements["VPC_NAME"] = resource_name + "-2"
    replacements["CIDR_BLOCK"] = "10.1.0.0/16"
    
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
    replacements["VPC_REF_NAME"] = resource_name + "-1"
    replacements["PEER_VPC_REF_NAME"] = resource_name + "-2"

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
    wait_for_vpc_peering_connection_status(ref)
    assert cr["status"]["status"]["code"] == "active" 

    yield (ref, cr)

    # Delete VPC Peering Connection k8s resource 
    try:
        _, deleted = k8s.delete_custom_resource(ref, 3, 10)
        assert deleted
    except:
        pass

    time.sleep(DELETE_WAIT_AFTER_SECONDS)

    # Delete 2 x VPC resources
    try: 
        _, vpc_1_deleted = k8s.delete_custom_resource(vpc_1_ref, 3, 10)
        _, vpc_2_deleted = k8s.delete_custom_resource(vpc_2_ref, 3, 10)
        assert vpc_1_deleted is True
        assert vpc_2_deleted is True
    except:
        pass

def wait_for_vpc_peering_connection_status(ref, timeout_seconds=120):
    start_time = time.time()
    while time.time() - start_time < timeout_seconds:
        resource = k8s.wait_resource_consumed_by_controller(ref)
        if resource["status"]["status"]["code"] == "active":
            logging.debug("VPC Peering Connection Status Code is 'active'", resource)
            return
        time.sleep(5)
    print("CR contents", resource)
    raise TimeoutError(f"Timed out waiting for VPC Peering Connection status to become 'active'", "Current status code", resource["status"]["status"]["code"])

@service_marker
@pytest.mark.canary
class TestVPCPeeringConnections:
    def test_create_delete_ref(self, ec2_client, ref_vpc_peering_connection):
        (ref, cr) = ref_vpc_peering_connection
        vpc_peering_connection_id = cr["status"]["vpcPeeringConnectionID"]

        # Check VPC Peering Connection exists
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_vpc_peering_connection(vpc_peering_connection_id)

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref, 2, 5)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check VPC Peering Connection no longer exists in AWS
        ec2_validator.assert_vpc_peering_connection(vpc_peering_connection_id, exists=False)

    # def test_create_delete(self, ec2_client, simple_vpc_peering_connection):
    #     (ref, cr) = simple_vpc_peering_connection
    #     vpc_peering_connection_id = cr["status"]["vpcPeeringConnectionID"]

    #     # Check VPC Peering Connection exists
    #     ec2_validator = EC2Validator(ec2_client)
    #     ec2_validator.assert_vpc_peering_connection(vpc_peering_connection_id)

    #     # Delete k8s resource
    #     _, deleted = k8s.delete_custom_resource(ref, 2, 5)
    #     assert deleted is True

    #     time.sleep(DELETE_WAIT_AFTER_SECONDS)

    #     # Check VPC Peering Connection no longer exists in AWS
    #     ec2_validator.assert_vpc_peering_connection(vpc_peering_connection_id, exists=False)

    # def test_crud_tags(self, ec2_client, simple_vpc_peering_connection):
    #     (ref, cr) = simple_vpc_peering_connection

    #     resource = k8s.get_resource(ref)
    #     resource_id = cr["status"]["vpcPeeringConnectionID"]

    #     time.sleep(CREATE_WAIT_AFTER_SECONDS)

    #     # Check VPC Peering Connection exists in AWS
    #     ec2_validator = EC2Validator(ec2_client)
    #     ec2_validator.assert_vpc_peering_connection(resource_id)

    #     # Check system and user tags exist for VPC Peering Connection resource
    #     vpc_peering_connection = ec2_validator.get_vpc_peering_connection(resource_id)
    #     user_tags = {
    #         "initialtagkey": "initialtagvalue"
    #     }
    #     tags.assert_ack_system_tags(
    #        tags=vpc_peering_connection["Tags"],
    #     )
    #     tags.assert_equal_without_ack_tags(
    #         expected=user_tags,
    #         actual=vpc_peering_connection["Tags"],
    #     )
        
    #     # Update tags
    #     update_tags = [
    #         {
    #             "key": "updatedtagkey",
    #             "value": "updatedtagvalue",
    #         }
    #     ]

    #     # Patch the VPCPeeringConnection, updating the tags with a new pair
    #     updates = {
    #         "spec": {"tags": update_tags},
    #     }

    #     k8s.patch_custom_resource(ref, updates)
    #     time.sleep(MODIFY_WAIT_AFTER_SECONDS)

    #     # Check resource synced successfully
    #     assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)

    #     # Check for updated user tags; system tags should persist
    #     vpc_peering_connection = ec2_validator.get_vpc_peering_connection(resource_id)
    #     updated_tags = {
    #         "updatedtagkey": "updatedtagvalue"
    #     }
    #     tags.assert_ack_system_tags(
    #         tags=vpc_peering_connection["Tags"],
    #     )
    #     tags.assert_equal_without_ack_tags(
    #         expected=updated_tags,
    #         actual=vpc_peering_connection["Tags"],
    #     )

    #     # Patch the VPCPeeringConnection resource, deleting the tags
    #     updates = {
    #         "spec": {"tags": []},
    #     }

    #     k8s.patch_custom_resource(ref, updates)
    #     time.sleep(MODIFY_WAIT_AFTER_SECONDS)

    #     # Check resource synced successfully
    #     assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)

    #     # Check for removed user tags; system tags should persist
    #     vpc_peering_connection = ec2_validator.get_vpc_peering_connection(resource_id)
    #     tags.assert_ack_system_tags(
    #         tags=vpc_peering_connection["Tags"],
    #     )
    #     tags.assert_equal_without_ack_tags(
    #         expected=[],
    #         actual=vpc_peering_connection["Tags"],
    #     )

    #     # Check user tags are removed from Spec
    #     resource = k8s.get_resource(ref)
    #     assert len(resource["spec"]["tags"]) == 0

    #     # Delete k8s resource
    #     try:
    #         _, deleted = k8s.delete_custom_resource(ref, 3, 10)
    #         assert deleted
    #     except:
    #         pass
    #     time.sleep(DELETE_WAIT_AFTER_SECONDS)

    #     # Check VPC Peering Connection no longer exists in AWS
    #     ec2_validator.assert_vpc_peering_connection(resource_id, exists=False)


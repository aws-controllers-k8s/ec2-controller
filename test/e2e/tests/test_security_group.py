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

"""Integration tests for the SecurityGroup API.
"""

import logging
import resource
import time

import pytest
from acktest import tags
from acktest.k8s import resource as k8s
from acktest.resources import random_suffix_name
from e2e import CRD_GROUP, CRD_VERSION, load_ec2_resource, service_marker
from e2e.bootstrap_resources import get_bootstrap_resources
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e.tests.helper import EC2Validator
from acktest.aws.identity import get_account_id

RESOURCE_PLURAL = "securitygroups"

CREATE_WAIT_AFTER_SECONDS = 10
DELETE_WAIT_AFTER_SECONDS = 10
MODIFY_WAIT_AFTER_SECONDS = 5

CREATE_CYCLIC_REF_AFTER_SECONDS = 60
DELETE_CYCLIC_REF_AFTER_SECONDS = 30


@pytest.fixture
def simple_security_group(request):
    resource_name = random_suffix_name("security-group-test", 24)
    resource_file = "security_group"
    test_vpc = get_bootstrap_resources().SharedTestVPC

    replacements = REPLACEMENT_VALUES.copy()
    replacements["SECURITY_GROUP_NAME"] = resource_name
    replacements["VPC_ID"] = test_vpc.vpc_id
    replacements["SECURITY_GROUP_DESCRIPTION"] = "TestSecurityGroup"

    marker = request.node.get_closest_marker("resource_data")
    if marker is not None:
        data = marker.args[0]
        if "resource_file" in data:
            resource_file = data["resource_file"]
            replacements.update(data)
        if "tag_key" in data:
            replacements["TAG_KEY"] = data["tag_key"]
        if "tag_value" in data:
            replacements["TAG_VALUE"] = data["tag_value"]

    # Load Security Group CR
    resource_data = load_ec2_resource(
        resource_file,
        additional_replacements=replacements,
    )
    logging.debug(resource_data)

    # Create k8s resource
    ref = k8s.CustomResourceReference(
        CRD_GROUP,
        CRD_VERSION,
        RESOURCE_PLURAL,
        resource_name,
        namespace="default",
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
def security_group_with_vpc(request, simple_vpc):
    (_, vpc_cr) = simple_vpc
    vpc_id = vpc_cr["status"]["vpcID"]

    assert vpc_id is not None

    resource_name = random_suffix_name("security-group-vpc", 24)
    resource_file = "security_group"

    replacements = REPLACEMENT_VALUES.copy()
    replacements["SECURITY_GROUP_NAME"] = resource_name
    replacements["VPC_ID"] = vpc_id
    replacements["SECURITY_GROUP_DESCRIPTION"] = "TestSecurityGroup"

    marker = request.node.get_closest_marker("resource_data")
    if marker is not None:
        data = marker.args[0]
        if "resource_file" in data:
            resource_file = data["resource_file"]
            replacements.update(data)
        if "tag_key" in data:
            replacements["TAG_KEY"] = data["tag_key"]
        if "tag_value" in data:
            replacements["TAG_VALUE"] = data["tag_value"]

    # Load Security Group CR
    resource_data = load_ec2_resource(
        resource_file,
        additional_replacements=replacements,
    )
    logging.debug(resource_data)

    # Create k8s resource
    ref = k8s.CustomResourceReference(
        CRD_GROUP,
        CRD_VERSION,
        RESOURCE_PLURAL,
        resource_name,
        namespace="default",
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


def create_security_group_with_sg_ref(resource_name, reference_name):
    replacements = REPLACEMENT_VALUES.copy()
    replacements["VPC_ID"] = get_bootstrap_resources().SharedTestVPC.vpc_id
    replacements["SECURITY_GROUP_NAME"] = resource_name
    replacements["SECURITY_GROUP_REF_NAME"] = reference_name

    # Load Security Group CR
    resource_data = load_ec2_resource(
        "security_group_with_sg_ref",
        additional_replacements=replacements,
    )
    logging.debug(resource_data)

    # Create k8s resource
    ref = k8s.CustomResourceReference(
        CRD_GROUP,
        CRD_VERSION,
        RESOURCE_PLURAL,
        resource_name,
        namespace="default",
    )
    k8s.create_custom_resource(ref, resource_data)

    return ref


def create_security_group_self_owner_userid(resource_name):
    """Create an SG with an omitted-group self-ref pair that also carries the
    redundant owner account in userID (exercises the field-drop path in
    canonicalizeGroupPair)."""
    replacements = REPLACEMENT_VALUES.copy()
    replacements["VPC_ID"] = get_bootstrap_resources().SharedTestVPC.vpc_id
    replacements["SECURITY_GROUP_NAME"] = resource_name
    replacements["USER_ID"] = str(get_account_id())

    resource_data = load_ec2_resource(
        "security_group_self_owner_userid",
        additional_replacements=replacements,
    )
    logging.debug(resource_data)

    ref = k8s.CustomResourceReference(
        CRD_GROUP,
        CRD_VERSION,
        RESOURCE_PLURAL,
        resource_name,
        namespace="default",
    )
    k8s.create_custom_resource(ref, resource_data)

    return ref


@pytest.fixture
def security_groups_cyclic_ref():
    resource_name_1 = random_suffix_name("security-group-test", 24)
    resource_name_2 = random_suffix_name("security-group-test", 24)
    resource_name_3 = random_suffix_name("security-group-test", 24)

    ref_1 = create_security_group_with_sg_ref(resource_name_1, resource_name_2)
    ref_2 = create_security_group_with_sg_ref(resource_name_2, resource_name_3)
    ref_3 = create_security_group_with_sg_ref(resource_name_3, resource_name_1)

    time.sleep(CREATE_CYCLIC_REF_AFTER_SECONDS)

    cr_1 = k8s.wait_resource_consumed_by_controller(ref_1)
    cr_2 = k8s.wait_resource_consumed_by_controller(ref_2)
    cr_3 = k8s.wait_resource_consumed_by_controller(ref_3)
    assert cr_1 is not None
    assert cr_2 is not None
    assert cr_3 is not None

    yield [(ref_1, cr_1), (ref_2, cr_2), (ref_3, cr_3)]

    try:
        k8s.delete_custom_resource(ref, 3, 10)
        k8s.delete_custom_resource(ref, 3, 10)
        k8s.delete_custom_resource(ref, 3, 10)

        time.sleep(DELETE_CYCLIC_REF_AFTER_SECONDS)

        assert not k8s.get_resource_exists(ref_1)
        assert not k8s.get_resource_exists(ref_2)
        assert not k8s.get_resource_exists(ref_3)
    except:
        pass


def _sg_status_rule_ids(ref):
    """Return the sorted securityGroupRuleIDs currently in the CR's status."""
    latest = k8s.get_resource(ref)
    rules = latest.get("status", {}).get("rules", []) or []
    ids = sorted(r.get("securityGroupRuleID") for r in rules if r.get("securityGroupRuleID"))
    assert ids, f"no securityGroupRuleIDs populated in status: {rules}"
    return ids


def _assert_no_perpetual_diff(ref):
    """Force a reconcile unrelated to the rules and assert the security group
    rule IDs in status do not change.

    A tag edit bumps metadata.generation, forcing an immediate reconcile that
    recomputes the rule delta. A spurious diff revokes and re-authorizes the
    rules, so AWS assigns new securityGroupRuleIDs. Stable IDs are the definitive
    signal of no rule churn -- ACK.ResourceSynced=True is not, since the #2822
    bug churned rules while still reporting Synced=True.
    """
    ids_before = _sg_status_rule_ids(ref)

    # Unrelated update: add a tag. This does not touch any ingress/egress rule.
    k8s.patch_custom_resource(
        ref, {"spec": {"tags": [{"key": "force-reconcile", "value": "1"}]}}
    )
    time.sleep(MODIFY_WAIT_AFTER_SECONDS)
    assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)

    ids_after = _sg_status_rule_ids(ref)
    assert ids_after == ids_before, (
        f"security group rules churned after an unrelated (tag) update: "
        f"{ids_before} -> {ids_after}; a perpetual delta re-authorized the rules"
    )


def _assert_groupref_retained(ref, expected_ref_name):
    """Assert the user's groupRef survives in the persisted spec (ingress and
    egress) and that no canonical groupID leaked into it."""
    spec = k8s.get_resource(ref)["spec"]
    for rules_field in ("ingressRules", "egressRules"):
        for rule in spec.get(rules_field, []):
            for pair in rule.get("userIDGroupPairs", []):
                assert (
                    pair.get("groupRef", {}).get("from", {}).get("name")
                    == expected_ref_name
                ), f"{rules_field}: user groupRef must survive in the spec, got: {pair}"
                assert "groupID" not in pair, (
                    f"{rules_field}: canonical groupID must not leak into the "
                    f"spec, got: {pair}"
                )


@service_marker
@pytest.mark.canary
class TestSecurityGroup:
    def test_create_delete(self, ec2_client, simple_security_group):
        (ref, cr) = simple_security_group
        resource_id = cr["status"]["id"]

        # Check Security Group exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_security_group(resource_id)

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check Security Group no longer exists in AWS
        ec2_validator.assert_security_group(resource_id, exists=False)

    def test_create_with_vpc_add_egress_rule(self, ec2_client, security_group_with_vpc):
        (ref, cr) = security_group_with_vpc
        resource_id = cr["status"]["id"]

        # Check resource is synced successfully
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)

        # Check Security Group exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_security_group(resource_id)

        # Add a new Egress rule via patch
        new_egress_rule = {
            "ipProtocol": "-1",
            "ipRanges": [
                {
                    "cidrIP": "0.0.0.0/0",
                    "description": "Allow traffic from all IPs - test",
                }
            ],
        }
        patch = {"spec": {"egressRules": [new_egress_rule]}}
        _ = k8s.patch_custom_resource(ref, patch)

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # Check resource gets into synced state
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)

        # assert patched state
        cr = k8s.get_resource(ref)
        assert len(cr["status"]["rules"]) == 1

        # Check egress rule exists
        sg_group = ec2_validator.get_security_group(resource_id)
        assert len(sg_group["IpPermissions"]) == 0
        assert len(sg_group["IpPermissionsEgress"]) == 1

        # Check egress rule data
        assert sg_group["IpPermissionsEgress"][0]["IpProtocol"] == "-1"
        assert len(sg_group["IpPermissionsEgress"][0]["IpRanges"]) == 1
        ip_range = sg_group["IpPermissionsEgress"][0]["IpRanges"][0]
        assert ip_range["CidrIp"] == "0.0.0.0/0"
        assert ip_range["Description"] == "Allow traffic from all IPs - test"

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check Security Group no longer exists in AWS
        # Deleting Security Group will also delete rules
        ec2_validator.assert_security_group(resource_id, exists=False)

    @pytest.mark.resource_data(
        {
            "resource_file": "security_group_rule",
            "IP_PROTOCOL": "tcp",
            "FROM_PORT": "80",
            "TO_PORT": "80",
            "CIDR_IP": "172.31.0.0/16",
            "DESCRIPTION_INGRESS": "test ingress rule",
        }
    )
    def test_rules_create_update_delete(self, ec2_client, simple_security_group):
        (ref, cr) = simple_security_group
        resource_id = cr["status"]["id"]

        # Check resource is synced successfully
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)

        # Check Security Group exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_security_group(resource_id)

        # Hook code should update Spec rules using data from ReadOne resp
        assert len(cr["spec"]["ingressRules"]) == 1

        # Check ingress rule added
        assert len(cr["status"]["rules"]) == 1
        sg_group = ec2_validator.get_security_group(resource_id)
        assert len(sg_group["IpPermissions"]) == 1

        # Add Egress rule via patch
        new_egress_rule = {
            "ipProtocol": "tcp",
            "fromPort": 25,
            "toPort": 25,
            "ipRanges": [{"cidrIP": "172.31.0.0/16", "description": "test egress"}],
        }
        # Add Egress rule via patch
        new_egress_rule_with_sg_pair = {
            "ipProtocol": "tcp",
            "fromPort": 40,
            "toPort": 40,
            "ipRanges": [{"cidrIP": "172.31.0.0/12", "description": "test egress"}],
            "userIDGroupPairs": [
                {
                    "description": "test userIDGroupPairs",
                    "userID": str(get_account_id()),
                }
            ],
        }
        patch = {
            "spec": {"egressRules": [new_egress_rule, new_egress_rule_with_sg_pair]}
        }
        _ = k8s.patch_custom_resource(ref, patch)

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # Check resource gets into synced state
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)

        # Check ingress and egress rules exist
        sg_group = ec2_validator.get_security_group(resource_id)
        assert len(sg_group["IpPermissions"]) == 1
        assert len(sg_group["IpPermissionsEgress"]) == 2

        # Check egress rule data
        assert sg_group["IpPermissionsEgress"][0]["IpProtocol"] == "tcp"
        assert sg_group["IpPermissionsEgress"][0]["FromPort"] == 25
        assert sg_group["IpPermissionsEgress"][0]["ToPort"] == 25
        assert (
            sg_group["IpPermissionsEgress"][0]["IpRanges"][0]["Description"]
            == "test egress"
        )

        assert sg_group["IpPermissionsEgress"][1]["IpProtocol"] == "tcp"
        assert sg_group["IpPermissionsEgress"][1]["FromPort"] == 40
        assert sg_group["IpPermissionsEgress"][1]["ToPort"] == 40
        assert (
            sg_group["IpPermissionsEgress"][1]["IpRanges"][0]["Description"]
            == "test egress"
        )
        assert len(sg_group["IpPermissionsEgress"][1]["UserIdGroupPairs"]) == 1
        assert (
            sg_group["IpPermissionsEgress"][1]["UserIdGroupPairs"][0]["Description"]
            == "test userIDGroupPairs"
        )
        assert (
            sg_group["IpPermissionsEgress"][1]["UserIdGroupPairs"][0]["GroupId"]
            == resource_id
        )

        # Remove Ingress rule
        patch = {"spec": {"ingressRules": []}}
        _ = k8s.patch_custom_resource(ref, patch)
        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # assert patched state
        cr = k8s.get_resource(ref)
        assert len(cr["status"]["rules"]) == 3

        # Check ingress rule removed; egress rule remains
        sg_group = ec2_validator.get_security_group(resource_id)
        assert len(sg_group["IpPermissions"]) == 0
        assert len(sg_group["IpPermissionsEgress"]) == 2

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check Security Group no longer exists in AWS
        # Deleting Security Group will also delete rules
        ec2_validator.assert_security_group(resource_id, exists=False)

    @pytest.mark.resource_data(
        {"tag_key": "initialtagkey", "tag_value": "initialtagvalue"}
    )
    def test_crud_tags(self, ec2_client, simple_security_group):
        (ref, cr) = simple_security_group

        resource = k8s.get_resource(ref)
        resource_id = cr["status"]["id"]

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # Check SecurityGroup exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_security_group(resource_id)

        # Check system and user tags exist for security group resource
        security_group = ec2_validator.get_security_group(resource_id)
        user_tags = {"initialtagkey": "initialtagvalue"}
        tags.assert_ack_system_tags(
            tags=security_group["Tags"],
        )
        tags.assert_equal_without_ack_tags(
            expected=user_tags,
            actual=security_group["Tags"],
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

        # Patch the SecurityGroup, updating the tags with new pair
        updates = {
            "spec": {"tags": update_tags},
        }

        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # Check resource synced successfully
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)

        # Check for updated user tags; system tags should persist
        security_group = ec2_validator.get_security_group(resource_id)
        updated_tags = {"updatedtagkey": "updatedtagvalue"}
        tags.assert_ack_system_tags(
            tags=security_group["Tags"],
        )
        tags.assert_equal_without_ack_tags(
            expected=updated_tags,
            actual=security_group["Tags"],
        )

        # Only user tags should be present in Spec
        resource = k8s.get_resource(ref)
        assert len(resource["spec"]["tags"]) == 1
        assert resource["spec"]["tags"][0]["key"] == "updatedtagkey"
        assert resource["spec"]["tags"][0]["value"] == "updatedtagvalue"

        # Patch the SecurityGroup resource, deleting the tags
        updates = {
            "spec": {"tags": []},
        }

        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # Check resource synced successfully
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)

        # Check for removed user tags; system tags should persist
        security_group = ec2_validator.get_security_group(resource_id)
        tags.assert_ack_system_tags(
            tags=security_group["Tags"],
        )
        tags.assert_equal_without_ack_tags(
            expected=[],
            actual=security_group["Tags"],
        )

        # Check user tags are removed from Spec
        resource = k8s.get_resource(ref)
        assert len(resource["spec"]["tags"]) == 0

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check SecurityGroup no longer exists in AWS
        ec2_validator.assert_security_group(resource_id, exists=False)

    def test_cyclic_ref(self, ec2_client, security_groups_cyclic_ref):
        sgs = security_groups_cyclic_ref
        (ref_1, cr_1) = sgs[0]
        (ref_2, cr_2) = sgs[1]
        (ref_3, cr_3) = sgs[2]

        # Check Security Groups exists in AWS
        resource_id_1 = cr_1["status"]["id"]
        resource_id_2 = cr_2["status"]["id"]
        resource_id_3 = cr_3["status"]["id"]

        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_security_group(resource_id_1)
        ec2_validator.assert_security_group(resource_id_2)
        ec2_validator.assert_security_group(resource_id_3)

        # Check resources are synced successfully
        assert k8s.wait_on_condition(
            ref_1, "ACK.ResourceSynced", "True", wait_periods=5
        )
        assert k8s.wait_on_condition(
            ref_2, "ACK.ResourceSynced", "True", wait_periods=5
        )
        assert k8s.wait_on_condition(
            ref_3, "ACK.ResourceSynced", "True", wait_periods=5
        )

        sg_group_1 = ec2_validator.get_security_group(resource_id_1)
        sg_group_2 = ec2_validator.get_security_group(resource_id_2)
        sg_group_3 = ec2_validator.get_security_group(resource_id_3)

        # Check ingress rules exist
        assert len(sg_group_1["IpPermissions"]) == 1
        assert len(sg_group_2["IpPermissions"]) == 1
        assert len(sg_group_3["IpPermissions"]) == 1

        # Check egress rules exist
        assert len(sg_group_1["IpPermissionsEgress"]) == 1
        assert len(sg_group_2["IpPermissionsEgress"]) == 1
        assert len(sg_group_3["IpPermissionsEgress"]) == 1

        # Check ingress rules cyclic data
        assert (
            sg_group_1["IpPermissions"][0]["UserIdGroupPairs"][0]["GroupId"]
            == resource_id_2
        )
        assert (
            sg_group_2["IpPermissions"][0]["UserIdGroupPairs"][0]["GroupId"]
            == resource_id_3
        )
        assert (
            sg_group_3["IpPermissions"][0]["UserIdGroupPairs"][0]["GroupId"]
            == resource_id_1
        )

        # Check egress rules cyclic data
        assert (
            sg_group_1["IpPermissionsEgress"][0]["UserIdGroupPairs"][0]["GroupId"]
            == resource_id_2
        )
        assert (
            sg_group_2["IpPermissionsEgress"][0]["UserIdGroupPairs"][0]["GroupId"]
            == resource_id_3
        )
        assert (
            sg_group_3["IpPermissionsEgress"][0]["UserIdGroupPairs"][0]["GroupId"]
            == resource_id_1
        )

        # Delete k8s resources
        k8s.delete_custom_resource(ref_1)
        k8s.delete_custom_resource(ref_2)
        k8s.delete_custom_resource(ref_3)

        time.sleep(DELETE_CYCLIC_REF_AFTER_SECONDS)

        assert not k8s.get_resource_exists(ref_1)
        assert not k8s.get_resource_exists(ref_2)
        assert not k8s.get_resource_exists(ref_3)

        # Check Security Group no longer exists in AWS
        ec2_validator.assert_security_group(resource_id_1, exists=False)
        ec2_validator.assert_security_group(resource_id_2, exists=False)
        ec2_validator.assert_security_group(resource_id_3, exists=False)

    @pytest.mark.resource_data({"resource_file": "security_group_self_ref"})
    def test_self_ref_rule_no_perpetual_diff(self, ec2_client, simple_security_group):
        # 1. Create the SecurityGroup (self-reference expressed by omitting
        #    groupID entirely -- the #2822 pattern).
        (ref, cr) = simple_security_group
        resource_id = cr["status"]["id"]

        # 2. Wait for the resource to sync.
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=8)

        # 3. Verify via the AWS API that the security group looks correct: the
        #    self-referencing ingress rule exists with GroupId auto-filled to
        #    the SG's own ID.
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_security_group(resource_id)
        sg_group = ec2_validator.get_security_group(resource_id)
        assert len(sg_group["IpPermissions"]) == 1
        rule = sg_group["IpPermissions"][0]
        assert rule["IpProtocol"] == "tcp"
        assert rule["FromPort"] == 443
        assert rule["ToPort"] == 443
        assert len(rule["UserIdGroupPairs"]) == 1
        assert rule["UserIdGroupPairs"][0]["GroupId"] == resource_id

        # 4 + 5. Force a reconcile with an unrelated (tag) update and confirm
        #        the self-ref rule ID in status does not change.
        _assert_no_perpetual_diff(ref)

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True
        time.sleep(DELETE_WAIT_AFTER_SECONDS)
        ec2_validator.assert_security_group(resource_id, exists=False)

    def test_self_ref_groupref_no_perpetual_diff(self, ec2_client):
        # Self-reference via a groupRef pointing at the SG itself -- a distinct
        # path from the omitted-groupID form: GroupID resolves from GroupRef only
        # once the SG's own ID is known (second reconcile).
        # NB: the name doubles as the EC2 GroupName, which AWS rejects in the
        # "sg-*" format, so it must not start with "sg-".
        name = random_suffix_name("selfref-groupref", 24)
        ref = create_security_group_with_sg_ref(name, name)  # references itself
        try:
            time.sleep(CREATE_CYCLIC_REF_AFTER_SECONDS)
            assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=8)

            cr = k8s.get_resource(ref)
            resource_id = cr["status"]["id"]
            ec2_validator = EC2Validator(ec2_client)
            ec2_validator.assert_security_group(resource_id)
            sg = ec2_validator.get_security_group(resource_id)

            # Both the ingress and egress tcp/443 self-ref rules resolve to the
            # SG's own ID (also exercises egress self-reference).
            for perms in (sg["IpPermissions"], sg["IpPermissionsEgress"]):
                match = [p for p in perms if p.get("FromPort") == 443]
                assert len(match) == 1, f"expected one tcp/443 rule, got {perms}"
                pairs = match[0]["UserIdGroupPairs"]
                assert len(pairs) == 1
                assert pairs[0]["GroupId"] == resource_id

            # The delta canonicalizes copies (clearing GroupRef, stamping
            # groupID); that must never leak into the persisted spec. Verify the
            # user's groupRef survives -- both after the initial sync and after a
            # forced update-path reconcile -- with no canonical groupID injected.
            _assert_groupref_retained(ref, name)
            _assert_no_perpetual_diff(ref)
            _assert_groupref_retained(ref, name)
        finally:
            k8s.delete_custom_resource(ref)
            time.sleep(DELETE_WAIT_AFTER_SECONDS)

    def test_cross_sg_userid_no_perpetual_diff(self, ec2_client, simple_security_group):
        # A rule referencing a *different* (peer) SG in the same account: the
        # user omits userID, AWS auto-fills the owner account. Clearing that
        # owner userID while keeping the peer GroupID avoids a perpetual diff.
        (peer_ref, _) = simple_security_group
        assert k8s.wait_on_condition(peer_ref, "ACK.ResourceSynced", "True", wait_periods=8)
        peer_id = k8s.get_resource(peer_ref)["status"]["id"]

        # See note above: the name doubles as the EC2 GroupName, which cannot
        # be in the "sg-*" format.
        name = random_suffix_name("cross-ref", 24)
        ref = create_security_group_with_sg_ref(name, peer_ref.name)  # references the peer
        try:
            time.sleep(CREATE_CYCLIC_REF_AFTER_SECONDS)
            assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=8)

            cr = k8s.get_resource(ref)
            resource_id = cr["status"]["id"]
            ec2_validator = EC2Validator(ec2_client)
            sg = ec2_validator.get_security_group(resource_id)

            ing = [p for p in sg["IpPermissions"] if p.get("FromPort") == 443]
            assert len(ing) == 1
            pairs = ing[0]["UserIdGroupPairs"]
            assert len(pairs) == 1
            assert pairs[0]["GroupId"] == peer_id, "cross-SG GroupID must be preserved"
            # AWS auto-fills the owner account id even though the spec omits it.
            assert pairs[0]["UserId"] == str(get_account_id())

            _assert_no_perpetual_diff(ref)
            # The peer groupRef must survive the forced reconcile intact (not
            # rewritten to a concrete groupID).
            _assert_groupref_retained(ref, peer_ref.name)
        finally:
            k8s.delete_custom_resource(ref)
            time.sleep(DELETE_WAIT_AFTER_SECONDS)

    @pytest.mark.resource_data({"resource_file": "security_group_allproto"})
    def test_all_protocol_ports_no_perpetual_diff(self, ec2_client, simple_security_group):
        # Isolated: AWS drops the port range for an all-protocols ("-1") rule.
        (ref, cr) = simple_security_group
        resource_id = cr["status"]["id"]
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=8)

        ec2_validator = EC2Validator(ec2_client)
        sg = ec2_validator.get_security_group(resource_id)
        egress = [e for e in sg["IpPermissionsEgress"] if e["IpProtocol"] == "-1"]
        assert len(egress) == 1
        assert "FromPort" not in egress[0]
        assert "ToPort" not in egress[0]

        _assert_no_perpetual_diff(ref)

        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True
        time.sleep(DELETE_WAIT_AFTER_SECONDS)
        ec2_validator.assert_security_group(resource_id, exists=False)

    @pytest.mark.resource_data({"resource_file": "security_group_nonstandard_proto"})
    def test_nonstandard_protocol_ports_no_perpetual_diff(self, ec2_client, simple_security_group):
        # Isolated: a protocol outside {tcp, udp, icmp, icmpv6} -- here ESP (IP
        # protocol 50) -- has its port range dropped by AWS on read-back, even
        # though the spec carries the -1/-1 sentinel. This is distinct from the
        # "-1" all-protocols drop: it covers every other IP protocol number.
        # Canonicalization must drop the spec ports to match, or the rule churns
        # every reconcile.
        (ref, cr) = simple_security_group
        resource_id = cr["status"]["id"]
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=8)

        ec2_validator = EC2Validator(ec2_client)
        sg = ec2_validator.get_security_group(resource_id)
        # Protocol 50 has no well-known name, so AWS returns it as the number.
        esp = [e for e in sg["IpPermissionsEgress"] if e.get("IpProtocol") == "50"]
        assert len(esp) == 1, f"expected one ESP egress rule, got {sg['IpPermissionsEgress']}"
        # The backend contract this fix depends on: ports are omitted for
        # protocols outside {tcp, udp, icmp, icmpv6}.
        assert "FromPort" not in esp[0]
        assert "ToPort" not in esp[0]

        _assert_no_perpetual_diff(ref)

        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True
        time.sleep(DELETE_WAIT_AFTER_SECONDS)
        ec2_validator.assert_security_group(resource_id, exists=False)

    @pytest.mark.resource_data({"resource_file": "security_group_icmpv6_no_typecode"})
    def test_icmpv6_no_typecode_no_perpetual_diff(self, ec2_client, simple_security_group):
        # Isolated: icmpv6 does not require a type/code (it is excluded from the
        # backend's must-receive-type/code set), so the spec may omit
        # fromPort/toPort. The EC2 backend treats protocol 58 with no type/code
        # as the "all types and codes" wildcard and returns it as
        # FromPort=-1/ToPort=-1 on read-back -- a path distinct from the "-1"
        # all-protocols port drop, since ICMP/ICMPv6 overload the ports as
        # type/code.
        (ref, cr) = simple_security_group
        resource_id = cr["status"]["id"]
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=8)

        ec2_validator = EC2Validator(ec2_client)
        sg = ec2_validator.get_security_group(resource_id)
        # AWS may report protocol 58 as the name "icmpv6" or the number "58".
        icmpv6 = [p for p in sg["IpPermissions"] if p.get("IpProtocol") in ("icmpv6", "58")]
        assert len(icmpv6) == 1, f"expected one icmpv6 rule, got {sg['IpPermissions']}"
        # Document the backend contract this fix depends on: the omitted
        # type/code reads back as the -1/-1 wildcard. Canonicalisation must
        # collapse it to the omitted spec form, or the rule churns every
        # reconcile.
        assert icmpv6[0].get("FromPort") == -1
        assert icmpv6[0].get("ToPort") == -1

        _assert_no_perpetual_diff(ref)

        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True
        time.sleep(DELETE_WAIT_AFTER_SECONDS)
        ec2_validator.assert_security_group(resource_id, exists=False)

    @pytest.mark.resource_data({"resource_file": "security_group_protocol_notation"})
    def test_protocol_notation_no_perpetual_diff(self, ec2_client, simple_security_group):
        # Isolated: numeric protocol "6" is stored/returned by AWS as "tcp".
        (ref, cr) = simple_security_group
        resource_id = cr["status"]["id"]
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=8)

        ec2_validator = EC2Validator(ec2_client)
        sg = ec2_validator.get_security_group(resource_id)
        rule = [p for p in sg["IpPermissions"] if p.get("FromPort") == 8006]
        assert len(rule) == 1, f"expected one tcp/8006 rule, got {sg['IpPermissions']}"
        assert rule[0]["IpProtocol"] == "tcp"

        _assert_no_perpetual_diff(ref)

        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True
        time.sleep(DELETE_WAIT_AFTER_SECONDS)
        ec2_validator.assert_security_group(resource_id, exists=False)

    @pytest.mark.resource_data({"resource_file": "security_group_cidr_canon"})
    def test_cidr_canonicalization_no_perpetual_diff(self, ec2_client, simple_security_group):
        # Isolated: AWS canonicalizes CIDRs. IPv4 and IPv6 are on separate
        # ports so a regression in one cannot mask the other.
        (ref, cr) = simple_security_group
        resource_id = cr["status"]["id"]
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=8)

        ec2_validator = EC2Validator(ec2_client)
        sg = ec2_validator.get_security_group(resource_id)

        ipv4 = [p for p in sg["IpPermissions"] if p.get("FromPort") == 8010]
        assert len(ipv4) == 1
        assert ipv4[0]["IpRanges"][0]["CidrIp"] == "172.16.0.0/16"

        ipv6 = [p for p in sg["IpPermissions"] if p.get("FromPort") == 8011]
        assert len(ipv6) == 1
        assert ipv6[0]["Ipv6Ranges"][0]["CidrIpv6"] == "2001:db8:abcd:12::/64"

        _assert_no_perpetual_diff(ref)

        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True
        time.sleep(DELETE_WAIT_AFTER_SECONDS)
        ec2_validator.assert_security_group(resource_id, exists=False)

    @pytest.mark.resource_data({"resource_file": "security_group_aggregation"})
    def test_grant_aggregation_no_perpetual_diff(self, ec2_client, simple_security_group):
        # Isolated: two spec rules sharing (tcp, 8020, 8020) are returned by
        # AWS as a single aggregated IpPermission carrying both CIDRs.
        (ref, cr) = simple_security_group
        resource_id = cr["status"]["id"]
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=8)

        ec2_validator = EC2Validator(ec2_client)
        sg = ec2_validator.get_security_group(resource_id)
        agg = [p for p in sg["IpPermissions"] if p.get("FromPort") == 8020]
        assert len(agg) == 1, f"expected one aggregated permission, got {sg['IpPermissions']}"
        cidrs = sorted(r["CidrIp"] for r in agg[0]["IpRanges"])
        assert cidrs == ["10.2.0.0/16", "10.3.0.0/16"]

        _assert_no_perpetual_diff(ref)

        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True
        time.sleep(DELETE_WAIT_AFTER_SECONDS)
        ec2_validator.assert_security_group(resource_id, exists=False)

    @pytest.mark.resource_data({"resource_file": "security_group_multi_grant"})
    def test_multiple_grants_sorted_no_perpetual_diff(self, ec2_client, simple_security_group):
        # A single rule carrying several CIDRs must be ordered deterministically
        # so read-back order never drives a perpetual diff. (A nil grant element
        # can't occur end-to-end -- the schema rejects null entries and the
        # read-back setter always allocates -- so only the populated sort path
        # is reachable here.)
        (ref, cr) = simple_security_group
        resource_id = cr["status"]["id"]
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=8)

        ec2_validator = EC2Validator(ec2_client)
        sg = ec2_validator.get_security_group(resource_id)
        rule = [p for p in sg["IpPermissions"] if p.get("FromPort") == 9090]
        assert len(rule) == 1, f"expected one tcp/9090 rule, got {sg['IpPermissions']}"
        v4 = sorted(r["CidrIp"] for r in rule[0].get("IpRanges", []))
        assert v4 == ["10.10.0.0/16", "10.20.0.0/16", "10.30.0.0/16"]
        v6 = sorted(r["CidrIpv6"] for r in rule[0].get("Ipv6Ranges", []))
        assert v6 == ["2001:db8:1::/48", "2001:db8:2::/48"]

        # A forced reconcile must not churn despite the many grants in the rule.
        _assert_no_perpetual_diff(ref)

        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True
        time.sleep(DELETE_WAIT_AFTER_SECONDS)
        ec2_validator.assert_security_group(resource_id, exists=False)

    @pytest.mark.resource_data({"resource_file": "security_group_combined_canon"})
    def test_combined_canonicalizations_update_applied(self, ec2_client, simple_security_group):
        # Broad coverage: a single SecurityGroup that exercises every
        # canonicalization path at once -- self-reference (omitted groupID),
        # numeric protocol notation, IPv4 CIDR canonicalization, the icmpv6
        # -1/-1 wildcard, grant aggregation, all-protocol ("-1") egress, and a
        # non-standard (ESP/50) egress. It confirms the combined create has no
        # perpetual diff, then applies an update that touches several canonical
        # paths simultaneously and verifies the edits land while the untouched
        # rules survive intact. (Supersedes the narrower self-ref-only
        # port-change test; the self-ref port change is retained as one of the
        # paths.)
        #
        # NOTE: sync operates on the raw spec rules, so a non-canonically-spelled
        # untouched rule may be revoked and re-authorized (its ruleID churns)
        # during an update; it still ends up present and correct. The invariant
        # asserted here is "not lost", not "same ruleID". Steady-state no-churn
        # (no update pending) is covered by _assert_no_perpetual_diff.
        (ref, cr) = simple_security_group
        resource_id = cr["status"]["id"]
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=8)

        ec2_validator = EC2Validator(ec2_client)
        sg = ec2_validator.get_security_group(resource_id)

        # --- Each canonical form must land as AWS returns it ---
        # self-ref tcp/443 -> groupID auto-filled to self
        selfref = [p for p in sg["IpPermissions"] if p.get("FromPort") == 443]
        assert len(selfref) == 1, f"expected one tcp/443 rule, got {sg['IpPermissions']}"
        assert selfref[0]["IpProtocol"] == "tcp"
        assert selfref[0]["UserIdGroupPairs"][0]["GroupId"] == resource_id
        # numeric "6" -> "tcp"; CIDR 172.16.5.9/16 -> 172.16.0.0/16
        numeric = [p for p in sg["IpPermissions"] if p.get("FromPort") == 8006]
        assert len(numeric) == 1
        assert numeric[0]["IpProtocol"] == "tcp"
        assert numeric[0]["IpRanges"][0]["CidrIp"] == "172.16.0.0/16"
        # icmpv6 with no type/code -> -1/-1 wildcard
        icmpv6 = [p for p in sg["IpPermissions"] if p.get("IpProtocol") in ("icmpv6", "58")]
        assert len(icmpv6) == 1
        assert icmpv6[0].get("FromPort") == -1 and icmpv6[0].get("ToPort") == -1
        # grant aggregation -> one tcp/9090 rule carrying both CIDRs
        agg = [p for p in sg["IpPermissions"] if p.get("FromPort") == 9090]
        assert len(agg) == 1, f"expected one aggregated tcp/9090 rule, got {sg['IpPermissions']}"
        assert sorted(r["CidrIp"] for r in agg[0]["IpRanges"]) == ["10.10.0.0/16", "10.20.0.0/16"]
        # egress: ESP (50) returned without a port range
        esp = [e for e in sg["IpPermissionsEgress"] if e["IpProtocol"] == "50"]
        assert len(esp) == 1
        assert "FromPort" not in esp[0] and "ToPort" not in esp[0]

        # The full combined spec must not churn on a forced reconcile.
        _assert_no_perpetual_diff(ref)

        # --- Update touching multiple canonical paths at once ---
        # * self-ref port 443 -> 8443 (self-ref canon + a genuine change)
        # * numeric-proto rule CIDR 172.16.5.9/16 -> non-canonical 192.168.5.9/24
        #   (protocol-notation + CIDR canon; canonicalises to 192.168.5.0/24)
        # icmpv6 and the two aggregation grants are repeated unchanged; egress is
        # omitted from the merge patch so it stays put -- together verifying those
        # paths are not lost while other rules change.
        patch = {
            "spec": {
                "ingressRules": [
                    {
                        "fromPort": 8443,
                        "toPort": 8443,
                        "ipProtocol": "tcp",
                        "userIDGroupPairs": [{"description": "self-referencing rule"}],
                    },
                    {
                        "fromPort": 8006,
                        "toPort": 8006,
                        "ipProtocol": "6",
                        "ipRanges": [
                            {"cidrIP": "192.168.5.9/24", "description": "numeric proto + non-canonical cidr"}
                        ],
                    },
                    {
                        "ipProtocol": "icmpv6",
                        "ipv6Ranges": [{"cidrIPv6": "::/0", "description": "icmpv6 all types and codes"}],
                    },
                    {
                        "fromPort": 9090,
                        "toPort": 9090,
                        "ipProtocol": "tcp",
                        "ipRanges": [{"cidrIP": "10.10.0.0/16", "description": "aggregation A"}],
                    },
                    {
                        "fromPort": 9090,
                        "toPort": 9090,
                        "ipProtocol": "tcp",
                        "ipRanges": [{"cidrIP": "10.20.0.0/16", "description": "aggregation B"}],
                    },
                ]
            }
        }
        k8s.patch_custom_resource(ref, patch)
        time.sleep(CREATE_WAIT_AFTER_SECONDS)
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=8)

        sg = ec2_validator.get_security_group(resource_id)
        # self-ref moved 443 -> 8443
        assert not [
            p for p in sg["IpPermissions"] if p.get("FromPort") == 443
        ], "old tcp/443 self-ref rule should have been revoked"
        updated = [p for p in sg["IpPermissions"] if p.get("FromPort") == 8443]
        assert len(updated) == 1, f"port change to 8443 was not applied: {sg['IpPermissions']}"
        assert updated[0]["UserIdGroupPairs"][0]["GroupId"] == resource_id
        # numeric-proto rule CIDR changed and canonicalised: 192.168.5.9/24 -> 192.168.5.0/24
        numeric = [p for p in sg["IpPermissions"] if p.get("FromPort") == 8006]
        assert len(numeric) == 1
        assert numeric[0]["IpRanges"][0]["CidrIp"] == "192.168.5.0/24", (
            f"numeric-proto rule CIDR not updated/canonicalised: {numeric[0]['IpRanges']}"
        )
        # untouched paths still present and intact
        assert [
            p for p in sg["IpPermissions"] if p.get("IpProtocol") in ("icmpv6", "58")
        ], "icmpv6 rule should have survived the update"
        agg = [p for p in sg["IpPermissions"] if p.get("FromPort") == 9090]
        assert len(agg) == 1 and len(agg[0]["IpRanges"]) == 2, "aggregated tcp/9090 rule should be intact"
        assert [
            e for e in sg["IpPermissionsEgress"] if e["IpProtocol"] == "50"
        ], "ESP egress rule should have survived the update"

        # The post-update combined state must itself be free of perpetual diff.
        # Use a distinct tag value so the reconcile is genuinely re-triggered
        # (the pre-update _assert_no_perpetual_diff already set force-reconcile=1).
        ids_before = _sg_status_rule_ids(ref)
        k8s.patch_custom_resource(
            ref, {"spec": {"tags": [{"key": "force-reconcile", "value": "2"}]}}
        )
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)
        assert _sg_status_rule_ids(ref) == ids_before, (
            "combined security group rules churned after the update settled"
        )

        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True
        time.sleep(DELETE_WAIT_AFTER_SECONDS)
        ec2_validator.assert_security_group(resource_id, exists=False)

    def test_self_ref_owner_userid_no_data_loss(self, ec2_client):
        # An omitted-group self-ref that also carries the redundant owner
        # account in userID is classified as self, and canonicalizeGroupPair
        # drops that userID. This is lossless: {GroupId=self} and
        # {GroupId=self,UserId=owner} produce the identical rule (AWS auto-fills
        # the owner). A foreign userID is only dropped when paired with the SG's
        # own GroupID -- an input AWS rejects; a cross-account ref uses
        # GroupId=peer and is preserved (unit test
        # TestCustomPostCompare_CrossAccount_NotSuppressed).
        name = random_suffix_name("selfref-owner-uid", 24)
        ref = create_security_group_self_owner_userid(name)
        try:
            time.sleep(CREATE_WAIT_AFTER_SECONDS)
            assert k8s.wait_on_condition(
                ref, "ACK.ResourceSynced", "True", wait_periods=8
            )
            resource_id = k8s.get_resource(ref)["status"]["id"]

            ec2_validator = EC2Validator(ec2_client)
            sg = ec2_validator.get_security_group(resource_id)
            rule = [p for p in sg["IpPermissions"] if p.get("FromPort") == 443]
            assert len(rule) == 1, f"expected one tcp/443 rule, got {sg['IpPermissions']}"
            pairs = rule[0]["UserIdGroupPairs"]
            assert len(pairs) == 1
            # Correct self-reference; the owner account is present exactly as it
            # would be for a bare self-ref -- the dropped userID lost nothing.
            assert pairs[0]["GroupId"] == resource_id
            assert pairs[0]["UserId"] == str(get_account_id())

            # The dropped owner userID must not drive a perpetual diff.
            _assert_no_perpetual_diff(ref)
        finally:
            k8s.delete_custom_resource(ref)
            time.sleep(DELETE_WAIT_AFTER_SECONDS)

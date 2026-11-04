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

"""Integration tests for the KeyPair API.
"""

import pytest
import time

from acktest import tags
from acktest.resources import random_suffix_name
from acktest.k8s import resource as k8s
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_ec2_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e.tests.helper import EC2Validator

RESOURCE_PLURAL = "keypairs"

CREATE_WAIT_AFTER_SECONDS = 10
DELETE_WAIT_AFTER_SECONDS = 10
MODIFY_WAIT_AFTER_SECONDS = 5

# A throwaway ed25519 public key; the private half was discarded after
# generating it, so it cannot log in anywhere.
PUBLIC_KEY_MATERIAL = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIPfh4lRy/xJUbMAlfUO6qYyTTnKLDwducZGNA7VIgbz5 ack-e2e-keypair"

@pytest.fixture
def simple_key_pair(request):
    resource_name = random_suffix_name("key-pair-test", 24)

    replacements = REPLACEMENT_VALUES.copy()
    replacements["KEY_PAIR_NAME"] = resource_name
    replacements["PUBLIC_KEY_MATERIAL"] = PUBLIC_KEY_MATERIAL
    replacements["TAG_KEY"] = "initialtagkey"
    replacements["TAG_VALUE"] = "initialtagvalue"

    resource_data = load_ec2_resource(
        "key_pair",
        additional_replacements=replacements,
    )

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

    try:
        _, deleted = k8s.delete_custom_resource(ref, 3, 10)
        assert deleted
    except:
        pass

@service_marker
@pytest.mark.canary
class TestKeyPair:
    def test_create_delete(self, ec2_client, simple_key_pair):
        (ref, cr) = simple_key_pair

        key_name = cr["spec"]["keyName"]

        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)

        cr = k8s.get_resource(ref)
        assert "keyPairID" in cr["status"]
        assert "keyFingerprint" in cr["status"]

        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_key_pair(key_name)

        key_pair = ec2_validator.get_key_pair(key_name)
        assert key_pair["KeyPairId"] == cr["status"]["keyPairID"]
        assert key_pair["KeyType"] == "ed25519"

        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        ec2_validator.assert_key_pair(key_name, exists=False)

    def test_crud_tags(self, ec2_client, simple_key_pair):
        (ref, cr) = simple_key_pair

        key_name = cr["spec"]["keyName"]

        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)

        ec2_validator = EC2Validator(ec2_client)
        key_pair = ec2_validator.get_key_pair(key_name)
        assert key_pair is not None

        user_tags = {
            "initialtagkey": "initialtagvalue"
        }
        tags.assert_ack_system_tags(
            tags=key_pair["Tags"],
        )
        tags.assert_equal_without_ack_tags(
            expected=user_tags,
            actual=key_pair["Tags"],
        )

        resource = k8s.get_resource(ref)
        assert len(resource["spec"]["tags"]) == 1
        assert resource["spec"]["tags"][0]["key"] == "initialtagkey"
        assert resource["spec"]["tags"][0]["value"] == "initialtagvalue"

        updates = {
            "spec": {"tags": [{"key": "updatedtagkey", "value": "updatedtagvalue"}]},
        }
        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)

        key_pair = ec2_validator.get_key_pair(key_name)
        updated_tags = {
            "updatedtagkey": "updatedtagvalue"
        }
        tags.assert_ack_system_tags(
            tags=key_pair["Tags"],
        )
        tags.assert_equal_without_ack_tags(
            expected=updated_tags,
            actual=key_pair["Tags"],
        )

        resource = k8s.get_resource(ref)
        assert len(resource["spec"]["tags"]) == 1
        assert resource["spec"]["tags"][0]["key"] == "updatedtagkey"
        assert resource["spec"]["tags"][0]["value"] == "updatedtagvalue"

        updates = {
            "spec": {"tags": []},
        }
        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)

        key_pair = ec2_validator.get_key_pair(key_name)
        tags.assert_ack_system_tags(
            tags=key_pair["Tags"],
        )
        tags.assert_equal_without_ack_tags(
            expected={},
            actual=key_pair["Tags"],
        )

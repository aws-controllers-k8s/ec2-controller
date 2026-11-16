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

"""Integration tests for the KeyPair Adoption API.
"""

import pytest
import time
import logging

from acktest.resources import random_suffix_name
from acktest.k8s import resource as k8s
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_ec2_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e.tests.helper import EC2Validator
from e2e.tests.test_key_pair import PUBLIC_KEY_MATERIAL

KEY_PAIR_RESOURCE_PLURAL = "keypairs"

CREATE_WAIT_AFTER_SECONDS = 10
DELETE_WAIT_AFTER_SECONDS = 10

@pytest.fixture
def key_pair_adoption(request, ec2_client):
    replacements = REPLACEMENT_VALUES.copy()
    resource_name = random_suffix_name("key-pair-adoption", 32)

    # The key pair exists before the CR; DescribeKeyPairs never returns the
    # public key, so the adopted spec carries only the name.
    imported = ec2_client.import_key_pair(
        KeyName=resource_name,
        PublicKeyMaterial=PUBLIC_KEY_MATERIAL.encode(),
    )
    key_pair_id = imported["KeyPairId"]

    replacements["KEY_PAIR_ADOPTION_NAME"] = resource_name
    replacements["ADOPTION_POLICY"] = "adopt"
    replacements["ADOPTION_FIELDS"] = f"{{\\\"keyName\\\": \\\"{resource_name}\\\"}}"

    resource_data = load_ec2_resource(
        "key_pair_adoption",
        additional_replacements=replacements,
    )
    logging.debug(resource_data)

    ref = k8s.CustomResourceReference(
        CRD_GROUP, CRD_VERSION, KEY_PAIR_RESOURCE_PLURAL,
        resource_name, namespace="default",
    )
    k8s.create_custom_resource(ref, resource_data)
    time.sleep(CREATE_WAIT_AFTER_SECONDS)

    cr = k8s.wait_resource_consumed_by_controller(ref)
    assert cr is not None
    assert k8s.get_resource_exists(ref)

    yield (ref, cr, key_pair_id)

    try:
        _, deleted = k8s.delete_custom_resource(ref, 3, 10)
        assert deleted
    except:
        pass
    try:
        ec2_client.delete_key_pair(KeyName=resource_name)
    except ec2_client.exceptions.ClientError:
        pass


@service_marker
@pytest.mark.canary
class TestKeyPairAdoption:
    def test_key_pair_adopt_delete(self, ec2_client, key_pair_adoption):
        (ref, cr, key_pair_id) = key_pair_adoption

        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)

        cr = k8s.get_resource(ref)
        assert cr["spec"]["keyName"] == ref.name
        assert "publicKeyMaterial" not in cr["spec"]
        assert cr["status"]["keyPairID"] == key_pair_id

        # Deleting the adopted CR deletes the key pair, so the name is read
        # back and the finalizer is not dropped on a not-found read
        _, deleted = k8s.delete_custom_resource(ref, 3, 10)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_key_pair(ref.name, exists=False)

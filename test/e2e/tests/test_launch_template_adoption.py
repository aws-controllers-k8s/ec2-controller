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

"""Integration tests for the LaunchTemplate adopt-or-create flow.
"""

import pytest
import time
import logging

from acktest.resources import random_suffix_name
from acktest.k8s import resource as k8s
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_ec2_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e.tests.helper import EC2Validator

RESOURCE_PLURAL = "launchtemplates"

CREATE_WAIT_AFTER_SECONDS = 10
DELETE_WAIT_AFTER_SECONDS = 10


@pytest.fixture
def adopt_or_create_existing_launch_template(request, ec2_client):
    resource_name = random_suffix_name("lt-adopt", 24)
    existing = ec2_client.create_launch_template(
        LaunchTemplateName=resource_name,
        LaunchTemplateData={"InstanceType": "t2.nano"},
    )
    existing_id = existing["LaunchTemplate"]["LaunchTemplateId"]

    replacements = REPLACEMENT_VALUES.copy()
    replacements["LAUNCH_TEMPLATE_NAME"] = resource_name

    resource_data = load_ec2_resource(
        "launch_template_adopt_or_create",
        additional_replacements=replacements,
    )
    logging.debug(resource_data)

    ref = k8s.CustomResourceReference(
        CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
        resource_name, namespace="default",
    )
    k8s.create_custom_resource(ref, resource_data)
    time.sleep(CREATE_WAIT_AFTER_SECONDS)

    cr = k8s.wait_resource_consumed_by_controller(ref)
    assert cr is not None
    assert k8s.get_resource_exists(ref)

    yield (ref, cr, existing_id)

    try:
        _, deleted = k8s.delete_custom_resource(ref, 3, 10)
        assert deleted
    except:
        pass
    try:
        ec2_client.delete_launch_template(LaunchTemplateId=existing_id)
    except:
        pass


@service_marker
@pytest.mark.canary
class TestLaunchTemplateAdoption:
    def test_adopt_or_create_by_name(self, ec2_client, adopt_or_create_existing_launch_template):
        (ref, cr, existing_id) = adopt_or_create_existing_launch_template

        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=10)

        cr = k8s.get_resource(ref)
        assert cr["status"]["id"] == existing_id
        assert cr["spec"]["data"]["instanceType"] == "t2.nano"

        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_launch_template(existing_id)

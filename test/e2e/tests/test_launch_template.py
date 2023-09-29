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

"""Integration tests for the LaunchTemplate API.
"""

import pytest
import time
import logging

from acktest import tags
from acktest.resources import random_suffix_name
from acktest.k8s import resource as k8s
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_ec2_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e.tests.helper import EC2Validator

RESOURCE_PLURAL = "launchtemplates"

DELETE_WAIT_AFTER_SECONDS= 10
CREATE_WAIT_AFTER_SECONDS= 10


@pytest.fixture
def simple_launch_template(request):
    resource_name = random_suffix_name("lt-ack-test", 24)
    resource_file = "launch_template"
   
    replacements = REPLACEMENT_VALUES.copy()
    replacements["LAUNCH_TEMPLATE_NAME"] = resource_name
    replacements["VERSION_DESCRIPTION"] = "THIS IS TEST LAUNCH TEMPLATE"
    
    # Load LaunchTemplate CR
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
class TestLaunchTemplate:
    def test_crud(self, ec2_client, simple_launch_template):
        (ref, cr) = simple_launch_template

        resource_id = cr["status"]["launchTemplateID"]   
       

        # Check LaunchTemplate exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_launch_template(resource_id)

        # Validate LaunchTemplate
        launch_template = ec2_validator.get_launch_template(resource_id)
        assert launch_template["LaunchTemplateId"] == resource_id
        
        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # Update tags
        update_tags = [
                {
                    "key": "newtagkey",
                    "value": "newtagvalue",
                }
            ]

        # Patch the launchtemplate, updating the tags with new pair
        updates = {
            "spec": {"tags": update_tags},
        }

        k8s.patch_custom_resource(ref, updates)
        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # Check resource synced successfully
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)

        # Only user tags should be present in Spec
        resource = k8s.get_resource(ref)
        assert len(resource["spec"]["tags"]) == 1
        assert resource["spec"]["tags"][0]["key"] == "newtagkey"
        assert resource["spec"]["tags"][0]["value"] == "newtagvalue"

        # Check user and ack tags on aws 
        launch_template = ec2_validator.get_launch_template(resource_id)
        assert len(launch_template['Tags']) == 3

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref, 2, 5)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check LaunchTemplate no longer exists in AWS
        ec2_validator.assert_launch_template(resource_id, exists=False)
    


   
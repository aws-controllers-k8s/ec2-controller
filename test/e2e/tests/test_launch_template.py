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
from acktest import tags

RESOURCE_PLURAL = "launchtemplates"

DELETE_WAIT_AFTER_SECONDS= 10
CREATE_WAIT_AFTER_SECONDS= 10
MODIFY_WAIT_AFTER_SECONDS= 10


@pytest.fixture
def simple_launch_template(request):
    resource_name = random_suffix_name("lt-ack-test", 24)
    resource_file = "launch_template"

    replacements = REPLACEMENT_VALUES.copy()
    replacements["LAUNCH_TEMPLATE_NAME"] = resource_name

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

        time.sleep(CREATE_WAIT_AFTER_SECONDS) 
                
        resource_id = cr["status"]["id"]   
        
        # Check LaunchTemplate exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_launch_template(resource_id)

        launch_template = ec2_validator.get_launch_template(launch_template_id=resource_id)
        default_version = launch_template['DefaultVersionNumber']
        latest_version = launch_template['LatestVersionNumber']

        assert latest_version == default_version == 1

        assert not cr['spec']['data']['monitoring']['enabled']

        # Update enabling monitoring
        updates = {
            "spec": {
                "data": {
                    "monitoring": {
                        "enabled": True
                    }
                }
            }
        }

        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # Check resource synced successfully
        assert k8s.wait_on_condition(ref, "Ready", "True", wait_periods=5)

        cr = k8s.get_resource(ref)

        launch_template = ec2_validator.get_launch_template(launch_template_id=resource_id)
        default_version = launch_template['DefaultVersionNumber']
        latest_version = launch_template['LatestVersionNumber']

        # assert default version hasn't changed yet
        assert default_version == 1
        assert latest_version == 2

        assert default_version == cr['spec']['defaultVersion']
        assert latest_version == cr['status']['latestVersion']

        launch_template_v2 = ec2_validator.get_launch_template_version(launch_template_id=resource_id, version='2')

        # check latest launch_template has monitoring true 
        assert launch_template_v2['LaunchTemplateData']['Monitoring']['Enabled']
        # and CR also has true
        assert cr['spec']['data']['monitoring']['enabled']

        # update defaultVersion
        updates = {
            'spec': {
                'defaultVersion': 2
            }
        }

        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # Check resource synced successfully
        assert k8s.wait_on_condition(ref, "Ready", "True", wait_periods=5)

        cr = k8s.get_resource(ref)

        launch_template = ec2_validator.get_launch_template(launch_template_id=resource_id)
        default_version = launch_template['DefaultVersionNumber']
        latest_version = launch_template['LatestVersionNumber']
        
        # latest_version was asserted to be 2 earlier
        assert default_version == latest_version == 2

    
    @pytest.mark.resource_data({'tag_key': 'initialtagkey', 'tag_value': 'initialtagvalue'})
    def test_crud_tags(self, ec2_client, simple_launch_template):
        (ref, cr) = simple_launch_template

        time.sleep(CREATE_WAIT_AFTER_SECONDS) 

        resource_id = cr["status"]["id"]

        # Validate LaunchTemplate Tags
        ec2_validator = EC2Validator(ec2_client)
        launch_template = ec2_validator.get_launch_template(resource_id)
        assert launch_template is not None
        latest_tags = launch_template["Tags"]

        tags.assert_ack_system_tags(
            tags=latest_tags,
        )

        assert 'tags' in cr['spec']
        user_tags = cr["spec"]["tags"]
        user_tags = [{"Key": d["key"], "Value": d["value"]} for d in user_tags]
        tags.assert_equal_without_ack_tags(
            expected=user_tags,
            actual=latest_tags,
        )


        # Update tags
        update_tags =  [
            {
                "key": "newKey",
                "value": "newValue",
            }
        ]

        # Patch the launchtemplate, updating the tags with new pair
        updates = {
            "spec": {"tags": update_tags},
        }

        k8s.patch_custom_resource(ref, updates)
        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # Check resource synced successfully
        assert k8s.wait_on_condition(ref, "Ready", "True", wait_periods=5)

        cr = k8s.get_resource(ref)
        assert 'tags' in cr['spec']
        user_tags = cr["spec"]["tags"]

        ec2_validator = EC2Validator(ec2_client)
        launch_template = ec2_validator.get_launch_template(resource_id)
        assert launch_template is not None
        latest_tags = launch_template["Tags"]

        tags.assert_ack_system_tags(
            tags=latest_tags,
        )

        user_tags = [{"Key": d["key"], "Value": d["value"]} for d in user_tags]
        tags.assert_equal_without_ack_tags(
            expected=user_tags,
            actual=latest_tags,
        )

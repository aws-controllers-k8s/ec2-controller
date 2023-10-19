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
from e2e.tests.test_launch_template import simple_launch_template

RESOURCE_PLURAL = "launchtemplateversions"

DELETE_WAIT_AFTER_SECONDS= 10
CREATE_WAIT_AFTER_SECONDS= 10

@pytest.fixture
def simple_launch_template_version(request,simple_launch_template):
    
    resource_name = random_suffix_name("ltv-ack-test", 24)
    resource_file = "launch_template_version"


    replacements = REPLACEMENT_VALUES.copy()
    replacements["LAUNCH_TEMPLATE_ID"] = simple_launch_template[1]['status']['launchTemplateID']
    replacements["LAUNCH_TEMPLATE_NAME"] = resource_name
    replacements["VERSION_DESCRIPTION"] = "THIS IS TEST LAUNCH TEMPLATE VERSION"
    
    
    marker = request.node.get_closest_marker("resource_data")

    if marker is not None:
        data = marker.args[0]
        if 'sourceversion' in data:
            replacements["SOURCE_VERSION"] = "'" + str(data["sourceversion"]) + "'"
            replacements["DISABLE_API_TERMINATION"] = str(True)
            replacements["INSTANCE_TYPE"] = ""
    else:
        replacements["INSTANCE_TYPE"] = "t2.medium" 
        replacements["DISABLE_API_TERMINATION"] = str(False)
        replacements["SOURCE_VERSION"] = ""

    # Load LaunchTemplateVersion CR
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
class TestLaunchTemplateVersion:
    def test_launch_template_version_without_source(self, ec2_client,simple_launch_template_version):
        
 
        ## Create launch template version using launch template id
        (refv ,crv) = simple_launch_template_version

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        launch_template_id_of_version = crv['spec']['launchTemplateID']

        # Check LaunchTemplateVersion exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_launch_template_version(launch_template_id_of_version, launch_template_version=["2"])

        # Validate LaunchTemplate
        launch_template_version = ec2_validator.get_launch_template_version(launch_template_id_of_version,launch_template_version=["2"])
        assert launch_template_version["VersionNumber"] == 2

    @pytest.mark.resource_data({'sourceversion': 1})
    def test_launch_template_version_with_source(self, ec2_client,simple_launch_template_version):
        
        ## Create launch template version using launch template id
        (refv ,crv) = simple_launch_template_version

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        launch_template_id_of_version = crv['spec']['launchTemplateID']
       

        # Check LaunchTemplateVersion exists in AWS
        ec2_validator = EC2Validator(ec2_client)
        ec2_validator.assert_launch_template_version(launch_template_id_of_version, launch_template_version=["2"])

        # Validate LaunchTemplate
        launch_template_version = ec2_validator.get_launch_template_version(launch_template_id_of_version,launch_template_version=["2"])
        
        assert launch_template_version["VersionNumber"] == 2
        assert launch_template_version["LaunchTemplateData"]["InstanceType"] == "t2.micro"
        

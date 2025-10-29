# Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License"). You may
# not use this file except in compliance with the License. A copy of the
# License is located at
#
#     http://aws.amazon.com/apache2.0/
#
# or in the "license" file accompanying this file. This file is distributed
# on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
# express or implied. See the License for the specific language governing
# permissions and limitations under the License.

"""Integration tests for Managed Prefix List API.
"""

import pytest
import time
import logging

from acktest.resources import random_suffix_name
from acktest.k8s import resource as k8s
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_ec2_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e.bootstrap_resources import get_bootstrap_resources

RESOURCE_PLURAL = "managedprefixlists"

CREATE_WAIT_AFTER_SECONDS = 10
UPDATE_WAIT_AFTER_SECONDS = 10
DELETE_WAIT_AFTER_SECONDS = 10

@pytest.fixture(scope="module")
def prefix_list_ipv4():
    resource_name = random_suffix_name("managed-prefix-list-ipv4", 32)
    
    replacements = REPLACEMENT_VALUES.copy()
    replacements["PREFIX_LIST_NAME"] = resource_name
    replacements["TAG_KEY"] = "test-key"
    replacements["TAG_VALUE"] = "test-value"

    # Load the resource
    resource_data = load_ec2_resource(
        "managed_prefix_list",
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

    # Teardown
    try:
        _, deleted = k8s.delete_custom_resource(ref, 3, 10)
        assert deleted
    except:
        pass


@pytest.fixture(scope="module")
def prefix_list_ipv6():
    resource_name = random_suffix_name("managed-prefix-list-ipv6", 32)
    
    replacements = REPLACEMENT_VALUES.copy()
    replacements["PREFIX_LIST_NAME_IPV6"] = resource_name
    replacements["TAG_KEY"] = "test-key"
    replacements["TAG_VALUE"] = "test-value"

    # Load the resource
    resource_data = load_ec2_resource(
        "managed_prefix_list_ipv6",
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

    # Teardown
    try:
        _, deleted = k8s.delete_custom_resource(ref, 3, 10)
        assert deleted
    except:
        pass


@service_marker
@pytest.mark.canary
class TestManagedPrefixList:
    def test_create_delete_ipv4(self, prefix_list_ipv4):
        """Test creation and deletion of an IPv4 managed prefix list."""
        (ref, cr) = prefix_list_ipv4

        # Check that the resource was created
        assert cr is not None
        assert 'status' in cr
        assert 'prefixListID' in cr['status']
        
        prefix_list_id = cr['status']['prefixListID']
        assert prefix_list_id is not None
        assert prefix_list_id.startswith('pl-')

        # Check state
        assert 'state' in cr['status']
        state = cr['status']['state']
        assert state in ['create-in-progress', 'create-complete']

        # Wait for the prefix list to be in a synced state
        time.sleep(CREATE_WAIT_AFTER_SECONDS)
        
        # Get updated resource
        cr = k8s.get_resource(ref)
        assert cr['status']['state'] == 'create-complete'

        # Check that version was set
        assert 'version' in cr['status']
        assert cr['status']['version'] is not None

    def test_create_delete_ipv6(self, prefix_list_ipv6):
        """Test creation and deletion of an IPv6 managed prefix list."""
        (ref, cr) = prefix_list_ipv6

        # Check that the resource was created
        assert cr is not None
        assert 'status' in cr
        assert 'prefixListID' in cr['status']
        
        prefix_list_id = cr['status']['prefixListID']
        assert prefix_list_id is not None
        assert prefix_list_id.startswith('pl-')

        # Verify address family in spec
        assert cr['spec']['addressFamily'] == 'IPv6'

        # Wait for completion
        time.sleep(CREATE_WAIT_AFTER_SECONDS)
        cr = k8s.get_resource(ref)
        assert cr['status']['state'] == 'create-complete'

    def test_update_entries(self, prefix_list_ipv4):
        """Test updating prefix list entries."""
        (ref, cr) = prefix_list_ipv4

        # Wait for initial creation to complete
        time.sleep(CREATE_WAIT_AFTER_SECONDS)
        cr = k8s.get_resource(ref)
        assert cr['status']['state'] == 'create-complete'
        
        initial_version = cr['status']['version']

        # Update the entries - add a new CIDR block
        cr['spec']['entries'].append({
            'cidr': '10.0.2.0/24',
            'description': 'New network C'
        })

        # Apply the update
        k8s.patch_custom_resource(ref, cr)
        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        # Get the updated resource
        cr = k8s.get_resource(ref)
        
        # Check that version was incremented
        assert 'version' in cr['status']
        updated_version = cr['status']['version']
        assert updated_version > initial_version

        # Verify entries were updated
        assert len(cr['spec']['entries']) == 4  # Original 3 + 1 new

    def test_update_tags(self, prefix_list_ipv4):
        """Test updating prefix list tags."""
        (ref, cr) = prefix_list_ipv4

        # Add a new tag
        new_tag = {
            'key': 'Environment',
            'value': 'Test'
        }
        if 'tags' not in cr['spec']:
            cr['spec']['tags'] = []
        cr['spec']['tags'].append(new_tag)

        # Apply the update
        k8s.patch_custom_resource(ref, cr)
        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        # Get the updated resource
        cr = k8s.get_resource(ref)
        
        # Verify tag was added
        tags = cr['spec'].get('tags', [])
        assert any(tag['key'] == 'Environment' and tag['value'] == 'Test' for tag in tags)

    def test_prefix_list_fields(self, prefix_list_ipv4):
        """Test that all expected fields are present."""
        (ref, cr) = prefix_list_ipv4

        # Check spec fields
        assert 'prefixListName' in cr['spec']
        assert 'addressFamily' in cr['spec']
        assert 'maxEntries' in cr['spec']
        assert 'entries' in cr['spec']

        # Check status fields
        assert 'prefixListID' in cr['status']
        assert 'state' in cr['status']
        assert 'version' in cr['status']
        assert 'ownerID' in cr['status']

        # Validate entry structure
        for entry in cr['spec']['entries']:
            assert 'cidr' in entry
            # Description is optional



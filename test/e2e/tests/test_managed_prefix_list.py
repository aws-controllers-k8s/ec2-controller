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
import boto3

from acktest.resources import random_suffix_name
from acktest.k8s import resource as k8s
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_ec2_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e.bootstrap_resources import get_bootstrap_resources
from e2e.tests.helper import EC2Validator

RESOURCE_PLURAL = "managedprefixlists"

CREATE_WAIT_AFTER_SECONDS = 10
UPDATE_WAIT_AFTER_SECONDS = 10
DELETE_WAIT_AFTER_SECONDS = 10

@pytest.fixture(scope="module")
def ec2_validator():
    """Fixture to provide EC2 validator for AWS API calls."""
    ec2_client = boto3.client("ec2")
    return EC2Validator(ec2_client)

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
    def test_create_delete_ipv4(self, prefix_list_ipv4, ec2_validator):
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

        # Wait for AWS to complete creation
        state_reached = ec2_validator.wait_managed_prefix_list_state(
            prefix_list_id,
            'create-complete',
            max_wait_seconds=180
        )
        assert state_reached, f"Prefix list {prefix_list_id} did not reach create-complete state within timeout"

        # Wait for K8s controller to sync the state from AWS
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=30), \
            "Resource did not sync within timeout"

        # Verify final state
        cr = k8s.get_resource(ref)
        assert cr['status'].get('state') == 'create-complete', \
            f"Expected state create-complete, got {cr['status'].get('state')}"

        # Check that version was set
        assert 'version' in cr['status']
        assert cr['status']['version'] is not None

    def test_create_delete_ipv6(self, prefix_list_ipv6, ec2_validator):
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

        # Wait for AWS to complete creation
        state_reached = ec2_validator.wait_managed_prefix_list_state(
            prefix_list_id,
            'create-complete',
            max_wait_seconds=180
        )
        assert state_reached, f"Prefix list {prefix_list_id} did not reach create-complete state within timeout"

        # Wait for K8s controller to sync the state from AWS
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=30), \
            "Resource did not sync within timeout"

        # Verify final state
        cr = k8s.get_resource(ref)
        assert cr['status'].get('state') == 'create-complete', \
            f"Expected state create-complete, got {cr['status'].get('state')}"

    def test_update_entries(self, prefix_list_ipv4, ec2_validator):
        """Test adding and removing prefix list entries."""
        (ref, cr) = prefix_list_ipv4

        # Get the prefix list ID
        assert 'prefixListID' in cr['status'], "PrefixListID should be present in status"
        prefix_list_id = cr['status']['prefixListID']

        # Wait for the controller to process and sync the change
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=40), \
            "Resource did not sync after creation"

        # Wait for initial creation to complete in AWS
        state_reached = ec2_validator.wait_managed_prefix_list_state(
            prefix_list_id,
            'create-complete',
            max_wait_seconds=180
        )
        assert state_reached, f"Prefix list {prefix_list_id} did not reach create-complete state"

        # Verify initial state - should have 3 entries
        aws_prefix_list = ec2_validator.get_managed_prefix_list(prefix_list_id)
        initial_count = len(aws_prefix_list.get('Entries', []))
        assert initial_count == 3, f"Expected 3 initial entries, got {initial_count}"

        # Get the latest resource
        cr = k8s.get_resource(ref)
        assert cr['status']['state'] == 'create-complete', f"K8s state is {cr['status']['state']}, expected create-complete"

        # ===== TEST 1: Add an entry (3 → 4) =====
        cr['spec']['entries'].append({
            'cidr': '10.0.2.0/24',
            'description': 'New network C'
        })

        # Apply the update
        k8s.patch_custom_resource(ref, cr)

        # Give AWS time to process the async modification
        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        # Wait for the controller to process and sync the change
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=40), \
            "Resource did not sync after add"

        # Now wait for AWS to complete the modification
        state_reached = ec2_validator.wait_managed_prefix_list_state(
            prefix_list_id,
            'modify-complete',
            max_wait_seconds=180
        )
        assert state_reached, f"Prefix list {prefix_list_id} did not reach modify-complete state after add"

        # Verify in AWS
        aws_prefix_list = ec2_validator.get_managed_prefix_list(prefix_list_id)
        after_add_count = len(aws_prefix_list.get('Entries', []))
        assert after_add_count == 4, f"Expected 4 entries after add, got {after_add_count}"

        # ===== TEST 2: Remove an entry (4 → 3) =====
        cr = k8s.get_resource(ref)
        original_entries = cr['spec']['entries'][:]

        # Remove the entry we just added
        entry_to_remove = '10.0.2.0/24'
        cr['spec']['entries'] = [e for e in original_entries if e['cidr'] != entry_to_remove]

        # Apply the update
        k8s.patch_custom_resource(ref, cr)

        # Give AWS time to process the async modification
        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        # Wait for the controller to process and sync the change
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=40), \
            "Resource did not sync after removal"

        # Now wait for AWS to complete the modification
        state_reached = ec2_validator.wait_managed_prefix_list_state(
            prefix_list_id,
            'modify-complete',
            max_wait_seconds=180
        )
        assert state_reached, f"Prefix list {prefix_list_id} did not reach modify-complete state after removal"

        # Verify in AWS
        aws_prefix_list = ec2_validator.get_managed_prefix_list(prefix_list_id)
        after_remove_entries = aws_prefix_list.get('Entries', [])
        after_remove_count = len(after_remove_entries)
        aws_cidrs = [e['Cidr'] for e in after_remove_entries]

        # Verify deletion happened
        assert after_remove_count == 3, f"Expected 3 entries after removal, got {after_remove_count}"
        assert entry_to_remove not in aws_cidrs, \
            f"Entry {entry_to_remove} should have been removed but is still in AWS: {aws_cidrs}"

        # Final verification - check K8s matches AWS
        cr = k8s.get_resource(ref)
        k8s_cidrs = [e['cidr'] for e in cr['spec'].get('entries', [])]
        assert len(k8s_cidrs) == 3, f"Expected 3 entries in K8s, got {len(k8s_cidrs)}"
        assert entry_to_remove not in k8s_cidrs, \
            f"Entry {entry_to_remove} should not be in K8s: {k8s_cidrs}"

    def test_update_tags(self, prefix_list_ipv4, ec2_validator):
        """Test adding, updating, and removing prefix list tags."""
        (ref, cr) = prefix_list_ipv4

        # Get the prefix list ID
        prefix_list_id = cr['status']['prefixListID']

        # Get the latest version of the resource to avoid conflicts
        cr = k8s.get_resource(ref)

        # ===== TEST 1: Add a new tag =====
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

        # Wait for the resource to be synced
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5), \
            "Resource did not sync after adding tag"

        # Get the updated resource
        cr = k8s.get_resource(ref)

        # Verify tag was added in K8s
        tags = cr['spec'].get('tags', [])
        assert any(tag['key'] == 'Environment' and tag['value'] == 'Test' for tag in tags), \
            "Environment tag should be added in K8s"

        # Verify tag was added in AWS
        aws_prefix_list = ec2_validator.get_managed_prefix_list(prefix_list_id)
        aws_tags = aws_prefix_list.get('Tags', [])
        assert any(tag['Key'] == 'Environment' and tag['Value'] == 'Test' for tag in aws_tags), \
            "Environment tag should be added in AWS"

        # ===== TEST 2: Update an existing tag =====
        for tag in cr['spec']['tags']:
            if tag['key'] == 'Environment':
                tag['value'] = 'Development'

        # Apply the update
        k8s.patch_custom_resource(ref, cr)
        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        # Wait for the resource to be synced
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5), \
            "Resource did not sync after updating tag"

        # Get the updated resource
        cr = k8s.get_resource(ref)

        # Verify tag was updated in K8s
        tags = cr['spec'].get('tags', [])
        assert any(tag['key'] == 'Environment' and tag['value'] == 'Development' for tag in tags), \
            "Environment tag should be updated to Development in K8s"

        # Verify tag was updated in AWS
        aws_prefix_list = ec2_validator.get_managed_prefix_list(prefix_list_id)
        aws_tags = aws_prefix_list.get('Tags', [])
        assert any(tag['Key'] == 'Environment' and tag['Value'] == 'Development' for tag in aws_tags), \
            "Environment tag should be updated to Development in AWS"

        # ===== TEST 3: Remove a tag =====
        cr['spec']['tags'] = [tag for tag in cr['spec']['tags'] if tag['key'] != 'Environment']

        # Apply the update
        k8s.patch_custom_resource(ref, cr)
        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        # Wait for the resource to be synced
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5), \
            "Resource did not sync after removing tag"

        # Get the updated resource
        cr = k8s.get_resource(ref)

        # Verify tag was removed in K8s
        tags = cr['spec'].get('tags', [])
        assert not any(tag['key'] == 'Environment' for tag in tags), \
            "Environment tag should be removed from K8s"

        # Verify tag was removed in AWS
        aws_prefix_list = ec2_validator.get_managed_prefix_list(prefix_list_id)
        aws_tags = aws_prefix_list.get('Tags', [])
        assert not any(tag['Key'] == 'Environment' for tag in aws_tags), \
            "Environment tag should be removed from AWS"

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

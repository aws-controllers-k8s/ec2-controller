# Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License"). You may
# not use this file except in compliance with the License. A copy of the
# License is located at
#
#	 http://aws.amazon.com/apache2.0/
#
# or in the "license" file accompanying this file. This file is distributed
# on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
# express or implied. See the License for the specific language governing
# permissions and limitations under the License.

"""Helper functions for ec2 tests
"""

from typing import Union, Dict
import time


class EC2Validator:
    def __init__(self, ec2_client):
        self.ec2_client = ec2_client

    def get_dhcp_options(self, dhcp_options_id: str):
        try:
            aws_res = self.ec2_client.describe_dhcp_options(DhcpOptionsIds=[dhcp_options_id])
            if len(aws_res["DhcpOptions"]) > 0:
                return aws_res["DhcpOptions"][0]
            return None
        except self.ec2_client.exceptions.ClientError:
            return None

    def assert_dhcp_options(self, dhcp_options_id: str, exists=True):
        res_found = False
        try:
            aws_res = self.ec2_client.describe_dhcp_options(DhcpOptionsIds=[dhcp_options_id])
            res_found = len(aws_res["DhcpOptions"]) > 0
        except self.ec2_client.exceptions.ClientError:
            pass
        assert res_found is exists

    def assert_dhcp_vpc_association(self, dhcp_options_id: str, vpc_id: str):
        print("data:",vpc_id)
        try:
            vpc = self.get_vpc(vpc_id)
        except self.ec2_client.exceptions.ClientError:
            pass
        assert vpc["DhcpOptionsId"] == dhcp_options_id

    def get_internet_gateway(self, igw_id: str):
        try:
            aws_res = self.ec2_client.describe_internet_gateways(InternetGatewayIds=[igw_id])
            if len(aws_res["InternetGateways"]) > 0:
                return aws_res["InternetGateways"][0]
            return None
        except self.ec2_client.exceptions.ClientError:
            return None

    def assert_internet_gateway(self, igw_id: str, exists=True):
        assert (self.get_internet_gateway(igw_id) is not None) == exists

    def get_nat_gateway(self, ngw_id: str):
        try:
            aws_res = self.ec2_client.describe_nat_gateways(NatGatewayIds=[ngw_id])
            if len(aws_res["NatGateways"]) > 0:
                return aws_res["NatGateways"][0]
            return None
        except self.ec2_client.exceptions.ClientError:
            return None

    def assert_nat_gateway(self, ngw_id: str, exists=True):
        res_found = False
        try:
            aws_res = self.ec2_client.describe_nat_gateways(NatGatewayIds=[ngw_id])
            assert len(aws_res["NatGateways"]) > 0
            ngw = aws_res["NatGateways"][0]
            # NATGateway may take awhile to be removed server-side, so 
            # treat 'deleting' and 'deleted' states as resource no longer existing
            res_found = ngw is not None and ngw['State'] != "deleting" and ngw['State'] != "deleted"
        except self.ec2_client.exceptions.ClientError:
            pass
        assert res_found is exists

    def assert_route(self, route_table_id: str, gateway_id: str, origin: str, exists=True):
        res_found = False
        try:
            aws_res = self.ec2_client.describe_route_tables(RouteTableIds=[route_table_id])
            routes = aws_res["RouteTables"][0]["Routes"]
            for route in routes:
                if route["Origin"] == origin and route["GatewayId"] == gateway_id:
                    res_found = True
        except self.ec2_client.exceptions.ClientError:
            pass
        assert res_found is exists

    def assert_route_table_association(self, route_table_id: str, gateway_id: str, AssociationState: str, exists=True):
        res_found = False
        try:
            aws_res = self.ec2_client.describe_route_tables(
                RouteTableIds=[route_table_id]
            )
            associations = aws_res["RouteTables"][0]["Associations"]
            for association in associations:
                if (association["AssociationState"]["State"] == AssociationState and
                    association["GatewayId"] == gateway_id):
                    res_found = True
        except self.ec2_client.exceptions.ClientError:
            pass
        assert res_found is exists

    def assert_entry(self, network_acl_id: str, rule_number: int, egress: str, exists=True):
        res_found = False
        try:
            aws_res = self.ec2_client.describe_network_acls(NetworkAclIds=[network_acl_id])
            entries = aws_res["NetworkAcls"][0]["Entries"]
            for entry in entries:
                if entry["RuleNumber"] == rule_number and str(entry["Egress"]) == egress:
                    res_found = True
        except self.ec2_client.exceptions.ClientError:
            pass
        assert res_found is exists
    
    def assert_association(self, network_acl_id: str, subnet_id: str, exists=True):
        res_found = False
        try:
            aws_res = self.ec2_client.describe_network_acls(NetworkAclIds=[network_acl_id])
            associations = aws_res["NetworkAcls"][0]["Associations"]
            for association in associations:
                if association["SubnetId"] == subnet_id:
                    res_found = True
        except self.ec2_client.exceptions.ClientError:
            pass
        assert res_found is exists

    def get_route_table(self, route_table_id: str) -> Union[None, Dict]:
        try:
            aws_res = self.ec2_client.describe_route_tables(RouteTableIds=[route_table_id])
            if len(aws_res["RouteTables"]) > 0:
                return aws_res["RouteTables"][0]
            return None
        except self.ec2_client.exceptions.ClientError:
            return None
        
    def get_network_acl(self, network_acl_id: str) -> Union[None, Dict]:
        try:
            aws_res = self.ec2_client.describe_network_acls(NetworkAclIds=[network_acl_id])
            if len(aws_res["NetworkAcls"]) > 0:
                return aws_res["NetworkAcls"][0]
            return None
        except self.ec2_client.exceptions.ClientError:
            return None

    def assert_route_table(self, route_table_id: str, exists=True):
        assert (self.get_route_table(route_table_id) is not None) == exists

    def assert_network_acl(self, network_acl_id: str, exists=True):
        assert (self.get_network_acl(network_acl_id) is not None) == exists

    def get_route_table_association(self, route_table_id: str, subnet_id: str) -> Union[None, Dict]:
        rt = self.get_route_table(route_table_id)
        for assoc in rt["Associations"]:
            if assoc["SubnetId"] == subnet_id:
                return assoc
        return None

    def assert_security_group(self, sg_id: str, exists=True):
        res_found = False
        try:
            aws_res = self.ec2_client.describe_security_groups(GroupIds=[sg_id])
            res_found = len(aws_res["SecurityGroups"]) > 0
        except self.ec2_client.exceptions.ClientError:
            pass
        assert res_found is exists

    def get_security_group(self, sg_id: str) -> Union[None, Dict]:
        try:
            aws_res = self.ec2_client.describe_security_groups(GroupIds=[sg_id])
            if len(aws_res["SecurityGroups"]) > 0:
                return aws_res["SecurityGroups"][0]
            return None
        except self.ec2_client.exceptions.ClientError:
            return None

    def get_subnet(self, subnet_id: str) -> Union[None, Dict]:
        try:
            aws_res = self.ec2_client.describe_subnets(SubnetIds=[subnet_id])
            if len(aws_res["Subnets"]) > 0:
                return aws_res["Subnets"][0]
            return None
        except self.ec2_client.exceptions.ClientError:
            return None

    def assert_subnet(self, subnet_id: str, exists=True):
        assert (self.get_subnet(subnet_id) is not None) == exists

    def get_transit_gateway(self, tgw_id: str) -> Union[None, Dict]:
        try:
            aws_res = self.ec2_client.describe_transit_gateways(TransitGatewayIds=[tgw_id])
            if len(aws_res["TransitGateways"]) > 0:
                return aws_res["TransitGateways"][0]
            return None
        except self.ec2_client.exceptions.ClientError:
            return None

    def assert_transit_gateway(self, tgw_id: str, exists=True):
        res_found = False
        try:
            aws_res = self.ec2_client.describe_transit_gateways(TransitGatewayIds=[tgw_id])
            tgw = aws_res["TransitGateways"][0]
            # TransitGateway may take awhile to be removed server-side, so 
            # treat 'deleting' and 'deleted' states as resource no longer existing
            res_found = tgw is not None and tgw['State'] != "deleting" and tgw['State'] != "deleted"
        except self.ec2_client.exceptions.ClientError:
            pass
        assert res_found is exists

    def wait_transit_gateway_state(self, tgw_id: str, state: str):
        is_state = False
        max_tries = 5
        try:
            for tries in range(max_tries):
                transit_gateway = self.ec2_client.describe_transit_gateways(TransitGatewayIds=[tgw_id])
                if transit_gateway['TransitGateways'][0]['State'] == state:
                    is_state=True
                    break
                else:
                    time.sleep(30)
        except:
            pass
        return is_state

    def get_transit_gateway_vpc_attachment(self, attachment_id: str) -> Union[None, Dict]:
        try:
            aws_res = self.ec2_client.describe_transit_gateway_vpc_attachments(TransitGatewayAttachmentIds=[attachment_id])
            if len(aws_res["TransitGatewayVpcAttachments"]) > 0:
                return aws_res["TransitGatewayVpcAttachments"][0]
            return None
        except self.ec2_client.exceptions.ClientError:
            return None

    def assert_transit_gateway_vpc_attachment(self, tgw_id: str, exists=True):
        res_found = False
        try:
            aws_res = self.ec2_client.describe_transit_gateway_vpc_attachments(TransitGatewayAttachmentIds=[tgw_id])
            tgw = aws_res["TransitGateways"][0]
            # TransitGateway may take awhile to be removed server-side, so 
            # treat 'deleting' and 'deleted' states as resource no longer existing
            res_found = tgw is not None and tgw['State'] != "deleting" and tgw['State'] != "deleted"
        except self.ec2_client.exceptions.ClientError:
            pass
        assert res_found is exists

    def get_vpc(self, vpc_id: str) -> Union[None, Dict]:
        try:
            aws_res = self.ec2_client.describe_vpcs(VpcIds=[vpc_id])
            if len(aws_res["Vpcs"]) > 0:
                return aws_res["Vpcs"][0]
            return None
        except self.ec2_client.exceptions.ClientError:
            return None
            
    def assert_vpc(self, vpc_id: str, exists=True):
        res_found = False
        try:
            aws_res = self.ec2_client.describe_vpcs(VpcIds=[vpc_id])
            res_found = len(aws_res["Vpcs"]) > 0
        except self.ec2_client.exceptions.ClientError:
            pass
        assert res_found is exists

    def get_vpc_endpoint(self, vpc_endpoint_id: str) -> Union[None, Dict]:
        try:
            aws_res = self.ec2_client.describe_vpc_endpoints(VpcEndpointIds=[vpc_endpoint_id])
            if len(aws_res["VpcEndpoints"]) > 0:
                return aws_res["VpcEndpoints"][0]
            return None
        except self.ec2_client.exceptions.ClientError:
            return None

    def assert_vpc_endpoint(self, vpc_endpoint_id: str, exists=True):
        res_found = False
        try:
            aws_res = self.ec2_client.describe_vpc_endpoints(VpcEndpointIds=[vpc_endpoint_id])
            res_found = len(aws_res["VpcEndpoints"]) > 0
        except self.ec2_client.exceptions.ClientError:
            pass
        assert res_found is exists

    def get_vpc_endpoint_service_configuration(self, vpc_endpoint_service_configuration_id: str) -> Union[None, Dict]:
        try:
            aws_res = self.ec2_client.describe_vpc_endpoint_service_configurations(ServiceIds=[vpc_endpoint_service_configuration_id])
            if len(aws_res["ServiceConfigurations"]) > 0:
                return aws_res["ServiceConfigurations"][0]
            return None
        except self.ec2_client.exceptions.ClientError:
            return None

    def assert_vpc_endpoint_service_configuration(self, vpc_endpoint_service_configuration_id: str, exists=True):
        res_found = False
        try:
            aws_res = self.ec2_client.describe_vpc_endpoint_service_configurations(ServiceIds=[vpc_endpoint_service_configuration_id])
            res_found = len(aws_res["ServiceConfigurations"]) > 0
        except self.ec2_client.exceptions.ClientError:
            pass
        assert res_found is exists
    
    def get_vpc_endpoint_service_permissions(self, vpc_endpoint_service_configuration_id: str) -> Union[None, Dict]:
        try:
            aws_res = self.ec2_client.describe_vpc_endpoint_service_permissions(ServiceId=vpc_endpoint_service_configuration_id)
            if len(aws_res["AllowedPrincipals"]) > 0:
                return aws_res["AllowedPrincipals"]
            return None
        except self.ec2_client.exceptions.ClientError:
            return None
        
    def get_vpc_peering_connection(self, vpc_peering_connection_id: str) -> Union[None, Dict]:
        try:
            aws_res = self.ec2_client.describe_vpc_peering_connections(
                VpcPeeringConnectionIds=[vpc_peering_connection_id]
            )
            if len(aws_res["VpcPeeringConnections"]) > 0:
                return aws_res["VpcPeeringConnections"][0]
            return None
        except self.ec2_client.exceptions.ClientError:
            return None

    def assert_vpc_peering_connection(self, vpc_peering_connection_id: str, exists=True):
        res_found = False
        try:
            aws_res = self.ec2_client.describe_vpc_peering_connections(VpcPeeringConnectionIds=[vpc_peering_connection_id])
            res_found = len(aws_res["VpcPeeringConnections"]) > 0
        except self.ec2_client.exceptions.ClientError:
            pass
        assert (res_found is exists 
                or
                aws_res["VpcPeeringConnections"][0]["Status"]["Code"] == "deleted")
        
    def get_capacity_reservation(self, capacity_reservation_id: str) -> Union[None, Dict]:
        try:
            aws_res = self.ec2_client.describe_capacity_reservations(
                CapacityReservationIds=[capacity_reservation_id]
            )
            if len(aws_res["CapacityReservations"]) > 0:
                return aws_res["CapacityReservations"][0]
            return None
        except self.ec2_client.exceptions.ClientError:
            return None

    def assert_capacity_reservation(self, capacity_reservation_id: str, exists=True):
        res_found = False
        try:
            aws_res = self.ec2_client.describe_capacity_reservations(CapacityReservationIds=[capacity_reservation_id])
            res_found = len(aws_res["CapacityReservations"]) > 0
        except self.ec2_client.exceptions.ClientError:
            pass
        assert (res_found is exists 
                or
                aws_res["CapacityReservations"][0]["State"] == "cancelled")

    def assert_launch_template(self, launch_template_id: str, exists=True):
        res_found = False
        try:
            aws_res = self.ec2_client.describe_launch_templates(LaunchTemplateIds=[launch_template_id])
            res_found = len(aws_res["LaunchTemplates"]) > 0
        except self.ec2_client.exceptions.ClientError:
            pass
        assert res_found is exists

    def get_launch_template(self, launch_template_id: str) -> Union[None, Dict]:
        try:
            aws_res = self.ec2_client.describe_launch_templates(LaunchTemplateIds=[launch_template_id])
            if len(aws_res["LaunchTemplates"]) > 0:
                return aws_res["LaunchTemplates"][0]
            return None
        except self.ec2_client.exceptions.ClientError:
            return None

    def get_launch_template_version(self, launch_template_id: str, version: str) -> Union[None, Dict]:
        try:
            aws_res = self.ec2_client.describe_launch_template_versions(LaunchTemplateId=launch_template_id, Versions=[version])
            if len(aws_res["LaunchTemplateVersions"]) > 0:
                return aws_res["LaunchTemplateVersions"][0]
            return None
        except self.ec2_client.exceptions.ClientError:
            return None

    def get_managed_prefix_list(self, prefix_list_id: str) -> Union[None, Dict]:
        """Get a managed prefix list by ID."""
        try:
            aws_res = self.ec2_client.describe_managed_prefix_lists(PrefixListIds=[prefix_list_id])
            if len(aws_res["PrefixLists"]) > 0:
                return aws_res["PrefixLists"][0]
            return None
        except self.ec2_client.exceptions.ClientError:
            return None

    def wait_managed_prefix_list_state(self, prefix_list_id: str, expected_state: str, max_wait_seconds: int = 120) -> bool:
        """Wait for a managed prefix list to reach the expected state.

        Args:
            prefix_list_id: The ID of the prefix list to wait for
            expected_state: The expected state (e.g., 'create-complete', 'modify-complete')
            max_wait_seconds: Maximum time to wait in seconds

        Returns:
            True if the expected state was reached, False otherwise
        """
        interval = 5
        max_tries = max_wait_seconds // interval

        for tries in range(max_tries):
            prefix_list = self.get_managed_prefix_list(prefix_list_id)
            if prefix_list is None:
                return False

            current_state = prefix_list.get('State', '')
            if current_state == expected_state:
                return True

            # Check if we're in an error state
            if 'failed' in current_state.lower():
                return False

            time.sleep(interval)

        return False

    def assert_managed_prefix_list(self, prefix_list_id: str, exists=True):
        """Assert that a managed prefix list exists or doesn't exist."""
        res_found = False
        try:
            aws_res = self.ec2_client.describe_managed_prefix_lists(PrefixListIds=[prefix_list_id])
            res_found = len(aws_res["PrefixLists"]) > 0
        except self.ec2_client.exceptions.ClientError:
            pass
        assert res_found is exists

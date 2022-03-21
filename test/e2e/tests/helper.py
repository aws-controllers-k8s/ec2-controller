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

class EC2Validator:
    def __init__(self, ec2_client):
        self.ec2_client = ec2_client

    def assert_dhcp_options(self, dhcp_options_id: str, exists=True):
        res_found = False
        try:
            aws_res = self.ec2_client.describe_dhcp_options(DhcpOptionsIds=[dhcp_options_id])
            res_found = len(aws_res["DhcpOptions"]) > 0
        except self.ec2_client.exceptions.ClientError:
            pass
        assert res_found is exists

    def assert_internet_gateway(self, ig_id: str, exists=True):
        res_found = False
        try:
            aws_res = self.ec2_client.describe_internet_gateways(InternetGatewayIds=[ig_id])
            res_found = len(aws_res["InternetGateways"]) > 0
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

    def assert_route_table(self, route_table_id: str, exists=True):
        res_found = False
        try:
            aws_res = self.ec2_client.describe_route_tables(RouteTableIds=[route_table_id])
            res_found = len(aws_res["RouteTables"]) > 0
        except self.ec2_client.exceptions.ClientError:
            pass
        assert res_found is exists

    def assert_security_group(self, sg_id: str, exists=True):
        res_found = False
        try:
            aws_res = self.ec2_client.describe_security_groups(GroupIds=[sg_id])
            res_found = len(aws_res["SecurityGroups"]) > 0
        except self.ec2_client.exceptions.ClientError:
            pass
        assert res_found is exists

    def assert_subnet(self, subnet_id: str, exists=True):
        res_found = False
        try:
            aws_res = self.ec2_client.describe_subnets(SubnetIds=[subnet_id])
            res_found = len(aws_res["Subnets"]) > 0
        except self.ec2_client.exceptions.ClientError:
            pass
        assert res_found is exists

    def assert_transit_gateway(self, tgw_id: str, exists=True):
        res_found = False
        try:
            aws_res = self.ec2_client.describe_transit_gateways(TransitGatewayIds=[tgw_id])
            res_found = len(aws_res["TransitGateways"]) > 0
        except self.ec2_client.exceptions.ClientError:
            pass
        assert res_found is exists

    def assert_vpc(self, vpc_id: str, exists=True):
        res_found = False
        try:
            aws_res = self.ec2_client.describe_vpcs(VpcIds=[vpc_id])
            res_found = len(aws_res["Vpcs"]) > 0
        except self.ec2_client.exceptions.ClientError:
            pass
        assert res_found is exists

    def assert_vpc_endpoint(self, vpc_endpoint_id: str, exists=True):
        res_found = False
        try:
            aws_res = self.ec2_client.describe_vpc_endpoints(VpcEndpointIds=[vpc_endpoint_id])
            res_found = len(aws_res["VpcEndpoints"]) > 0
        except self.ec2_client.exceptions.ClientError:
            pass
        assert res_found is exists
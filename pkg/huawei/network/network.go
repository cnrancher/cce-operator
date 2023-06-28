package network

import (
	"github.com/cnrancher/cce-operator/pkg/huawei/common"
	"github.com/cnrancher/cce-operator/pkg/utils"
	dns "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dns/v2"
	dns_model "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dns/v2/model"
	dns_region "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dns/v2/region"
	eipv2 "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/eip/v2"
	eipv2_model "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/eip/v2/model"
	eipv2_region "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/eip/v2/region"
	eip "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/eip/v3"
	eip_model "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/eip/v3/model"
	eip_region "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/eip/v3/region"
	vpc "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/vpc/v2"
	vpc_model "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/vpc/v2/model"
	vpc_region "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/vpc/v2/region"
	vpcep "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/vpcep/v1"
	vpcep_model "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/vpcep/v1/model"
	vpcep_region "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/vpcep/v1/region"
)

var (
	DefaultVpcCIDR              = "10.10.0.0/16"
	DefaultSubnetCIDR           = "10.10.2.0/24"
	DefaultSubnetGateway        = "10.10.2.1"
	DefaultContainerNetworkMode = "eni"
	DefaultContainerNetworkCIDR = "10.101.0.0/16"
)

func NewVpcClient(c *common.ClientAuth) *vpc.VpcClient {
	return vpc.NewVpcClient(
		vpc.VpcClientBuilder().
			WithRegion(vpc_region.ValueOf(c.Region)).
			WithCredential(c.Credential).
			Build())
}

func NewEipClient(c *common.ClientAuth) *eip.EipClient {
	return eip.NewEipClient(
		eip.EipClientBuilder().
			WithRegion(eip_region.ValueOf(c.Region)).
			WithCredential(c.Credential).
			Build())
}

func NewEipV2Client(c *common.ClientAuth) *eipv2.EipClient {
	return eipv2.NewEipClient(
		eipv2.EipClientBuilder().
			WithRegion(eipv2_region.ValueOf(c.Region)).
			WithCredential(c.Credential).
			Build())
}

func NewVpcepClient(c *common.ClientAuth) *vpcep.VpcepClient {
	return vpcep.NewVpcepClient(
		vpcep.VpcepClientBuilder().
			WithRegion(vpcep_region.ValueOf(c.Region)).
			WithCredential(c.Credential).
			Build())
}

func NewDnsClient(c *common.ClientAuth) *dns.DnsClient {
	return dns.NewDnsClient(
		dns.DnsClientBuilder().
			WithRegion(dns_region.ValueOf(c.Region)).
			WithCredential(c.Credential).
			Build())
}

func GetPublicIP(client *eip.EipClient, ID string) (*eip_model.ShowPublicipResponse, error) {
	return client.ShowPublicip(&eip_model.ShowPublicipRequest{
		PublicipId: ID,
	})
}

func UpdatePublicIP(client *eipv2.EipClient, ID string) (*eipv2_model.UpdatePublicipResponse, error) {
	return client.UpdatePublicip(&eipv2_model.UpdatePublicipRequest{
		PublicipId: ID,
		Body: &eipv2_model.UpdatePublicipsRequestBody{
			Publicip: &eipv2_model.UpdatePublicipOption{},
		},
	})
}

func GetVPC(client *vpc.VpcClient, ID string) (*vpc_model.ShowVpcResponse, error) {
	return client.ShowVpc(&vpc_model.ShowVpcRequest{
		VpcId: ID,
	})
}

func CreateVPC(client *vpc.VpcClient, name, cidr string) (*vpc_model.CreateVpcResponse, error) {
	return client.CreateVpc(&vpc_model.CreateVpcRequest{
		Body: &vpc_model.CreateVpcRequestBody{
			Vpc: &vpc_model.CreateVpcOption{
				Name: &name,
				Cidr: &cidr,
			},
		},
	})
}

func GetSubnet(client *vpc.VpcClient, ID string) (*vpc_model.ShowSubnetResponse, error) {
	return client.ShowSubnet(&vpc_model.ShowSubnetRequest{
		SubnetId: ID,
	})
}

func CreateSubnet(client *vpc.VpcClient, name, vpcID, pDNS, sDNS string) (*vpc_model.CreateSubnetResponse, error) {
	return client.CreateSubnet(&vpc_model.CreateSubnetRequest{
		Body: &vpc_model.CreateSubnetRequestBody{
			Subnet: &vpc_model.CreateSubnetOption{
				Name:         name,
				Cidr:         DefaultSubnetCIDR,
				GatewayIp:    DefaultSubnetGateway,
				VpcId:        vpcID,
				PrimaryDns:   &pDNS,
				SecondaryDns: &sDNS,
				DhcpEnable:   utils.GetPtr(true),
			},
		},
	})
}

func DeleteVPC(client *vpc.VpcClient, ID string) (*vpc_model.DeleteVpcResponse, error) {
	return client.DeleteVpc(&vpc_model.DeleteVpcRequest{
		VpcId: ID,
	})
}

func DeleteSubnet(
	client *vpc.VpcClient, vpcID string, subnetID string,
) (*vpc_model.DeleteSubnetResponse, error) {
	return client.DeleteSubnet(&vpc_model.DeleteSubnetRequest{
		VpcId:    vpcID,
		SubnetId: subnetID,
	})
}

func GetVpcRoutes(
	client *vpc.VpcClient, VpcID string,
) (*vpc_model.ListVpcRoutesResponse, error) {
	return client.ListVpcRoutes(&vpc_model.ListVpcRoutesRequest{
		VpcId: &VpcID,
	})
}

func GetVpcRoute(client *vpc.VpcClient, RouteID string) (*vpc_model.ShowVpcRouteResponse, error) {
	return client.ShowVpcRoute(&vpc_model.ShowVpcRouteRequest{
		RouteId: RouteID,
	})
}

func DeleteRoute(client *vpc.VpcClient, RouteID string) (*vpc_model.DeleteVpcRouteResponse, error) {
	return client.DeleteVpcRoute(&vpc_model.DeleteVpcRouteRequest{
		RouteId: RouteID,
	})
}

func GetRouteTables(
	client *vpc.VpcClient, RtID, VpcID, SubnetID string,
) (*vpc_model.ListRouteTablesResponse, error) {
	return client.ListRouteTables(&vpc_model.ListRouteTablesRequest{
		Id:       &RtID,
		VpcId:    &VpcID,
		SubnetId: &SubnetID,
	})
}

func GetVpcepServices(
	client *vpcep.VpcepClient, svcID string,
) (*vpcep_model.ListEndpointServiceResponse, error) {
	return client.ListEndpointService(&vpcep_model.ListEndpointServiceRequest{
		Id: &svcID,
	})
}

func GetVpcepService(client *vpcep.VpcepClient, svcID string) (*vpcep_model.ListServiceDetailsResponse, error) {
	return client.ListServiceDetails(&vpcep_model.ListServiceDetailsRequest{
		VpcEndpointServiceId: svcID,
	})
}

func DeleteVpcepService(client *vpcep.VpcepClient, ID string) (*vpcep_model.DeleteEndpointServiceResponse, error) {
	return client.DeleteEndpointService(&vpcep_model.DeleteEndpointServiceRequest{
		VpcEndpointServiceId: ID,
	})
}

func ListNameServers(client *dns.DnsClient, region string) (*dns_model.ListNameServersResponse, error) {
	return client.ListNameServers(&dns_model.ListNameServersRequest{
		Region: &region,
	})
}

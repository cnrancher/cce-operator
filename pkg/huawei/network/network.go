package network

import (
	ccev1 "github.com/cnrancher/cce-operator/pkg/apis/cce.pandaria.io/v1"
	"github.com/cnrancher/cce-operator/pkg/huawei/common"
	"github.com/cnrancher/cce-operator/pkg/utils"
	dns "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dns/v2"
	dns_model "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dns/v2/model"
	dns_region "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dns/v2/region"
	eip "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/eip/v2"
	eip_model "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/eip/v2/model"
	eip_region "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/eip/v2/region"
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

func CreatePublicIP(
	client *eip.EipClient, param *ccev1.CCEClusterPublicIP,
) (*eip_model.CreatePublicipResponse, error) {
	body := &eip_model.CreatePublicipRequestBody{
		Bandwidth: &eip_model.CreatePublicipBandwidthOption{
			ChargeMode: nil, // bandwidth, traffic
			Id:         nil,
			ShareType:  eip_model.GetCreatePublicipBandwidthOptionShareTypeEnum().PER,
			Size:       &param.Eip.Bandwidth.Size,
			Name:       utils.GetPtr(common.GenResourceName("bandwidth")),
		},
		Publicip: &eip_model.CreatePublicipOption{
			Type:  param.Eip.Iptype,
			Alias: utils.GetPtr(common.GenResourceName("eip")),
		},
	}
	var chargeMode eip_model.CreatePublicipBandwidthOptionChargeMode
	switch param.Eip.Bandwidth.ChargeMode {
	case "bandwidth":
		chargeMode = eip_model.GetCreatePublicipBandwidthOptionChargeModeEnum().BANDWIDTH
	case "traffic":
		chargeMode = eip_model.GetCreatePublicipBandwidthOptionChargeModeEnum().TRAFFIC
	default:
		chargeMode = eip_model.GetCreatePublicipBandwidthOptionChargeModeEnum().BANDWIDTH
	}
	body.Bandwidth.ChargeMode = &chargeMode
	var shareType eip_model.CreatePublicipBandwidthOptionShareType
	switch param.Eip.Bandwidth.ShareType {
	case "PER":
		shareType = eip_model.GetCreatePublicipBandwidthOptionShareTypeEnum().PER
	case "WHOLE":
		shareType = eip_model.GetCreatePublicipBandwidthOptionShareTypeEnum().WHOLE
	default:
		shareType = eip_model.GetCreatePublicipBandwidthOptionShareTypeEnum().PER
	}
	body.Bandwidth.ShareType = shareType

	return client.CreatePublicip(&eip_model.CreatePublicipRequest{
		Body: body,
	})
}

func GetPublicIP(client *eip.EipClient, ID string) (*eip_model.ShowPublicipResponse, error) {
	return client.ShowPublicip(&eip_model.ShowPublicipRequest{
		PublicipId: ID,
	})
}

func DeletePublicIP(client *eip.EipClient, ID string) (*eip_model.DeletePublicipResponse, error) {
	return client.DeletePublicip(&eip_model.DeletePublicipRequest{
		PublicipId: ID,
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

func ListSecurityGroups(client *vpc.VpcClient, vpcID string) (*vpc_model.ListSecurityGroupsResponse, error) {
	response, err := client.ListSecurityGroups(&vpc_model.ListSecurityGroupsRequest{
		VpcId: &vpcID,
	})
	return response, err
}

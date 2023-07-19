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
	"github.com/sirupsen/logrus"
)

var (
	DefaultVpcCIDR              = "10.10.0.0/16"
	DefaultSubnetCIDR           = "10.10.0.0/16"
	DefaultSubnetGateway        = "10.10.0.1"
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

	res, err := client.CreatePublicip(&eip_model.CreatePublicipRequest{
		Body: body,
	})
	if err != nil {
		logrus.Debugf("CreatePublicip failed: %v", utils.PrintObject(body))
	}
	return res, err
}

func GetPublicIP(client *eip.EipClient, ID string) (*eip_model.ShowPublicipResponse, error) {
	res, err := client.ShowPublicip(&eip_model.ShowPublicipRequest{
		PublicipId: ID,
	})
	if err != nil {
		logrus.Debugf("ShowPublicip failed: PublicIP ID [%v]", ID)
	}
	return res, err
}

func DeletePublicIP(client *eip.EipClient, ID string) (*eip_model.DeletePublicipResponse, error) {
	res, err := client.DeletePublicip(&eip_model.DeletePublicipRequest{
		PublicipId: ID,
	})
	if err != nil {
		logrus.Debugf("DeletePublicip failed: PublicIP ID [%s]", ID)
	}
	return res, err
}

func GetVPC(client *vpc.VpcClient, ID string) (*vpc_model.ShowVpcResponse, error) {
	res, err := client.ShowVpc(&vpc_model.ShowVpcRequest{
		VpcId: ID,
	})
	if err != nil {
		logrus.Debugf("ShowVpc failed: VPC ID [%s]", ID)
	}
	return res, err
}

func CreateVPC(client *vpc.VpcClient, name, cidr string) (*vpc_model.CreateVpcResponse, error) {
	request := &vpc_model.CreateVpcRequest{
		Body: &vpc_model.CreateVpcRequestBody{
			Vpc: &vpc_model.CreateVpcOption{
				Name: &name,
				Cidr: &cidr,
			},
		},
	}
	res, err := client.CreateVpc(request)
	if err != nil {
		logrus.Debugf("CreateVpc failed: %v", utils.PrintObject(res))
	}
	return res, err
}

func GetSubnet(client *vpc.VpcClient, ID string) (*vpc_model.ShowSubnetResponse, error) {
	res, err := client.ShowSubnet(&vpc_model.ShowSubnetRequest{
		SubnetId: ID,
	})
	if err != nil {
		logrus.Debugf("ShowSubnet failed: subnet ID [%v]", ID)
	}
	return res, err
}

func CreateSubnet(client *vpc.VpcClient, name, vpcID, pDNS, sDNS string) (*vpc_model.CreateSubnetResponse, error) {
	request := &vpc_model.CreateSubnetRequest{
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
	}
	res, err := client.CreateSubnet(request)
	if err != nil {
		logrus.Debugf("CreateSubnet failed: %v", utils.PrintObject(request))
	}
	return res, err
}

func DeleteVPC(client *vpc.VpcClient, ID string) (*vpc_model.DeleteVpcResponse, error) {
	res, err := client.DeleteVpc(&vpc_model.DeleteVpcRequest{
		VpcId: ID,
	})
	if err != nil {
		logrus.Debugf("DeleteVpc failed: VPC ID [%s]", ID)
	}
	return res, err
}

func DeleteSubnet(
	client *vpc.VpcClient, vpcID string, subnetID string,
) (*vpc_model.DeleteSubnetResponse, error) {
	res, err := client.DeleteSubnet(&vpc_model.DeleteSubnetRequest{
		VpcId:    vpcID,
		SubnetId: subnetID,
	})
	if err != nil {
		logrus.Debugf("DeleteSubnet failed: VPC ID [%s], Subnet ID [%s]", vpcID, subnetID)
	}
	return res, err
}

func GetVpcRoutes(
	client *vpc.VpcClient, vpcID string,
) (*vpc_model.ListVpcRoutesResponse, error) {
	res, err := client.ListVpcRoutes(&vpc_model.ListVpcRoutesRequest{
		VpcId: &vpcID,
	})
	if err != nil {
		logrus.Debugf("ListVpcRoutes failed: VPC ID [%s]", vpcID)
	}
	return res, err
}

func GetVpcRoute(client *vpc.VpcClient, RouteID string) (*vpc_model.ShowVpcRouteResponse, error) {
	res, err := client.ShowVpcRoute(&vpc_model.ShowVpcRouteRequest{
		RouteId: RouteID,
	})
	if err != nil {
		logrus.Debugf("ShowVpcRoute failed: RouteID [%s]", RouteID)
	}
	return res, err
}

func DeleteRoute(client *vpc.VpcClient, RouteID string) (*vpc_model.DeleteVpcRouteResponse, error) {
	res, err := client.DeleteVpcRoute(&vpc_model.DeleteVpcRouteRequest{
		RouteId: RouteID,
	})
	if err != nil {
		logrus.Debugf("DeleteVpcRoute failed: RouteID [%s]", RouteID)
	}
	return res, err
}

func GetRouteTables(
	client *vpc.VpcClient, RtID, VpcID, SubnetID string,
) (*vpc_model.ListRouteTablesResponse, error) {
	request := &vpc_model.ListRouteTablesRequest{
		Id:       &RtID,
		VpcId:    &VpcID,
		SubnetId: &SubnetID,
	}
	res, err := client.ListRouteTables(request)
	if err != nil {
		logrus.Debugf("ListRouteTables failed: %v", utils.PrintObject(request))
	}
	return res, err
}

func GetVpcepServices(
	client *vpcep.VpcepClient, svcID string,
) (*vpcep_model.ListEndpointServiceResponse, error) {
	res, err := client.ListEndpointService(&vpcep_model.ListEndpointServiceRequest{
		Id: &svcID,
	})
	if err != nil {
		logrus.Debugf("ListEndpointService failed: service ID %v", svcID)
	}
	return res, err
}

func GetVpcepService(client *vpcep.VpcepClient, svcID string) (*vpcep_model.ListServiceDetailsResponse, error) {
	res, err := client.ListServiceDetails(&vpcep_model.ListServiceDetailsRequest{
		VpcEndpointServiceId: svcID,
	})
	if err != nil {
		logrus.Debugf("ListServiceDetails failed: service ID %v", svcID)
	}
	return res, err
}

func DeleteVpcepService(client *vpcep.VpcepClient, ID string) (*vpcep_model.DeleteEndpointServiceResponse, error) {
	res, err := client.DeleteEndpointService(&vpcep_model.DeleteEndpointServiceRequest{
		VpcEndpointServiceId: ID,
	})
	if err != nil {
		logrus.Debugf("DeleteEndpointService failed: ID %v", ID)
	}
	return res, err
}

func ListNameServers(client *dns.DnsClient, region string) (*dns_model.ListNameServersResponse, error) {
	res, err := client.ListNameServers(&dns_model.ListNameServersRequest{
		Region: &region,
	})
	if err != nil {
		logrus.Debugf("ListNameServers failed: Region %q", region)
	}
	return res, err
}

func ListSecurityGroups(client *vpc.VpcClient, vpcID string) (*vpc_model.ListSecurityGroupsResponse, error) {
	res, err := client.ListSecurityGroups(&vpc_model.ListSecurityGroupsRequest{
		VpcId: &vpcID,
	})
	if err != nil {
		logrus.Debugf("ListSecurityGroups failed: vpc ID %q", vpcID)
	}
	return res, err
}

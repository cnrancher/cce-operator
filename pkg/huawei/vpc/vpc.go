package vpc

import (
	"github.com/cnrancher/cce-operator/pkg/huawei/common"
	"github.com/cnrancher/cce-operator/pkg/utils"
	vpc "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/vpc/v2"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/vpc/v2/model"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/vpc/v2/region"
	"github.com/sirupsen/logrus"
)

var (
	DefaultVpcCIDR       = "10.224.0.0/16"
	DefaultSubnetCIDR    = "10.224.0.0/16"
	DefaultSubnetGateway = "10.224.0.1"
)

func NewVpcClient(c *common.ClientAuth) *vpc.VpcClient {
	return vpc.NewVpcClient(
		vpc.VpcClientBuilder().
			WithRegion(region.ValueOf(c.Region)).
			WithCredential(c.Credential).
			Build())
}

func ShowVPC(client *vpc.VpcClient, ID string) (*model.ShowVpcResponse, error) {
	res, err := client.ShowVpc(&model.ShowVpcRequest{
		VpcId: ID,
	})
	if err != nil {
		logrus.Debugf("ShowVpc failed: VPC ID [%s]", ID)
	}
	return res, err
}

func CreateVPC(client *vpc.VpcClient, name, cidr string) (*model.CreateVpcResponse, error) {
	request := &model.CreateVpcRequest{
		Body: &model.CreateVpcRequestBody{
			Vpc: &model.CreateVpcOption{
				Name:        &name,
				Cidr:        &cidr,
				Description: &common.DefaultResourceDescription,
			},
		},
	}
	res, err := client.CreateVpc(request)
	if err != nil {
		logrus.Debugf("CreateVpc failed: %v", utils.PrintObject(res))
	}
	return res, err
}

func DeleteVPC(client *vpc.VpcClient, ID string) (*model.DeleteVpcResponse, error) {
	res, err := client.DeleteVpc(&model.DeleteVpcRequest{
		VpcId: ID,
	})
	if err != nil {
		logrus.Debugf("DeleteVpc failed: VPC ID [%s]", ID)
	}
	return res, err
}

func GetVpcRoutes(
	client *vpc.VpcClient, vpcID string,
) (*model.ListVpcRoutesResponse, error) {
	res, err := client.ListVpcRoutes(&model.ListVpcRoutesRequest{
		VpcId: &vpcID,
	})
	if err != nil {
		logrus.Debugf("ListVpcRoutes failed: VPC ID [%s]", vpcID)
	}
	return res, err
}

func ShowVpcRoute(client *vpc.VpcClient, RouteID string) (*model.ShowVpcRouteResponse, error) {
	res, err := client.ShowVpcRoute(&model.ShowVpcRouteRequest{
		RouteId: RouteID,
	})
	if err != nil {
		logrus.Debugf("ShowVpcRoute failed: RouteID [%s]", RouteID)
	}
	return res, err
}

func DeleteVpcRoute(client *vpc.VpcClient, RouteID string) (*model.DeleteVpcRouteResponse, error) {
	res, err := client.DeleteVpcRoute(&model.DeleteVpcRouteRequest{
		RouteId: RouteID,
	})
	if err != nil {
		logrus.Debugf("DeleteVpcRoute failed: RouteID [%s]", RouteID)
	}
	return res, err
}

func ListRouteTables(
	client *vpc.VpcClient, RtID, VpcID, SubnetID string,
) (*model.ListRouteTablesResponse, error) {
	request := &model.ListRouteTablesRequest{
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

func ListSecurityGroups(client *vpc.VpcClient, vpcID string) (*model.ListSecurityGroupsResponse, error) {
	res, err := client.ListSecurityGroups(&model.ListSecurityGroupsRequest{
		VpcId: &vpcID,
	})
	if err != nil {
		logrus.Debugf("ListSecurityGroups failed: vpc ID %q", vpcID)
	}
	return res, err
}

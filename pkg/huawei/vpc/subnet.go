package vpc

import (
	"github.com/cnrancher/cce-operator/pkg/utils"
	vpc "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/vpc/v2"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/vpc/v2/model"
	"github.com/sirupsen/logrus"
)

func ShowSubnet(client *vpc.VpcClient, ID string) (*model.ShowSubnetResponse, error) {
	res, err := client.ShowSubnet(&model.ShowSubnetRequest{
		SubnetId: ID,
	})
	if err != nil {
		logrus.Debugf("ShowSubnet failed: subnet ID [%v]", ID)
	}
	return res, err
}

func CreateSubnet(client *vpc.VpcClient, name, vpcID, pDNS, sDNS string) (*model.CreateSubnetResponse, error) {
	request := &model.CreateSubnetRequest{
		Body: &model.CreateSubnetRequestBody{
			Subnet: &model.CreateSubnetOption{
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

func DeleteSubnet(
	client *vpc.VpcClient, vpcID string, subnetID string,
) (*model.DeleteSubnetResponse, error) {
	res, err := client.DeleteSubnet(&model.DeleteSubnetRequest{
		VpcId:    vpcID,
		SubnetId: subnetID,
	})
	if err != nil {
		logrus.Debugf("DeleteSubnet failed: VPC ID [%s], Subnet ID [%s]", vpcID, subnetID)
	}
	return res, err
}

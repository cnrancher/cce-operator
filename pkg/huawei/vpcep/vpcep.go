package vpcep

import (
	"github.com/cnrancher/cce-operator/pkg/huawei/common"
	vpcep "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/vpcep/v1"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/vpcep/v1/model"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/vpcep/v1/region"
	"github.com/sirupsen/logrus"
)

func NewVpcepClient(c *common.ClientAuth) *vpcep.VpcepClient {
	return vpcep.NewVpcepClient(
		vpcep.VpcepClientBuilder().
			WithRegion(region.ValueOf(c.Region)).
			WithCredential(c.Credential).
			Build())
}

func ListEndpointService(
	client *vpcep.VpcepClient, svcID string,
) (*model.ListEndpointServiceResponse, error) {
	res, err := client.ListEndpointService(&model.ListEndpointServiceRequest{
		Id: &svcID,
	})
	if err != nil {
		logrus.Debugf("ListEndpointService failed: service ID %v", svcID)
	}
	return res, err
}

func ListServiceDetails(client *vpcep.VpcepClient, svcID string) (*model.ListServiceDetailsResponse, error) {
	res, err := client.ListServiceDetails(&model.ListServiceDetailsRequest{
		VpcEndpointServiceId: svcID,
	})
	if err != nil {
		logrus.Debugf("ListServiceDetails failed: service ID %v", svcID)
	}
	return res, err
}

func DeleteVpcepService(client *vpcep.VpcepClient, ID string) (*model.DeleteEndpointServiceResponse, error) {
	res, err := client.DeleteEndpointService(&model.DeleteEndpointServiceRequest{
		VpcEndpointServiceId: ID,
	})
	if err != nil {
		logrus.Debugf("DeleteEndpointService failed: ID %v", ID)
	}
	return res, err
}

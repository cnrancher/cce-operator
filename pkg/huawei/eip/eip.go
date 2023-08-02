package eip

import (
	ccev1 "github.com/cnrancher/cce-operator/pkg/apis/cce.pandaria.io/v1"
	"github.com/cnrancher/cce-operator/pkg/huawei/common"
	"github.com/cnrancher/cce-operator/pkg/utils"
	eip "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/eip/v2"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/eip/v2/model"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/eip/v2/region"
	"github.com/sirupsen/logrus"
)

func NewEipClient(c *common.ClientAuth) *eip.EipClient {
	return eip.NewEipClient(
		eip.EipClientBuilder().
			WithRegion(region.ValueOf(c.Region)).
			WithCredential(c.Credential).
			Build())
}

func CreatePublicIP(
	client *eip.EipClient, param *ccev1.CCEEip,
) (*model.CreatePublicipResponse, error) {
	body := &model.CreatePublicipRequestBody{
		Bandwidth: &model.CreatePublicipBandwidthOption{
			ChargeMode: nil, // bandwidth, traffic
			Id:         nil,
			ShareType:  model.GetCreatePublicipBandwidthOptionShareTypeEnum().PER,
			Size:       &param.Bandwidth.Size,
			Name:       utils.GetPtr(common.GenResourceName("bandwidth")),
		},
		Publicip: &model.CreatePublicipOption{
			Type:  param.Iptype,
			Alias: utils.GetPtr(common.GenResourceName("eip")),
		},
	}
	var chargeMode model.CreatePublicipBandwidthOptionChargeMode
	switch param.Bandwidth.ChargeMode {
	case "bandwidth":
		chargeMode = model.GetCreatePublicipBandwidthOptionChargeModeEnum().BANDWIDTH
	case "traffic":
		chargeMode = model.GetCreatePublicipBandwidthOptionChargeModeEnum().TRAFFIC
	default:
		chargeMode = model.GetCreatePublicipBandwidthOptionChargeModeEnum().BANDWIDTH
	}
	body.Bandwidth.ChargeMode = &chargeMode
	var shareType model.CreatePublicipBandwidthOptionShareType
	switch param.Bandwidth.ShareType {
	case "PER":
		shareType = model.GetCreatePublicipBandwidthOptionShareTypeEnum().PER
	case "WHOLE":
		shareType = model.GetCreatePublicipBandwidthOptionShareTypeEnum().WHOLE
	default:
		shareType = model.GetCreatePublicipBandwidthOptionShareTypeEnum().PER
	}
	body.Bandwidth.ShareType = shareType

	res, err := client.CreatePublicip(&model.CreatePublicipRequest{
		Body: body,
	})
	if err != nil {
		logrus.Debugf("CreatePublicip failed: %v", utils.PrintObject(body))
	}
	return res, err
}

func ShowPublicip(client *eip.EipClient, ID string) (*model.ShowPublicipResponse, error) {
	res, err := client.ShowPublicip(&model.ShowPublicipRequest{
		PublicipId: ID,
	})
	if err != nil {
		logrus.Debugf("ShowPublicip failed: PublicIP ID [%v]", ID)
	}
	return res, err
}

func DeletePublicIP(client *eip.EipClient, ID string) (*model.DeletePublicipResponse, error) {
	res, err := client.DeletePublicip(&model.DeletePublicipRequest{
		PublicipId: ID,
	})
	if err != nil {
		logrus.Debugf("DeletePublicip failed: PublicIP ID [%s]", ID)
	}
	return res, err
}

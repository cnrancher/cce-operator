package elb

import (
	"github.com/cnrancher/cce-operator/pkg/huawei/common"
	"github.com/cnrancher/cce-operator/pkg/utils"
	cce_model "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cce/v3/model"
	elb "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/elb/v2"
	elb_model "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/elb/v2/model"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/elb/v2/region"
	"github.com/sirupsen/logrus"
)

func NewElbClient(c *common.ClientAuth) *elb.ElbClient {
	client := elb.NewElbClient(
		elb.ElbClientBuilder().
			WithRegion(region.ValueOf(c.Region)).
			WithCredential(c.Credential).
			Build())

	return client
}

func CreateELB(
	client *elb.ElbClient, name, desc, subnetID string,
) (*elb_model.CreateLoadbalancerResponse, error) {
	request := &elb_model.CreateLoadbalancerRequest{
		Body: &elb_model.CreateLoadbalancerRequestBody{
			Loadbalancer: &elb_model.CreateLoadbalancerReq{
				Name:        &name,
				Description: &desc,
				VipSubnetId: subnetID,
			},
		},
	}
	res, err := client.CreateLoadbalancer(request)
	if err != nil {
		logrus.Debugf("CreateLoadbalancer failed: %v", utils.PrintObject(res))
	}
	return res, err
}

func GetLoadBalancer(client *elb.ElbClient, ID string) (*elb_model.ShowLoadbalancerResponse, error) {
	request := &elb_model.ShowLoadbalancerRequest{
		LoadbalancerId: ID,
	}
	res, err := client.ShowLoadbalancer(request)
	if err != nil {
		logrus.Debugf("ShowLoadbalancer failed: %v", utils.PrintObject(request))
	}
	return res, err
}

func ListListeners(client *elb.ElbClient) (*elb_model.ListListenersResponse, error) {
	request := &elb_model.ListListenersRequest{
		Limit: utils.Pointer(int32(1000)),
	}
	res, err := client.ListListeners(request)
	if err != nil {
		logrus.Debugf("ListListeners failed: %v", utils.PrintObject(request))
	}
	return res, err
}

func UpdateListener(client *elb.ElbClient, ID string) (*elb_model.UpdateListenerResponse, error) {
	request := &elb_model.UpdateListenerRequest{
		ListenerId: ID,
		Body: &elb_model.UpdateListenerRequestBody{
			Listener: &elb_model.UpdateListenerReq{},
		},
	}
	res, err := client.UpdateListener(request)
	if err != nil {
		logrus.Debugf("UpdateListener failed: %v", utils.PrintObject(request))
	}
	return res, err
}

func DeleteListener(client *elb.ElbClient, ID string) (*elb_model.DeleteListenerResponse, error) {
	request := &elb_model.DeleteListenerRequest{
		ListenerId: ID,
	}
	res, err := client.DeleteListener(request)
	if err != nil {
		logrus.Debugf("DeleteListener failed: %v", utils.PrintObject(request))
	}
	return res, err
}

func CreateListener(client *elb.ElbClient, ELBID, name, desc string) (*elb_model.CreateListenerResponse, error) {
	request := &elb_model.CreateListenerRequest{
		Body: &elb_model.CreateListenerRequestBody{
			Listener: &elb_model.CreateListenerReq{
				LoadbalancerId: ELBID,
				Protocol:       elb_model.GetCreateListenerReqProtocolEnum().TCP,
				ProtocolPort:   5443,
				Name:           &name,
				Description:    &desc,
			},
		},
	}
	resp, err := client.CreateListener(request)
	if err != nil {
		logrus.Debugf("CreateListener failed: %v", utils.PrintObject(request))
	}
	return resp, nil
}

func AddBackends(
	client *elb.ElbClient, listerID, elbID, subnetID, poolID string, backends *[]cce_model.Node,
) (*elb_model.CreatePoolResponse, error) {
	request := &elb_model.CreatePoolRequest{
		Body: &elb_model.CreatePoolRequestBody{
			Pool: &elb_model.CreatePoolReq{
				Protocol:       elb_model.GetCreatePoolReqProtocolEnum().TCP,
				LbAlgorithm:    "ROUND_ROBIN",
				ListenerId:     &listerID,
				LoadbalancerId: &elbID,
			},
		},
	}
	backendGroup, err := client.CreatePool(request)
	if err != nil {
		return nil, err
	}
	for _, backend := range *backends {
		createMemberReq := &elb_model.CreateMemberRequest{
			Body: &elb_model.CreateMemberRequestBody{
				Member: &elb_model.CreateMemberReq{
					Address:      *backend.Status.PrivateIP,
					ProtocolPort: 3389,
					SubnetId:     subnetID,
				},
			},
			PoolId: poolID,
		}
		if _, err = client.CreateMember(createMemberReq); err != nil {
			return nil, err
		}
	}
	return backendGroup, err
}

func ShowPool(client *elb.ElbClient, ID string) (*elb_model.ShowPoolResponse, error) {
	request := &elb_model.ShowPoolRequest{
		PoolId: ID,
	}
	response, err := client.ShowPool(request)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func DeleteHealthcheck(client *elb.ElbClient, ID string) (*elb_model.DeleteHealthmonitorResponse, error) {
	request := &elb_model.DeleteHealthmonitorRequest{
		HealthmonitorId: ID,
	}
	return client.DeleteHealthmonitor(request)
}

func DeleteMember(client *elb.ElbClient, poolID string, memberID string) (*elb_model.DeleteMemberResponse, error) {
	request := &elb_model.DeleteMemberRequest{
		PoolId:   poolID,
		MemberId: memberID,
	}
	return client.DeleteMember(request)
}

func DeletePool(client *elb.ElbClient, ID string) (*elb_model.DeletePoolResponse, error) {
	request := &elb_model.DeletePoolRequest{
		PoolId: ID,
	}
	return client.DeletePool(request)
}

func DeleteLoadBalancer(client *elb.ElbClient, ID string) (*elb_model.DeleteLoadbalancerResponse, error) {
	request := &elb_model.DeleteLoadbalancerRequest{
		LoadbalancerId: ID,
	}
	return client.DeleteLoadbalancer(request)
}

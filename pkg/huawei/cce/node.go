package cce

import (
	ccev1 "github.com/cnrancher/cce-operator/pkg/apis/cce.pandaria.io/v1"
	"github.com/cnrancher/cce-operator/pkg/utils"
	cce "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cce/v3"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cce/v3/model"
	"github.com/sirupsen/logrus"
)

const (
	NodePoolIDAnnotationKey = "kubernetes.io/node-pool.id"
)

func CreateNodePool(
	client *cce.CceClient, clusterID string, nodePool *ccev1.CCENodePool,
) (*model.CreateNodePoolResponse, error) {
	req, err := GetCreateNodePoolRequest(clusterID, nodePool)
	if err != nil {
		return nil, err
	}
	res, err := client.CreateNodePool(req)
	if err != nil {
		logrus.Debugf("CreateNodePool failed: %v", utils.PrintObject(req))
	}
	return res, err
}

func ListNodes(client *cce.CceClient, clusterID string) (*model.ListNodesResponse, error) {
	request := &model.ListNodesRequest{
		ClusterId: clusterID,
	}
	res, err := client.ListNodes(request)
	if err != nil {
		logrus.Debugf("ListNodes failed: %v", utils.PrintObject(request))
	}
	return res, err
}

func ListNodePools(
	client *cce.CceClient, clusterID string, showDefaultNP bool,
) (*model.ListNodePoolsResponse, error) {
	var sdnp *string
	if showDefaultNP {
		sdnp = utils.Pointer("true")
	}
	res, err := client.ListNodePools(&model.ListNodePoolsRequest{
		ClusterId:           clusterID,
		ShowDefaultNodePool: sdnp,
	})
	if err != nil {
		logrus.Debugf("ListNodePools failed: clusterID [%s]", clusterID)
	}
	return res, err
}

func ShowNode(
	client *cce.CceClient, clusterID, nodeID string,
) (*model.ShowNodeResponse, error) {
	request := &model.ShowNodeRequest{
		ClusterId: clusterID,
		NodeId:    nodeID,
	}
	res, err := client.ShowNode(request)
	if err != nil {
		logrus.Debugf("ShowNode failed: %v", utils.PrintObject(request))
	}
	return res, err
}

func ShowNodePool(
	client *cce.CceClient, clusterID, npID string,
) (*model.ShowNodePoolResponse, error) {
	res, err := client.ShowNodePool(&model.ShowNodePoolRequest{
		ClusterId:  clusterID,
		NodepoolId: npID,
	})
	if err != nil {
		logrus.Debugf("ShowNodePool failed: clusterID [%s], nodePoolID [%s]",
			clusterID, npID)
	}
	return res, err
}

func UpdateNodePool(
	client *cce.CceClient, clusterID string, nodePool *ccev1.CCENodePool,
) (*model.UpdateNodePoolResponse, error) {
	req := GetUpdateNodePoolRequest(clusterID, nodePool)
	res, err := client.UpdateNodePool(req)
	if err != nil {
		logrus.Debugf("UpdateNodePool failed: %v",
			utils.PrintObject(req))
	}
	return res, err
}

func GetUpdateNodePoolRequest(
	clusterID string, nodePool *ccev1.CCENodePool,
) *model.UpdateNodePoolRequest {
	req := &model.UpdateNodePoolRequest{
		ClusterId:  clusterID,
		NodepoolId: nodePool.ID,
		Body: &model.NodePoolUpdate{
			Metadata: &model.NodePoolMetadataUpdate{
				Name: nodePool.Name,
			},
			Spec: &model.NodePoolSpecUpdate{
				NodeTemplate:     &model.NodeSpecUpdate{},
				InitialNodeCount: nodePool.InitialNodeCount,
				Autoscaling: &model.NodePoolNodeAutoscaling{
					Enable:                &nodePool.Autoscaling.Enable,
					MinNodeCount:          &nodePool.Autoscaling.MinNodeCount,
					MaxNodeCount:          &nodePool.Autoscaling.MaxNodeCount,
					ScaleDownCooldownTime: &nodePool.Autoscaling.ScaleDownCooldownTime,
					Priority:              &nodePool.Autoscaling.Priority,
				},
			},
		},
	}
	return req
}

func DeleteNode(
	client *cce.CceClient, clusterID string, nodeID string,
) (*model.DeleteNodeResponse, error) {
	res, err := client.DeleteNode(&model.DeleteNodeRequest{
		ClusterId: clusterID,
		NodeId:    nodeID,
	})
	if err != nil {
		logrus.Debugf("DeleteNode failed: %v", utils.PrintObject(res))
	}
	return res, err
}

func DeleteNodePool(
	client *cce.CceClient, clusterID, npID string,
) (*model.DeleteNodePoolResponse, error) {
	res, err := client.DeleteNodePool(&model.DeleteNodePoolRequest{
		ClusterId:  clusterID,
		NodepoolId: npID,
	})
	if err != nil {
		logrus.Debugf("DeleteNodePool failed: %v", utils.PrintObject(res))
	}
	return res, err
}

func GetCreateNodePoolRequest(
	clusterID string, np *ccev1.CCENodePool,
) (*model.CreateNodePoolRequest, error) {
	nodePoolBody := &model.NodePool{
		Kind:       "NodePool",
		ApiVersion: "v3",
		Metadata: &model.NodePoolMetadata{
			Name: np.Name,
		},
		Spec: &model.NodePoolSpec{
			NodeTemplate: &model.NodeSpec{
				Flavor: np.NodeTemplate.Flavor,
				Az:     np.NodeTemplate.AvailableZone,
				Os:     &np.NodeTemplate.OperatingSystem,
				Login: &model.Login{
					SshKey: &np.NodeTemplate.SSHKey,
				},
				RootVolume: &model.Volume{
					Size:       np.NodeTemplate.RootVolume.Size,
					Volumetype: np.NodeTemplate.RootVolume.Type,
				},
				DataVolumes: make([]model.Volume, 0, len(np.NodeTemplate.DataVolumes)),
				PublicIP:    &model.NodePublicIp{},
				Count:       utils.Pointer(int32(1)),
				BillingMode: &np.NodeTemplate.BillingMode,
				ExtendParam: &model.NodeExtendParam{
					PeriodType:  &np.NodeTemplate.ExtendParam.PeriodType,
					PeriodNum:   &np.NodeTemplate.ExtendParam.PeriodNum,
					IsAutoRenew: &np.NodeTemplate.ExtendParam.IsAutoRenew,
				},
			},
			InitialNodeCount: &np.InitialNodeCount,
			Autoscaling: &model.NodePoolNodeAutoscaling{
				Enable:                &np.Autoscaling.Enable,
				MinNodeCount:          &np.Autoscaling.MinNodeCount,
				MaxNodeCount:          &np.Autoscaling.MaxNodeCount,
				ScaleDownCooldownTime: &np.Autoscaling.ScaleDownCooldownTime,
				Priority:              &np.Autoscaling.Priority,
			},
		},
	}
	var npType model.NodePoolSpecType
	switch np.Type {
	case "vm":
		npType = model.GetNodePoolSpecTypeEnum().VM
	case "pm":
		npType = model.GetNodePoolSpecTypeEnum().PM
	case "ElasticBMS":
		npType = model.GetNodePoolSpecTypeEnum().ELASTIC_BMS
	default:
		npType = model.GetNodePoolSpecTypeEnum().VM
	}
	nodePoolBody.Spec.Type = &npType
	for _, dv := range np.NodeTemplate.DataVolumes {
		nodePoolBody.Spec.NodeTemplate.DataVolumes = append(nodePoolBody.Spec.NodeTemplate.DataVolumes,
			model.Volume{
				Size:       dv.Size,
				Volumetype: dv.Type,
			},
		)
	}

	if len(np.NodeTemplate.PublicIP.IDs) > 0 {
		nodePoolBody.Spec.NodeTemplate.PublicIP.Ids = &np.NodeTemplate.PublicIP.IDs
	}
	chargeMode := "traffic"
	if np.NodeTemplate.PublicIP.Eip.Bandwidth.ChargeMode != "traffic" {
		chargeMode = ""
	}
	if np.NodeTemplate.PublicIP.Count > 0 {
		nodePoolBody.Spec.NodeTemplate.PublicIP.Count = &np.NodeTemplate.PublicIP.Count
		nodePoolBody.Spec.NodeTemplate.PublicIP.Eip = &model.NodeEipSpec{
			Iptype: np.NodeTemplate.PublicIP.Eip.Iptype,
			Bandwidth: &model.NodeBandwidth{
				Chargemode: &chargeMode,
				Size:       &np.NodeTemplate.PublicIP.Eip.Bandwidth.Size,
				Sharetype:  &np.NodeTemplate.PublicIP.Eip.Bandwidth.ShareType,
			},
		}
	}
	var runtime model.RuntimeName
	switch np.NodeTemplate.Runtime {
	case "docker":
		runtime = model.GetRuntimeNameEnum().DOCKER
	case "containerd":
		runtime = model.GetRuntimeNameEnum().CONTAINERD
	default:
		runtime = model.GetRuntimeNameEnum().DOCKER
	}
	nodePoolBody.Spec.NodeTemplate.Runtime = &model.Runtime{
		Name: &runtime,
	}
	if len(np.CustomSecurityGroups) > 0 {
		nodePoolBody.Spec.CustomSecurityGroups = &np.CustomSecurityGroups
	}
	request := &model.CreateNodePoolRequest{
		ClusterId: clusterID,
		Body:      nodePoolBody,
	}
	return request, nil
}

package cce

import (
	ccev1 "github.com/cnrancher/cce-operator/pkg/apis/cce.pandaria.io/v1"
	"github.com/cnrancher/cce-operator/pkg/utils"
	cce "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cce/v3"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cce/v3/model"
	"github.com/sirupsen/logrus"
)

func CreateNodePool(
	client *cce.CceClient, clusterID string, nodePool *ccev1.NodePool,
) (*model.CreateNodePoolResponse, error) {
	createNodePoolReq, err := getNodePoolRequirement(clusterID, nodePool)
	if err != nil {
		return nil, err
	}
	cnpr, err := client.CreateNodePool(createNodePoolReq)
	if err != nil {
		logrus.Debugf("createNodePool failed, request: %v", utils.PrintObject(createNodePoolReq))
	}
	return cnpr, err
}

func GetClusterNodes(client *cce.CceClient, clusterID string) (*model.ListNodesResponse, error) {
	request := &model.ListNodesRequest{
		ClusterId: clusterID,
	}
	return client.ListNodes(request)
}

func GetClusterNodePools(
	client *cce.CceClient, clusterID string, showDefaultNP bool,
) (*model.ListNodePoolsResponse, error) {
	var sdnp *string
	if showDefaultNP {
		sdnp = utils.GetPtr("true")
	}
	return client.ListNodePools(&model.ListNodePoolsRequest{
		ClusterId:           clusterID,
		ShowDefaultNodePool: sdnp,
	})
}

func GetNode(
	client *cce.CceClient, clusterID, nodeID string,
) (*model.ShowNodeResponse, error) {
	request := &model.ShowNodeRequest{
		ClusterId: clusterID,
		NodeId:    nodeID,
	}
	return client.ShowNode(request)
}

func GetNodePool(
	client *cce.CceClient, clusterID, npID string,
) (*model.ShowNodePoolResponse, error) {
	return client.ShowNodePool(&model.ShowNodePoolRequest{
		ClusterId:  clusterID,
		NodepoolId: npID,
	})
}

func DeleteNode(
	client *cce.CceClient, clusterID string, nodeID string,
) (*model.DeleteNodeResponse, error) {
	res, err := client.DeleteNode(&model.DeleteNodeRequest{
		ClusterId: clusterID,
		NodeId:    nodeID,
	})
	if err != nil {
		logrus.Debugf("failed to delete node, request: %v", utils.PrintObject(res))
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
		logrus.Debugf("failed to delete node pool, request: %v", utils.PrintObject(res))
	}
	return res, err
}

func getNodePoolRequirement(
	clusterID string, np *ccev1.NodePool,
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
				Count:       &np.NodeTemplate.Count,
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

	if len(np.NodeTemplate.PublicIP.Ids) > 0 {
		nodePoolBody.Spec.NodeTemplate.PublicIP.Ids = &np.NodeTemplate.PublicIP.Ids
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

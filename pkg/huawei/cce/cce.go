package cce

import (
	"fmt"

	ccev1 "github.com/cnrancher/cce-operator/pkg/apis/cce.pandaria.io/v1"
	"github.com/cnrancher/cce-operator/pkg/huawei/common"
	"github.com/cnrancher/cce-operator/pkg/utils"
	huawei_utils "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/utils"
	cce "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cce/v3"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cce/v3/model"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cce/v3/region"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	ClusterStatusAvailable      = "Available"      // 可用
	ClusterStatusUnavailable    = "Unavailable"    // 不可用，集群异常，需手动删除
	ClusterStatusScalingUp      = "ScalingUp"      // 扩容中
	ClusterStatusScalingDown    = "ScalingDown"    // 缩容中
	ClusterStatusCreating       = "Creating"       // 创建中
	ClusterStatusDeleting       = "Deleting"       // 删除中
	ClusterStatusUpgrading      = "Upgrading"      // 升级中
	ClusterStatusResizing       = "Resizing"       // 规格变更中
	ClusterStatusRollingBack    = "RollingBack"    // 回滚中
	ClusterStatusRollbackFailed = "RollbackFailed" // 回滚异常
	ClusterStatusEmpty          = "Empty"          // 集群无任何资源
)

func NewCCEClient(auth *common.ClientAuth) *cce.CceClient {
	return cce.NewCceClient(
		cce.CceClientBuilder().
			WithRegion(region.ValueOf(auth.Region)).
			WithCredential(auth.Credential).
			Build())
}

func CreateCluster(
	client *cce.CceClient, config *ccev1.CCEClusterConfig,
) (*model.CreateClusterResponse, error) {
	clusterReq := common.GetClusterRequestFromCCECCSpec(config)
	return client.CreateCluster(clusterReq)
}

func GetCluster(client *cce.CceClient, ID string) (*model.ShowClusterResponse, error) {
	return client.ShowCluster(&model.ShowClusterRequest{
		ClusterId: ID,
	})
}

func ListClusters(client *cce.CceClient) (*model.ListClustersResponse, error) {
	return client.ListClusters(&model.ListClustersRequest{})
}

func DeleteCluster(client *cce.CceClient, ID string) (*model.DeleteClusterResponse, error) {
	return client.DeleteCluster(&model.DeleteClusterRequest{
		ClusterId: ID,
	})
}

func CreateNode(
	client *cce.CceClient, clusterID string, nodeConfig *ccev1.NodeConfig,
) (*model.CreateNodeResponse, error) {
	createNodeReq, err := getNodeRequirement(clusterID, nodeConfig)
	if err != nil {
		return nil, err
	}
	createNodeRes, err := client.CreateNode(createNodeReq)
	if err != nil {
		logrus.Warnf("error: %v, retry to create node for cluster %q...",
			err, clusterID)
		createNodeRes, err = client.CreateNode(createNodeReq)
		if err != nil {
			return nil, fmt.Errorf("failed to create node(s) for cluster: %w", err)
		}
	}
	return createNodeRes, nil
}

func GetClusterNodes(client *cce.CceClient, clusterID string) (*model.ListNodesResponse, error) {
	request := &model.ListNodesRequest{
		ClusterId: clusterID,
	}
	return client.ListNodes(request)
}

func GetNode(client *cce.CceClient, clusterID, nodeID string) (*model.ShowNodeResponse, error) {
	request := &model.ShowNodeRequest{
		ClusterId: clusterID,
		NodeId:    nodeID,
	}
	return client.ShowNode(request)
}

func DeleteNode(
	client *cce.CceClient, clusterID string, nodeID string,
) (*model.DeleteNodeResponse, error) {
	return client.DeleteNode(&model.DeleteNodeRequest{
		ClusterId: clusterID,
		NodeId:    nodeID,
	})
}

func getNodeRequirement(
	clusterID string, nc *ccev1.NodeConfig,
) (*model.CreateNodeRequest, error) {
	nodeCreateReq := &model.NodeCreateRequest{
		Kind:       "Node",
		ApiVersion: "v3",
		Metadata: &model.NodeMetadata{
			Name:   utils.GetPtr(common.GenResourceName("node")),
			Labels: nc.Labels,
		},
		Spec: &model.NodeSpec{
			Flavor: nc.Flavor,
			Az:     nc.AvailableZone,
			Os:     &nc.OperatingSystem,
			Login: &model.Login{
				SshKey:       &nc.SSHKey,
				UserPassword: &model.UserPassword{},
			},
			RootVolume: &model.Volume{
				Size:       nc.RootVolume.Size,
				Volumetype: nc.RootVolume.Type,
			},
			DataVolumes: []model.Volume{},
			PublicIP:    &model.NodePublicIp{},
			Count:       utils.GetPtr(int32(nc.Count)),
			BillingMode: utils.GetPtr(int32(nc.BillingMode)),
			ExtendParam: nil,
		},
	}
	for _, dv := range nc.DataVolumes {
		nodeCreateReq.Spec.DataVolumes = append(nodeCreateReq.Spec.DataVolumes, model.Volume{
			Size:       dv.Size,
			Volumetype: dv.Type,
		})
	}
	extendParam := &model.NodeExtendParam{
		PeriodType:  &nc.ExtendParam.BMSPeriodType,
		PeriodNum:   utils.GetPtr(int32(nc.ExtendParam.BMSPeriodNum)),
		IsAutoRenew: &nc.ExtendParam.BMSIsAutoRenew,
	}

	if nc.ExtendParam.BMSPeriodType != "" &&
		nc.ExtendParam.BMSPeriodNum != 0 &&
		nc.ExtendParam.BMSIsAutoRenew != "" {
		nodeCreateReq.Spec.ExtendParam = extendParam
	}

	if len(nc.PublicIP.Ids) > 0 {
		nodeCreateReq.Spec.PublicIP.Ids = &nc.PublicIP.Ids
	}
	chargeMode := "traffic"
	if nc.PublicIP.Eip.Bandwidth.ChargeMode != "traffic" {
		chargeMode = ""
	}
	if nc.PublicIP.Count > 0 {
		nodeCreateReq.Spec.PublicIP.Count = utils.GetPtr(int32(nc.PublicIP.Count))
		nodeCreateReq.Spec.PublicIP.Eip = &model.NodeEipSpec{
			Iptype: nc.PublicIP.Eip.Iptype,
			Bandwidth: &model.NodeBandwidth{
				Chargemode: &chargeMode,
				Size:       utils.GetPtr(int32(nc.PublicIP.Eip.Bandwidth.Size)),
				Sharetype:  &nc.PublicIP.Eip.Bandwidth.ShareType,
			},
		}
	}
	request := &model.CreateNodeRequest{
		ClusterId: clusterID,
		Body:      nodeCreateReq,
	}
	return request, nil
}

func GetClusterClient(client *cce.CceClient, cluster *model.ShowClusterResponse) (kubernetes.Interface, error) {
	clusterCert, err := GetClusterCert(client, cluster)
	if err != nil {
		return nil, err
	}
	data, err := huawei_utils.Marshal(clusterCert)
	if err != nil {
		return nil, fmt.Errorf("error creating clientset: %v", err)
	}

	config, err := clientcmd.RESTConfigFromKubeConfig(data)
	if err != nil {
		logrus.Infof("Generate config Failed %+v", err)
		return nil, err
	}
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error creating clientset: %v", err)
	}

	return clientSet, nil
}

func GetClusterCert(
	client *cce.CceClient, cluster *model.ShowClusterResponse,
) (*model.CreateKubernetesClusterCertResponse, error) {
	if cluster == nil || client == nil {
		return nil, fmt.Errorf("cluster or cce client is nil")
	}
	request := &model.CreateKubernetesClusterCertRequest{
		ClusterId: *cluster.Metadata.Uid,
		Body: &model.CertDuration{
			Duration: int32(365),
		},
	}
	return client.CreateKubernetesClusterCert(request)
}

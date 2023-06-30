package cce

import (
	"fmt"

	ccev1 "github.com/cnrancher/cce-operator/pkg/apis/cce.pandaria.io/v1"
	"github.com/cnrancher/cce-operator/pkg/huawei/common"
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

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
	"k8s.io/client-go/rest"
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
	req := common.GetCreateClusterRequest(config)
	res, err := client.CreateCluster(req)
	if err != nil {
		logrus.Debugf("CreateCluster failed: %v", utils.PrintObject(req))
	}
	return res, err
}

func GetCluster(client *cce.CceClient, ID string) (*model.ShowClusterResponse, error) {
	res, err := client.ShowCluster(&model.ShowClusterRequest{
		ClusterId: ID,
	})
	if err != nil {
		logrus.Debugf("ShowCluster failed: clusterID [%s]", ID)
	}
	return res, err
}

func ListClusters(client *cce.CceClient) (*model.ListClustersResponse, error) {
	res, err := client.ListClusters(&model.ListClustersRequest{})
	if err != nil {
		logrus.Debugf("ListClusters failed")
	}
	return res, err
}

func UpdateCluster(
	client *cce.CceClient, config *ccev1.CCEClusterConfig,
) (*model.UpdateClusterResponse, error) {
	req := common.GetUpdateClusterRequest(config)
	res, err := client.UpdateCluster(req)
	if err != nil {
		logrus.Debugf("UpdateCluster failed: %v", utils.PrintObject(req))
	}
	return res, err
}

func UpgradeCluster(
	client *cce.CceClient, config *ccev1.CCEClusterConfig,
) (*model.UpgradeClusterResponse, error) {
	req := common.GetUpgradeClusterRequest(config)
	res, err := client.UpgradeCluster(req)
	if err != nil {
		logrus.Debugf("UpgradeCluster failed: %v", utils.PrintObject(req))
	}
	return res, err
}

func ShowUpgradeClusterTask(
	client *cce.CceClient, clusterID string, taskID string,
) (*model.ShowUpgradeClusterTaskResponse, error) {
	req := &model.ShowUpgradeClusterTaskRequest{
		ClusterId: clusterID,
		TaskId:    taskID,
	}
	res, err := client.ShowUpgradeClusterTask(req)
	if err != nil {
		logrus.Debugf("ShowUpgradeClusterTask failed: %v", utils.PrintObject(req))
	}
	return res, err
}

func DeleteCluster(client *cce.CceClient, ID string) (*model.DeleteClusterResponse, error) {
	res, err := client.DeleteCluster(&model.DeleteClusterRequest{
		ClusterId: ID,
	})
	if err != nil {
		logrus.Debugf("DeleteCluster failed: clusterID [%s]", ID)
	}
	return res, err
}

func GetClusterRestConfig(
	client *cce.CceClient, clusterID string, duration int32,
) (*rest.Config, error) {
	clusterCert, err := GetClusterCert(client, clusterID, duration)
	if err != nil {
		return nil, err
	}
	data, err := huawei_utils.Marshal(clusterCert)
	if err != nil {
		return nil, fmt.Errorf("error creating clientset: %v", err)
	}

	config, err := clientcmd.RESTConfigFromKubeConfig(data)
	if err != nil {
		return nil, err
	}
	return config, nil
}

func GetClusterClient(
	client *cce.CceClient, clusterID string, duration int32,
) (kubernetes.Interface, error) {
	config, err := GetClusterRestConfig(client, clusterID, duration)
	if err != nil {
		return nil, err
	}
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error creating clientset: %v", err)
	}

	return clientSet, nil
}

func GetClusterCert(
	client *cce.CceClient, clusterID string, duration int32,
) (*model.CreateKubernetesClusterCertResponse, error) {
	if duration > 365*30 || duration < -1 {
		return nil, fmt.Errorf(
			"invalid duration '%d'(days), should be <= 365*30", duration)
	} else if duration == 0 {
		duration = -1
	}
	request := &model.CreateKubernetesClusterCertRequest{
		ClusterId: clusterID,
		Body: &model.CertDuration{
			Duration: duration,
		},
	}
	res, err := client.CreateKubernetesClusterCert(request)
	if err != nil {
		logrus.Debugf("CreateKubernetesClusterCert failed: %v", utils.PrintObject(request))
	}
	return res, err
}

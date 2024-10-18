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
	req := GetCreateClusterRequest(config)
	res, err := client.CreateCluster(req)
	if err != nil {
		logrus.Debugf("CreateCluster failed: %v", utils.PrintObject(req))
	}
	return res, err
}

func GetCreateClusterRequest(config *ccev1.CCEClusterConfig) *model.CreateClusterRequest {
	spec := &config.Spec
	status := &config.Status
	var containerNetworkMode model.ContainerNetworkMode
	switch spec.ContainerNetwork.Mode {
	case "overlay_l2":
		containerNetworkMode = model.GetContainerNetworkModeEnum().OVERLAY_L2
	case "vpc-router":
		containerNetworkMode = model.GetContainerNetworkModeEnum().VPC_ROUTER
	case "eni":
		containerNetworkMode = model.GetContainerNetworkModeEnum().ENI
	default:
		containerNetworkMode = model.GetContainerNetworkModeEnum().VPC_ROUTER
	}

	var clusterSpecType model.ClusterSpecType
	switch spec.Type {
	case "ARM64":
		clusterSpecType = model.GetClusterSpecTypeEnum().ARM64
	case "VirtualMachine":
		clusterSpecType = model.GetClusterSpecTypeEnum().VIRTUAL_MACHINE
	default:
		clusterSpecType = model.GetClusterSpecTypeEnum().VIRTUAL_MACHINE
	}

	var clusterSpecCategory model.ClusterSpecCategory
	switch spec.Category {
	case "CCE":
		clusterSpecCategory = model.GetClusterSpecCategoryEnum().CCE
	case "Turbo":
		clusterSpecCategory = model.GetClusterSpecCategoryEnum().TURBO
	default:
		clusterSpecCategory = model.GetClusterSpecCategoryEnum().CCE
	}
	var kubeProxyMode model.ClusterSpecKubeProxyMode
	switch spec.KubeProxyMode {
	case "iptables":
		kubeProxyMode = model.GetClusterSpecKubeProxyModeEnum().IPTABLES
	case "ipvs":
		kubeProxyMode = model.GetClusterSpecKubeProxyModeEnum().IPVS
	default:
		kubeProxyMode = model.GetClusterSpecKubeProxyModeEnum().IPTABLES
	}

	var clusterTags = make([]model.ResourceTag, 0)
	for k, v := range spec.Tags {
		clusterTags = append(clusterTags, model.ResourceTag{
			Key:   utils.Pointer(k),
			Value: utils.Pointer(v),
		})
	}

	clusterReq := &model.Cluster{
		Kind:       "cluster",
		ApiVersion: "v3",
		Metadata: &model.ClusterMetadata{
			Name:   spec.Name,
			Labels: spec.Labels,
		},
		Spec: &model.ClusterSpec{
			Category:    &clusterSpecCategory,
			Type:        &clusterSpecType,
			Flavor:      spec.Flavor,
			Version:     &spec.Version,
			Description: &spec.Description,
			Ipv6enable:  &spec.Ipv6Enable,
			HostNetwork: &model.HostNetwork{
				Vpc:           spec.HostNetwork.VpcID,
				Subnet:        spec.HostNetwork.SubnetID,
				SecurityGroup: &spec.HostNetwork.SecurityGroup,
			},
			ContainerNetwork: &model.ContainerNetwork{
				Mode: containerNetworkMode,
				Cidr: &spec.ContainerNetwork.CIDR,
			},
			EniNetwork: &model.EniNetwork{},
			Authentication: &model.Authentication{
				Mode: &spec.Authentication.Mode,
			},
			BillingMode: &spec.BillingMode,
			ServiceNetwork: &model.ServiceNetwork{
				IPv4CIDR: &spec.KubernetesSvcIPRange,
			},
			ClusterTags:   &clusterTags,
			KubeProxyMode: &kubeProxyMode,
			ExtendParam: &model.ClusterExtendParam{
				ClusterAZ:         &spec.ExtendParam.ClusterAZ,
				ClusterExternalIP: &status.ClusterExternalIP,
			},
		},
	}
	if len(spec.EniNetwork.Subnets) > 0 {
		for _, v := range spec.EniNetwork.Subnets {
			s := model.NetworkSubnet{
				SubnetID: v,
			}
			clusterReq.Spec.EniNetwork.Subnets = append(clusterReq.Spec.EniNetwork.Subnets, s)
		}
	}
	if spec.Authentication.Mode == "authenticating_proxy" {
		clusterReq.Spec.Authentication.AuthenticatingProxy.Ca =
			&spec.Authentication.AuthenticatingProxy.Ca
		clusterReq.Spec.Authentication.AuthenticatingProxy.Cert =
			&spec.Authentication.AuthenticatingProxy.Cert
		clusterReq.Spec.Authentication.AuthenticatingProxy.PrivateKey =
			&spec.Authentication.AuthenticatingProxy.PrivateKey
	}
	if spec.BillingMode != 0 {
		clusterReq.Spec.ExtendParam.PeriodType = &spec.ExtendParam.PeriodType
		clusterReq.Spec.ExtendParam.PeriodNum = &spec.ExtendParam.PeriodNum
		clusterReq.Spec.ExtendParam.IsAutoRenew = &spec.ExtendParam.IsAutoRenew
		clusterReq.Spec.ExtendParam.IsAutoPay = &spec.ExtendParam.IsAutoPay
	}
	request := &model.CreateClusterRequest{
		Body: clusterReq,
	}

	return request
}

func ShowCluster(client *cce.CceClient, ID string) (*model.ShowClusterResponse, error) {
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
	req := GetUpdateClusterRequest(config)
	res, err := client.UpdateCluster(req)
	if err != nil {
		logrus.Debugf("UpdateCluster failed: %v", utils.PrintObject(req))
	}
	return res, err
}

func GetUpdateClusterRequest(config *ccev1.CCEClusterConfig) *model.UpdateClusterRequest {
	req := &model.UpdateClusterRequest{
		ClusterId: config.Spec.ClusterID,
		Body: &model.ClusterInformation{
			Metadata: &model.ClusterMetadataForUpdate{
				Alias: &config.Spec.Name,
			},
			Spec: &model.ClusterInformationSpec{
				Description: &config.Spec.Description,
				EniNetwork: &model.EniNetworkUpdate{
					Subnets: nil,
				},
				HostNetwork: &model.ClusterInformationSpecHostNetwork{
					SecurityGroup: &config.Spec.HostNetwork.SecurityGroup,
				},
			},
		},
	}

	return req
}

func UpgradeCluster(
	client *cce.CceClient, config *ccev1.CCEClusterConfig,
) (*model.UpgradeClusterResponse, error) {
	req := GetUpgradeClusterRequest(config)
	res, err := client.UpgradeCluster(req)
	if err != nil {
		logrus.Debugf("UpgradeCluster failed: %v", utils.PrintObject(req))
	}
	return res, err
}

func GetUpgradeClusterRequest(config *ccev1.CCEClusterConfig) *model.UpgradeClusterRequest {
	req := &model.UpgradeClusterRequest{
		ClusterId: config.Spec.ClusterID,
		Body: &model.UpgradeClusterRequestBody{
			Metadata: &model.UpgradeClusterRequestMetadata{
				ApiVersion: "v3",
				Kind:       "UpgradeTask",
			},
			Spec: &model.UpgradeSpec{
				ClusterUpgradeAction: &model.ClusterUpgradeAction{
					Addons:        nil,
					NodeOrder:     nil,
					NodePoolOrder: nil,
					Strategy: &model.UpgradeStrategy{
						Type: "inPlaceRollingUpdate",
						InPlaceRollingUpdate: &model.InPlaceRollingUpdate{
							UserDefinedStep: utils.Pointer(int32(20)),
						},
					},
					TargetVersion: config.Spec.Version,
				},
			},
		},
	}
	return req
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

func ResizeCluster(
	client *cce.CceClient, ID, flavor, isAutoPay string,
) (*model.ResizeClusterResponse, error) {
	req := &model.ResizeClusterRequest{
		ClusterId: ID,
		Body: &model.ResizeClusterRequestBody{
			FlavorResize: flavor,
			ExtendParam: &model.ResizeClusterRequestBodyExtendParam{
				IsAutoPay: &isAutoPay,
			},
		},
	}
	res, err := client.ResizeCluster(req)
	if err != nil {
		logrus.Debugf("ResizeCluster failed: %v", utils.PrintObject(req))
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

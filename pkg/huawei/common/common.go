package common

import (
	"fmt"

	ccev1 "github.com/cnrancher/cce-operator/pkg/apis/cce.pandaria.io/v1"
	"github.com/cnrancher/cce-operator/pkg/utils"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cce/v3/model"
)

const (
	resourceNamePrefix = "rancher-managed"
)

type ClientAuth struct {
	Region     string
	Credential *basic.Credentials
}

func NewClientAuth(ak, sk, region, projectID string) *ClientAuth {
	return &ClientAuth{
		Region: region,
		Credential: basic.NewCredentialsBuilder().
			WithAk(ak).
			WithSk(sk).
			WithProjectId(projectID).
			Build(),
	}
}

func GetCreateClusterRequest(config *ccev1.CCEClusterConfig) *model.CreateClusterRequest {
	spec := &config.Spec
	status := &config.Status
	var containerNetWorkMode model.ContainerNetworkMode
	switch spec.ContainerNetwork.Mode {
	case "overlay_l2":
		containerNetWorkMode = model.GetContainerNetworkModeEnum().OVERLAY_L2
	case "vpc-router":
		containerNetWorkMode = model.GetContainerNetworkModeEnum().VPC_ROUTER
	case "eni":
		containerNetWorkMode = model.GetContainerNetworkModeEnum().ENI
	default:
		containerNetWorkMode = model.GetContainerNetworkModeEnum().ENI
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

	var clusterTags []model.ResourceTag
	for k, v := range spec.Tags {
		clusterTags = append(clusterTags, model.ResourceTag{
			Key:   utils.GetPtr(k),
			Value: utils.GetPtr(v),
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
				Mode: containerNetWorkMode,
				Cidr: &spec.ContainerNetwork.CIDR,
			},
			Authentication: &model.Authentication{
				Mode: &spec.Authentication.Mode,
			},
			BillingMode:          &spec.BillingMode,
			KubernetesSvcIpRange: &spec.KubernetesSvcIPRange,
			ClusterTags:          &clusterTags,
			KubeProxyMode:        &kubeProxyMode,
			ExtendParam: &model.ClusterExtendParam{
				ClusterAZ:         &spec.ExtendParam.ClusterAZ,
				ClusterExternalIP: &status.ClusterExternalIP,
			},
		},
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

func GetUpdateClusterRequest(config *ccev1.CCEClusterConfig) *model.UpdateClusterRequest {
	req := &model.UpdateClusterRequest{
		ClusterId: config.Spec.ClusterID,
		Body: &model.ClusterInformation{
			Metadata: &model.ClusterMetadataForUpdate{
				Alias: &config.Spec.Name, // operator does not support update cluster name
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
							UserDefinedStep: utils.GetPtr(int32(20)),
						},
					},
					TargetVersion: config.Spec.Version,
				},
			},
		},
	}
	return req
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

// GenResourceName generates the name of resource.
// vpc: rancher-managed-vpc-[RANDOM_STR]
// subnet: rancher-managed-subnet-[RANDOM_STR]
func GenResourceName(name string) string {
	return fmt.Sprintf("%s-%s-%s",
		resourceNamePrefix, name, utils.RandomString(5))
}

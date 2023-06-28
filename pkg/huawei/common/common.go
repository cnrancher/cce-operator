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

func GetClusterRequestFromCCECCSpec(config *ccev1.CCEClusterConfig) *model.CreateClusterRequest {
	spec := config.Spec
	status := config.Status
	var containerNetWorkMode model.ContainerNetworkMode
	switch status.ContainerNetwork.Mode {
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
				Vpc:    status.HostNetwork.VpcID,
				Subnet: status.HostNetwork.SubnetID,
			},
			ContainerNetwork: &model.ContainerNetwork{
				Mode: containerNetWorkMode,
				Cidr: &status.ContainerNetwork.CIDR,
			},
			Authentication: &model.Authentication{
				Mode: &spec.Authentication.Mode,
			},
			BillingMode:          &spec.BillingMode,
			KubernetesSvcIpRange: &spec.KubernetesSvcIPRange,
			ClusterTags:          &clusterTags,
		},
	}
	if spec.Authentication.Mode == "authenticating_proxy" {
		clusterReq.Spec.Authentication.AuthenticatingProxy.Ca =
			&spec.Authentication.AuthenticatingProxy.Ca
	}
	request := &model.CreateClusterRequest{
		Body: clusterReq,
	}

	return request
}

// GenResourceName generates the name of resource.
// vpc: rancher-managed-vpc-[RANDOM_STR]
// subnet: rancher-managed-subnet-[RANDOM_STR]
func GenResourceName(name string) string {
	return fmt.Sprintf("%s-%s-%s",
		resourceNamePrefix, name, utils.RandomString(5))
}

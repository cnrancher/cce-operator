package controller

import (
	"fmt"

	ccev1 "github.com/cnrancher/cce-operator/pkg/apis/cce.pandaria.io/v1"
	"github.com/cnrancher/cce-operator/pkg/utils"
	huawei_cce_model "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cce/v3/model"
)

func BuildUpstreamClusterState(
	c *huawei_cce_model.ShowClusterResponse,
	nodePools *huawei_cce_model.ListNodePoolsResponse,
) (*ccev1.CCEClusterConfigSpec, error) {
	if c == nil || nodePools == nil {
		return nil, fmt.Errorf("BuildUpstreamClusterState: cluster or nodePool is nil pointer")
	}
	if c.Metadata == nil || c.Spec == nil || c.Spec.Type == nil || c.Spec.Category == nil {
		return nil, fmt.Errorf(
			"failed to get cluster from CCE API: Metadata or Spec is nil")
	}
	spec := &ccev1.CCEClusterConfigSpec{
		HuaweiCredentialSecret: "",
		Category:               c.Spec.Category.Value(),
		Imported:               false,
		Name:                   c.Metadata.Name,
		Labels:                 c.Metadata.Labels,
		Type:                   c.Spec.Type.Value(),
		Flavor:                 c.Spec.Flavor,
		Version:                utils.GetValue(c.Spec.Version),
		BillingMode:            utils.GetValue(c.Spec.BillingMode),
		KubernetesSvcIPRange:   utils.GetValue(c.Spec.KubernetesSvcIpRange),
		Tags:                   make(map[string]string),
		KubeProxyMode:          c.Spec.KubeProxyMode.Value(),
		PublicAccess:           false,
	}
	if c.Spec.HostNetwork != nil {
		spec.HostNetwork.VpcID = c.Spec.HostNetwork.Vpc
		spec.HostNetwork.SubnetID = c.Spec.HostNetwork.Subnet
		spec.HostNetwork.SecurityGroup = utils.GetValue(c.Spec.HostNetwork.SecurityGroup)
	}
	if c.Spec.ContainerNetwork != nil {
		spec.ContainerNetwork.Mode = c.Spec.ContainerNetwork.Mode.Value()
		spec.ContainerNetwork.CIDR = utils.GetValue(c.Spec.ContainerNetwork.Cidr)
	}
	if c.Spec.Authentication != nil {
		spec.Authentication.Mode = utils.GetValue(c.Spec.Authentication.Mode)
		if c.Spec.Authentication.AuthenticatingProxy != nil &&
			c.Spec.Authentication.AuthenticatingProxy.Ca != nil {
			spec.Authentication.AuthenticatingProxy.Ca = utils.GetValue(
				c.Spec.Authentication.AuthenticatingProxy.Ca)
		}
	}
	if c.Spec.ClusterTags != nil && len(*c.Spec.ClusterTags) > 0 {
		for _, ct := range *c.Spec.ClusterTags {
			spec.Tags[utils.GetValue(ct.Key)] = utils.GetValue(ct.Value)
		}
	}
	if c.Spec.ExtendParam != nil {
		spec.ExtendParam = ccev1.CCEClusterExtendParam{
			ClusterAZ:         utils.GetValue(c.Spec.ExtendParam.ClusterAZ),
			ClusterExternalIP: utils.GetValue(c.Spec.ExtendParam.ClusterExternalIP),
			PeriodType:        utils.GetValue(c.Spec.ExtendParam.PeriodType),
			PeriodNum:         utils.GetValue(c.Spec.ExtendParam.PeriodNum),
			IsAutoRenew:       utils.GetValue(c.Spec.ExtendParam.IsAutoRenew),
			IsAutoPay:         utils.GetValue(c.Spec.ExtendParam.IsAutoPay),
		}
	}
	if c.Status != nil && c.Status.Endpoints != nil {
		for _, endpoint := range *c.Status.Endpoints {
			if endpoint.Type != nil && *endpoint.Type == "External" {
				spec.PublicAccess = true
			}
		}
	}
	var err error
	spec.NodePools, err = BuildUpstreamNodePoolConfigs(nodePools)
	if err != nil {
		return nil, err
	}
	return spec, nil
}

func BuildUpstreamNodePoolConfigs(
	nodePools *huawei_cce_model.ListNodePoolsResponse,
) ([]ccev1.CCENodePool, error) {
	if nodePools == nil || nodePools.Items == nil {
		return nil, fmt.Errorf("BuildUpstreamNodePoolConfigs: invalid nil parameter")
	}
	var nps []ccev1.CCENodePool = make([]ccev1.CCENodePool, 0, len(*nodePools.Items))
	if len(*nodePools.Items) == 0 {
		return nps, nil
	}

	for _, np := range *nodePools.Items {
		if np.Metadata == nil || np.Spec == nil || np.Spec.Type == nil ||
			np.Spec.NodeTemplate == nil || np.Spec.Autoscaling == nil {
			continue
		}
		config := ccev1.CCENodePool{
			Name: np.Metadata.Name,
			Type: np.Spec.Type.Value(),
			ID:   utils.GetValue(np.Metadata.Uid),
			NodeTemplate: ccev1.CCENodeTemplate{
				Flavor:          np.Spec.NodeTemplate.Flavor,
				AvailableZone:   np.Spec.NodeTemplate.Az,
				OperatingSystem: utils.GetValue(np.Spec.NodeTemplate.Os),
				BillingMode:     utils.GetValue(np.Spec.NodeTemplate.BillingMode),
			},
			InitialNodeCount: utils.GetValue(np.Spec.InitialNodeCount),
			Autoscaling: ccev1.CCENodePoolNodeAutoscaling{
				Enable:                utils.GetValue(np.Spec.Autoscaling.Enable),
				MinNodeCount:          utils.GetValue(np.Spec.Autoscaling.MinNodeCount),
				MaxNodeCount:          utils.GetValue(np.Spec.Autoscaling.MaxNodeCount),
				ScaleDownCooldownTime: utils.GetValue(np.Spec.Autoscaling.ScaleDownCooldownTime),
				Priority:              utils.GetValue(np.Spec.Autoscaling.Priority),
			},
		}
		if np.Spec.NodeTemplate.Login != nil && np.Spec.NodeTemplate.Login.SshKey != nil {
			config.NodeTemplate.SSHKey = *np.Spec.NodeTemplate.Login.SshKey
		}
		if np.Spec.NodeTemplate.RootVolume != nil {
			config.NodeTemplate.RootVolume = ccev1.CCENodeVolume{
				Size: np.Spec.NodeTemplate.RootVolume.Size,
				Type: np.Spec.NodeTemplate.RootVolume.Volumetype,
			}
		}
		if len(np.Spec.NodeTemplate.DataVolumes) > 0 {
			for _, v := range np.Spec.NodeTemplate.DataVolumes {
				config.NodeTemplate.DataVolumes = append(config.NodeTemplate.DataVolumes,
					ccev1.CCENodeVolume{
						Size: v.Size,
						Type: v.Volumetype,
					},
				)
			}
		}
		if np.Spec.NodeTemplate.PublicIP != nil {
			config.NodeTemplate.PublicIP.Ids = utils.GetValue(np.Spec.NodeTemplate.PublicIP.Ids)
			config.NodeTemplate.PublicIP.Count = utils.GetValue(np.Spec.NodeTemplate.Count)
			if np.Spec.NodeTemplate.PublicIP.Eip != nil {
				config.NodeTemplate.PublicIP.Eip.Iptype = np.Spec.NodeTemplate.PublicIP.Eip.Iptype
				if np.Spec.NodeTemplate.PublicIP.Eip.Bandwidth != nil {
					config.NodeTemplate.PublicIP.Eip.Bandwidth = ccev1.CCEEipBandwidth{
						ChargeMode: utils.GetValue(np.Spec.NodeTemplate.PublicIP.Eip.Bandwidth.Chargemode),
						Size:       utils.GetValue(np.Spec.NodeTemplate.PublicIP.Eip.Bandwidth.Size),
						ShareType:  utils.GetValue(np.Spec.NodeTemplate.PublicIP.Eip.Bandwidth.Sharetype),
					}
				}
			}
		}
		if np.Spec.NodeTemplate.Runtime != nil && np.Spec.NodeTemplate.Runtime.Name != nil {
			config.NodeTemplate.Runtime = np.Spec.NodeTemplate.Runtime.Name.Value()
		}
		if np.Spec.CustomSecurityGroups != nil && len(*np.Spec.CustomSecurityGroups) > 0 {
			config.CustomSecurityGroups = append(config.CustomSecurityGroups, *np.Spec.CustomSecurityGroups...)
		}
		nps = append(nps, config)
	}
	return nps, nil
}

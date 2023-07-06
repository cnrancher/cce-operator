package controller

import (
	"fmt"

	ccev1 "github.com/cnrancher/cce-operator/pkg/apis/cce.pandaria.io/v1"
	"github.com/cnrancher/cce-operator/pkg/utils"
	huawei_cce_model "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cce/v3/model"
)

func BuildUpstreamClusterState(
	cluster *huawei_cce_model.ShowClusterResponse,
	nodePools *huawei_cce_model.ListNodePoolsResponse,
) (*ccev1.CCEClusterConfigSpec, error) {
	if cluster == nil || nodePools == nil {
		return nil, fmt.Errorf("BuildUpstreamClusterState: cluster or nodes is nil pointer")
	}
	if cluster.Metadata == nil || cluster.Spec == nil || cluster.Spec.Type == nil {
		return nil, fmt.Errorf(
			"failed to get cluster from CCE API: Metadata or Spec is nil")
	}
	spec := &ccev1.CCEClusterConfigSpec{
		HuaweiCredentialSecret: "",
		RegionID:               utils.GetValue(cluster.Spec.Az),
		Imported:               false,
		Name:                   cluster.Metadata.Name,
		Labels:                 cluster.Metadata.Labels,
		Type:                   cluster.Spec.Type.Value(),
		Flavor:                 cluster.Spec.Flavor,
		Version:                utils.GetValue(cluster.Spec.Version),
		BillingMode:            utils.GetValue(cluster.Spec.BillingMode),
		KubernetesSvcIPRange:   utils.GetValue(cluster.Spec.KubernetesSvcIpRange),
		KubeProxyMode:          cluster.Spec.KubeProxyMode.Value(),
		PublicAccess:           false,
	}
	if cluster.Spec.HostNetwork != nil {
		spec.HostNetwork.VpcID = cluster.Spec.HostNetwork.Vpc
		spec.HostNetwork.SubnetID = cluster.Spec.HostNetwork.Subnet
	}
	if cluster.Spec.ContainerNetwork != nil {
		spec.ContainerNetwork.Mode = cluster.Spec.ContainerNetwork.Mode.Value()
		spec.ContainerNetwork.CIDR = utils.GetValue(cluster.Spec.ContainerNetwork.Cidr)
	}
	if cluster.Spec.Authentication != nil {
		spec.Authentication.Mode = utils.GetValue(cluster.Spec.Authentication.Mode)
		if cluster.Spec.Authentication.AuthenticatingProxy != nil &&
			cluster.Spec.Authentication.AuthenticatingProxy.Ca != nil {
			spec.Authentication.AuthenticatingProxy.Ca = utils.GetValue(
				cluster.Spec.Authentication.AuthenticatingProxy.Ca)
		}
	}
	if cluster.Spec.ExtendParam != nil {
		spec.ExtendParam = ccev1.CCEClusterExtendParam{
			ClusterAZ:         utils.GetValue(cluster.Spec.ExtendParam.ClusterAZ),
			ClusterExternalIP: utils.GetValue(cluster.Spec.ExtendParam.ClusterExternalIP),
			PeriodType:        utils.GetValue(cluster.Spec.ExtendParam.PeriodType),
			PeriodNum:         utils.GetValue(cluster.Spec.ExtendParam.PeriodNum),
			IsAutoRenew:       utils.GetValue(cluster.Spec.ExtendParam.IsAutoRenew),
			IsAutoPay:         utils.GetValue(cluster.Spec.ExtendParam.IsAutoPay),
		}
	}
	if spec.ExtendParam.ClusterExternalIP != "" {
		spec.PublicAccess = true
	}
	var err error
	spec.NodePools, err = BuildUpstreamNodePoolConfigs(nodePools)
	if err != nil {
		return nil, err
	}
	return spec, nil
}

func BuildUpstreamNodePoolConfigs(
	nodePoolsRes *huawei_cce_model.ListNodePoolsResponse,
) ([]ccev1.CCENodePool, error) {
	if nodePoolsRes == nil || nodePoolsRes.Items == nil {
		return nil, fmt.Errorf("BuildUpstreamNodePoolConfigs: invalid nil parameter")
	}
	var nodePools []ccev1.CCENodePool = make([]ccev1.CCENodePool, 0, len(*nodePoolsRes.Items))
	if len(*nodePoolsRes.Items) == 0 {
		return nodePools, nil
	}

	for _, n := range *nodePoolsRes.Items {
		if n.Metadata == nil || n.Spec == nil || n.Spec.Type == nil ||
			n.Spec.NodeTemplate == nil || n.Spec.Autoscaling == nil {
			continue
		}
		config := ccev1.CCENodePool{
			Name: n.Metadata.Name,
			Type: n.Spec.Type.Value(),
			ID:   utils.GetValue(n.Metadata.Uid),
			NodeTemplate: ccev1.CCENodeTemplate{
				Flavor:          n.Spec.NodeTemplate.Flavor,
				AvailableZone:   n.Spec.NodeTemplate.Az,
				OperatingSystem: utils.GetValue(n.Spec.NodeTemplate.Os),
				Count:           utils.GetValue(n.Spec.NodeTemplate.Count),
				BillingMode:     utils.GetValue(n.Spec.NodeTemplate.BillingMode),
			},
			InitialNodeCount: utils.GetValue(n.Spec.InitialNodeCount),
			Autoscaling: ccev1.CCENodePoolNodeAutoscaling{
				Enable:                utils.GetValue(n.Spec.Autoscaling.Enable),
				MinNodeCount:          utils.GetValue(n.Spec.Autoscaling.MinNodeCount),
				MaxNodeCount:          utils.GetValue(n.Spec.Autoscaling.MaxNodeCount),
				ScaleDownCooldownTime: utils.GetValue(n.Spec.Autoscaling.ScaleDownCooldownTime),
				Priority:              utils.GetValue(n.Spec.Autoscaling.Priority),
			},
		}
		if n.Spec.NodeTemplate.Login != nil && n.Spec.NodeTemplate.Login.SshKey != nil {
			config.NodeTemplate.SSHKey = *n.Spec.NodeTemplate.Login.SshKey
		}
		if n.Spec.NodeTemplate.RootVolume != nil {
			config.NodeTemplate.RootVolume = ccev1.CCENodeVolume{
				Size: n.Spec.NodeTemplate.RootVolume.Size,
				Type: n.Spec.NodeTemplate.RootVolume.Volumetype,
			}
		}
		if len(n.Spec.NodeTemplate.DataVolumes) > 0 {
			for _, v := range n.Spec.NodeTemplate.DataVolumes {
				config.NodeTemplate.DataVolumes = append(config.NodeTemplate.DataVolumes,
					ccev1.CCENodeVolume{
						Size: v.Size,
						Type: v.Volumetype,
					},
				)
			}
		}
		if n.Spec.NodeTemplate.PublicIP != nil {
			config.NodeTemplate.PublicIP.Ids = utils.GetValue(n.Spec.NodeTemplate.PublicIP.Ids)
			config.NodeTemplate.PublicIP.Count = utils.GetValue(n.Spec.NodeTemplate.Count)
			if n.Spec.NodeTemplate.PublicIP.Eip != nil {
				config.NodeTemplate.PublicIP.Eip.Iptype = n.Spec.NodeTemplate.PublicIP.Eip.Iptype
				if n.Spec.NodeTemplate.PublicIP.Eip.Bandwidth != nil {
					config.NodeTemplate.PublicIP.Eip.Bandwidth = ccev1.CCEEipBandwidth{
						ChargeMode: utils.GetValue(n.Spec.NodeTemplate.PublicIP.Eip.Bandwidth.Chargemode),
						Size:       utils.GetValue(n.Spec.NodeTemplate.PublicIP.Eip.Bandwidth.Size),
						ShareType:  utils.GetValue(n.Spec.NodeTemplate.PublicIP.Eip.Bandwidth.Sharetype),
					}
				}
			}
		}
		if n.Spec.NodeTemplate.Runtime != nil && n.Spec.NodeTemplate.Runtime.Name != nil {
			config.NodeTemplate.Runtime = n.Spec.NodeTemplate.Runtime.Name.Value()
		}
		if n.Spec.CustomSecurityGroups != nil && len(*n.Spec.CustomSecurityGroups) > 0 {
			config.CustomSecurityGroups = append(config.CustomSecurityGroups, *n.Spec.CustomSecurityGroups...)
		}
		nodePools = append(nodePools, config)
	}
	return nodePools, nil
}

func CompareNodePool(a, b *ccev1.CCENodePool) bool {
	if a.Name != b.Name || a.Type != b.Type {
		return false
	}
	// TODO: compare Autoscaling, PodSecurityGroups, CustomSecurityGroups...
	// Compare NodeTemplate
	at := a.NodeTemplate
	bt := b.NodeTemplate
	if at.Flavor != bt.Flavor ||
		at.AvailableZone != bt.AvailableZone ||
		at.SSHKey != bt.SSHKey ||
		at.BillingMode != bt.BillingMode ||
		at.OperatingSystem != bt.OperatingSystem {
		return false
	}

	if !CompareVolume(&at.RootVolume, &bt.RootVolume) {
		return false
	}

	if len(at.DataVolumes) != len(bt.DataVolumes) {
		return false
	}

	if len(at.DataVolumes) == 0 {
		return true
	}

	for _, ad := range at.DataVolumes {
		var found = false
		for _, bd := range bt.DataVolumes {
			if CompareVolume(&ad, &bd) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

func CompareVolume(a, b *ccev1.CCENodeVolume) bool {
	if a.Size != b.Size || a.Type != b.Type {
		return false
	}
	return true
}

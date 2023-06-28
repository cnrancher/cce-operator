package controller

import (
	"fmt"

	ccev1 "github.com/cnrancher/cce-operator/pkg/apis/cce.pandaria.io/v1"
	"github.com/cnrancher/cce-operator/pkg/utils"
	huawei_cce "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cce/v3"
	huawei_cce_model "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cce/v3/model"
)

func BuildUpstreamClusterState(
	client *huawei_cce.CceClient,
	cluster *huawei_cce_model.ShowClusterResponse,
	nodes *huawei_cce_model.ListNodesResponse,
) (*ccev1.CCEClusterConfigSpec, error) {
	if cluster == nil || nodes == nil {
		return nil, fmt.Errorf("cluster or nodes is nil pointer")
	}
	if cluster.Metadata == nil || cluster.Spec == nil {
		return nil, fmt.Errorf(
			"failed to get cluster from CCE API: Metadata or Spec is nil")
	}
	newSpec := &ccev1.CCEClusterConfigSpec{
		CredentialSecret: "",
		RegionID:         utils.GetValue(cluster.Spec.Az),
		Imported:         false,
		Name:             cluster.Metadata.Name,
		Labels:           cluster.Metadata.Labels,
		Type:             cluster.Spec.Type.Value(),
		Flavor:           cluster.Spec.Flavor,
		Version:          utils.GetValue(cluster.Spec.Version),
		BillingMode:      utils.GetValue(cluster.Spec.BillingMode),
		// ExternalServiceEnabled: false, // TODO: always false
		KubernetesSvcIPRange: utils.GetValue(cluster.Spec.KubernetesSvcIpRange),
	}
	if cluster.Spec.HostNetwork != nil {
		newSpec.HostNetwork.VpcID = cluster.Spec.HostNetwork.Vpc
		newSpec.HostNetwork.SubnetID = cluster.Spec.HostNetwork.Subnet
	}
	if cluster.Spec.ContainerNetwork != nil {
		newSpec.ContainerNetwork.Mode = cluster.Spec.ContainerNetwork.Mode.Value()
		newSpec.ContainerNetwork.CIDR = utils.GetValue(cluster.Spec.ContainerNetwork.Cidr)
	}
	if cluster.Spec.Authentication != nil {
		newSpec.Authentication.Mode = utils.GetValue(cluster.Spec.Authentication.Mode)
		if cluster.Spec.Authentication.AuthenticatingProxy != nil &&
			cluster.Spec.Authentication.AuthenticatingProxy.Ca != nil {
			newSpec.Authentication.AuthenticatingProxy.Ca = utils.GetValue(
				cluster.Spec.Authentication.AuthenticatingProxy.Ca)
		}
	}
	var err error
	newSpec.NodeConfigs, err = BuildUpstreamNodeConfigs(client, nodes)
	if err != nil {
		return nil, err
	}
	return newSpec, nil
}

func BuildUpstreamNodeConfigs(
	client *huawei_cce.CceClient, nodes *huawei_cce_model.ListNodesResponse,
) ([]ccev1.NodeConfig, error) {
	if nodes == nil {
		return nil, fmt.Errorf("nodes is nil pointer")
	}
	nodeConfigs := []ccev1.NodeConfig{}
	if nodes.Items == nil || len(*nodes.Items) == 0 {
		return nodeConfigs, nil
	}

	for _, n := range *nodes.Items {
		if n.Metadata == nil || n.Spec == nil {
			continue
		}
		config := ccev1.NodeConfig{
			Name:            utils.GetValue(n.Metadata.Name),
			NodeID:          utils.GetValue(n.Metadata.Uid),
			Flavor:          n.Spec.Flavor,
			AvailableZone:   n.Spec.Az,
			Count:           utils.GetValue(n.Spec.Count),
			BillingMode:     utils.GetValue(n.Spec.BillingMode),
			OperatingSystem: utils.GetValue(n.Spec.Os),
		}
		if n.Spec.Login != nil && n.Spec.Login.SshKey != nil {
			config.SSHKey = *n.Spec.Login.SshKey
		}
		if n.Spec.RootVolume != nil {
			config.RootVolume = ccev1.Volume{
				Size: n.Spec.RootVolume.Size,
				Type: n.Spec.RootVolume.Volumetype,
			}
		}
		if len(n.Spec.DataVolumes) > 0 {
			for _, v := range n.Spec.DataVolumes {
				config.DataVolumes = append(config.DataVolumes, ccev1.Volume{
					Size: v.Size,
					Type: v.Volumetype,
				})
			}
		}
		if n.Spec.PublicIP != nil {
			config.PublicIP.Ids = utils.GetValue(n.Spec.PublicIP.Ids)
			config.PublicIP.Count = utils.GetValue(n.Spec.Count)
			if n.Spec.PublicIP.Eip != nil {
				config.PublicIP.Eip.Iptype = n.Spec.PublicIP.Eip.Iptype
				if n.Spec.PublicIP.Eip.Bandwidth != nil {
					config.PublicIP.Eip.Bandwidth = ccev1.Bandwidth{
						ChargeMode: utils.GetValue(n.Spec.PublicIP.Eip.Bandwidth.Chargemode),
						Size:       utils.GetValue(n.Spec.PublicIP.Eip.Bandwidth.Size),
						ShareType:  utils.GetValue(n.Spec.PublicIP.Eip.Bandwidth.Sharetype),
					}
				}
			}
		}
		nodeConfigs = append(nodeConfigs, config)
	}
	return nodeConfigs, nil
}

func CompareNode(a, b *ccev1.NodeConfig) bool {
	if a.Flavor != b.Flavor ||
		a.AvailableZone != b.AvailableZone ||
		a.SSHKey != b.SSHKey ||
		a.BillingMode != b.BillingMode ||
		a.OperatingSystem != b.OperatingSystem {
		return false
	}

	if !CompareVolume(&a.RootVolume, &b.RootVolume) {
		return false
	}

	if len(a.DataVolumes) != len(b.DataVolumes) {
		return false
	}

	if len(a.DataVolumes) == 0 {
		return true
	}

	for _, ad := range a.DataVolumes {
		var found = false
		for _, bd := range b.DataVolumes {
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

func CompareVolume(a, b *ccev1.Volume) bool {
	if a.Size != b.Size || a.Type != b.Type {
		return false
	}
	return true
}

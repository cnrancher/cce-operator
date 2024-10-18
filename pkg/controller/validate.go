package controller

import (
	"fmt"

	"github.com/Masterminds/semver/v3"
	ccev1 "github.com/cnrancher/cce-operator/pkg/apis/cce.pandaria.io/v1"
	"github.com/cnrancher/cce-operator/pkg/huawei"
	"github.com/cnrancher/cce-operator/pkg/huawei/cce"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	cannotBeEmptyError = "field [%s] cannot be empty for non-import cluster [%s]"
)

func validateNodePool(config *ccev1.CCEClusterConfig) error {
	nodePoolNames := map[string]bool{}
	for _, pool := range config.Spec.NodePools {
		if pool.Name == "" {
			return fmt.Errorf(cannotBeEmptyError, "nodePool.name", config.Name)
		}
		if nodePoolNames[pool.Name] {
			return fmt.Errorf("nodePool.name should be unique, duplicated detected: %q", pool.Name)
		}
		nodePoolNames[pool.Name] = true
		nt := pool.NodeTemplate
		if nt.Flavor == "" {
			return fmt.Errorf(cannotBeEmptyError, "nodePool.nodeTemplate.flavor", config.Name)
		}
		if nt.AvailableZone == "" {
			return fmt.Errorf(cannotBeEmptyError, "nodePool.nodeTemplate.availableZone", config.Name)
		}
		if nt.SSHKey == "" {
			return fmt.Errorf(cannotBeEmptyError, "nodePool.nodeTemplate.sshKey", config.Name)
		}
		if nt.RootVolume.Size == 0 || nt.RootVolume.Type == "" {
			return fmt.Errorf(cannotBeEmptyError, "nodePool.nodeTemplate.rootVolume", config.Name)
		}
		if len(nt.DataVolumes) == 0 {
			return fmt.Errorf(cannotBeEmptyError, "nodePool.nodeTemplate.dataVolumes", config.Name)
		}
		for _, dv := range nt.DataVolumes {
			if dv.Size == 0 || dv.Type == "" {
				return fmt.Errorf(cannotBeEmptyError, "nodePool.nodeTemplate.dataVolumes", config.Name)
			}
		}
		if nt.OperatingSystem == "" {
			return fmt.Errorf(cannotBeEmptyError, "nodePool.nodeTemplate.operatingSystem", config.Name)
		}
	}
	return nil
}

func (h *Handler) validateCreate(config *ccev1.CCEClusterConfig) error {
	driver := h.drivers[config.Spec.HuaweiCredentialSecret]
	// Check for existing cceclusterconfigs with the same display name
	cceConfigs, err := h.cceCC.List(config.Namespace, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("cannot list cceclusterconfigs for display name check")
	}
	for _, c := range cceConfigs.Items {
		if c.Spec.Name == config.Spec.Name && c.Name != config.Name {
			return fmt.Errorf("cannot create cluster [%s] because an cceclusterconfig "+
				"exists with the same name", config.Spec.Name)
		}
	}

	if config.Spec.HuaweiCredentialSecret == "" {
		return fmt.Errorf(cannotBeEmptyError, "huaweiCredentialSecret", config.Name)
	}
	if config.Spec.RegionID == "" {
		return fmt.Errorf(cannotBeEmptyError, "regionID", config.Name)
	}
	if config.Spec.Name == "" {
		return fmt.Errorf(cannotBeEmptyError, "name", config.Name)
	}

	if config.Spec.Imported {
		if config.Spec.ClusterID == "" {
			return fmt.Errorf(cannotBeEmptyError, "clusterID", config.Name)
		}
		_, err := cce.ShowCluster(driver.CCE, config.Spec.ClusterID)
		if err != nil {
			hwerr, _ := huawei.NewHuaweiError(err)
			if hwerr.StatusCode == 404 {
				return fmt.Errorf("failed to find cluster [%s]: %v",
					config.Spec.ClusterID, hwerr.ErrorMessage)
			}
			return err
		}
	} else {
		// Cluster may already created, skip validation.
		if config.Spec.ClusterID != "" {
			return nil
		}
		listClustersRes, err := cce.ListClusters(driver.CCE)
		if err != nil {
			return err
		}
		if listClustersRes == nil || listClustersRes.Items == nil {
			return fmt.Errorf("ListClusters returns invalid data")
		}
		for _, cluster := range *listClustersRes.Items {
			if config.Spec.Name == cluster.Metadata.Name {
				return fmt.Errorf("cannot create cluster [%s] because a cluster"+
					" in CCE exists with the same name", cluster.Metadata.Name)
			}
		}
		if config.Spec.Type == "" {
			return fmt.Errorf(cannotBeEmptyError, "type", config.Name)
		}
		if config.Spec.Flavor == "" {
			return fmt.Errorf(cannotBeEmptyError, "flavor", config.Name)
		}
		if config.Spec.Version == "" {
			return fmt.Errorf(cannotBeEmptyError, "version", config.Name)
		}
		if config.Spec.KubernetesSvcIPRange == "" {
			return fmt.Errorf(cannotBeEmptyError, "kubernetesSvcIPRange", config.Name)
		}
		if config.Spec.ExtendParam.ClusterExternalIP != "" || config.Spec.PublicIP.CreateEIP {
			if !config.Spec.PublicAccess {
				return fmt.Errorf("'publicAccess' can not be 'false' when 'clusterExternalIP' provided " +
					"or 'publicIP.createEIP' is true")
			}
		}
		if config.Spec.PublicAccess {
			if config.Spec.ExtendParam.ClusterExternalIP == "" && !config.Spec.PublicIP.CreateEIP {
				return fmt.Errorf(
					"should provide 'clusterExternalIP' or setup 'publicIP' if 'publicAccess' is true")
			}
			if config.Spec.PublicIP.CreateEIP && config.Spec.PublicIP.Eip.Bandwidth.Size == 0 {
				return fmt.Errorf(
					"'publicIP.eip.bandwidth.size' should be configured when 'createEIP' is true")
			}
		}
		if config.Spec.NatGateway.Enabled {
			if config.Spec.NatGateway.ExistingEIPID == "" && config.Spec.NatGateway.SNatRuleEIP.Bandwidth.Size == 0 {
				return fmt.Errorf(
					"'natGateway.publicIP' should be configured when NAT enabled and 'existingEIPID' not provided")
			}
		}
		if len(config.Spec.NodePools) == 0 {
			return fmt.Errorf(cannotBeEmptyError, "nodePools", config.Name)
		}
		if err = validateNodePool(config); err != nil {
			return err
		}
	}

	return nil
}

func validateUpdate(config *ccev1.CCEClusterConfig) error {
	if config.Spec.Version != "" {
		var err error
		_, err = semver.NewVersion(config.Spec.Version + ".0")
		if err != nil {
			return fmt.Errorf("improper version format for cluster [%s]: %s, %v",
				config.Spec.Name, config.Spec.Version, err)
		}
	}
	if config.Spec.Name == "" {
		return fmt.Errorf(cannotBeEmptyError, "name", config.Name)
	}
	if config.Spec.RegionID == "" {
		return fmt.Errorf(cannotBeEmptyError, "regionID", config.Name)
	}
	if config.Spec.HuaweiCredentialSecret == "" {
		return fmt.Errorf(cannotBeEmptyError, "huaweiCredentialSecret", config.Name)
	}
	if config.Spec.Imported {
		if config.Spec.ClusterID == "" {
			return fmt.Errorf(cannotBeEmptyError, "clusterID", config.Name)
		}
		return nil
	}
	if config.Spec.Name == "" {
		return fmt.Errorf(cannotBeEmptyError, "name", config.Name)
	}
	if len(config.Spec.NodePools) == 0 {
		return fmt.Errorf(cannotBeEmptyError, "nodePools", config.Name)
	}

	return validateNodePool(config)
}

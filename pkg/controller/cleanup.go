package controller

import (
	"fmt"
	"time"

	ccev1 "github.com/cnrancher/cce-operator/pkg/apis/cce.pandaria.io/v1"
	"github.com/cnrancher/cce-operator/pkg/huawei"
	"github.com/cnrancher/cce-operator/pkg/huawei/cce"
	"github.com/cnrancher/cce-operator/pkg/huawei/network"
	"github.com/cnrancher/cce-operator/pkg/utils"
	cce_model "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cce/v3/model"
	"github.com/sirupsen/logrus"
)

func (h *Handler) OnCCEConfigRemoved(_ string, config *ccev1.CCEClusterConfig) (*ccev1.CCEClusterConfig, error) {
	h.log = logrus.WithFields(logrus.Fields{
		"cluster": config.Name,
		"phase":   "remove",
	})
	if config.Spec.Imported {
		h.log.Infof("cluster [%s] is imported, will not delete CCE cluster", config.Name)
		return config, nil
	}
	if err := h.newDriver(h.secretsCache, &config.Spec); err != nil {
		return config, fmt.Errorf("error creating new CCE services: %w", err)
	}
	h.log.Infof("start deleting cluster [%s] resources", config.Name)

	var (
		refresh bool
		err     error
	)
	for refresh = true; refresh; {
		config, refresh, err = h.deleteCCEClusterNodePools(config)
		if err != nil {
			return config, err
		}
		if refresh {
			time.Sleep(10 * time.Second)
		}
	}

	for refresh = true; refresh; {
		config, refresh, err = h.deleteCCECluster(config)
		if err != nil {
			return config, err
		}
		if refresh {
			time.Sleep(20 * time.Second)
		}
	}

	for refresh = true; refresh; {
		config, refresh, err = h.deleteNetworkResources(config)
		if err != nil {
			return config, err
		}
		if refresh {
			time.Sleep(5 * time.Second)
		}
	}

	h.log.Infof("finished clean-up resources of cluster [%s]", config.Name)

	return config, nil
}

func (h *Handler) deleteCCEClusterNodePools(
	config *ccev1.CCEClusterConfig,
) (*ccev1.CCEClusterConfig, bool, error) {
	// Cluster or nodes were already deleted.
	if config.Status.ClusterID == "" {
		return config, false, nil
	}

	nodes, err := cce.GetClusterNodes(h.driver.CCE, config.Status.ClusterID)
	if err != nil {
		if hwerr, _ := huawei.NewHuaweiError(err); hwerr.StatusCode == 404 {
			// Cluster was deleted and failed to query nodes.
			return config, false, nil
		}
		return config, false, fmt.Errorf("failed to get cluster nodes: %v", err)
	}
	if nodes.Items == nil {
		return config, false, fmt.Errorf("cce.GetClusterNodes returns invalid value")
	}
	for _, node := range *nodes.Items {
		if node.Status == nil || node.Status.Phase == nil || node.Metadata == nil {
			continue
		}
		// Ensure nodes are available to delete
		switch *node.Status.Phase {
		case cce_model.GetNodeStatusPhaseEnum().INSTALLING,
			// cce_model.GetNodeStatusPhaseEnum().INSTALLED,
			// cce_model.GetNodeStatusPhaseEnum().SHUT_DOWN,
			cce_model.GetNodeStatusPhaseEnum().UPGRADING,
			// cce_model.GetNodeStatusPhaseEnum().ABNORMAL,
			// cce_model.GetNodeStatusPhaseEnum().ERROR,
			cce_model.GetNodeStatusPhaseEnum().BUILD,
			cce_model.GetNodeStatusPhaseEnum().DELETING:
			h.log.Infof("waiting for node [%s] status: %v",
				utils.GetValue(node.Metadata.Name), node.Status.Phase.Value())
			return config, true, nil
		}
	}
	// Cluster nodes are available to delete.
	nodePools, err := cce.GetClusterNodePools(h.driver.CCE, config.Status.ClusterID, false)
	if err != nil {
		return config, false, err
	}
	if nodePools.Items == nil {
		return config, true, fmt.Errorf("cce.GetClusterNodePools returns invalid value")
	}
	var enqueueNode bool = false
	for _, np := range *nodePools.Items {
		if np.Metadata == nil || np.Metadata.Uid == nil {
			continue
		}
		if _, err = cce.DeleteNodePool(h.driver.CCE, config.Status.ClusterID, *np.Metadata.Uid); err != nil {
			return config, false, fmt.Errorf("error delete node pool [%s]: %w",
				np.Metadata.Name, err)
		}
		h.log.Infof("request to delete node pool [%s], ID [%s]",
			np.Metadata.Name, utils.GetValue(np.Metadata.Uid))
		enqueueNode = true
	}
	if enqueueNode {
		// Requeue to waiting for node pools deleted.
		return config, true, nil
	}

	// Cluster nodes were deleted, update status.
	config = config.DeepCopy()
	config.Status.NodePools = []ccev1.CCENodePool{}
	config, err = h.cceCC.UpdateStatus(config)
	return config, false, err
}

func (h *Handler) deleteCCECluster(
	config *ccev1.CCEClusterConfig,
) (*ccev1.CCEClusterConfig, bool, error) {
	if config.Status.ClusterID == "" {
		// Cluster was already deleted.
		return config, false, nil
	}

	cluster, err := cce.GetCluster(h.driver.CCE, config.Status.ClusterID)
	if hwerr, _ := huawei.NewHuaweiError(err); hwerr.StatusCode == 404 {
		// Cluster deleted, update status.
		h.log.Infof("deleted cluster [%s]", config.Spec.Name)
		config = config.DeepCopy()
		config.Status.ClusterID = ""
		config, err = h.cceCC.UpdateStatus(config)
		return config, false, err
	} else if err != nil {
		return config, false, err
	}
	if cluster.Status == nil || cluster.Metadata == nil {
		return config, false, fmt.Errorf("cce.GetCluster rerturns invalid value")
	}
	switch utils.GetValue(cluster.Status.Phase) {
	case cce.ClusterStatusDeleting,
		cce.ClusterStatusCreating,
		cce.ClusterStatusUpgrading,
		cce.ClusterStatusResizing,
		cce.ClusterStatusScalingDown,
		cce.ClusterStatusScalingUp,
		cce.ClusterStatusRollingBack:
		h.log.Infof("waiting for cluster [%s] status: %s",
			config.Spec.Name, *cluster.Status.Phase)
		return config, true, nil
	}

	if _, err = cce.DeleteCluster(h.driver.CCE, config.Status.ClusterID); err != nil {
		return config, false, fmt.Errorf("failed to delete cluster: %w", err)
	}
	h.log.Infof("requested to delete cluster [%s]", config.Spec.Name)

	return config, true, nil
}

func (h *Handler) deleteNetworkResources(
	config *ccev1.CCEClusterConfig,
) (*ccev1.CCEClusterConfig, bool, error) {
	if config.Status.ClusterExternalIPID != "" {
		eipID := config.Status.ClusterExternalIPID
		_, err := network.GetPublicIP(h.driver.EIP, eipID)
		if hwerr, _ := huawei.NewHuaweiError(err); hwerr.StatusCode == 404 {
			config = config.DeepCopy()
			config.Status.ClusterExternalIPID = ""
			config.Status.ClusterExternalIP = ""
			config, err = h.cceCC.UpdateStatus(config)
			if err != nil {
				return config, false, err
			}
			h.log.Infof("EIP [%s] deleted", eipID)
		} else if err != nil {
			return config, false, err
		} else {
			_, err = network.DeletePublicIP(h.driver.EIP, eipID)
			if err != nil {
				return config, false, err
			}
			h.log.Infof("request to delete EIP [%v]", eipID)
			return config, true, nil
		}
	}

	var subnetID, vpcID string
	if config.Spec.HostNetwork.VpcID == "" {
		subnetID = config.Status.HostNetwork.SubnetID
		vpcID = config.Status.HostNetwork.VpcID
	} else if config.Spec.HostNetwork.SubnetID == "" {
		subnetID = config.Status.HostNetwork.SubnetID
	} else {
		// HostNetwork provided, skip vpc & subnet deletion.
		return config, false, nil
	}

	// HostNetwork resources were deleted.
	if vpcID == "" && subnetID == "" {
		return config, false, nil
	}

	var err error
	if subnetID != "" {
		_, err = network.GetSubnet(h.driver.VPC, subnetID)
		if hwerr, _ := huawei.NewHuaweiError(err); hwerr.StatusCode == 404 {
			config = config.DeepCopy()
			config.Status.HostNetwork.SubnetID = ""
			config, err = h.cceCC.UpdateStatus(config)
			if err != nil {
				return config, false, err
			}
			h.log.Infof("subnet [%s] deleted", subnetID)
		} else if err != nil {
			return config, false, fmt.Errorf("failed to get subnet: %w", err)
		} else {
			_, err := network.DeleteSubnet(h.driver.VPC, vpcID, subnetID)
			if err != nil {
				return config, false, err
			}
			h.log.Infof("request to delete subnet [%s]", subnetID)
			return config, true, nil
		}
	}

	if vpcID != "" {
		vpceps, err := network.GetVpcepServices(h.driver.VPCEP, "")
		if err != nil {
			return config, false, fmt.Errorf("failed to get VpcepSvc: %v", err)
		}
		// Ensure VPC does not have associated VpcEndpointService (vpcepsvc).
		var vpcepsvcID string
		if vpceps.EndpointServices != nil && len(*vpceps.EndpointServices) > 0 {
			for _, v := range *vpceps.EndpointServices {
				if v.VpcId == nil || *v.VpcId != config.Status.HostNetwork.VpcID {
					continue
				}
				vpcepsvcID = utils.GetValue(v.Id)
				break
			}
		}
		// VPC has associated VpcEndpointService, delete vpcepsvc before delete VPC.
		if vpcepsvcID != "" {
			_, err = network.DeleteVpcepService(h.driver.VPCEP, vpcepsvcID)
			if err != nil {
				return config, false, fmt.Errorf("failed to delete vpcep: %w", err)
			}
			h.log.Infof("request to delete VpcEndpointService [%s]", vpcepsvcID)
			return config, true, nil
		}

		_, err = network.GetVPC(h.driver.VPC, vpcID)
		if hwerr, _ := huawei.NewHuaweiError(err); hwerr.StatusCode == 404 {
			config = config.DeepCopy()
			config.Status.HostNetwork.VpcID = ""
			config, err = h.cceCC.UpdateStatus(config)
			if err != nil {
				return config, false, err
			}
			h.log.Infof("vpc [%s] deleted", vpcID)
		} else if err != nil {
			return config, false, fmt.Errorf("failed to get vpc: %w", err)
		} else {
			_, err = network.DeleteVPC(h.driver.VPC, config.Status.HostNetwork.VpcID)
			if err != nil {
				return config, false, fmt.Errorf("failed to delete VPC: %w", err)
			}
			h.log.Infof("request to delete vpc [%s]", vpcID)
			return config, true, nil
		}
	}

	return config, false, nil
}

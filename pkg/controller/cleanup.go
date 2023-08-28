package controller

import (
	"fmt"
	"time"

	ccev1 "github.com/cnrancher/cce-operator/pkg/apis/cce.pandaria.io/v1"
	"github.com/cnrancher/cce-operator/pkg/huawei"
	"github.com/cnrancher/cce-operator/pkg/huawei/cce"
	"github.com/cnrancher/cce-operator/pkg/huawei/eip"
	"github.com/cnrancher/cce-operator/pkg/huawei/nat"
	"github.com/cnrancher/cce-operator/pkg/huawei/vpc"
	"github.com/cnrancher/cce-operator/pkg/huawei/vpcep"
	"github.com/cnrancher/cce-operator/pkg/utils"
	cce_model "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cce/v3/model"
	"github.com/sirupsen/logrus"
)

func (h *Handler) OnCCEConfigRemoved(_ string, config *ccev1.CCEClusterConfig) (*ccev1.CCEClusterConfig, error) {
	var err error
	if config.Spec.Imported {
		logrus.WithFields(logrus.Fields{
			"cluster": config.Name,
			"phase":   "remove",
		}).Infof("cluster [%s] is imported, will not delete CCE cluster", config.Name)
		return config, nil
	}

	// Ensure the driver in h.drivers map exists.
	if err := h.setupHuaweiDriver(&config.Spec); err != nil {
		return config, err
	}

	logrus.WithFields(logrus.Fields{
		"cluster": config.Name,
		"phase":   "remove",
	}).Infof("start deleting cluster [%s] resources", config.Name)

	var refresh bool
	for refresh = true; refresh; {
		config, refresh, err = h.ensureCCEClusterDeletable(config)
		if err != nil {
			time.Sleep(5 * time.Second) // Avoid rate limit.
			return config, err
		}
		if refresh {
			time.Sleep(10 * time.Second)
		}
	}

	for refresh = true; refresh; {
		config, refresh, err = h.deleteCCECluster(config)
		if err != nil {
			time.Sleep(5 * time.Second) // Avoid rate limit.
			return config, err
		}
		if refresh {
			time.Sleep(20 * time.Second)
		}
	}

	for refresh = true; refresh; {
		config, refresh, err = h.deleteNetworkResources(config)
		if err != nil {
			time.Sleep(5 * time.Second) // Avoid rate limit.
			return config, err
		}
		if refresh {
			time.Sleep(5 * time.Second)
		}
	}

	logrus.WithFields(logrus.Fields{
		"cluster": config.Name,
		"phase":   "remove",
	}).Infof("finished clean-up resources of cluster [%s]", config.Name)

	return config, nil
}

func (h *Handler) ensureCCEClusterDeletable(
	config *ccev1.CCEClusterConfig,
) (*ccev1.CCEClusterConfig, bool, error) {
	driver := h.drivers[config.Spec.HuaweiCredentialSecret]
	// Cluster was already deleted.
	if config.Spec.ClusterID == "" {
		return config, false, nil
	}

	nodes, err := cce.ListNodes(driver.CCE, config.Spec.ClusterID)
	if err != nil {
		// Cluster was deleted and failed to query nodes.
		return config, false, nil
	}
	if nodes == nil || nodes.Items == nil {
		return config, false, fmt.Errorf("cce.GetClusterNodes returns invalid data")
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
			logrus.WithFields(logrus.Fields{
				"cluster": config.Name,
				"phase":   "remove",
			}).Infof("waiting for node [%s] status: %v",
				utils.Value(node.Metadata.Name), node.Status.Phase.Value())
			return config, true, nil
		}
	}
	return config, false, err
}

func (h *Handler) deleteCCECluster(
	config *ccev1.CCEClusterConfig,
) (*ccev1.CCEClusterConfig, bool, error) {
	driver := h.drivers[config.Spec.HuaweiCredentialSecret]
	if config.Spec.ClusterID == "" {
		// Cluster was already deleted.
		return config, false, nil
	}

	cluster, err := cce.ShowCluster(driver.CCE, config.Spec.ClusterID)
	if hwerr, _ := huawei.NewHuaweiError(err); hwerr.StatusCode == 404 {
		// Cluster deleted, update status.
		logrus.WithFields(logrus.Fields{
			"cluster": config.Name,
			"phase":   "remove",
		}).Infof("deleted cluster [%s]", config.Spec.Name)
		config = config.DeepCopy()
		config.Spec.ClusterID = ""
		config, err = h.cceCC.Update(config)
		if err != nil {
			return config, false, err
		}
		config = config.DeepCopy()
		config.Status.ClusterExternalIP = ""
		config, err = h.cceCC.UpdateStatus(config)
		return config, false, err
	} else if err != nil {
		return config, false, err
	}
	if cluster == nil || cluster.Status == nil || cluster.Metadata == nil {
		return config, false, fmt.Errorf("cce.GetCluster returns invalid data")
	}
	switch utils.Value(cluster.Status.Phase) {
	case cce.ClusterStatusDeleting,
		cce.ClusterStatusCreating,
		cce.ClusterStatusUpgrading,
		cce.ClusterStatusResizing,
		cce.ClusterStatusScalingDown,
		cce.ClusterStatusScalingUp,
		cce.ClusterStatusRollingBack:
		logrus.WithFields(logrus.Fields{
			"cluster": config.Name,
			"phase":   "remove",
		}).Infof("waiting for cluster [%s] status: %s",
			config.Spec.Name, utils.Value(cluster.Status.Phase))
		return config, true, nil
	}

	if _, err = cce.DeleteCluster(driver.CCE, config.Spec.ClusterID); err != nil {
		return config, false, err
	}
	logrus.WithFields(logrus.Fields{
		"cluster": config.Name,
		"phase":   "remove",
	}).Infof("request to delete cluster [%s]", config.Spec.Name)

	return config, true, nil
}

func (h *Handler) deleteNetworkResources(
	config *ccev1.CCEClusterConfig,
) (*ccev1.CCEClusterConfig, bool, error) {
	driver := h.drivers[config.Spec.HuaweiCredentialSecret]
	var err error
	if config.Status.CreatedNatGatewayID != "" {
		natID := config.Status.CreatedNatGatewayID
		// Delete SNAT Rules before delete NAT Gateway.
		snatRulesRes, err := nat.ListNatGatewaySnatRules(driver.NAT, []string{natID})
		if err != nil {
			return config, false, err
		}
		if snatRulesRes == nil || snatRulesRes.SnatRules == nil {
			return config, false, fmt.Errorf("ListNatGatewaySnatRules returns invalid data")
		}
		for _, sr := range *snatRulesRes.SnatRules {
			if _, err = nat.DeleteNatGatewaySnatRule(driver.NAT, sr.Id, natID); err != nil {
				return config, false, err
			}
			logrus.WithFields(logrus.Fields{
				"cluster": config.Name,
				"phase":   "remove",
			}).Infof("request to delete SNAT Rule [%s] from NAT [%s]",
				sr.Id, natID)
		}
		if len(*snatRulesRes.SnatRules) > 0 {
			// Requeue to wait for SNAT Rules were deleted from NAT Gateway.
			return config, true, nil
		}

		// Delete NatGateway.
		_, err = nat.ShowNatGateway(driver.NAT, config.Status.CreatedNatGatewayID)
		if hwerr, _ := huawei.NewHuaweiError(err); hwerr.StatusCode == 404 {
			logrus.WithFields(logrus.Fields{
				"cluster": config.Name,
				"phase":   "remove",
			}).Infof("NAT Gateway [%s] deleted", natID)
			config = config.DeepCopy()
			config.Status.CreatedNatGatewayID = ""
			config.Status.CreatedSNATRuleID = ""
			config, err = h.cceCC.UpdateStatus(config)
			return config, true, err
		} else if err != nil {
			return config, false, err
		}
		_, err = nat.DeleteNatGateway(driver.NAT, natID)
		if err != nil {
			return config, false, err
		}
		logrus.WithFields(logrus.Fields{
			"cluster": config.Name,
			"phase":   "remove",
		}).Infof("request to delete NAT Gateway [%v]", natID)
		// Requeue to wait for NAT Gateway were deleted.
		return config, true, nil
	}

	// Delete EIP.
	if config.Status.CreatedClusterEIPID != "" {
		eipID := config.Status.CreatedClusterEIPID
		_, err = eip.ShowPublicip(driver.EIP, eipID)
		if hwerr, _ := huawei.NewHuaweiError(err); hwerr.StatusCode == 404 {
			config = config.DeepCopy()
			config.Status.CreatedClusterEIPID = ""
			config, err = h.cceCC.UpdateStatus(config)
			logrus.WithFields(logrus.Fields{
				"cluster": config.Name,
				"phase":   "remove",
			}).Infof("cluster EIP [%s] deleted", eipID)
			return config, true, err
		} else if err != nil {
			return config, false, err
		}
		if _, err = eip.DeletePublicIP(driver.EIP, eipID); err != nil {
			return config, false, err
		}
		logrus.WithFields(logrus.Fields{
			"cluster": config.Name,
			"phase":   "remove",
		}).Infof("request to delete EIP [%v]", eipID)
		return config, true, nil
	}
	if config.Status.CreatedSNatRuleEIPID != "" {
		eipID := config.Status.CreatedSNatRuleEIPID
		_, err = eip.ShowPublicip(driver.EIP, eipID)
		if hwerr, _ := huawei.NewHuaweiError(err); hwerr.StatusCode == 404 {
			config = config.DeepCopy()
			config.Status.CreatedSNatRuleEIPID = ""
			config, err = h.cceCC.UpdateStatus(config)
			logrus.WithFields(logrus.Fields{
				"cluster": config.Name,
				"phase":   "remove",
			}).Infof("SNAT Rule EIP [%s] deleted", eipID)
			return config, true, err
		} else if err != nil {
			return config, false, err
		}
		if _, err = eip.DeletePublicIP(driver.EIP, eipID); err != nil {
			return config, false, err
		}
		logrus.WithFields(logrus.Fields{
			"cluster": config.Name,
			"phase":   "remove",
		}).Infof("request to delete EIP [%v]", eipID)
		return config, true, nil
	}

	var (
		vpcID    = config.Status.CreatedVpcID
		subnetID = config.Status.CreatedSubnetID
	)
	if subnetID != "" {
		_, err = vpc.ShowSubnet(driver.VPC, subnetID)
		if hwerr, _ := huawei.NewHuaweiError(err); hwerr.StatusCode == 404 {
			logrus.WithFields(logrus.Fields{
				"cluster": config.Name,
				"phase":   "remove",
			}).Infof("subnet [%s] deleted", subnetID)
			config = config.DeepCopy()
			config.Status.CreatedSubnetID = ""
			config, err = h.cceCC.UpdateStatus(config)
			return config, true, err
		} else if err != nil {
			return config, false, err
		}
		_, err := vpc.DeleteSubnet(driver.VPC, vpcID, subnetID)
		if err != nil {
			return config, false, err
		}
		logrus.WithFields(logrus.Fields{
			"cluster": config.Name,
			"phase":   "remove",
		}).Infof("request to delete subnet [%s]", subnetID)
		return config, true, nil
	}
	if vpcID != "" {
		vpceps, err := vpcep.ListEndpointService(driver.VPCEP, "")
		if err != nil {
			return config, false, err
		}
		// Ensure VPC does not have associated VpcEndpointService (vpcepsvc).
		var vpcepsvcID string
		if vpceps.EndpointServices != nil && len(*vpceps.EndpointServices) > 0 {
			for _, v := range *vpceps.EndpointServices {
				if utils.Value(v.VpcId) != vpcID {
					continue
				}
				vpcepsvcID = utils.Value(v.Id)
				break
			}
		}
		// VPC has associated VpcEndpointService, delete vpcepsvc before delete VPC.
		if vpcepsvcID != "" {
			_, err = vpcep.DeleteVpcepService(driver.VPCEP, vpcepsvcID)
			if err != nil {
				return config, false, err
			}
			logrus.WithFields(logrus.Fields{
				"cluster": config.Name,
				"phase":   "remove",
			}).Infof("request to delete VpcEndpointService [%s]", vpcepsvcID)
			return config, true, nil
		}
		_, err = vpc.ShowVPC(driver.VPC, vpcID)
		if hwerr, _ := huawei.NewHuaweiError(err); hwerr.StatusCode == 404 {
			logrus.WithFields(logrus.Fields{
				"cluster": config.Name,
				"phase":   "remove",
			}).Infof("vpc [%s] deleted", vpcID)
			config = config.DeepCopy()
			config.Status.CreatedVpcID = ""
			config, err = h.cceCC.UpdateStatus(config)
			return config, true, err
		} else if err != nil {
			return config, false, err
		}
		_, err = vpc.DeleteVPC(driver.VPC, vpcID)
		if err != nil {
			return config, false, err
		}
		logrus.WithFields(logrus.Fields{
			"cluster": config.Name,
			"phase":   "remove",
		}).Infof("request to delete vpc [%s]", vpcID)
		return config, true, nil
	}
	return config, false, nil
}

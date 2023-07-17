package controller

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/Masterminds/semver/v3"
	ccev1 "github.com/cnrancher/cce-operator/pkg/apis/cce.pandaria.io/v1"
	ccecontrollers "github.com/cnrancher/cce-operator/pkg/generated/controllers/cce.pandaria.io/v1"
	"github.com/cnrancher/cce-operator/pkg/huawei"
	"github.com/cnrancher/cce-operator/pkg/huawei/cce"
	"github.com/cnrancher/cce-operator/pkg/huawei/common"
	"github.com/cnrancher/cce-operator/pkg/huawei/network"
	"github.com/cnrancher/cce-operator/pkg/utils"
	cce_model "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cce/v3/model"
	wranglerv1 "github.com/rancher/wrangler/pkg/generated/controllers/core/v1"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

const (
	controllerName           = "cce-operator"
	controllerRemoveName     = "cce-operator-remove"
	cceConfigCreatingPhase   = "creating"
	cceConfigNotCreatedPhase = ""
	cceConfigActivePhase     = "active"
	cceConfigUpdatingPhase   = "updating"
	cceConfigImportingPhase  = "importing"
	cceClusterConfigKind     = "CCEClusterConfig"
)

type Handler struct {
	cceCC           ccecontrollers.CCEClusterConfigClient
	cceEnqueueAfter func(namespace, name string, duration time.Duration)
	cceEnqueue      func(namespace, name string)
	secrets         wranglerv1.SecretClient
	secretsCache    wranglerv1.SecretCache
	driver          HuaweiDriver
}

func Register(
	ctx context.Context,
	secrets wranglerv1.SecretController,
	cce ccecontrollers.CCEClusterConfigController,
) {
	h := &Handler{
		cceCC:           cce,
		cceEnqueue:      cce.Enqueue,
		cceEnqueueAfter: cce.EnqueueAfter,
		secretsCache:    secrets.Cache(),
		secrets:         secrets,
	}

	// Register handlers
	cce.OnChange(ctx, controllerName, h.recordError(h.OnCCEConfigChanged))
	cce.OnRemove(ctx, controllerRemoveName, h.OnCCEConfigRemoved)
}

func (h *Handler) OnCCEConfigChanged(_ string, config *ccev1.CCEClusterConfig) (*ccev1.CCEClusterConfig, error) {
	if config == nil {
		return nil, nil
	}
	if config.Name == "" {
		return config, nil
	}

	if config.DeletionTimestamp != nil {
		return nil, nil
	}

	if err := h.newDriver(h.secretsCache, &config.Spec); err != nil {
		return config, fmt.Errorf("error creating new CCE services: %w", err)
	}

	switch config.Status.Phase {
	case cceConfigImportingPhase:
		return h.importCluster(config)
	case cceConfigNotCreatedPhase:
		return h.create(config)
	case cceConfigCreatingPhase:
		return h.waitForCreationComplete(config)
	case cceConfigActivePhase, cceConfigUpdatingPhase:
		return h.checkAndUpdate(config)
	}

	return config, nil
}

// recordError writes the error return by onChange to the failureMessage field on status. If there is no error, then
// empty string will be written to status
func (h *Handler) recordError(
	onChange func(key string, config *ccev1.CCEClusterConfig) (*ccev1.CCEClusterConfig, error),
) func(key string, config *ccev1.CCEClusterConfig) (*ccev1.CCEClusterConfig, error) {
	return func(key string, config *ccev1.CCEClusterConfig) (*ccev1.CCEClusterConfig, error) {
		var err error
		var message string
		config, err = onChange(key, config)
		if config == nil {
			// CCE config is likely deleting
			return config, err
		}
		if err != nil {
			logrus.Warnf("%v", err)
			if huawei.IsHuaweiError(err) {
				hwerr, _ := huawei.NewHuaweiError(err)
				hwerr.RequestID = ""
				message = hwerr.String()
			} else {
				message = err.Error()
			}
		}

		if config.Name == "" {
			return config, err
		}

		if config.Status.FailureMessage == message {
			// Avoid trigger the HWCloud API rate limit.
			if message != "" {
				time.Sleep(time.Second * 5)
			}
			return config, err
		}

		config = config.DeepCopy()
		if message != "" && config.Status.Phase == cceConfigActivePhase {
			// can assume an update is failing
			config.Status.Phase = cceConfigUpdatingPhase
		}
		config.Status.FailureMessage = message

		var recordErr error
		config, recordErr = h.cceCC.UpdateStatus(config)
		if recordErr != nil {
			logrus.Errorf("Error recording cce cluster config [%s] failure message: %v",
				config.Name, recordErr)
		}

		return config, err
	}
}

func (h *Handler) create(config *ccev1.CCEClusterConfig) (*ccev1.CCEClusterConfig, error) {
	if err := h.validateCreate(config); err != nil {
		return config, err
	}

	if config.Spec.Imported {
		config = config.DeepCopy()
		config.Status.Phase = cceConfigImportingPhase
		return h.cceCC.UpdateStatus(config)
	}

	var err error
	if config, err = h.generateAndSetNetworking(config); err != nil {
		return config, err
	}

	// create cluster
	if config.Status.ClusterID != "" {
		_, err := cce.GetCluster(h.driver.CCE, config.Status.ClusterID)
		if err == nil {
			logrus.WithFields(logrus.Fields{
				"cluster": config.Name,
			}).Infof("cluster [%s] already created, switch to creating phase",
				config.Status.ClusterID)
			config = config.DeepCopy()
			config.Status.Phase = cceConfigCreatingPhase
			return h.cceCC.UpdateStatus(config)
		}
	}
	logrus.WithFields(logrus.Fields{
		"cluster": config.Name,
	}).Infof("creating cluster [%s]", config.Spec.Name)
	cluster, err := cce.CreateCluster(h.driver.CCE, config)
	if err != nil {
		return config, err
	}
	if cluster.Metadata == nil || cluster.Metadata.Uid == nil {
		return config, fmt.Errorf("cce.CreateCluster return invalid value")
	}
	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		config, err = h.cceCC.Get(config.Namespace, config.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		config = config.DeepCopy()
		config.Status.ClusterID = *cluster.Metadata.Uid
		config.Status.Phase = cceConfigCreatingPhase
		config.Status.FailureMessage = ""
		config, err = h.cceCC.UpdateStatus(config)
		return err
	})
	logrus.WithFields(logrus.Fields{
		"cluster": config.Name,
	}).Infof("created cluster %q", config.Status.ClusterID)
	return config, err
}

func (h *Handler) generateAndSetNetworking(config *ccev1.CCEClusterConfig) (*ccev1.CCEClusterConfig, error) {
	var err error
	if config.Status.ContainerNetwork.Mode == "" || config.Status.ContainerNetwork.CIDR == "" {
		configUpdate := config.DeepCopy()
		if config.Spec.ContainerNetwork.Mode != "" {
			configUpdate.Status.ContainerNetwork.Mode = config.Spec.ContainerNetwork.Mode
		} else {
			configUpdate.Status.ContainerNetwork.Mode = network.DefaultContainerNetworkMode
		}
		if config.Spec.ContainerNetwork.CIDR != "" {
			configUpdate.Status.ContainerNetwork.CIDR = config.Spec.ContainerNetwork.CIDR
		} else {
			configUpdate.Status.ContainerNetwork.CIDR = network.DefaultContainerNetworkCIDR
		}
		config, err = h.cceCC.UpdateStatus(configUpdate)
		if err != nil {
			return config, err
		}
	}

	// Create EIP.
	if config.Spec.PublicAccess && config.Spec.PublicIP.CreateEIP && config.Status.ClusterExternalIP == "" {
		res, err := network.CreatePublicIP(h.driver.EIP, &config.Spec.PublicIP)
		if err != nil {
			return config, err
		}
		if res.Publicip == nil {
			return config, fmt.Errorf("network.CreatePublicIP returns invalid value")
		}
		logrus.WithFields(logrus.Fields{
			"cluster": config.Name,
		}).Infof("created EIP [%s], address [%s]",
			utils.GetValue(res.Publicip.Alias), utils.GetValue(res.Publicip.PublicIpAddress))
		configUpdate := config.DeepCopy()
		configUpdate.Status.ClusterExternalIP = utils.GetValue(res.Publicip.PublicIpAddress)
		configUpdate.Status.ClusterExternalIPID = utils.GetValue(res.Publicip.Id)
		config, err = h.cceCC.UpdateStatus(configUpdate)
		if err != nil {
			return config, err
		}
	}
	// Do not create EIP and use existing EIP address.
	if config.Spec.PublicAccess && config.Status.ClusterExternalIP == "" && config.Spec.ExtendParam.ClusterExternalIP != "" {
		configUpdate := config.DeepCopy()
		configUpdate.Status.ClusterExternalIP = config.Spec.ExtendParam.ClusterExternalIP
		config, err = h.cceCC.UpdateStatus(configUpdate)
		if err != nil {
			return config, err
		}
	}

	// HostNetwork configured, skip.
	if config.Status.HostNetwork.VpcID != "" && config.Status.HostNetwork.SubnetID != "" {
		return config, nil
	}

	// Configure VPC & Subnet.
	if config.Spec.HostNetwork.VpcID == "" {
		logrus.WithFields(logrus.Fields{
			"cluster": config.Name,
		}).Infof("VPC ID not provided, will create VPC and subnet...")
		logrus.WithFields(logrus.Fields{
			"cluster": config.Name,
		}).Infof("creating VPC...")
		vpcRes, err := network.CreateVPC(
			h.driver.VPC,
			common.GenResourceName("vpc"),
			network.DefaultVpcCIDR,
		)
		if err != nil {
			return config, err
		}
		logrus.WithFields(logrus.Fields{
			"cluster": config.Name,
		}).Infof("querying DNS server of region [%s]", config.Spec.RegionID)
		dnsServers, err := network.ListNameServers(h.driver.DNS, config.Spec.RegionID)
		if err != nil {
			return config, err
		}
		if dnsServers.Nameservers == nil || len(*dnsServers.Nameservers) == 0 {
			return config, fmt.Errorf("network.ListNameServers returns invalid value")
		}
		var dnsRecords []string = make([]string, 2)
		for _, nameserver := range *dnsServers.Nameservers {
			if nameserver.NsRecords == nil || len(*nameserver.NsRecords) == 0 {
				continue
			}
			for i := 0; i < len(*nameserver.NsRecords) && i < 2; i++ {
				ns := (*nameserver.NsRecords)[i]
				dnsRecords[i] = utils.GetValue(ns.Address)
			}
		}
		logrus.WithFields(logrus.Fields{
			"cluster": config.Name,
		}).Infof("found DNS server of region [%s] %v",
			config.Spec.RegionID, dnsRecords)
		logrus.WithFields(logrus.Fields{
			"cluster": config.Name,
		}).Infof("creating subnet...")
		subnetRes, err := network.CreateSubnet(
			h.driver.VPC,
			common.GenResourceName("subnet"),
			vpcRes.Vpc.Id,
			dnsRecords[0],
			dnsRecords[1],
		)
		if err != nil {
			return config, err
		}
		// Update status.
		configUpdate := config.DeepCopy()
		configUpdate.Status.HostNetwork.VpcID = vpcRes.Vpc.Id
		configUpdate.Status.HostNetwork.SubnetID = subnetRes.Subnet.Id
		if config, err = h.cceCC.UpdateStatus(configUpdate); err != nil {
			return config, err
		}
		logrus.WithFields(logrus.Fields{
			"cluster": config.Name,
		}).Infof("created VPC [%s] [%s]",
			vpcRes.Vpc.Name, config.Status.HostNetwork.VpcID)
		logrus.WithFields(logrus.Fields{
			"cluster": config.Name,
		}).Infof("created subnet [%s] [%s]",
			subnetRes.Subnet.Name, config.Status.HostNetwork.SubnetID)
	} else if config.Spec.HostNetwork.SubnetID == "" {
		// VPC ID provided but subnet ID not provided.
		// Create a subnet based on the provided VPC.
		// Ensure provided VPC exists first.
		_, err = network.GetVPC(h.driver.VPC, config.Spec.HostNetwork.VpcID)
		if err != nil {
			return config, err
		}
		logrus.WithFields(logrus.Fields{
			"cluster": config.Name,
		}).Infof("VPC ID provided [%s], will create subnet for this VPC",
			config.Spec.HostNetwork.VpcID)
		logrus.WithFields(logrus.Fields{
			"cluster": config.Name,
		}).Infof("querying DNS server of region [%s]", config.Spec.RegionID)
		dnsServers, err := network.ListNameServers(h.driver.DNS, config.Spec.RegionID)
		if err != nil {
			return config, err
		}
		if dnsServers.Nameservers == nil || len(*dnsServers.Nameservers) == 0 {
			return config, fmt.Errorf("network.ListNameServers returns invalid value")
		}
		var dnsRecords []string = make([]string, 2)
		for _, nameserver := range *dnsServers.Nameservers {
			if nameserver.NsRecords == nil || len(*nameserver.NsRecords) == 0 {
				continue
			}
			for i := 0; i < len(*nameserver.NsRecords) && i < 2; i++ {
				ns := (*nameserver.NsRecords)[i]
				dnsRecords[i] = utils.GetValue(ns.Address)
			}
		}
		logrus.WithFields(logrus.Fields{
			"cluster": config.Name,
		}).Infof("found DNS server of region [%s] %v",
			config.Spec.RegionID, dnsRecords)
		subnetRes, err := network.CreateSubnet(
			h.driver.VPC,
			common.GenResourceName("subnet"),
			config.Spec.HostNetwork.VpcID,
			dnsRecords[0],
			dnsRecords[1],
		)
		if err != nil {
			return config, err
		}
		// Update status.
		configUpdate := config.DeepCopy()
		configUpdate.Status.HostNetwork.SubnetID = subnetRes.Subnet.Id
		if config, err = h.cceCC.UpdateStatus(configUpdate); err != nil {
			return config, err
		}
		logrus.WithFields(logrus.Fields{
			"cluster": config.Name,
		}).Infof("created subnet [%s] [%s]",
			subnetRes.Subnet.Name, config.Status.HostNetwork.SubnetID)
	} else {
		// Both VPC ID and subnet ID are provided.
		// Ensure provided VPC and subnet exists.
		_, err = network.GetVPC(h.driver.VPC, config.Spec.HostNetwork.VpcID)
		if err != nil {
			return config, err
		}
		_, err = network.GetSubnet(h.driver.VPC, config.Spec.HostNetwork.SubnetID)
		if err != nil {
			return config, err
		}
		// Update status.
		configUpdate := config.DeepCopy()
		configUpdate.Status.HostNetwork.VpcID = config.Spec.HostNetwork.VpcID
		configUpdate.Status.HostNetwork.SubnetID = config.Spec.HostNetwork.SubnetID
		if config, err = h.cceCC.UpdateStatus(configUpdate); err != nil {
			return config, err
		}
		logrus.WithFields(logrus.Fields{
			"cluster": config.Name,
		}).Infof("VPC [%s] and subnet [%s] are provided",
			config.Status.HostNetwork.VpcID, config.Status.HostNetwork.SubnetID)
	}

	config, err = h.cceCC.UpdateStatus(config)
	if err != nil {
		return config, err
	}
	return config, err
}

func (h *Handler) waitForCreationComplete(config *ccev1.CCEClusterConfig) (*ccev1.CCEClusterConfig, error) {
	cluster, err := cce.GetCluster(h.driver.CCE, config.Status.ClusterID)
	if err != nil {
		return config, fmt.Errorf("waitForCreationComplete: %w", err)
	}

	if cluster.Status == nil || cluster.Metadata == nil || cluster.Spec == nil {
		return config, fmt.Errorf("cce.GetCluster returns invalid value")
	}

	if *cluster.Status.Phase == cce.ClusterStatusUnavailable {
		return config, fmt.Errorf("creation failed for cluster %q: %v",
			cluster.Metadata.Name, utils.GetValue(cluster.Status.Reason))
	}

	if *cluster.Status.Phase == cce.ClusterStatusAvailable {
		if err := h.createCASecret(config); err != nil {
			return config, fmt.Errorf("createCASecret: %w", err)
		}
		logrus.WithFields(logrus.Fields{
			"cluster": config.Name,
			"phase":   config.Status.Phase,
		}).Infof("cluster [%s] created successfully", config.Name)
		config = config.DeepCopy()
		config.Status.Phase = cceConfigUpdatingPhase
		config, err = h.cceCC.UpdateStatus(config)
		if err != nil {
			return config, err
		}
		return config, nil
	}

	logrus.WithFields(logrus.Fields{
		"cluster": config.Name,
		"phase":   config.Status.Phase,
	}).Infof("waiting for cluster [%s] status %q",
		config.Spec.Name, *cluster.Status.Phase)
	h.cceEnqueueAfter(config.Namespace, config.Name, 30*time.Second)

	return config, nil
}

func (h *Handler) checkAndUpdate(config *ccev1.CCEClusterConfig) (*ccev1.CCEClusterConfig, error) {
	if err := validateUpdate(config); err != nil {
		// validation failed, will be considered a failing update until resolved
		config = config.DeepCopy()
		config.Status.Phase = cceConfigUpdatingPhase
		var updateErr error
		config, updateErr = h.cceCC.UpdateStatus(config)
		if updateErr != nil {
			return config, updateErr
		}
		return config, err
	}

	// Check cluster upgrade status.
	if config.Status.UpgradeClusterTaskID != "" {
		res, err := cce.ShowUpgradeClusterTask(h.driver.CCE, config.Status.ClusterID, config.Status.UpgradeClusterTaskID)
		if err != nil {
			hwerr, _ := huawei.NewHuaweiError(err)
			if hwerr.StatusCode == 404 {
				config = config.DeepCopy()
				config.Status.UpgradeClusterTaskID = ""
				return h.cceCC.UpdateStatus(config)
			} else {
				return config, err
			}
		}
		if res != nil && res.Spec != nil && res.Status != nil {
			switch utils.GetValue(res.Status.Phase) {
			case "Success", "":
				logrus.WithFields(logrus.Fields{
					"cluster": config.Name,
					"phase":   config.Status.Phase,
				}).Infof("cluster [%s] finished upgrade",
					config.Spec.Name)
				config = config.DeepCopy()
				config.Status.UpgradeClusterTaskID = ""
				return h.cceCC.UpdateStatus(config)
			default:
				logrus.WithFields(logrus.Fields{
					"cluster": config.Name,
					"phase":   config.Status.Phase,
				}).Infof("waiting for cluster [%s] upgrade task status %q",
					config.Spec.Name, utils.GetValue(res.Status.Phase))
			}
			h.cceEnqueueAfter(config.Namespace, config.Name, 30*time.Second)
			return config, nil
		}
	}

	// Check cluster status.
	cluster, err := cce.GetCluster(h.driver.CCE, config.Status.ClusterID)
	if err != nil {
		return config, err
	}
	if cluster.Status == nil || cluster.Spec == nil {
		return config, fmt.Errorf("cce.GetCluster returns invalid data")
	}
	switch utils.GetValue(cluster.Status.Phase) {
	case cce.ClusterStatusDeleting,
		cce.ClusterStatusResizing,
		cce.ClusterStatusUpgrading:
		logrus.WithFields(logrus.Fields{
			"cluster": config.Name,
			"phase":   config.Status.Phase,
		}).Infof("waiting for cluster [%s] finish status %q",
			config.Spec.Name, utils.GetValue(cluster.Status.Phase))
		if config.Status.Phase != cceConfigUpdatingPhase {
			configUpdate := config.DeepCopy()
			configUpdate.Status.Phase = cceConfigUpdatingPhase
			if config, err = h.cceCC.UpdateStatus(configUpdate); err != nil {
				return config, err
			}
		}
		h.cceEnqueueAfter(config.Namespace, config.Name, 30*time.Second)
		return config, nil
	}
	if config.Status.AvailableZone != utils.GetValue(cluster.Spec.Az) {
		configUpdate := config.DeepCopy()
		configUpdate.Status.AvailableZone = utils.GetValue(cluster.Spec.Az)
		if config, err = h.cceCC.UpdateStatus(configUpdate); err != nil {
			return config, err
		}
	}

	// Get the created node pools and build upstream cluster state.
	nodePools, err := cce.GetClusterNodePools(h.driver.CCE, config.Status.ClusterID, false)
	if err != nil {
		return config, err
	}
	if nodePools == nil || nodePools.Items == nil {
		return config, fmt.Errorf("checkAndUpdate: failed to get cluster nodePools: Items is nil")
	}
	if len(*nodePools.Items) == 0 {
		logrus.WithFields(logrus.Fields{
			"cluster": config.Name,
			"phase":   config.Status.Phase,
		}).Infof("cluster [%s] does not have nodePool", config.Spec.Name)
	}
	for _, np := range *nodePools.Items {
		if np.Status == nil || np.Status.Phase == nil || np.Metadata == nil || np.Spec == nil {
			continue
		}
		switch *np.Status.Phase {
		case cce_model.GetNodePoolStatusPhaseEnum().SYNCHRONIZED,
			cce_model.GetNodePoolStatusPhaseEnum().SYNCHRONIZING,
			cce_model.GetNodePoolStatusPhaseEnum().DELETING,
			cce_model.GetNodePoolStatusPhaseEnum().ERROR,
			cce_model.GetNodePoolStatusPhaseEnum().SOLD_OUT:
			logrus.WithFields(logrus.Fields{
				"cluster": config.Name,
				"phase":   config.Status.Phase,
			}).Infof("waiting for nodepool %q %q status: %q",
				np.Metadata.Name, utils.GetValue(np.Metadata.Uid), np.Status.Phase.Value())
			if config.Status.Phase != cceConfigUpdatingPhase {
				config = config.DeepCopy()
				config.Status.Phase = cceConfigUpdatingPhase
				if config, err = h.cceCC.UpdateStatus(config); err != nil {
					return config, err
				}
			}
			h.cceEnqueueAfter(config.Namespace, config.Name, 30*time.Second)
			return config, nil
		}
	}
	upstreamSpec, err := BuildUpstreamClusterState(cluster, nodePools)
	if err != nil {
		return config, err
	}

	return h.updateUpstreamClusterState(upstreamSpec, config)
}

// updateUpstreamClusterState compares the upstream spec with the config spec,
// then updates the upstream CCE cluster to match the config spec.
// Function often returns after a single update because once the cluster is
// in updating phase in CCE, no more updates will be accepted until the current
// update is finished.
func (h *Handler) updateUpstreamClusterState(
	upstreamSpec *ccev1.CCEClusterConfigSpec, config *ccev1.CCEClusterConfig,
) (*ccev1.CCEClusterConfig, error) {
	logrus.WithFields(logrus.Fields{
		"cluster": config.Name,
		"phase":   config.Status.Phase,
	}).Debugf("start updateUpstreamClusterState")

	var err error
	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		config, err = h.cceCC.Get(config.Namespace, config.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		configUpdate := config.DeepCopy()
		configUpdate.Status.HostNetwork = upstreamSpec.HostNetwork
		configUpdate.Status.ContainerNetwork = upstreamSpec.ContainerNetwork
		config, err = h.cceCC.UpdateStatus(configUpdate)
		return err
	})
	if err != nil {
		return config, err
	}

	if config.Spec.Imported {
		if config.Status.Phase != cceConfigActivePhase {
			logrus.WithFields(logrus.Fields{
				"cluster": config.Name,
				"phase":   config.Status.Phase,
			}).Infof("cluster [%s] finished updating", config.Spec.Name)
			config = config.DeepCopy()
			config.Status.Phase = cceConfigActivePhase
			config, err = h.cceCC.UpdateStatus(config)
			if err != nil {
				return config, err
			}
			return config, nil
		}
		return config, nil
	}

	// Check kubernetes version for upgrade cluster.
	if upstreamSpec.Version != config.Spec.Version {
		oldVer, err := semver.NewVersion(upstreamSpec.Version)
		if err != nil {
			return config, fmt.Errorf("invalid version %q: %w", upstreamSpec.Version, err)
		}
		newVer, err := semver.NewVersion(config.Spec.Version)
		if err != nil {
			return config, fmt.Errorf("invalid version %q: %w", config.Spec.Version, err)
		}
		if oldVer.Compare(newVer) >= 0 {
			return config, fmt.Errorf("unsupported to downgrade cluster from %q to %q",
				upstreamSpec.Version, config.Spec.Version)
		}

		res, err := cce.UpgradeCluster(h.driver.CCE, config)
		if err != nil {
			return config, err
		}
		if res == nil || res.Uid == nil {
			return config, fmt.Errorf("UpgradeCluster returns invalid data")
		}
		logrus.WithFields(logrus.Fields{
			"cluster": config.Name,
			"phase":   config.Status.Phase,
		}).Infof("start upgrade cluster [%s] from version %q to %q, task id [%s]",
			config.Spec.Name, config.Spec.Version, upstreamSpec.Version, *res.Uid)
		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			config, err = h.cceCC.Get(config.Namespace, config.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			configUpdate := config.DeepCopy()
			configUpdate.Status.UpgradeClusterTaskID = *res.Uid
			config, err = h.cceCC.UpdateStatus(configUpdate)
			return err
		})
		if err != nil {
			return config, err
		}
		return h.enqueueUpdate(config)
	}

	// Update cluster info.
	if _, err = cce.UpdateCluster(h.driver.CCE, config); err != nil {
		return config, err
	}
	// Update nodePool infos.
	for _, np := range config.Spec.NodePools {
		if np.ID == "" {
			continue
		}
		_, err := cce.UpdateNodePool(h.driver.CCE, config.Status.ClusterID, &np)
		if err != nil {
			return config, err
		}
	}

	// Compare nodePools between upstream & config spec.
	enqueueNodePool := false
	upstreamNodePools := make(map[string]bool, len(upstreamSpec.NodePools))
	specNodePools := make(map[string]bool, len(config.Spec.NodePools))
	for _, np := range upstreamSpec.NodePools {
		upstreamNodePools[np.ID] = true
	}
	for i := 0; i < len(config.Spec.NodePools); i++ {
		np := &config.Spec.NodePools[i]
		if np.ID != "" {
			specNodePools[np.ID] = true
		}
		// This for loop create nodePool exists in spec but not exists in upstream.
		if upstreamNodePools[np.ID] {
			logrus.WithFields(logrus.Fields{
				"cluster": config.Name,
				"phase":   config.Status.Phase,
			}).Debugf("found nodePool [%s] ID [%s] exists in cce cluster [%s]",
				np.Name, np.ID, config.Spec.Name)
			continue
		}
		// Create nodePool if not fount in upstream spec.
		res, err := cce.CreateNodePool(
			h.driver.CCE, config.Status.ClusterID, np)
		if err != nil {
			return config, err
		}
		if res.Metadata == nil {
			return config, fmt.Errorf("createNodePool returns invalid data")
		}
		// Update nodePool ID.
		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			config, err = h.cceCC.Get(config.Namespace, config.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			configUpdate := config.DeepCopy()
			configUpdate.Spec.NodePools[i].ID = utils.GetValue(res.Metadata.Uid)
			config, err = h.cceCC.Update(configUpdate)
			return err
		})
		if err != nil {
			return config, err
		}
		enqueueNodePool = true
		logrus.WithFields(logrus.Fields{
			"cluster": config.Name,
			"phase":   config.Status.Phase,
		}).Infof("request to create nodePool [%s] ID [%s]",
			res.Metadata.Name, utils.GetValue(res.Metadata.Uid))
	}
	for _, np := range upstreamSpec.NodePools {
		// This for loop deletes nodePool exists in upstream but not exists in spec.
		if specNodePools[np.ID] {
			continue
		}
		logrus.WithFields(logrus.Fields{
			"cluster": config.Name,
			"phase":   config.Status.Phase,
		}).Debugf("nodePool [%s] ID [%s] exists in upstream but not exists in config spec",
			np.Name, np.ID)
		// Delete nodePool.
		res, err := cce.DeleteNodePool(
			h.driver.CCE, config.Status.ClusterID, np.ID)
		if err != nil {
			return config, err
		}
		if res.Metadata == nil {
			return config, fmt.Errorf("deleteNodePool returns invalid data")
		}
		enqueueNodePool = true
		logrus.WithFields(logrus.Fields{
			"cluster": config.Name,
			"phase":   config.Status.Phase,
		}).Infof("request to delete nodePool [%s] ID [%s]",
			np.Name, np.ID)
	}
	if enqueueNodePool {
		return h.enqueueUpdate(config)
	}

	if config.Status.Phase != cceConfigActivePhase {
		logrus.WithFields(logrus.Fields{
			"cluster": config.Name,
			"phase":   config.Status.Phase,
		}).Infof("cluster [%s] finished updating", config.Spec.Name)
		config = config.DeepCopy()
		config.Status.Phase = cceConfigActivePhase
		config, err = h.cceCC.UpdateStatus(config)
		if err != nil {
			return config, err
		}
		return config, nil
	}
	return config, nil
}

func (h *Handler) importCluster(config *ccev1.CCEClusterConfig) (*ccev1.CCEClusterConfig, error) {
	cluster, err := cce.GetCluster(h.driver.CCE, config.Spec.ClusterID)
	if err != nil {
		return config, err
	}
	nodePools, err := cce.GetClusterNodePools(h.driver.CCE, config.Spec.ClusterID, false)
	if err != nil {
		return config, err
	}

	upstreamConfig, err := BuildUpstreamClusterState(cluster, nodePools)
	if err != nil {
		return config, err
	}

	var clusterExternalIP string
	if cluster.Status != nil && cluster.Status.Endpoints != nil {
		for _, endpoint := range *cluster.Status.Endpoints {
			if endpoint.Type == nil || endpoint.Url == nil {
				continue
			}
			if *endpoint.Type == "External" {
				u, err := url.Parse(*endpoint.Url)
				if err != nil {
					continue
				}
				clusterExternalIP = u.Hostname()
				logrus.WithFields(logrus.Fields{
					"cluster": config.Name,
					"phase":   config.Status.Phase,
				}).Infof("imported cluster [%s] external IP: %q",
					config.Spec.Name, clusterExternalIP)
			}
		}
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		config, err = h.cceCC.Get(config.Namespace, config.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		configUpdate := config.DeepCopy()
		configUpdate.Status.ClusterID = config.Spec.ClusterID
		configUpdate.Status.HostNetwork = upstreamConfig.HostNetwork
		configUpdate.Status.ContainerNetwork = upstreamConfig.ContainerNetwork
		configUpdate.Status.ClusterExternalIP = clusterExternalIP
		config, err = h.cceCC.UpdateStatus(configUpdate)
		return err
	})

	if err = h.createCASecret(config); err != nil {
		return config, err
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		config, err = h.cceCC.Get(config.Namespace, config.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		configUpdate := config.DeepCopy()
		configUpdate.Status.Phase = cceConfigActivePhase
		config, err = h.cceCC.UpdateStatus(configUpdate)
		return err
	})

	return config, nil
}

// createCASecret creates a secret containing a CA and endpoint for use in generating a kubeconfig file.
func (h *Handler) createCASecret(config *ccev1.CCEClusterConfig) error {
	// TODO: refresh cluster certs after 3 years
	certs, err := cce.GetClusterCert(h.driver.CCE, config.Status.ClusterID, 365*3)
	if err != nil {
		return err
	}
	if certs == nil || len(*certs.Clusters) == 0 {
		return fmt.Errorf("createCASecret failed: no clusters returned from cce.GetClusterCert")
	}

	var clusterCert *cce_model.Clusters
	for _, c := range *certs.Clusters {
		if config.Spec.PublicAccess && utils.GetValue(c.Name) == "externalClusterTLSVerify" {
			clusterCert = &c
			break
		}
		if utils.GetValue(c.Name) == "internalCluster" {
			clusterCert = &c
		}
	}
	if clusterCert == nil {
		return fmt.Errorf("failed to find cluster endpoint")
	}

	endpoint := utils.GetValue(clusterCert.Cluster.Server)
	ca := utils.GetValue(clusterCert.Cluster.CertificateAuthorityData)
	logrus.WithFields(logrus.Fields{
		"cluster": config.Name,
		"phase":   config.Status.Phase,
	}).Infof("create secret [%s]", config.Name)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.Name,
			Namespace: config.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: ccev1.SchemeGroupVersion.String(),
					Kind:       cceClusterConfigKind,
					UID:        config.UID,
					Name:       config.Name,
				},
			},
		},
		Data: map[string][]byte{
			"endpoint": []byte(endpoint),
			"ca":       []byte(ca),
		},
	}
	_, err = h.secrets.Get(config.Namespace, config.Name, metav1.GetOptions{})
	if err != nil {
		// Secret does not created yet
		_, err = h.secrets.Create(secret)
	}

	return err
}

// enqueueUpdate enqueues the config if it is already in the updating phase. Otherwise, the
// phase is updated to "updating". This is important because the object needs to reenter the
// onChange handler to start waiting on the update.
func (h *Handler) enqueueUpdate(config *ccev1.CCEClusterConfig) (*ccev1.CCEClusterConfig, error) {
	if config.Status.Phase == cceConfigUpdatingPhase {
		h.cceEnqueue(config.Namespace, config.Name)
		return config, nil
	}
	var err error
	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		config, err = h.cceCC.Get(config.Namespace, config.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		config = config.DeepCopy()
		config.Status.Phase = cceConfigUpdatingPhase
		config, err = h.cceCC.UpdateStatus(config)
		return err
	})
	return config, err
}

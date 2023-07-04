package controller

import (
	"context"
	"fmt"
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
)

const (
	controllerName           = "cce-operator"
	controllerRemoveName     = "cce-operator-remove"
	cceConfigCreatingPhase   = "creating"
	cceConfigNotCreatedPhase = ""
	cceConfigActivePhase     = "active"
	cceConfigUpdatingPhase   = "updating"
	cceConfigImportingPhase  = "importing"
	allOpen                  = "0.0.0.0/0"
	cceClusterConfigKind     = "CCEClusterConfig"
)

type Handler struct {
	cceCC           ccecontrollers.CCEClusterConfigClient
	cceEnqueueAfter func(namespace, name string, duration time.Duration)
	cceEnqueue      func(namespace, name string)
	secrets         wranglerv1.SecretClient
	secretsCache    wranglerv1.SecretCache
	driver          HuaweiDriver
	log             *logrus.Entry
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
		log:             logrus.WithFields(nil),
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

	if config.Status.Phase == "" {
		h.log = logrus.WithFields(logrus.Fields{
			"cluster": config.Name,
			"phase":   "unknow",
		})
	} else {
		h.log = logrus.WithFields(logrus.Fields{
			"cluster": config.Name,
			"phase":   config.Status.Phase,
		})
	}

	if err := h.newDriver(h.secretsCache, config.Spec); err != nil {
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
			h.log.Warnf("onChange error: %v", err)
			if huawei.IsHuaweiError(err) {
				hwerr, _ := huawei.NewHuaweiError(err)
				message = hwerr.String()
			} else {
				message = err.Error()
			}
		}

		if config.Status.FailureMessage == message {
			return config, err
		}

		if message != "" {
			config = config.DeepCopy()
			if config.Status.Phase == cceConfigActivePhase {
				// can assume an update is failing
				config.Status.Phase = cceConfigUpdatingPhase
			}
		}
		config.Status.FailureMessage = message

		var recordErr error
		config, recordErr = h.cceCC.UpdateStatus(config)
		if recordErr != nil {
			logrus.Errorf("Error recording cce cluster config [%s] failure message: %v",
				config.Name, recordErr)
		}

		time.Sleep(time.Millisecond * 500)

		return config, err
	}
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
	cluster, err := cce.GetCluster(h.driver.CCE, config.Status.ClusterID)
	if err != nil {
		return config, err
	}
	switch *cluster.Status.Phase {
	case cce.ClusterStatusDeleting,
		cce.ClusterStatusResizing,
		cce.ClusterStatusUpgrading:
		h.log.Infof("waiting for cluster [%s] finish status %q",
			config.Spec.Name, *cluster.Status.Phase)
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

	nodes, err := cce.GetClusterNodes(h.driver.CCE, config.Status.ClusterID)
	if err != nil {
		return config, err
	}
	if nodes.Items == nil {
		return config, fmt.Errorf("checkAndUpdate: failed to get cluster nodes: Items is nil")
	}
	for _, node := range *nodes.Items {
		if node.Metadata == nil || node.Status == nil || node.Status.Phase == nil {
			continue
		}
		switch *node.Status.Phase {
		case cce_model.GetNodeStatusPhaseEnum().INSTALLING,
			// cce_model.GetNodeStatusPhaseEnum().INSTALLED,
			// cce_model.GetNodeStatusPhaseEnum().SHUT_DOWN,
			cce_model.GetNodeStatusPhaseEnum().UPGRADING,
			// cce_model.GetNodeStatusPhaseEnum().ABNORMAL,
			// cce_model.GetNodeStatusPhaseEnum().ERROR,
			cce_model.GetNodeStatusPhaseEnum().BUILD,
			cce_model.GetNodeStatusPhaseEnum().DELETING:
			if config.Status.Phase != cceConfigUpdatingPhase {
				config = config.DeepCopy()
				config.Status.Phase = cceConfigUpdatingPhase
				if config, err = h.cceCC.UpdateStatus(config); err != nil {
					return config, err
				}
			}
			h.log.Infof("waiting for cluster [%s] update node [%s], status: %+v",
				config.Spec.Name, *node.Metadata.Name, node.Status.Phase.Value())
			h.cceEnqueueAfter(config.Namespace, config.Name, 30*time.Second)
			return config, nil
		}
	}
	if len(*nodes.Items) == 0 {
		h.log.Infof("cluster [%s] does not have nodes", config.Spec.Name)
	}

	// Get the created node pools and build upstream cluster state.
	nodePools, err := cce.GetClusterNodePools(h.driver.CCE, config.Status.ClusterID, false)
	if err != nil {
		return config, err
	}
	if nodePools.Items == nil {
		return config, fmt.Errorf("checkAndUpdate: failed to get cluster nodePools: Items is nil")
	}
	upstreamSpec, err := BuildUpstreamClusterState(h.driver.CCE, cluster, nodePools)
	if err != nil {
		return config, err
	}

	return h.updateUpstreamClusterState(upstreamSpec, config)
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

	return nil
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
		return config, fmt.Errorf("error generating and setting network: %w", err)
	}

	// create cluster
	if config.Status.ClusterID != "" {
		_, err := cce.GetCluster(h.driver.CCE, config.Status.ClusterID)
		if err == nil {
			h.log.Infof("cluster [%s] already created, switch to creating phase",
				config.Status.ClusterID)
			config = config.DeepCopy()
			config.Status.Phase = cceConfigCreatingPhase
			return h.cceCC.UpdateStatus(config)
		}
	}
	h.log.Infof("creating cluster [%s]", config.Spec.Name)
	cluster, err := cce.CreateCluster(h.driver.CCE, config)
	if err != nil {
		return config, err
	}
	config = config.DeepCopy()
	config.Status.ClusterID = *cluster.Metadata.Uid
	config.Status.Phase = cceConfigCreatingPhase
	config.Status.FailureMessage = ""
	h.log.Infof("created cluster %q", config.Status.ClusterID)

	return h.cceCC.UpdateStatus(config)
}

func (h *Handler) validateCreate(config *ccev1.CCEClusterConfig) error {
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

	if !config.Spec.Imported {
		listClustersRes, err := cce.ListClusters(h.driver.CCE)
		if err != nil {
			return fmt.Errorf("ListClusters: %w", err)
		}
		for _, cluster := range *listClustersRes.Items {
			if config.Spec.Name == cluster.Metadata.Name {
				return fmt.Errorf("cannot create cluster [%s] because a cluster"+
					" in CCE exists with the same name", cluster.Metadata.Name)
			}
		}
		cannotBeEmptyError := "field [%s] cannot be empty for non-import cluster [%s]"
		if config.Spec.CredentialSecret == "" {
			return fmt.Errorf(cannotBeEmptyError, "credentialSecret", config.Name)
		}
		if config.Spec.Name == "" {
			return fmt.Errorf(cannotBeEmptyError, "name", config.Name)
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
		if len(config.Spec.NodePools) == 0 {
			return fmt.Errorf(cannotBeEmptyError, "nodePools", config.Name)
		}
		for _, node := range config.Spec.NodePools {
			nt := node.NodeTemplate
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
			if nt.Count == 0 {
				return fmt.Errorf(cannotBeEmptyError, "nodePool.nodeTemplate.Count", config.Name)
			}
		}
	}

	return nil
}

func (h *Handler) generateAndSetNetworking(config *ccev1.CCEClusterConfig) (*ccev1.CCEClusterConfig, error) {
	var err error
	if config.Status.ContainerNetwork.Mode == "" || config.Status.ContainerNetwork.CIDR == "" {
		config = config.DeepCopy()
		if config.Spec.ContainerNetwork.Mode != "" {
			config.Status.ContainerNetwork.Mode = config.Spec.ContainerNetwork.Mode
		} else {
			config.Status.ContainerNetwork.Mode = network.DefaultContainerNetworkMode
		}
		if config.Spec.ContainerNetwork.CIDR != "" {
			config.Status.ContainerNetwork.CIDR = config.Spec.ContainerNetwork.CIDR
		} else {
			config.Status.ContainerNetwork.CIDR = network.DefaultContainerNetworkCIDR
		}
		config, err = h.cceCC.UpdateStatus(config)
		if err != nil {
			return config, err
		}
	}

	if config.Status.HostNetwork.VpcID != "" && config.Status.HostNetwork.SubnetID != "" {
		return config, nil
	}

	// Configure VPC & Subnet.
	if config.Spec.HostNetwork.VpcID == "" {
		h.log.Infof("VPC ID not provided, will create VPC and subnet...")
		h.log.Infof("creating VPC...")
		vpcRes, err := network.CreateVPC(
			h.driver.VPC,
			common.GenResourceName("vpc"),
			network.DefaultVpcCIDR,
		)
		if err != nil {
			return config, err
		}
		h.log.Infof("querying DNS server of region [%s]", config.Spec.RegionID)
		dnsServers, err := network.ListNameServers(h.driver.DNS, config.Spec.RegionID)
		if err != nil {
			return config, fmt.Errorf("failed to ListNameServers: %w", err)
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
		h.log.Infof("found DNS server of region [%s] %v",
			config.Spec.RegionID, dnsRecords)
		h.log.Infof("creating subnet...")
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
		config = config.DeepCopy()
		config.Status.HostNetwork.VpcID = vpcRes.Vpc.Id
		config.Status.HostNetwork.SubnetID = subnetRes.Subnet.Id
		if config, err = h.cceCC.UpdateStatus(config); err != nil {
			return config, err
		}
		h.log.Infof("created VPC [%s] [%s]",
			vpcRes.Vpc.Name, config.Status.HostNetwork.VpcID)
		h.log.Infof("created subnet [%s] [%s]",
			subnetRes.Subnet.Name, config.Status.HostNetwork.SubnetID)
	} else if config.Spec.HostNetwork.SubnetID == "" {
		// VPC ID provided but subnet ID not provided.
		// Create a subnet based on the provided VPC.
		// Ensure provided VPC exists first.
		_, err = network.GetVPC(h.driver.VPC, config.Spec.HostNetwork.VpcID)
		if err != nil {
			return config, err
		}
		h.log.Infof("VPC ID provided [%s], will create subnet for this VPC",
			config.Spec.HostNetwork.VpcID)
		h.log.Infof("querying DNS server of region [%s]", config.Spec.RegionID)
		dnsServers, err := network.ListNameServers(h.driver.DNS, config.Spec.RegionID)
		if err != nil {
			return config, fmt.Errorf("failed to ListNameServers: %w", err)
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
		h.log.Infof("found DNS server of region [%s] %v",
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
		config = config.DeepCopy()
		config.Status.HostNetwork.SubnetID = subnetRes.Subnet.Id
		if config, err = h.cceCC.UpdateStatus(config); err != nil {
			return config, err
		}
		h.log.Infof("created subnet [%s] [%s]",
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
		config = config.DeepCopy()
		config.Status.HostNetwork.VpcID = config.Spec.HostNetwork.VpcID
		config.Status.HostNetwork.SubnetID = config.Spec.HostNetwork.SubnetID
		if config, err = h.cceCC.UpdateStatus(config); err != nil {
			return config, err
		}
		h.log.Infof("VPC [%s] and subnet [%s] are provided",
			config.Status.HostNetwork.VpcID, config.Status.HostNetwork.SubnetID)
	}

	return h.cceCC.UpdateStatus(config)
}

func (h *Handler) newDriver(secretsCache wranglerv1.SecretCache, spec ccev1.CCEClusterConfigSpec) error {
	auth, err := NewHuaweiClientAuth(secretsCache, spec)
	if err != nil {
		return err
	}
	h.driver = *NewHuaweiDriver(auth)

	return nil
}

func NewHuaweiClientAuth(
	secretsCache wranglerv1.SecretCache, spec ccev1.CCEClusterConfigSpec,
) (*common.ClientAuth, error) {
	region := spec.RegionID
	if region == "" {
		return nil, fmt.Errorf("regionID not provided")
	}
	ns, id := utils.Parse(spec.CredentialSecret)
	if spec.CredentialSecret == "" {
		return nil, fmt.Errorf("cce credential secret not provided")
	}

	secret, err := secretsCache.Get(ns, id)
	if err != nil {
		return nil, fmt.Errorf("error getting secret %s/%s: %w", ns, id, err)
	}

	accessKeyBytes := secret.Data["huaweicredentialConfig-accessKey"]
	secretKeyBytes := secret.Data["huaweicredentialConfig-secretKey"]
	projectIDBytes := secret.Data["huaweicredentialConfig-projectID"]
	if accessKeyBytes == nil || secretKeyBytes == nil || projectIDBytes == nil {
		return nil, fmt.Errorf("invalid huawei cloud credential")
	}
	accessKey := string(accessKeyBytes)
	secretKey := string(secretKeyBytes)
	projectID := string(projectIDBytes)
	return common.NewClientAuth(accessKey, secretKey, region, projectID), nil
}

func (h *Handler) waitForCreationComplete(config *ccev1.CCEClusterConfig) (*ccev1.CCEClusterConfig, error) {
	cluster, err := cce.GetCluster(h.driver.CCE, config.Status.ClusterID)
	if err != nil {
		return config, fmt.Errorf("waitForCreationComplete: %w", err)
	}

	if cluster.Status == nil {
		return config, fmt.Errorf("no cluster status was returned")
	}

	if *cluster.Status.Phase == cce.ClusterStatusUnavailable {
		return config, fmt.Errorf("creation failed for cluster %q",
			cluster.Metadata.Name)
	}

	if *cluster.Status.Phase == cce.ClusterStatusAvailable {
		if err := h.createCASecret(config, cluster); err != nil {
			return config, fmt.Errorf("createCASecret: %w", err)
		}
		h.log.Infof("cluster [%s] created successfully", config.Name)
		config = config.DeepCopy()
		config.Status.Phase = cceConfigActivePhase
		return h.cceCC.UpdateStatus(config)
	}

	h.log.Infof("waiting for cluster [%s], status %q",
		config.Spec.Name, *cluster.Status.Phase)
	h.cceEnqueueAfter(config.Namespace, config.Name, 30*time.Second)

	return h.cceCC.UpdateStatus(config)
}

// updateUpstreamClusterState compares the upstream spec with the config spec,
// then updates the upstream CCE cluster to match the config spec.
// Function often returns after a single update because once the cluster is
// in updating phase in CCE, no more updates will be accepted until the current
// update is finished.
func (h *Handler) updateUpstreamClusterState(
	upstreamSpec *ccev1.CCEClusterConfigSpec, config *ccev1.CCEClusterConfig,
) (*ccev1.CCEClusterConfig, error) {
	h.log.Debugf("start updateUpstreamClusterState")

	// Add UpdateCluster here if needed.

	var err error
	config = config.DeepCopy()
	config.Status.NodePools = upstreamSpec.NodePools
	config, err = h.cceCC.UpdateStatus(config)
	if err != nil {
		return config, err
	}

	if config.Spec.Imported {
		return config, nil
	}

	// Update node pools.
	// Compare nodeConfigs between status & spec.
	updateNode := false
	for _, specNP := range config.Spec.NodePools {
		// This for loop creates node pool exists in spec but not exists in status
		found := false
		for _, statusNP := range config.Status.NodePools {
			if CompareNodePool(&specNP, &statusNP) {
				h.log.Debugf("found nodepool [%s] exists in cce cluster [%s]",
					specNP.Name, config.Spec.Name)
				found = true
				break
			}
		}
		if !found {
			// Create node pool
			res, err := cce.CreateNodePool(
				h.driver.CCE, config.Status.ClusterID, &specNP)
			if err != nil {
				return config, fmt.Errorf("updateUpstreamClusterState createNodePool: %w", err)
			}
			if res.Metadata == nil {
				return config, fmt.Errorf("updateUpstreamClusterState: CreateNodePool returns invalid data")
			}
			updateNode = true
			h.log.Infof("request to create node pool [%s], ID [%s]",
				res.Metadata.Name, utils.GetValue(res.Metadata.Uid))
		}
	}
	for _, statusNP := range config.Status.NodePools {
		// This for loop deletes node pools exists in status but not exists in spec
		found := false
		for _, specNP := range config.Spec.NodePools {
			if CompareNodePool(&statusNP, &specNP) {
				found = true
			}
		}
		if !found {
			h.log.Debugf("nodepool [%s] exists in upstream spec but not exists in spec, should delete",
				statusNP.Name)
			// Delete node pool
			res, err := cce.DeleteNodePool(
				h.driver.CCE, config.Status.ClusterID, statusNP.ID)
			if err != nil {
				return config, fmt.Errorf("updateUpstreamClusterState deleteNodePool: %w", err)
			}
			if res.Metadata == nil {
				return config, fmt.Errorf("updateUpstreamClusterState: DeleteNodePool returns invalid data")
			}
			updateNode = true
			h.log.Infof("request to delete node group [%s], ID [%s]",
				statusNP.Name, statusNP.ID)
		}
	}
	if updateNode {
		if config.Status.Phase != cceConfigUpdatingPhase {
			config = config.DeepCopy()
			config.Status.Phase = cceConfigUpdatingPhase
			config, err = h.cceCC.UpdateStatus(config)
			if err != nil {
				return config, err
			}
		}
		h.cceEnqueueAfter(config.Namespace, config.Name, 30*time.Second)
	}

	if config.Status.Phase != cceConfigActivePhase {
		h.log.Infof("cluster [%s] finished updating", config.Spec.Name)
		config = config.DeepCopy()
		config.Status.Phase = cceConfigActivePhase
		return h.cceCC.UpdateStatus(config)
	}
	return config, nil
}

func (h *Handler) importCluster(config *ccev1.CCEClusterConfig) (*ccev1.CCEClusterConfig, error) {
	return h.cceCC.UpdateStatus(config)
}

// createCASecret creates a secret containing a CA and endpoint for use in generating a kubeconfig file.
func (h *Handler) createCASecret(config *ccev1.CCEClusterConfig, cluster *cce_model.ShowClusterResponse) error {
	certs, err := cce.GetClusterCert(h.driver.CCE, cluster)
	if err != nil {
		return err
	}
	if certs == nil || len(*certs.Clusters) == 0 {
		return fmt.Errorf("no cluster returned")
	}

	for _, cert := range *certs.Clusters {
		endpoint := *cert.Cluster.Server
		ca := *cert.Cluster.CertificateAuthorityData
		_, err = h.secrets.Create(
			&corev1.Secret{
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
			})
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
	config = config.DeepCopy()
	config.Status.Phase = cceConfigUpdatingPhase
	return h.cceCC.UpdateStatus(config)
}
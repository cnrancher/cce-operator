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
	"github.com/cnrancher/cce-operator/pkg/huawei/dns"
	"github.com/cnrancher/cce-operator/pkg/huawei/eip"
	"github.com/cnrancher/cce-operator/pkg/huawei/nat"
	"github.com/cnrancher/cce-operator/pkg/huawei/vpc"
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
	drivers         map[string]*HuaweiDriver
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
		drivers:         make(map[string]*HuaweiDriver),
	}

	// Register handlers
	cce.OnChange(ctx, controllerName, h.recordError(h.OnCCEConfigChanged))
	cce.OnRemove(ctx, controllerRemoveName, h.OnCCEConfigRemoved)
}

func (h *Handler) OnCCEConfigChanged(_ string, config *ccev1.CCEClusterConfig) (*ccev1.CCEClusterConfig, error) {
	if config == nil {
		return nil, nil
	}
	if config.Name == "" || config.DeletionTimestamp != nil {
		return config, nil
	}

	// Ensure the driver in h.drivers map exists.
	if err := h.setupHuaweiDriver(&config.Spec); err != nil {
		return config, err
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
	driver := h.drivers[config.Spec.HuaweiCredentialSecret]
	var err error
	if config, err = h.cceCC.Get(config.Namespace, config.Name, metav1.GetOptions{}); err != nil {
		return config, err
	}
	if err = h.validateCreate(config); err != nil {
		return config, err
	}

	if config.Spec.Imported {
		config = config.DeepCopy()
		config.Status.Phase = cceConfigImportingPhase
		return h.cceCC.UpdateStatus(config)
	}

	if config, err = h.generateAndSetNetworking(config); err != nil {
		return config, err
	}

	if config.Spec.ClusterID != "" {
		if _, err := cce.ShowCluster(driver.CCE, config.Spec.ClusterID); err == nil {
			logrus.WithFields(logrus.Fields{
				"cluster": config.Name,
				"phase":   "create",
			}).Infof("cluster [%s] ID [%s] created, switch to creating phase",
				config.Spec.Name, config.Spec.ClusterID)
			config = config.DeepCopy()
			config.Status.Phase = cceConfigCreatingPhase
			return h.cceCC.UpdateStatus(config)
		}
	}
	// Create cluster.
	cluster, err := cce.CreateCluster(driver.CCE, config)
	if err != nil {
		return config, err
	}
	if cluster == nil || cluster.Metadata == nil || cluster.Metadata.Uid == nil ||
		cluster.Spec == nil || cluster.Spec.HostNetwork == nil {
		return config, fmt.Errorf("cce.CreateCluster return invalid data")
	}
	// Use the RetryOnConflict to prevent repeated creation of cluster.
	// Update spec (ClusterID).
	if err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		config, err = h.cceCC.Get(config.Namespace, config.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		config = config.DeepCopy()
		config.Spec.ClusterID = utils.Value(cluster.Metadata.Uid)
		config, err = h.cceCC.Update(config)
		return err
	}); err != nil {
		return config, err
	}
	// Update status (Phase).
	if err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		config, err = h.cceCC.Get(config.Namespace, config.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		config = config.DeepCopy()
		config.Status.Phase = cceConfigCreatingPhase
		config.Status.FailureMessage = ""
		config, err = h.cceCC.UpdateStatus(config)
		return err
	}); err != nil {
		return config, err
	}
	logrus.WithFields(logrus.Fields{
		"cluster": config.Name,
		"phase":   "create",
	}).Infof("request to create cluster name [%s] ID [%s]",
		config.Spec.Name, config.Spec.ClusterID)
	return config, err
}

func (h *Handler) generateAndSetNetworking(config *ccev1.CCEClusterConfig) (*ccev1.CCEClusterConfig, error) {
	driver := h.drivers[config.Spec.HuaweiCredentialSecret]
	var err error
	// Create Cluster PublicIP.
	if config.Spec.PublicAccess && config.Spec.PublicIP.CreateEIP && config.Status.ClusterExternalIP == "" {
		res, err := eip.CreatePublicIP(driver.EIP, &config.Spec.PublicIP.Eip)
		if err != nil {
			return config, err
		}
		if res.Publicip == nil {
			return config, fmt.Errorf("CreatePublicIP returns invalid data")
		}
		logrus.WithFields(logrus.Fields{
			"cluster": config.Name,
			"phase":   "create",
		}).Infof("created cluster public IP [%s] address [%s]",
			utils.Value(res.Publicip.Alias), utils.Value(res.Publicip.PublicIpAddress))
		// Use the RetryOnConflict to prevent repeated creation of EIP.
		if err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			config, err = h.cceCC.Get(config.Namespace, config.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			configUpdate := config.DeepCopy()
			configUpdate.Status.ClusterExternalIP = utils.Value(res.Publicip.PublicIpAddress)
			configUpdate.Status.CreatedClusterEIPID = utils.Value(res.Publicip.Id)
			config, err = h.cceCC.UpdateStatus(configUpdate)
			return err
		}); err != nil {
			return config, err
		}
	}
	// Do not create cluster public IP and use existing EIP address.
	if config.Spec.PublicAccess && config.Status.ClusterExternalIP == "" && config.Spec.ExtendParam.ClusterExternalIP != "" {
		if err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			config, err = h.cceCC.Get(config.Namespace, config.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			configUpdate := config.DeepCopy()
			configUpdate.Status.ClusterExternalIP = config.Spec.ExtendParam.ClusterExternalIP
			config, err = h.cceCC.UpdateStatus(configUpdate)
			return err
		}); err != nil {
			return config, err
		}
	}

	// Configure VPC & Subnet.
	if config.Spec.HostNetwork.VpcID == "" {
		logrus.WithFields(logrus.Fields{
			"cluster": config.Name,
			"phase":   "create",
		}).Infof("VPC ID not provided, will create VPC and subnet")
		vpcRes, err := vpc.CreateVPC(
			driver.VPC,
			common.GenResourceName("vpc"),
			vpc.DefaultVpcCIDR,
		)
		if err != nil {
			return config, err
		}
		if vpcRes.Vpc == nil {
			return config, fmt.Errorf("CreateVPC returns invalid data")
		}
		logrus.WithFields(logrus.Fields{
			"cluster": config.Name,
			"phase":   "create",
		}).Infof("created VPC name [%s] ID [%s]", vpcRes.Vpc.Name, vpcRes.Vpc.Id)
		dnsServers, err := dns.ListNameServers(driver.DNS, config.Spec.RegionID)
		if err != nil {
			return config, err
		}
		if dnsServers.Nameservers == nil || len(*dnsServers.Nameservers) == 0 {
			return config, fmt.Errorf("ListNameServers returns invalid data")
		}
		var dnsRecords = make([]string, 2)
		for _, nameserver := range *dnsServers.Nameservers {
			if nameserver.NsRecords == nil || len(*nameserver.NsRecords) == 0 {
				continue
			}
			for i := 0; i < len(*nameserver.NsRecords) && i < 2; i++ {
				ns := (*nameserver.NsRecords)[i]
				dnsRecords[i] = utils.Value(ns.Address)
			}
		}
		logrus.WithFields(logrus.Fields{
			"cluster": config.Name,
			"phase":   "create",
		}).Infof("query DNS server of region [%s]: %s, %s",
			config.Spec.RegionID, dnsRecords[0], dnsRecords[1])
		subnetRes, err := vpc.CreateSubnet(
			driver.VPC,
			common.GenResourceName("subnet"),
			vpcRes.Vpc.Id,
			dnsRecords[0],
			dnsRecords[1],
		)
		if err != nil {
			return config, err
		}
		if subnetRes == nil || subnetRes.Subnet == nil {
			return config, fmt.Errorf("CreateSubnet returns invalid data")
		}
		logrus.WithFields(logrus.Fields{
			"cluster": config.Name,
			"phase":   "create",
		}).Infof("created subnet for VPC [%s]: name [%s] ID [%s]",
			vpcRes.Vpc.Name, subnetRes.Subnet.Name, subnetRes.Subnet.Id)
		// Update status.
		// Use the RetryOnConflict to prevent repeated creation of VPC & Subnet.
		if err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			config, err = h.cceCC.Get(config.Namespace, config.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			configUpdate := config.DeepCopy()
			configUpdate.Status.CreatedVpcID = vpcRes.Vpc.Id
			configUpdate.Status.CreatedSubnetID = subnetRes.Subnet.Id
			config, err = h.cceCC.UpdateStatus(configUpdate)
			return err
		}); err != nil {
			return config, err
		}
		// Update spec.
		if err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			config, err = h.cceCC.Get(config.Namespace, config.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			configUpdate := config.DeepCopy()
			configUpdate.Spec.HostNetwork.VpcID = vpcRes.Vpc.Id
			configUpdate.Spec.HostNetwork.SubnetID = subnetRes.Subnet.Id
			config, err = h.cceCC.Update(configUpdate)
			return err
		}); err != nil {
			return config, err
		}
	} else if config.Spec.HostNetwork.SubnetID == "" {
		// VPC ID provided but subnet ID not provided.
		// Create a subnet based on the provided VPC.
		// Ensure provided VPC exists first.
		vpcRes, err := vpc.ShowVPC(driver.VPC, config.Spec.HostNetwork.VpcID)
		if err != nil {
			return config, err
		}
		if vpcRes == nil || vpcRes.Vpc == nil {
			return config, fmt.Errorf("ShowVPC returns invalid data")
		}
		logrus.WithFields(logrus.Fields{
			"cluster": config.Name,
			"phase":   "create",
		}).Infof("VPC ID provided [%s], will create subnet for this VPC",
			config.Spec.HostNetwork.VpcID)
		logrus.WithFields(logrus.Fields{
			"cluster": config.Name,
			"phase":   "create",
		}).Infof("querying DNS server of region [%s]", config.Spec.RegionID)
		dnsServers, err := dns.ListNameServers(driver.DNS, config.Spec.RegionID)
		if err != nil {
			return config, err
		}
		if dnsServers.Nameservers == nil || len(*dnsServers.Nameservers) == 0 {
			return config, fmt.Errorf("ListNameServers returns invalid data")
		}
		var dnsRecords = make([]string, 2)
		for _, nameserver := range *dnsServers.Nameservers {
			if nameserver.NsRecords == nil || len(*nameserver.NsRecords) == 0 {
				continue
			}
			for i := 0; i < len(*nameserver.NsRecords) && i < 2; i++ {
				ns := (*nameserver.NsRecords)[i]
				dnsRecords[i] = utils.Value(ns.Address)
			}
		}
		logrus.WithFields(logrus.Fields{
			"cluster": config.Name,
			"phase":   "create",
		}).Infof("found DNS server of region [%s]: %s, %s",
			config.Spec.RegionID, dnsRecords[0], dnsRecords[1])
		subnetRes, err := vpc.CreateSubnet(
			driver.VPC,
			common.GenResourceName("subnet"),
			config.Spec.HostNetwork.VpcID,
			dnsRecords[0],
			dnsRecords[1],
		)
		if err != nil {
			return config, err
		}
		if subnetRes == nil || subnetRes.Subnet == nil {
			return config, fmt.Errorf("CreateSubnet returns invalid data")
		}
		logrus.WithFields(logrus.Fields{
			"cluster": config.Name,
			"phase":   "create",
		}).Infof("created subnet for VPC [%s]: name [%s] ID [%s]",
			vpcRes.Vpc.Name, subnetRes.Subnet.Name, subnetRes.Subnet.Id)
		// Update status.
		// Use the RetryOnConflict to prevent repeated creation of subnet.
		if err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			config, err = h.cceCC.Get(config.Namespace, config.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			configUpdate := config.DeepCopy()
			configUpdate.Status.CreatedSubnetID = subnetRes.Subnet.Id
			config, err = h.cceCC.UpdateStatus(configUpdate)
			return err
		}); err != nil {
			return config, err
		}
		// Update spec.
		if err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			config, err = h.cceCC.Get(config.Namespace, config.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			configUpdate := config.DeepCopy()
			configUpdate.Spec.HostNetwork.SubnetID = subnetRes.Subnet.Id
			config, err = h.cceCC.Update(configUpdate)
			return err
		}); err != nil {
			return config, err
		}
	} else {
		// Both VPC ID and subnet ID are provided.
		// Ensure provided VPC and subnet exists.
		_, err = vpc.ShowVPC(driver.VPC, config.Spec.HostNetwork.VpcID)
		if err != nil {
			return config, err
		}
		_, err = vpc.ShowSubnet(driver.VPC, config.Spec.HostNetwork.SubnetID)
		if err != nil {
			return config, err
		}
		logrus.WithFields(logrus.Fields{
			"cluster": config.Name,
			"phase":   "create",
		}).Infof("VPC [%s] and subnet [%s] are provided",
			config.Spec.HostNetwork.VpcID, config.Spec.HostNetwork.SubnetID)
	}

	// Add timeout to avoid huawei "The virSubnet is not in the vpc" error.
	time.Sleep(time.Second)

	// Configure NAT Gateway.
	if config.Spec.NatGateway.Enabled && config.Status.CreatedNatGatewayID == "" {
		natRes, err := nat.CreateNatGateway(driver.NAT, common.GenResourceName("nat"), &config.Spec)
		if err != nil {
			return config, err
		}
		if natRes.NatGateway == nil {
			return config, fmt.Errorf("CreateNatGateway returns invalid data")
		}
		logrus.WithFields(logrus.Fields{
			"cluster": config.Name,
			"phase":   "create",
		}).Infof("created NAT Gateway [%s] ID [%s]",
			natRes.NatGateway.Name, natRes.NatGateway.Id)
		// Use the RetryOnConflict to prevent repeated creation of NAT Gateway.
		if err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			config, err = h.cceCC.Get(config.Namespace, config.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			configUpdate := config.DeepCopy()
			configUpdate.Status.CreatedNatGatewayID = natRes.NatGateway.Id
			config, err = h.cceCC.UpdateStatus(configUpdate)
			return err
		}); err != nil {
			return config, err
		}
	}
	// Configure SNAT Rule for NAT Gateway.
	if config.Spec.NatGateway.Enabled && config.Status.CreatedSNATRuleID == "" {
		// Configure EIP for SNAT Rule.
		var snatEipID string
		if config.Spec.NatGateway.ExistingEIPID != "" {
			// Existing EIP ID provided, ensure provided EIP exists.
			snatEipID = config.Spec.NatGateway.ExistingEIPID
			if _, err = eip.ShowPublicip(driver.EIP, snatEipID); err != nil {
				return config, err
			}
			logrus.WithFields(logrus.Fields{
				"cluster": config.Name,
				"phase":   "create",
			}).Infof("use existing EIP ID [%s] for SNAT Rule", snatEipID)
		} else if config.Status.CreatedSNatRuleEIPID == "" {
			// Create EIP for SNAT Rule.
			eipRes, err := eip.CreatePublicIP(driver.EIP, &config.Spec.PublicIP.Eip)
			if err != nil {
				return config, err
			}
			if eipRes.Publicip == nil {
				return config, fmt.Errorf("CreatePublicIP returns invalid data")
			}
			logrus.WithFields(logrus.Fields{
				"cluster": config.Name,
				"phase":   "create",
			}).Infof("created public IP [%s] address [%s] for SNAT Rule",
				utils.Value(eipRes.Publicip.Alias), utils.Value(eipRes.Publicip.PublicIpAddress))
			snatEipID = utils.Value(eipRes.Publicip.Id)
			// Use the RetryOnConflict to prevent repeated creation of EIP used by SNAT Rule.
			if err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
				config, err = h.cceCC.Get(config.Namespace, config.Name, metav1.GetOptions{})
				if err != nil {
					return err
				}
				configUpdate := config.DeepCopy()
				configUpdate.Status.CreatedSNatRuleEIPID = utils.Value(eipRes.Publicip.Id)
				config, err = h.cceCC.UpdateStatus(configUpdate)
				return err
			}); err != nil {
				return config, err
			}
		} else {
			snatEipID = config.Status.CreatedSNatRuleEIPID
		}

		snatRuleRes, err := nat.CreateNatGatewaySnatRule(
			driver.NAT,
			config.Status.CreatedNatGatewayID,
			config.Spec.HostNetwork.SubnetID,
			snatEipID,
			0,
		)
		if err != nil {
			return config, err
		}
		if snatRuleRes == nil || snatRuleRes.SnatRule == nil {
			return config, fmt.Errorf("nat.CreateNatGatewaySnatRule returns invalid data")
		}
		logrus.WithFields(logrus.Fields{
			"cluster": config.Name,
			"phase":   "create",
		}).Infof("created SNAT Rule [%s]", snatRuleRes.SnatRule.Id)
		// Use the RetryOnConflict to prevent repeated creation of SNAT Rule.
		if err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			config, err = h.cceCC.Get(config.Namespace, config.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			configUpdate := config.DeepCopy()
			configUpdate.Status.CreatedSNATRuleID = snatRuleRes.SnatRule.Id
			config, err = h.cceCC.UpdateStatus(configUpdate)
			return err
		}); err != nil {
			return config, err
		}
	}

	return config, nil
}

func (h *Handler) waitForCreationComplete(config *ccev1.CCEClusterConfig) (*ccev1.CCEClusterConfig, error) {
	driver := h.drivers[config.Spec.HuaweiCredentialSecret]
	cluster, err := cce.ShowCluster(driver.CCE, config.Spec.ClusterID)
	if err != nil {
		return config, err
	}
	if cluster.Status == nil || cluster.Metadata == nil || cluster.Spec == nil {
		return config, fmt.Errorf("cce.GetCluster returns invalid data")
	}
	if utils.Value(cluster.Status.Phase) == cce.ClusterStatusUnavailable {
		return config, fmt.Errorf("creation failed for cluster %q: %v",
			cluster.Metadata.Name, utils.Value(cluster.Status.Reason))
	}
	if utils.Value(cluster.Status.Phase) == cce.ClusterStatusAvailable {
		if err := h.createCASecret(config); err != nil {
			return config, fmt.Errorf("createCASecret: %w", err)
		}
		logrus.WithFields(logrus.Fields{
			"cluster": config.Name,
			"phase":   config.Status.Phase,
		}).Infof("cluster [%s] ID [%s] created successfully",
			config.Spec.Name, config.Spec.ClusterID)
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
	}).Infof("waiting for cluster [%s] status [%s]",
		config.Spec.Name, utils.Value(cluster.Status.Phase))
	h.cceEnqueueAfter(config.Namespace, config.Name, 30*time.Second)

	return config, nil
}

func (h *Handler) checkAndUpdate(config *ccev1.CCEClusterConfig) (*ccev1.CCEClusterConfig, error) {
	driver := h.drivers[config.Spec.HuaweiCredentialSecret]
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

	// Get cluster status.
	cluster, err := cce.ShowCluster(driver.CCE, config.Spec.ClusterID)
	if err != nil {
		return config, err
	}
	if cluster == nil || cluster.Status == nil || cluster.Spec == nil || cluster.Spec.HostNetwork == nil {
		return config, fmt.Errorf("GetCluster returns invalid data")
	}

	// Check cluster upgrade status.
	if config.Status.UpgradeClusterTaskID != "" {
		ok, err := clusterUpgradeable(config.Spec.Version, *cluster.Spec.Version)
		if err != nil {
			return config, err
		}
		if !ok {
			// Delete UpgradeClusterTaskID if the config cluster version match
			// the upstream cluster version.
			config = config.DeepCopy()
			config.Status.UpgradeClusterTaskID = ""
			return h.cceCC.UpdateStatus(config)
		}

		res, err := cce.ShowUpgradeClusterTask(driver.CCE, config.Spec.ClusterID, config.Status.UpgradeClusterTaskID)
		if err != nil {
			hwerr, _ := huawei.NewHuaweiError(err)
			if hwerr.StatusCode == 404 {
				config = config.DeepCopy()
				config.Status.UpgradeClusterTaskID = ""
				return h.cceCC.UpdateStatus(config)
			}
			return config, err
		}
		if res != nil && res.Spec != nil && res.Status != nil {
			switch utils.Value(res.Status.Phase) {
			case "Success", "":
				logrus.WithFields(logrus.Fields{
					"cluster": config.Name,
					"phase":   config.Status.Phase,
				}).Infof("cluster [%s] upgrade to [%v]",
					config.Spec.Name, utils.Value(cluster.Spec.Version))
				config = config.DeepCopy()
				config.Status.UpgradeClusterTaskID = ""
				return h.cceCC.UpdateStatus(config)
			case "Failed":
				return config, fmt.Errorf("failed to upgrade cluster [%s] to %v, status [%s]",
					config.Spec.Name, config.Spec.Version, utils.Value(res.Status.Phase))
			default:
				logrus.WithFields(logrus.Fields{
					"cluster": config.Name,
					"phase":   config.Status.Phase,
				}).Infof("waiting for cluster [%s] upgrade task status [%s]",
					config.Spec.Name, utils.Value(res.Status.Phase))
			}
			h.cceEnqueueAfter(config.Namespace, config.Name, 30*time.Second)
			return config, nil
		}
	}

	// Check cluster status.
	switch utils.Value(cluster.Status.Phase) {
	case cce.ClusterStatusDeleting,
		cce.ClusterStatusResizing,
		cce.ClusterStatusUpgrading:
		logrus.WithFields(logrus.Fields{
			"cluster": config.Name,
			"phase":   config.Status.Phase,
		}).Infof("waiting for cluster [%s] finish status [%s]",
			config.Spec.Name, utils.Value(cluster.Status.Phase))
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
	configUpdate := config.DeepCopy()
	var updateStatus = false
	if config.Status.AvailableZone != utils.Value(cluster.Spec.Az) {
		configUpdate.Status.AvailableZone = utils.Value(cluster.Spec.Az)
		updateStatus = true
	}
	var updateEndpoints = false
	if cluster.Status.Endpoints != nil {
		if len(configUpdate.Status.Endpoints) == len(*cluster.Status.Endpoints) {
			for i := range *cluster.Status.Endpoints {
				if configUpdate.Status.Endpoints[i].Type != utils.Value((*cluster.Status.Endpoints)[i].Type) ||
					configUpdate.Status.Endpoints[i].URL != utils.Value((*cluster.Status.Endpoints)[i].Url) {
					updateEndpoints = true
				}
			}
		} else {
			updateEndpoints = true
		}
	}
	if updateEndpoints {
		configUpdate.Status.Endpoints = nil
		for _, e := range *cluster.Status.Endpoints {
			configUpdate.Status.Endpoints = append(configUpdate.Status.Endpoints, ccev1.CCEClusterEndpoints{
				URL:  utils.Value(e.Url),
				Type: utils.Value(e.Type),
			})
		}
		updateStatus = true
	}
	if updateStatus {
		if config, err = h.cceCC.UpdateStatus(configUpdate); err != nil {
			return config, err
		}
	}

	if config, err = h.cceCC.Get(config.Namespace, config.Name, metav1.GetOptions{}); err != nil {
		return config, err
	}
	if len(config.Spec.CreatedNodePoolIDs) > 0 {
		logrus.WithFields(logrus.Fields{
			"cluster": config.Name,
			"phase":   config.Status.Phase,
		}).Debugf("waiting for cce-operator-controller (Rancher) to update nodePool ID: %v",
			utils.PrintObject(config.Spec.CreatedNodePoolIDs))
		h.cceEnqueueAfter(config.Namespace, config.Name, 10*time.Second)
		return config, nil
	}

	// Get the created node pools and build upstream cluster state.
	nodePools, err := cce.ListNodePools(driver.CCE, config.Spec.ClusterID, false)
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
				np.Metadata.Name, utils.Value(np.Metadata.Uid), np.Status.Phase.Value())
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
	driver := h.drivers[config.Spec.HuaweiCredentialSecret]
	var err error
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

	// Init security group ID for created cluster if the security group wasn't
	// provided when creating the cluster.
	if config.Spec.HostNetwork.SecurityGroup == "" &&
		config.Spec.HostNetwork.SecurityGroup != upstreamSpec.HostNetwork.SecurityGroup {
		configUpdate := config.DeepCopy()
		configUpdate.Spec.HostNetwork.SecurityGroup = upstreamSpec.HostNetwork.SecurityGroup
		config, err = h.cceCC.Update(configUpdate)
		if err != nil {
			return config, err
		}
	}

	// Check kubernetes version for upgrade cluster.
	if ok, err := clusterUpgradeable(config.Spec.Version, upstreamSpec.Version); err != nil {
		return config, err
	} else if ok {
		return h.upgradeCluster(config)
	}
	// Check cluster flavor is resizable.
	var clusterResizable = false
	if config.Spec.Flavor != "" && config.Spec.Flavor != upstreamSpec.Flavor {
		cv, err := semver.NewVersion(config.Spec.Version)
		if err != nil {
			return config, err
		}
		minVersion := semver.New(1, 15, 0, "", "")
		if cv.Compare(minVersion) >= 0 {
			clusterResizable = true
		}
	}
	if clusterResizable {
		logrus.WithFields(logrus.Fields{
			"cluster": config.Name,
			"phase":   config.Status.Phase,
		}).Infof("cluster [%s] flavor change detected: %v -> %v",
			config.Spec.Name, upstreamSpec.Flavor, config.Spec.Flavor)

		res, err := cce.ResizeCluster(
			driver.CCE,
			config.Spec.ClusterID,
			config.Spec.Flavor,
			config.Spec.ExtendParam.IsAutoPay,
		)
		if err != nil {
			return config, err
		}
		if res == nil || res.JobID == nil {
			return config, fmt.Errorf("ResizeCluster returns invalid data")
		}
		logrus.WithFields(logrus.Fields{
			"cluster": config.Name,
			"phase":   config.Status.Phase,
		}).Infof("start resize cluster [%s] job ID %q",
			config.Spec.Name, utils.Value(res.JobID))
		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			config, err = h.cceCC.Get(config.Namespace, config.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			configUpdate := config.DeepCopy()
			configUpdate.Status.ResizeClusterJobID = utils.Value(res.JobID)
			config, err = h.cceCC.UpdateStatus(configUpdate)
			return err
		})
		if err != nil {
			return config, err
		}
		return h.enqueueUpdate(config)
	}

	// Update cluster info.
	if _, err = cce.UpdateCluster(driver.CCE, config); err != nil {
		return config, err
	}
	// Update nodePool infos.
	for _, np := range config.Spec.NodePools {
		if np.ID == "" {
			continue
		}
		_, err := cce.UpdateNodePool(driver.CCE, config.Spec.ClusterID, &np)
		if err != nil {
			return config, err
		}
	}

	// Compare nodePools between upstream & config spec.
	enqueueNodePool := false
	upstreamNodePoolIDs := make(map[string]bool, len(upstreamSpec.NodePools))
	upstreamNodePoolNames := make(map[string]bool, len(upstreamSpec.NodePools))
	specNodePoolIDs := make(map[string]bool, len(config.Spec.NodePools))
	specNodePoolNames := make(map[string]bool, len(config.Spec.NodePools))
	for _, np := range upstreamSpec.NodePools {
		upstreamNodePoolIDs[np.ID] = true
		upstreamNodePoolNames[np.Name] = true
	}
	createdNodePoolIDs := map[string]string{}
	for i := 0; i < len(config.Spec.NodePools); i++ {
		// This for loop create nodePool exists in spec but not exists in upstream.
		np := &config.Spec.NodePools[i]
		if np.ID != "" {
			specNodePoolIDs[np.ID] = true
			specNodePoolNames[np.Name] = true
		}
		if upstreamNodePoolIDs[np.ID] {
			logrus.WithFields(logrus.Fields{
				"cluster": config.Name,
				"phase":   config.Status.Phase,
			}).Debugf("found nodePool [%s] ID [%s] exists in cce cluster [%s]",
				np.Name, np.ID, config.Spec.Name)
			continue
		}
		if upstreamNodePoolNames[np.Name] {
			// Prevent creation of node pools with the same name.
			logrus.WithFields(logrus.Fields{
				"cluster": config.Name,
				"phase":   config.Status.Phase,
			}).Debugf("nodePool [%s] exists in cce cluster [%s], skip creation",
				np.Name, config.Spec.Name)
			continue
		}
		// Create nodePool if not found in upstream spec.
		res, err := cce.CreateNodePool(driver.CCE, config.Spec.ClusterID, np)
		if err != nil {
			return config, err
		}
		if res.Metadata == nil {
			return config, fmt.Errorf("CreateNodePool returns invalid data")
		}
		enqueueNodePool = true
		logrus.WithFields(logrus.Fields{
			"cluster": config.Name,
			"phase":   config.Status.Phase,
		}).Infof("request to create nodePool [%s] ID [%s]",
			res.Metadata.Name, utils.Value(res.Metadata.Uid))
		createdNodePoolIDs[res.Metadata.Name] = utils.Value(res.Metadata.Uid)
	}
	if len(createdNodePoolIDs) != 0 {
		// Update CreatedNodePoolIDs map to let cce-operator-controller (in Rancher)
		// know that some node pools were created by operator and update its ID properly.
		if err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			config, err = h.cceCC.Get(config.Namespace, config.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			configUpdate := config.DeepCopy()
			configUpdate.Spec.CreatedNodePoolIDs = createdNodePoolIDs
			config, err = h.cceCC.Update(configUpdate)
			return err
		}); err != nil {
			return config, err
		}
	}

	for _, np := range upstreamSpec.NodePools {
		// This for loop deletes nodePool exists in upstream but not exists in spec.
		if specNodePoolIDs[np.ID] {
			continue
		}
		logrus.WithFields(logrus.Fields{
			"cluster": config.Name,
			"phase":   config.Status.Phase,
		}).Debugf("nodePool [%s] ID [%s] exists in upstream but not exists in config spec",
			np.Name, np.ID)
		// Delete nodePool.
		if _, err := cce.DeleteNodePool(driver.CCE, config.Spec.ClusterID, np.ID); err != nil {
			return config, err
		}
		enqueueNodePool = true
		logrus.WithFields(logrus.Fields{
			"cluster": config.Name,
			"phase":   config.Status.Phase,
		}).Infof("request to delete nodePool [%s] ID [%s]", np.Name, np.ID)
	}
	if enqueueNodePool {
		if config.Status.Phase != cceConfigUpdatingPhase {
			if err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
				config, err = h.cceCC.Get(config.Namespace, config.Name, metav1.GetOptions{})
				if err != nil {
					return err
				}
				config = config.DeepCopy()
				config.Status.Phase = cceConfigUpdatingPhase
				config, err = h.cceCC.UpdateStatus(config)
				return err
			}); err != nil {
				return config, err
			}
		}
		h.cceEnqueueAfter(config.Namespace, config.Name, 10*time.Second)
		return config, nil
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
	driver := h.drivers[config.Spec.HuaweiCredentialSecret]
	cluster, err := cce.ShowCluster(driver.CCE, config.Spec.ClusterID)
	if err != nil {
		return config, err
	}
	var clusterExternalIP string
	if cluster == nil || cluster.Status == nil || cluster.Status.Endpoints == nil {
		return config, fmt.Errorf("ShowCluster returns invalid data, cluster ID [%s]", config.Spec.ClusterID)
	}

	// Check cluster status.
	switch utils.Value(cluster.Status.Phase) {
	case cce.ClusterStatusCreating,
		cce.ClusterStatusDeleting,
		cce.ClusterStatusResizing,
		cce.ClusterStatusUpgrading:
		logrus.WithFields(logrus.Fields{
			"cluster": config.Name,
			"phase":   config.Status.Phase,
		}).Infof("waiting for cluster [%s] finish status [%s]",
			config.Spec.Name, utils.Value(cluster.Status.Phase))
		h.cceEnqueueAfter(config.Namespace, config.Name, 15*time.Second)
		return config, nil
	}

	for _, endpoint := range *cluster.Status.Endpoints {
		if endpoint.Type == nil || endpoint.Url == nil {
			continue
		}
		if utils.Value(endpoint.Type) == "External" {
			u, err := url.Parse(utils.Value(endpoint.Url))
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
	if clusterExternalIP == "" {
		logrus.WithFields(logrus.Fields{
			"cluster": config.Name,
			"phase":   config.Status.Phase,
		}).Infof("cluster [%s] does not have external IP address",
			config.Spec.Name)
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		config, err = h.cceCC.Get(config.Namespace, config.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		configUpdate := config.DeepCopy()
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
	driver := h.drivers[config.Spec.HuaweiCredentialSecret]
	certs, err := cce.GetClusterCert(driver.CCE, config.Spec.ClusterID, 0)
	if err != nil {
		return err
	}
	if certs == nil || certs.Clusters == nil || len(*certs.Clusters) == 0 {
		return fmt.Errorf("createCASecret failed: no clusters returned from GetClusterCert")
	}

	var clusterCert *cce_model.Clusters
	for _, c := range *certs.Clusters {
		if config.Spec.PublicAccess && utils.Value(c.Name) == "externalClusterTLSVerify" {
			clusterCert = &c
			break
		}
		if utils.Value(c.Name) == "internalCluster" {
			clusterCert = &c
		}
	}
	if clusterCert == nil {
		return fmt.Errorf("createCASecret: failed to find cluster endpoint")
	}
	if clusterCert.Cluster == nil {
		return fmt.Errorf("createCASecret: ClusterCert is nil pointer")
	}

	if _, err = h.secrets.Get(config.Namespace, config.Name, metav1.GetOptions{}); err == nil {
		// Secret already created
		return nil
	}

	endpoint := utils.Value(clusterCert.Cluster.Server)
	ca := utils.Value(clusterCert.Cluster.CertificateAuthorityData)
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
	if _, err = h.secrets.Create(secret); err != nil {
		return err
	}
	logrus.WithFields(logrus.Fields{
		"cluster": config.Name,
		"phase":   config.Status.Phase,
	}).Infof("create secret [%s]", config.Name)

	return nil
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

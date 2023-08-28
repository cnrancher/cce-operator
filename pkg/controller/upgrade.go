package controller

import (
	"fmt"

	"github.com/Masterminds/semver/v3"
	ccev1 "github.com/cnrancher/cce-operator/pkg/apis/cce.pandaria.io/v1"
	"github.com/cnrancher/cce-operator/pkg/huawei/cce"
	"github.com/cnrancher/cce-operator/pkg/utils"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

func (h *Handler) upgradeCluster(
	config *ccev1.CCEClusterConfig,
) (*ccev1.CCEClusterConfig, error) {
	driver := h.drivers[config.Spec.HuaweiCredentialSecret]
	res, err := cce.UpgradeCluster(driver.CCE, config)
	if err != nil {
		return config, err
	}
	if res == nil || res.Metadata == nil {
		return config, fmt.Errorf("UpgradeCluster returns invalid data")
	}
	logrus.WithFields(logrus.Fields{
		"cluster": config.Name,
		"phase":   config.Status.Phase,
	}).Infof("start upgrade cluster [%s] to %q, task id [%s]",
		config.Spec.Name, config.Spec.Version, utils.Value(res.Metadata.Uid))
	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		config, err = h.cceCC.Get(config.Namespace, config.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		configUpdate := config.DeepCopy()
		configUpdate.Status.UpgradeClusterTaskID = utils.Value(res.Metadata.Uid)
		config, err = h.cceCC.UpdateStatus(configUpdate)
		return err
	})
	if err != nil {
		return config, err
	}
	return h.enqueueUpdate(config)
}

func clusterUpgradeable(oldVer, newVer string) (bool, error) {
	if oldVer == newVer {
		return false, nil
	}

	t, err := semver.NewVersion(oldVer)
	if err != nil {
		return false, fmt.Errorf("invalid version %q: %w", oldVer, err)
	}
	ov := semver.New(t.Major(), t.Minor(), 0, "", "")

	t, err = semver.NewVersion(newVer)
	if err != nil {
		return false, fmt.Errorf("invalid version %q: %w", newVer, err)
	}
	nv := semver.New(t.Major(), t.Minor(), 0, "", "")

	// Compare the major minor version only.
	if ov.Compare(nv) == 0 {
		return false, nil
	}
	if ov.Compare(nv) < 0 {
		return false, fmt.Errorf("unsupported to downgrade cluster from %q to %q",
			oldVer, newVer)
	}
	return true, nil
}

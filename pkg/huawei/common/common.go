package common

import (
	"fmt"

	"github.com/cnrancher/cce-operator/pkg/utils"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
)

var (
	resourceNamePrefix         = "rancher-managed"
	DefaultResourceDescription = "Managed by Rancher, do not edit!"
)

type ClientAuth struct {
	Region     string
	Credential *basic.Credentials
}

func NewClientAuth(ak, sk, region, projectID string) *ClientAuth {
	return &ClientAuth{
		Region: region,
		Credential: basic.NewCredentialsBuilder().
			WithAk(ak).
			WithSk(sk).
			WithProjectId(projectID).
			Build(),
	}
}

// GenResourceName generates the name of resource.
// vpc: rancher-managed-vpc-[RANDOM_STR]
// subnet: rancher-managed-subnet-[RANDOM_STR]
func GenResourceName(name string) string {
	return fmt.Sprintf("%s-%s-%s",
		resourceNamePrefix, name, utils.RandomString(5))
}

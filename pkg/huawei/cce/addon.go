package cce

import (
	"github.com/cnrancher/cce-operator/pkg/utils"
	cce "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cce/v3"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cce/v3/model"
)

func CreateAddonInstance(
	client *cce.CceClient,
) (*model.CreateAddonInstanceResponse, error) {
	return client.CreateAddonInstance(&model.CreateAddonInstanceRequest{
		Body: &model.InstanceRequest{
			Metadata: &model.AddonMetadata{
				Name:  utils.GetPtr(""),
				Alias: utils.GetPtr(""),
			},
			Spec: &model.InstanceRequestSpec{
				ClusterID:         "",
				AddonTemplateName: "",
				Values:            map[string]interface{}{},
			},
		},
	})
}

func ListAddonInstances(
	client *cce.CceClient, clusterID, addonName string,
) (*model.ListAddonInstancesResponse, error) {
	return client.ListAddonInstances(&model.ListAddonInstancesRequest{
		AddonTemplateName: &addonName,
		ClusterId:         clusterID,
	})
}

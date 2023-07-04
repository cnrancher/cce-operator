/*
Copyright 2023 [Rancher Labs, Inc](https://rancher.com).

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type CCEClusterConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CCEClusterConfigSpec   `json:"spec"`
	Status CCEClusterConfigStatus `json:"status"`
}

// CCEClusterConfigSpec is the spec for a CCEClusterConfig resource
type CCEClusterConfigSpec struct {
	CredentialSecret     string              `json:"credentialSecret"`
	Category             string              `json:"category,omitempty"` // 集群类别: CCE
	RegionID             string              `json:"regionID,omitempty"`
	ClusterID            string              `json:"clusterID,omitempty"` // ClusterID only used for import cluster
	Imported             bool                `json:"imported,omitempty"`
	Name                 string              `json:"name" norman:"noupdate"`
	Labels               map[string]string   `json:"labels,omitempty"`
	Type                 string              `json:"type"`
	Flavor               string              `json:"flavor" norman:"noupdate"`
	Version              string              `json:"version" norman:"noupdate"`
	Description          string              `json:"description,omitempty" norman:"noupdate"`
	Ipv6Enable           bool                `json:"ipv6Enable,omitempty"`
	HostNetwork          CCEHostNetwork      `json:"hostNetwork"`
	ContainerNetwork     CCEContainerNetwork `json:"containerNetwork"`
	EniNetwork           CCEEniNetwork       `json:"eniNetwork,omitempty"`
	Authentication       CCEAuthentication   `json:"authentication,omitempty"`
	BillingMode          int32               `json:"clusterBillingMode,omitempty" norman:"noupdate"`
	KubernetesSvcIPRange string              `json:"kubernetesSvcIPRange,omitempty" norman:"noupdate"`
	Tags                 map[string]string   `json:"tags"`
	KubeProxyMode        string              `json:"kubeProxyMode,omitempty"`
	NodePools            []CCENodePool       `json:"nodePools,omitempty"`
}

type CCEClusterConfigStatus struct {
	Phase          string `json:"phase"`
	FailureMessage string `json:"failureMessage"`

	ClusterID        string              `json:"clusterID"`
	HostNetwork      CCEHostNetwork      `json:"hostNetwork"`
	ContainerNetwork CCEContainerNetwork `json:"containerNetwork"`
	NodePools        []CCENodePool       `json:"nodePools,omitempty"`
}

type CCEHostNetwork struct {
	VpcID         string `json:"vpcID,omitempty"`
	SubnetID      string `json:"subnetID,omitempty"`
	SecurityGroup string `json:"securityGroup,omitempty"`
}

type CCEContainerNetwork struct {
	Mode string `json:"mode"`
	CIDR string `json:"cidr"`
	// CIDRs []string `json:"cidrs"` // 10.0.0.0/12~19, 172.16.0.0/16~19, 192.168.0.0/16~19
}

type CCEEniNetwork struct {
	Subnets []string `json:"subnets"`
}

type CCEAuthentication struct {
	Mode                string                 `json:"mode"`
	AuthenticatingProxy CCEAuthenticatingProxy `json:"authenticatingProxy"`
}

type CCEAuthenticatingProxy struct {
	Ca         string `json:"ca,omitempty"`
	Cert       string `json:"cert,omitempty"`
	PrivateKey string `json:"privateKey,omitempty"`
}

type CCENodePool struct {
	Name                 string                  `json:"name,omitempty"`   // 节点池名称
	Type                 string                  `json:"type"`             // 节点池类型：vm, ElasticBMS, pm (default: vm)
	ID                   string                  `json:"nodeID,omitempty"` // 节点池 ID，仅用于查询
	NodeTemplate         CCENodeTemplate         `json:"nodeTemplate"`
	InitialNodeCount     int32                   `json:"initialNodeCount"` // 节点池初始化节点个数。查询时为节点池目标节点数量。
	Autoscaling          NodePoolNodeAutoscaling `json:"autoscaling"`
	PodSecurityGroups    []string                `json:"podSecurityGroups"`
	CustomSecurityGroups []string                `json:"customSecurityGroups"` // 节点池自定义安全组相关配置，未指定安全组ID，新建节点将添加 Node 节点默认安全组。
}

type CCENodeTemplate struct {
	Flavor          string      `json:"flavor"`          // 节点池规格
	AvailableZone   string      `json:"availableZone"`   // 可用区
	OperatingSystem string      `json:"operatingSystem"` // 节点操作系统
	SSHKey          string      `json:"sshKey"`          // SSH 密钥名称（不支持帐号密码登录）
	RootVolume      Volume      `json:"rootVolume"`      // 节点的系统盘
	DataVolumes     []Volume    `json:"dataVolumes"`     // 节点的数据盘
	PublicIP        PublicIP    `json:"publicIP"`        // 节点公网IP
	Count           int32       `json:"count"`           // 批量创建节点时的数量
	BillingMode     int32       `json:"billingMode"`     // 节点计费模式
	Runtime         string      `json:"runtime"`         // 容器运行时，docker 或 containerd
	ExtendParam     ExtendParam `json:"extendParam"`     // 节点扩展参数
}

type NodePoolNodeAutoscaling struct {
	Enable                bool  `json:"enable"`                // 是否开启自动扩缩容
	MinNodeCount          int32 `json:"minNodeCount"`          // 若开启自动扩缩容，最小能缩容的节点个数
	MaxNodeCount          int32 `json:"maxNodeCount"`          // 若开启自动扩缩容，最大能扩容的节点个数
	ScaleDownCooldownTime int32 `json:"scaleDownCooldownTime"` // 节点保留时间，单位为分钟，扩容出来的节点在这个时间内不会被缩掉
	Priority              int32 `json:"priority"`              // 节点池权重，更高的权重在扩容时拥有更高的优先级
}

type Volume struct {
	Size int32  `json:"size"`
	Type string `json:"type"`
}

type Bandwidth struct {
	ChargeMode string `json:"chargeMode,omitempty"`
	Size       int32  `json:"size,omitempty"`
	ShareType  string `json:"shareType,omitempty"`
}

type Eip struct {
	Iptype    string    `json:"ipType,omitempty"`    // 弹性IP类型
	Bandwidth Bandwidth `json:"bandwidth,omitempty"` // 弹性IP的带宽参数
}

type PublicIP struct {
	Ids   []string `json:"ids,omitempty"`   // 已有的弹性IP的ID列表。数量不得大于待创建节点数
	Count int32    `json:"count,omitempty"` // 要动态创建的弹性IP个数。
	Eip   Eip      `json:"eip,omitempty"`   // 弹性IP参数。
}

type ExtendParam struct {
	PeriodType  string `json:"periodType,omitempty"`  // month / year, 作为请求参数，billingMode为1（包周期）或2（已废弃：自动付费包周期）时必选。
	PeriodNum   int32  `json:"periodNum,omitempty"`   // 订购周期数
	IsAutoRenew string `json:"isAutoRenew,omitempty"` // 是否自动续订
}
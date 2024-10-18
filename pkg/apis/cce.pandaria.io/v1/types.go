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
	HuaweiCredentialSecret string                `json:"huaweiCredentialSecret"`
	Category               string                `json:"category"` // 集群类别: CCE
	RegionID               string                `json:"regionID"`
	ClusterID              string                `json:"clusterID"` // 仅导入集群时需要提供
	Imported               bool                  `json:"imported"`
	Name                   string                `json:"name"`
	Labels                 map[string]string     `json:"labels,omitempty"`
	Type                   string                `json:"type"`
	Flavor                 string                `json:"flavor"`
	Version                string                `json:"version"`
	Description            string                `json:"description"`
	Ipv6Enable             bool                  `json:"ipv6Enable,omitempty"`
	HostNetwork            CCEHostNetwork        `json:"hostNetwork"`
	ContainerNetwork       CCEContainerNetwork   `json:"containerNetwork"`
	EniNetwork             CCEEniNetwork         `json:"eniNetwork,omitempty"`
	Authentication         CCEAuthentication     `json:"authentication,omitempty"`
	BillingMode            int32                 `json:"clusterBillingMode"`
	KubernetesSvcIPRange   string                `json:"kubernetesSvcIPRange"`
	Tags                   map[string]string     `json:"tags"`
	KubeProxyMode          string                `json:"kubeProxyMode"`
	PublicAccess           bool                  `json:"publicAccess"` // 若为 true，则创建集群时需提供已有的 ClusterExternalIP 或配置 PublicIP
	PublicIP               CCEClusterPublicIP    `json:"publicIP"`     // PublicAccess 为 true 且未提供已有的 ClusterExternalIP 时，创建公网 IP
	NatGateway             CCENatGateway         `json:"natGateway"`   // 使用 NAT 使节点访问公网
	ExtendParam            CCEClusterExtendParam `json:"extendParam,omitempty"`
	NodePools              []CCENodePool         `json:"nodePools"`

	// CreatedNodePoolIDs is a temporary map to store nodePool ID by nodePool name
	// and let cce-operator-controller (in Rancher) to know that some nodePools were
	// created by cce-operator and update its ID to CCE cluster config in Rancher.
	CreatedNodePoolIDs map[string]string `json:"createdNodePoolIDs"`
}

type CCEClusterConfigStatus struct {
	Phase          string `json:"phase"`
	FailureMessage string `json:"failureMessage"`

	ClusterExternalIP string                `json:"clusterExternalIP"` // master node public IP
	AvailableZone     string                `json:"availableZone"`     // master node region
	Endpoints         []CCEClusterEndpoints `json:"endpoints"`         // cluster Endpoints

	CreatedClusterEIPID string `json:"createdClusterEIPID"` // cluster EIP
	CreatedVpcID        string `json:"createdVpcID"`        // VPC ID
	CreatedSubnetID     string `json:"createdSubnetID"`     // Subnet ID

	CreatedNatGatewayID  string `json:"createdNatGatewayID"`  // NAT Gateway ID
	CreatedSNatRuleEIPID string `json:"createdSNatRuleEIPID"` // EIP ID for SNAT Rule
	CreatedSNATRuleID    string `json:"createdSNATRuleID"`    // SNAT Rule ID

	ResizeClusterJobID   string `json:"resizeClusterJobID"`   // resize cluster job ID
	UpgradeClusterTaskID string `json:"upgradeClusterTaskID"` // upgrade cluster task ID
}

type CCEHostNetwork struct {
	VpcID         string `json:"vpcID"`
	SubnetID      string `json:"subnetID"`
	SecurityGroup string `json:"securityGroup"`
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
	Name                 string                     `json:"name"`       // 节点池名称
	Type                 string                     `json:"type"`       // 节点池类型：vm, ElasticBMS, pm (default: vm)
	ID                   string                     `json:"nodePoolID"` // 节点池 ID，仅用于查询
	NodeTemplate         CCENodeTemplate            `json:"nodeTemplate"`
	InitialNodeCount     int32                      `json:"initialNodeCount"` // 节点池初始化节点个数。查询时为节点池目标节点数量。
	Autoscaling          CCENodePoolNodeAutoscaling `json:"autoscaling"`
	PodSecurityGroups    []string                   `json:"podSecurityGroups"`
	CustomSecurityGroups []string                   `json:"customSecurityGroups"` // 节点池自定义安全组相关配置，未指定安全组ID，新建节点将添加 Node 节点默认安全组。
}

type CCENodeTemplate struct {
	Flavor          string             `json:"flavor"`          // 节点池规格
	AvailableZone   string             `json:"availableZone"`   // 可用区
	OperatingSystem string             `json:"operatingSystem"` // 节点操作系统
	SSHKey          string             `json:"sshKey"`          // SSH 密钥名称（不支持帐号密码登录）
	RootVolume      CCENodeVolume      `json:"rootVolume"`      // 节点的系统盘
	DataVolumes     []CCENodeVolume    `json:"dataVolumes"`     // 节点的数据盘
	PublicIP        CCENodePublicIP    `json:"publicIP"`        // 节点公网IP
	BillingMode     int32              `json:"billingMode"`     // 节点计费模式
	Runtime         string             `json:"runtime"`         // 容器运行时，docker 或 containerd
	ExtendParam     CCENodeExtendParam `json:"extendParam"`     // 节点扩展参数
}

type CCENodePoolNodeAutoscaling struct {
	Enable                bool  `json:"enable"`                // 是否开启自动扩缩容
	MinNodeCount          int32 `json:"minNodeCount"`          // 若开启自动扩缩容，最小能缩容的节点个数
	MaxNodeCount          int32 `json:"maxNodeCount"`          // 若开启自动扩缩容，最大能扩容的节点个数
	ScaleDownCooldownTime int32 `json:"scaleDownCooldownTime"` // 节点保留时间，单位为分钟，扩容出来的节点在这个时间内不会被缩掉
	Priority              int32 `json:"priority"`              // 节点池权重，更高的权重在扩容时拥有更高的优先级
}

type CCENodeVolume struct {
	Size int32  `json:"size"`
	Type string `json:"type"`
}

type CCEEipBandwidth struct {
	ChargeMode string `json:"chargeMode,omitempty"` // 计费模式: bandwidth，traffic
	Size       int32  `json:"size,omitempty"`       // 带宽大小, 取值范围：默认 1Mbit/s ~ 2000Mbit/s
	ShareType  string `json:"shareType,omitempty"`  // PER为独占带宽，WHOLE是共享带宽
}

type CCEEip struct {
	Iptype    string          `json:"ipType,omitempty"`    // 弹性IP类型 5_telcom（电信），5_union（联通），5_bgp（全动态BGP），5_sbgp（静态BGP）
	Bandwidth CCEEipBandwidth `json:"bandwidth,omitempty"` // 弹性IP的带宽参数
}

type CCEClusterPublicIP struct {
	CreateEIP bool   `json:"createEIP,omitempty"` // 若为 false, 则必须填写 ClusterExternalIP
	Eip       CCEEip `json:"eip,omitempty"`
}

type CCENodePublicIP struct {
	IDs   []string `json:"ids,omitempty"`   // 已有的弹性IP的ID列表。数量不得大于待创建节点数
	Count int32    `json:"count,omitempty"` // 要动态创建的弹性IP个数。
	Eip   CCEEip   `json:"eip,omitempty"`   // 弹性IP参数。
}

type CCENodeExtendParam struct {
	PeriodType  string `json:"periodType,omitempty"`  // month / year, 作为请求参数，billingMode为1（包周期）或2（已废弃：自动付费包周期）时必选。
	PeriodNum   int32  `json:"periodNum,omitempty"`   // 订购周期数
	IsAutoRenew string `json:"isAutoRenew,omitempty"` // 是否自动续订
}

type CCEClusterExtendParam struct {
	ClusterAZ         string `json:"clusterAZ,omitempty"`         // 集群控制节点可用区
	ClusterExternalIP string `json:"clusterExternalIP,omitempty"` // master 弹性公网 IP 地址
	PeriodType        string `json:"periodType,omitempty"`        // month：月, year：年; billingMode为1（包周期）时生效，且为必选。
	PeriodNum         int32  `json:"periodNum,omitempty"`         // 订购周期数
	IsAutoRenew       string `json:"isAutoRenew,omitempty"`       // 是否自动续订
	IsAutoPay         string `json:"isAutoPay,omitempty"`         // 是否自动扣款
}

type CCEClusterEndpoints struct {
	URL  string `json:"url,omitempty"`
	Type string `json:"type,omitempty"`
}

type CCENatGateway struct {
	Enabled       bool   `json:"enabled"`       // 为集群节点启用 NAT
	SNatRuleEIP   CCEEip `json:"snatRuleEIP"`   // 配置 SNAT Rule 时新建 EIP 的参数
	ExistingEIPID string `json:"existingEIPID"` // 配置 SNAT Rule 时使用已有 EIP
}

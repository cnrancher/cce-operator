
# CCE Operator Parameter

````json
{   // 创建集群时填写的参数字段与华为云文档相对应：https://support.huaweicloud.com/api-cce/cce_02_0236.html#section4
    "credentialSecret": "cattle-global-data:cc-test", // secret ID
    "category": "CCE", // 保留选项，目前只支持 CCE，后续可以创建 Turbo
    "regionID": "cn-north-1", // CCE 集群的 Region
    "clusterID": "", // 创建集群时，此参数为空字符串，仅导入集群时需要此字段
    "imported": false, // 非导入集群
    "name": "cce-create-1", // CCE 集群名称，由用户填写
    "labels": {
        "key": "value", // Label
        "key2": "value2"
    },
    "type": "VirtualMachine", // 目前只支持 VirtualMachine：Master节点为x86架构服务器，后续支持 ARM64 鲲鹏
    "flavor": "cce.s1.small", // s1：单控制节点CCE集群。
                              // s2：多控制节点CCE集群 （高可用）。
                              // small (最大 50 节点), medium (200 节点), large (1k 节点), xlarge (2k 节点)
    "version": "v1.23", // v1.21, v1.23, v1.25
    "description": "example description", // 集群描述
    "ipv6Enable": false, // 不启用 IPv6，保留参数，永远为 False
    "hostNetwork": {
        "vpcID": "VPC-ID", // VPCID，若为空字符串，Operator 将新建一个 VPC
        "subnetID": "SUBNET-ID", // SubnetID，若为空字符串，Operator 将新建一个 Subnet
        "securityGroup": "SECURITY-GROUP-ID" // 安全组，若为空字符串，华为云在创建集群时会自动新建一个安全组
    },
    "containerNetwork": {
        "mode": "overlay_l2", // 容器网络类型：overlay_l2, vpc-router, eni
        "cidr": "172.16.123.0/24" // 容器网络网段
        // "cidrs": ["172.16.123.0/24"] // 后续华为云 API 升级可能会启用 cidr 字段改为 "cidrs" 字段
    },
    "eniNetwork": { // 创建CCE Turbo集群时指定，保留字段。
        "subnets": []
    },
    "authentication": { // 集群认证方式相关配置。
        "mode": "rbac", // 集群认证模式。
        "authenticatingProxy": { // 当集群认证模式为 authenticating_proxy 时，此项必须填写。
            "ca": "",
            "cert": "",
            "privateKey": ""
        }
    },
    "kubernetesSvcIPRange": "10.3.4.0/24", // 服务网段参数，kubernetes clusterIP取值范围
    "tags": {
        "cluster-key": "cluster-value" // 集群资源标签
    },
    "kubeProxyMode": "iptables", // 服务转发模式, iptables 或 ipvs (默认 iptables)
    "nodePools": [
        // 集群节点池的参数配置，与华为云文档相对应：https://support.huaweicloud.com/api-cce/cce_02_0242.html#section4
        {
            "name": "nodepool-1", // 节点池名称，由用户输入
            "type": "vm", // 节点池类型：vm, ElasticBMS, pm (default: vm)
            "nodeID": "NODE_ID-aaa-bbb-ccc", // 节点池 ID，仅查询时返回结果
            "nodeTemplate": { // 该节点池的每个节点的配置模板
                "flavor": "t6.large.2",  // 节点池规格
                "availableZone": "cn-north-1a", // 可用区, 可为 random
                "operatingSystem": "EulerOS 2.9",
                "sshKey": "SSH_KEY", // SSH 密钥，处于安全考虑，帐号密码登录的方式不被 Operator 支持
                "rootVolume": { // 节点系统盘
                    "size": 40,
                    "type": "SSD"
                },
                "dataVolumes": [ // 节点数据盘数组
                    {
                        "size": 100,
                        "type": "SSD"
                    }
                ],
                "publicIP": { // 节点公网IP配置
                    "count": 1,
                    "eip": {
                        "ipType": "5_bgp",
                        "bandwidth": {
                            "chargeMode": "traffic",
                            "size": 1,
                            "shareType": "PER"
                        }
                    }
                },
                "count": 1, // 批量创建节点的数量，不能为 0，永远为 1
                "billingMode": 0, // 节点计费模式
                "runtime": "containerd", // 容器运行时
                "extendParam": { // 节点扩展参数
                    "periodType": "month",
                    "periodNum": 1,
                    "isAutoRenew": "false"
                }
            },
            "initialNodeCount": 1, // 节点池初始化节点个数
            "autoscaling": {
                "enable": false, // 是否开启自动扩缩容
                "minNodeCount": 1, // 最小能缩容的节点个数
                "maxNodeCount": 1, // 最大能扩容的节点个数
                "scaleDownCooldownTime": 0, // 节点保留时间，单位为分钟
                "priority": 0 // 节点池权重，数值越大节点池优先级越高
            },
            "podSecurityGroups": [],
            "customSecurityGroups": [
                // 节点池自定义安全组相关配置，未指定安全组ID，新建节点将添加 Node 节点默认安全组。
                "SECURITY_GROUP_ID"
            ]
        }
    ]
}
````

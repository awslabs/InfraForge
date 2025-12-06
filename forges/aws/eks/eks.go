// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package eks

import (
	"fmt"
	"strings"

	"github.com/awslabs/InfraForge/core/config"
	"github.com/awslabs/InfraForge/core/interfaces"
	"github.com/awslabs/InfraForge/core/security"
	"github.com/awslabs/InfraForge/core/utils/types"
	"github.com/awslabs/InfraForge/core/utils/aws"
	"github.com/awslabs/InfraForge/forges/aws/eks/utils"
	"github.com/awslabs/InfraForge/core/dependency"
	"github.com/awslabs/InfraForge/core/partition"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awseks"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"

	"github.com/aws/jsii-runtime-go"
)

type EksInstanceConfig struct {
	config.BaseInstanceConfig
	EksVersion 		 string `json:"eksVersion"`
	KarpenterVersion	 string `json:"karpenterVersion"`
	KarpenterNodePools	 string `json:"karpenterNodePools"`
	KarpenterOsType		 string `json:"karpenterOsType"`
	InstanceTypes		 string `json:"instanceTypes"`
	// 移除 SharedKeyName，统一使用 KeyName
	OsType			 string `json:"osType"`
	WindowsType		 string `json:"windowsType"`
	KeyName                  string `json:"keyName,omitempty"`
	GpuType			 string `json:"gpuType"`
	OsArch			 string `json:"osArch"`
	MinSize			 int	`json:"minSize,omitempty"`
	MaxSize			 int	`json:"maxSize,omitempty"`
	DiskSize		 int    `json:"diskSize,omitempty"`
	NvidiaPluginVersion      string `json:"nvidiaPluginVersion,omitempty"` // 新增字段，用于指定 NVIDIA Device Plugin 版本
	EfaPluginVersion         string `json:"efaPluginVersion,omitempty"`    // 新增字段，用于指定 EFA Device Plugin 版本
	AdminUsers               string `json:"adminUsers,omitempty"` // 新增字段，逗号分隔的管理员用户列表
	ControlPlaneAzIndices    string `json:"controlPlaneAzIndices,omitempty"` // 新增字段，用于指定控制平面可用区索引，格式为"1,2,3"
	PodIdentityAgentVersion  string `json:"podIdentityAgentVersion,omitempty"` // Pod Identity Agent 版本
	
	// 用于支持 Training Operator
	DeployTrainingOperator   *bool  `json:"deployTrainingOperator,omitempty"` // 是否部署 Training Operator
	TrainingOperatorVersion  string `json:"trainingOperatorVersion,omitempty"` // Training Operator 版本
	UseModernTrainingOperator *bool `json:"useModernTrainingOperator,omitempty"` // 是否使用旧版 Training Operator，默认为 true

	// 用于支持 Storage 集成
	DependsOn                string `json:"dependsOn,omitempty"`
	DeployCsiDriver          *bool  `json:"deployCsiDriver,omitempty"`
	CreateStorageClass       *bool  `json:"createStorageClass,omitempty"`
	StorageClassName         string `json:"storageClassName,omitempty"`
	CreateStaticPV           *bool  `json:"createStaticPV,omitempty"`  // 用于指定是否创建静态 PV
	CreateDefaultPVC         *bool  `json:"createDefaultPVC,omitempty"`  // 用于指定是否创建默认 PVC
	DefaultPVCNamespace      string `json:"defaultPVCNamespace,omitempty"`  // 用于指定默认 PVC 的命名空间

	// 用于支持 Metrics Server
	MetricsServerVersion     string `json:"metricsServerVersion,omitempty"`    // Metrics Server 版本，不为空时安装

	// 用于支持 Cert Manager
	CertManagerVersion       string `json:"certManagerVersion,omitempty"`      // Cert Manager 版本，不为空时安装

	// 用于支持 AWS Load Balancer Controller
	AwsLoadBalancerControllerVersion string `json:"awsLoadBalancerControllerVersion,omitempty"` // AWS Load Balancer Controller 版本，不为空时安装

	// 用于支持 Mountpoint S3 CSI Driver
	MountpointS3CsiDriverVersion string `json:"mountpointS3CsiDriverVersion,omitempty"` // Mountpoint S3 CSI Driver 版本，不为空时安装
	S3BucketName                 string `json:"s3BucketName,omitempty"`                 // S3 存储桶名称

	// 用于支持 MLflow
	MlflowVersion                string `json:"mlflowVersion,omitempty"`                // MLflow 版本，不为空时安装

	// 用于支持 HyperPod 专用组件
	EnableHyperPodComponents     *bool  `json:"enableHyperPodComponents,omitempty"`     // 是否启用 HyperPod 专用组件
	NeuronDevicePluginVersion    string `json:"neuronDevicePluginVersion,omitempty"`   // Neuron Device Plugin 版本

	// CPU 节点池配置 - 扁平化
	KarpenterCpuInstanceTypes       string `json:"karpenterCpuInstanceTypes,omitempty"`       // "m5.large,c5.xlarge"
	KarpenterCpuInstanceFamilies    string `json:"karpenterCpuInstanceFamilies,omitempty"`    // "m5,c5,r5"
	KarpenterCpuInstanceCategories  string `json:"karpenterCpuInstanceCategories,omitempty"`  // "c,m,r"
	KarpenterCpuInstanceGenerations string `json:"karpenterCpuInstanceGenerations,omitempty"` // "5,6,7"
	KarpenterCpuCapacityTypes       string `json:"karpenterCpuCapacityTypes,omitempty"`       // "spot,on-demand"
	KarpenterCpuArchitectures       string `json:"karpenterCpuArchitectures,omitempty"`       // "amd64,arm64"
	KarpenterCpuDiskSize            int    `json:"karpenterCpuDiskSize,omitempty"`
	KarpenterCpuDiskType            string `json:"karpenterCpuDiskType,omitempty"`            // "gp3,gp2,io1,io2"
	KarpenterCpuDiskIops            int    `json:"karpenterCpuDiskIops,omitempty"`            // IOPS for gp3/io1/io2
	KarpenterCpuDiskThroughput      int    `json:"karpenterCpuDiskThroughput,omitempty"`      // Throughput for gp3 (MiB/s)
	KarpenterCpuUseInstanceStore    bool   `json:"karpenterCpuUseInstanceStore,omitempty"`    // Use instance store for ephemeral storage
	KarpenterCpuLabels              string `json:"karpenterCpuLabels,omitempty"`              // "key1=value1,key2=value2"
	KarpenterCpuTaints              string `json:"karpenterCpuTaints,omitempty"`              // "key1=value1:NoSchedule"
	
	// GPU 节点池配置 - 扁平化
	KarpenterGpuInstanceTypes       string `json:"karpenterGpuInstanceTypes,omitempty"`
	KarpenterGpuInstanceFamilies    string `json:"karpenterGpuInstanceFamilies,omitempty"`
	KarpenterGpuInstanceCategories  string `json:"karpenterGpuInstanceCategories,omitempty"`
	KarpenterGpuInstanceGenerations string `json:"karpenterGpuInstanceGenerations,omitempty"`
	KarpenterGpuCapacityTypes       string `json:"karpenterGpuCapacityTypes,omitempty"`
	KarpenterGpuArchitectures       string `json:"karpenterGpuArchitectures,omitempty"`
	KarpenterGpuDiskSize            int    `json:"karpenterGpuDiskSize,omitempty"`
	KarpenterGpuDiskType            string `json:"karpenterGpuDiskType,omitempty"`            // "gp3,gp2,io1,io2"
	KarpenterGpuDiskIops            int    `json:"karpenterGpuDiskIops,omitempty"`            // IOPS for gp3/io1/io2
	KarpenterGpuDiskThroughput      int    `json:"karpenterGpuDiskThroughput,omitempty"`      // Throughput for gp3 (MiB/s)
	KarpenterGpuUseInstanceStore    bool   `json:"karpenterGpuUseInstanceStore,omitempty"`    // Use instance store for ephemeral storage
	KarpenterGpuLabels              string `json:"karpenterGpuLabels,omitempty"`
	KarpenterGpuTaints              string `json:"karpenterGpuTaints,omitempty"`
	
	// Neuron 节点池配置 - 扁平化
	KarpenterNeuronInstanceTypes       string `json:"karpenterNeuronInstanceTypes,omitempty"`
	KarpenterNeuronInstanceFamilies    string `json:"karpenterNeuronInstanceFamilies,omitempty"`
	KarpenterNeuronInstanceCategories  string `json:"karpenterNeuronInstanceCategories,omitempty"`
	KarpenterNeuronInstanceGenerations string `json:"karpenterNeuronInstanceGenerations,omitempty"`
	KarpenterNeuronCapacityTypes       string `json:"karpenterNeuronCapacityTypes,omitempty"`
	KarpenterNeuronArchitectures       string `json:"karpenterNeuronArchitectures,omitempty"`
	KarpenterNeuronDiskSize            int    `json:"karpenterNeuronDiskSize,omitempty"`
	KarpenterNeuronDiskType            string `json:"karpenterNeuronDiskType,omitempty"`            // "gp3,gp2,io1,io2"
	KarpenterNeuronDiskIops            int    `json:"karpenterNeuronDiskIops,omitempty"`            // IOPS for gp3/io1/io2
	KarpenterNeuronDiskThroughput      int    `json:"karpenterNeuronDiskThroughput,omitempty"`      // Throughput for gp3 (MiB/s)
	KarpenterNeuronUseInstanceStore    bool   `json:"karpenterNeuronUseInstanceStore,omitempty"`    // Use instance store for ephemeral storage
	KarpenterNeuronLabels              string `json:"karpenterNeuronLabels,omitempty"`
	KarpenterNeuronTaints              string `json:"karpenterNeuronTaints,omitempty"`
}

type EksForge struct {
	eks        awseks.Cluster
	properties map[string]interface{}
}
func (e *EksForge) Create(ctx *interfaces.ForgeContext) interface{} {
	eksInstance, ok := (*ctx.Instance).(*EksInstanceConfig)
	if !ok {
		// 处理类型断言失败的情况
		return nil
	}

	// 获取所有可用区
	availabilityZones := ctx.VPC.AvailabilityZones()

	// 默认子网选择
	subnetSelection := []*awsec2.SubnetSelection{
		&awsec2.SubnetSelection{SubnetType: ctx.SubnetType},
	}

	// 检查是否指定了控制平面可用区
	if eksInstance.ControlPlaneAzIndices != "" {
		// 解析控制平面可用区索引
		// 格式为 "1,2,3"，表示使用第1、2、3个可用区（从1开始计数）
		// 例如，在 us-east-1 区域，可以使用 "1,2,3" 来指定 us-east-1a, us-east-1b, us-east-1c
		// 避免使用不支持 EKS 控制平面的可用区，如 us-east-1e
		azIndicesStr := strings.Split(eksInstance.ControlPlaneAzIndices, ",")
		var selectedAZs []*string

		for _, azIndexStr := range azIndicesStr {
			// 去除可能存在的空格
			azIndexStr = strings.TrimSpace(azIndexStr)
			if azIndexStr != "" {
				// 将字符串转换为整数
				var azIndex int
				_, err := fmt.Sscanf(azIndexStr, "%d", &azIndex)
				if err == nil && azIndex > 0 && azIndex <= len(*availabilityZones) {
					// 计算索引（用户输入从1开始，数组索引从0开始）
					selectedAZs = append(selectedAZs, (*availabilityZones)[azIndex-1])
				} else {
					fmt.Printf("警告: 可用区索引 %s 无效或超出范围 (1-%d)，将被忽略\n", 
					azIndexStr, len(*availabilityZones))
				}
			}
		}

		// 如果成功选择了可用区，则更新子网选择
		if len(selectedAZs) > 0 {
			subnetSelection = []*awsec2.SubnetSelection{
				&awsec2.SubnetSelection{
					SubnetType: ctx.SubnetType,
					AvailabilityZones: &selectedAZs,
				},
			}
			//fmt.Printf("信息: EKS 控制平面将使用指定的 %d 个可用区\n", len(selectedAZs))
		} else {
			fmt.Printf("警告: 未能选择有效的控制平面可用区，将使用系统自动选择\n")
		}
	}

	// 为安全组添加 Karpenter 发现标签
	// Karpenter 可以通过发现标签找到和使用这个安全组
	awscdk.Tags_Of(ctx.SecurityGroups.Default).Add(
		jsii.String("karpenter.sh/discovery"),
		jsii.String(eksInstance.GetID()),
		&awscdk.TagProps{},
	)

	// 为所选子网添加 Karpenter 发现标签
	selectedSubnets := ctx.VPC.SelectSubnets(&awsec2.SubnetSelection{
		SubnetType: ctx.SubnetType,
	}).Subnets

	// 为子网添加 Karpenter 发现标签
	// Karpenter 可以通过发现标签找到和使用这些子网
	// 解引用指针后再遍历
	for _, subnet := range *selectedSubnets {
		awscdk.Tags_Of(subnet).Add(
			jsii.String("karpenter.sh/discovery"),
			jsii.String(eksInstance.GetID()),
			&awscdk.TagProps{},
		)
	}

	// 创建一个 IAM 角色，用作 Masters Role, 需要添加管理策略， 否则 Cloudformation 不会有 Config Command and  Get Token Command 的 output
	// aws-infra-forge.eksConfigCommandDB09280A = aws eks update-kubeconfig --name eks --region ap-southeast-1 --role-arn arn:aws:iam::xxxxxxxxxxxx:role/aws-infra-forge-eksmastersroleXXXXXXXX-XXXXXXXXXXXX
	// aws-infra-forge.eksGetTokenCommand8952195F = aws eks get-token --cluster-name eks --region ap-southeast-1 --role-arn arn:aws:iam::xxxxxxxxxxxx:role/aws-infra-forge-eksmastersroleXXXXXXXX-XXXXXXXXXXXX
	mastersRole := awsiam.NewRole(ctx.Stack, jsii.String(fmt.Sprintf("%s-masters-role", eksInstance.GetID())), &awsiam.RoleProps{
		AssumedBy: awsiam.NewCompositePrincipal(
			awsiam.NewAccountPrincipal(ctx.Stack.Account()),
			awsiam.NewServicePrincipal(jsii.String("lambda.amazonaws.com"), nil), // 添加Lambda服务
		),
		Description: jsii.String("Role for EKS cluster administrators"),
		ManagedPolicies: &[]awsiam.IManagedPolicy{
			awsiam.ManagedPolicy_FromAwsManagedPolicyName(jsii.String("AmazonEKSClusterPolicy")),
		},
	})

	cluster := awseks.NewCluster(ctx.Stack, jsii.String(eksInstance.GetID()), &awseks.ClusterProps{
		ClusterName: jsii.String(eksInstance.GetID()),
		Version: awseks.KubernetesVersion_Of(&eksInstance.EksVersion),
		KubectlLayer: utils.GetKubectlLayer(ctx.Stack, "kubectl", eksInstance.EksVersion),
		DefaultCapacity: jsii.Number(0),
		OutputClusterName: jsii.Bool(true),
		OutputConfigCommand: jsii.Bool(true),
		OutputMastersRoleArn: jsii.Bool(true),
		Vpc: ctx.VPC,
		VpcSubnets: &subnetSelection, 
		MastersRole: mastersRole,
		SecurityGroup: ctx.SecurityGroups.Default,
		AuthenticationMode: awseks.AuthenticationMode_API_AND_CONFIG_MAP,
		Tags: &map[string]*string{
			"karpenter.sh/discovery": jsii.String(eksInstance.GetID()),
		},
	})


	// 简化：直接使用 KeyName，如果为空则使用默认值
	// 使用 CreateOrGetKeyPair 函数获取或创建密钥对
	keyPair := aws.CreateOrGetKeyPair(ctx.Stack, eksInstance.KeyName, eksInstance.OsType)

	// 假设 eksInstance.InstanceTypes 是一个逗号分隔的字符串，如 "m5.large,m5.xlarge,r5.large"
	instanceTypesStr := eksInstance.InstanceTypes

	// 将字符串分割为字符串切片
	instanceTypesList := strings.Split(instanceTypesStr, ",")

	// 创建 awsec2.InstanceType 切片
	var instanceTypes []awsec2.InstanceType
	for _, instanceType := range instanceTypesList {
		// 去除可能存在的空格
		instanceType = strings.TrimSpace(instanceType)
		if instanceType != "" {
			instanceTypes = append(instanceTypes, awsec2.NewInstanceType(jsii.String(instanceType)))
		}
	}

	// 创建启动模板
	// 使用 template 才能指定 default Node Pool 的 Security Group
	// 普通环境 CoreDNS 运行在 default Node Pool
	// 如果 Karpenter nodepool SG 和 default Node Pool SG 不在同一个 SG，需要特殊权限设置，否则会导致 karpenter 管理的节点无法访问 CoreDNS. 
	launchTemplate := awsec2.NewLaunchTemplate(ctx.Stack, jsii.String(fmt.Sprintf("%s.EksDefaultNodePool", eksInstance.GetID())), &awsec2.LaunchTemplateProps{
		LaunchTemplateName: jsii.String(fmt.Sprintf("%s-default-node-template", eksInstance.GetID())),
		SecurityGroup: ctx.SecurityGroups.Default,
		// 在启动模板中使用密钥对
		KeyPair: keyPair, // 直接传递密钥对对象
		BlockDevices: &[]*awsec2.BlockDevice{
			{
				DeviceName: jsii.String("/dev/xvda"),
				// 正确使用 BlockDeviceVolume.ebs 静态方法
				Volume: awsec2.BlockDeviceVolume_Ebs(jsii.Number(eksInstance.DiskSize), &awsec2.EbsDeviceOptions{
					VolumeType: awsec2.EbsDeviceVolumeType_GP3,
					// 可选: 其他 EBS 参数
					// Iops: jsii.Number(3000),
					// Throughput: jsii.Number(125),
				}),
			},
		},
	})

	cluster.AddNodegroupCapacity(jsii.String(fmt.Sprintf("%s-node-group-v%s", eksInstance.GetID(), strings.ReplaceAll(eksInstance.EksVersion, ".", ""))), &awseks.NodegroupOptions{
		InstanceTypes: &instanceTypes,
		MinSize: jsii.Number(eksInstance.MinSize),
		MaxSize: jsii.Number(eksInstance.MaxSize),
		//DiskSize: jsii.Number(eksInstance.DiskSize),
		AmiType: utils.SelectEksAmiType(eksInstance.OsType, eksInstance.OsArch, eksInstance.GpuType, eksInstance.WindowsType),
		// 使用启动模板
		LaunchTemplateSpec: &awseks.LaunchTemplateSpec{
			Id: launchTemplate.LaunchTemplateId(),
			Version: launchTemplate.LatestVersionNumber(),
		},
	})

	// 添加 Admin 角色到 aws-auth ConfigMap，支持所有 Admin/* 联合身份用户, 支持 Isengard 用户
	adminRole := awsiam.Role_FromRoleArn(ctx.Stack, jsii.String("AdminRole"), 
	jsii.String(fmt.Sprintf("arn:%s:iam::%s:role/Admin", partition.DefaultPartition, *ctx.Stack.Account())), nil)
	cluster.AwsAuth().AddRoleMapping(adminRole, &awseks.AwsAuthMapping{
		Groups: &[]*string{
			jsii.String("system:masters"),
		},
		Username: jsii.String("{{SessionName}}"), // 使用 SessionName 作为用户名，映射到联合身份用户的实际名称
	})

	// 解析 AdminUsers 字段并添加用户到 aws-auth ConfigMap, 用于支持 aws console 用户访问
	if eksInstance.AdminUsers != "" {
		// 将逗号分隔的字符串拆分为字符串数组
		userList := strings.Split(eksInstance.AdminUsers, ",")

		for _, username := range userList {
			// 去除可能存在的空格
			username = strings.TrimSpace(username)
			if username != "" {
				// 为每个用户创建一个唯一的逻辑 ID
				userLogicalId := fmt.Sprintf("%s-%s-User", eksInstance.GetID(), username)
				user := awsiam.User_FromUserName(ctx.Stack, jsii.String(userLogicalId), jsii.String(username))
				cluster.AwsAuth().AddUserMapping(user, &awseks.AwsAuthMapping{
					Groups: &[]*string{
						jsii.String("system:masters"),
					},
					Username: jsii.String(username),
				})
			}
		}
	}

	// 给 aws-auth ConfigMap 添加 mastersRole 依赖，确保删除顺序正确
	awsAuthConfigMap := cluster.AwsAuth()
	if awsAuthConfigMap != nil {
		awsAuthConfigMap.Node().AddDependency(mastersRole)
	}

	// 1. 首先部署 Pod Identity Agent（如果指定了版本）- 底层身份认证组件
	if eksInstance.PodIdentityAgentVersion != "" {
		podIdentityAgent := deployPodIdentityAgent(ctx.Stack, cluster, eksInstance.PodIdentityAgentVersion)
		if podIdentityAgent != nil {
			// 依赖 aws-auth ConfigMap 而不是直接依赖 mastersRole
			podIdentityAgent.Node().AddDependency(awsAuthConfigMap)
		}
	}

	// 创建 Karpenter IAM 资源
	karpenterIam := createKarpenterIamResources(ctx.Stack, "KarpenterIam", cluster)

	// 创建统一的 eks-admin ServiceAccount 和 ClusterRoleBinding（供各组件使用）
	eksAdminSA, eksAdminCRB := createEksAdminResources(ctx.Stack, "EksAdmin", cluster, "eks-admin")

	// 检查是否需要 Multi-NIC 支持
	nodePoolProps := &KarpenterNodePoolProps{
		GpuInstanceTypes:    eksInstance.KarpenterGpuInstanceTypes,
		CpuInstanceTypes:    eksInstance.KarpenterCpuInstanceTypes,
		NeuronInstanceTypes: eksInstance.KarpenterNeuronInstanceTypes,
	}
	
	needsMultiNic := needsMultiNicSupport(nodePoolProps)
	
	// 升级 VPC CNI 并配置 Multi-NIC（如果需要）
	var vpcCniManifest awseks.KubernetesManifest
	if needsMultiNic {
		vpcCniManifest = upgradeVpcCniAndConfigureMultiNic(ctx.Stack, "VpcCniUpgrade", &VpcCniUpgradeProps{
			Cluster:        cluster,
			ClusterName:    *cluster.ClusterName(),
			VpcCniVersion:  "v1.20.4",
			EnableMultiNic: true,
		}, eksAdminSA, eksAdminCRB)
	}

	// EFA Launch Templates 已移除（Karpenter v1 不支持自定义 Launch Template）

	// 部署 Karpenter Helm Chart
	karpenterChart := deployKarpenterHelm(ctx.Stack, "KarpenterHelm", &KarpenterHelmProps{
		ClusterName:      *cluster.ClusterName(),
		Cluster:          cluster,
		ControllerRoleArn: *karpenterIam.ControllerRole.RoleArn(),
		KarpenterVersion: eksInstance.KarpenterVersion,
	})
	
	if karpenterChart != nil {
		karpenterChart.Node().AddDependency(awsAuthConfigMap)
	}

	// 创建 Karpenter NodePool 和 EC2NodeClass
	nodePools := createKarpenterNodePool(ctx.Stack, "KarpenterNodePool", &KarpenterNodePoolProps{
		ClusterName: *cluster.ClusterName(),
		Cluster:     cluster,
		IamResources: karpenterIam,
		KubernetesVersion: eksInstance.EksVersion,
		KarpenterOsType: eksInstance.KarpenterOsType,
		NodePoolTypes: eksInstance.KarpenterNodePools, // 使用这个参数决定启用哪些节点池
		
		// CPU 配置
		CpuInstanceTypes:       eksInstance.KarpenterCpuInstanceTypes,
		CpuInstanceFamilies:    eksInstance.KarpenterCpuInstanceFamilies,
		CpuInstanceCategories:  eksInstance.KarpenterCpuInstanceCategories,
		CpuInstanceGenerations: eksInstance.KarpenterCpuInstanceGenerations,
		CpuCapacityTypes:       eksInstance.KarpenterCpuCapacityTypes,
		CpuArchitectures:       eksInstance.KarpenterCpuArchitectures,
		CpuDiskSize:            eksInstance.KarpenterCpuDiskSize,
		CpuDiskType:            eksInstance.KarpenterCpuDiskType,
		CpuDiskIops:            eksInstance.KarpenterCpuDiskIops,
		CpuDiskThroughput:      eksInstance.KarpenterCpuDiskThroughput,
		CpuUseInstanceStore:    eksInstance.KarpenterCpuUseInstanceStore,
		CpuLabels:              eksInstance.KarpenterCpuLabels,
		CpuTaints:              eksInstance.KarpenterCpuTaints,
		
		// GPU 配置
		GpuInstanceTypes:       eksInstance.KarpenterGpuInstanceTypes,
		GpuInstanceFamilies:    eksInstance.KarpenterGpuInstanceFamilies,
		GpuInstanceCategories:  eksInstance.KarpenterGpuInstanceCategories,
		GpuInstanceGenerations: eksInstance.KarpenterGpuInstanceGenerations,
		GpuCapacityTypes:       eksInstance.KarpenterGpuCapacityTypes,
		GpuArchitectures:       eksInstance.KarpenterGpuArchitectures,
		GpuDiskSize:            eksInstance.KarpenterGpuDiskSize,
		GpuDiskType:            eksInstance.KarpenterGpuDiskType,
		GpuDiskIops:            eksInstance.KarpenterGpuDiskIops,
		GpuDiskThroughput:      eksInstance.KarpenterGpuDiskThroughput,
		GpuUseInstanceStore:    eksInstance.KarpenterGpuUseInstanceStore,
		GpuLabels:              eksInstance.KarpenterGpuLabels,
		GpuTaints:              eksInstance.KarpenterGpuTaints,
		
		// Neuron 配置
		NeuronInstanceTypes:       eksInstance.KarpenterNeuronInstanceTypes,
		NeuronInstanceFamilies:    eksInstance.KarpenterNeuronInstanceFamilies,
		NeuronInstanceCategories:  eksInstance.KarpenterNeuronInstanceCategories,
		NeuronInstanceGenerations: eksInstance.KarpenterNeuronInstanceGenerations,
		NeuronCapacityTypes:       eksInstance.KarpenterNeuronCapacityTypes,
		NeuronArchitectures:       eksInstance.KarpenterNeuronArchitectures,
		NeuronDiskSize:            eksInstance.KarpenterNeuronDiskSize,
		NeuronDiskType:            eksInstance.KarpenterNeuronDiskType,
		NeuronDiskIops:            eksInstance.KarpenterNeuronDiskIops,
		NeuronDiskThroughput:      eksInstance.KarpenterNeuronDiskThroughput,
		NeuronUseInstanceStore:    eksInstance.KarpenterNeuronUseInstanceStore,
		NeuronLabels:              eksInstance.KarpenterNeuronLabels,
		NeuronTaints:              eksInstance.KarpenterNeuronTaints,
	})

	// 确保 Karpenter 在集群和角色创建后部署
	if karpenterChart != nil {
		karpenterChart.Node().AddDependency(cluster)
		karpenterChart.Node().AddDependency(karpenterIam.ControllerRole)
	}

	// 确保所有 NodePool 在 Karpenter 和 VPC CNI 升级后创建
	for _, nodePool := range nodePools {
		if karpenterChart != nil {
			nodePool.Node().AddDependency(karpenterChart)
		}
		if vpcCniManifest != nil {
			nodePool.Node().AddDependency(vpcCniManifest)
		}
	}

	// 部署 EFA Device Plugin（只依赖集群，与Karpenter并行）
	efaChart := deployEfaDevicePlugin(ctx.Stack, cluster, eksInstance.EfaPluginVersion)
	if efaChart != nil {
		efaChart.Node().AddDependency(cluster)
		efaChart.Node().AddDependency(awsAuthConfigMap)
	}

	// 如果启用了 GPU 节点池，部署 NVIDIA Device Plugin
	if strings.Contains(eksInstance.KarpenterNodePools, "gpu") || strings.Contains(eksInstance.KarpenterNodePools, "nvidia") {
		nvidiaPlugin := deployNvidiaDevicePlugin(ctx.Stack, cluster, eksInstance.NvidiaPluginVersion)
		if nvidiaPlugin != nil {
			nvidiaPlugin.Node().AddDependency(cluster)
			nvidiaPlugin.Node().AddDependency(awsAuthConfigMap)
		}
	}

	// 部署额外的 Kubernetes 组件（避免webhook冲突的简化依赖关系）
	// 
	// 依赖关系设计：
	// ALB Controller → Cert Manager → Metrics Server
	// Karpenter (并行部署，不参与上述依赖链)
	// 
	// 原因：避免复杂的循环依赖，让 Karpenter 与其他组件并行部署，
	// Cert Manager 直接依赖 ALB Controller，通过 ALB Controller 的配置优化减少 webhook 冲突。

	// 2. 部署 AWS Load Balancer Controller（如果指定了版本）
	// ALB Controller 与 Karpenter 并行部署，但 Cert Manager 会依赖 ALB Controller
	var awsLbControllerChart awseks.HelmChart
	if eksInstance.AwsLoadBalancerControllerVersion != "" {
		awsLbControllerChart = deployAwsLoadBalancerController(ctx.Stack, cluster, eksInstance.AwsLoadBalancerControllerVersion)
		if awsLbControllerChart != nil {
			awsLbControllerChart.Node().AddDependency(awsAuthConfigMap)
		}
	}

	// 3. 部署 Cert Manager（如果指定了版本）
	// 直接依赖 ALB Controller，通过 ALB Controller 的配置优化来减少 webhook 冲突
	var certManagerChart awseks.HelmChart
	if eksInstance.CertManagerVersion != "" {
		// 先创建cert-manager namespace并添加mastersRole依赖
		certManagerNamespace := cluster.AddManifest(jsii.String("cert-manager-namespace-with-deps"), &map[string]interface{}{
			"apiVersion": "v1",
			"kind": "Namespace",
			"metadata": map[string]interface{}{
				"name": "cert-manager",
			},
		})
		if certManagerNamespace != nil {
			certManagerNamespace.Node().AddDependency(awsAuthConfigMap)
		}
		
		certManagerChart = deployCertManager(ctx.Stack, cluster, eksInstance.CertManagerVersion)
		if certManagerChart != nil {
			certManagerChart.Node().AddDependency(awsAuthConfigMap)
			// 确保chart在我们创建的namespace之后部署
			if certManagerNamespace != nil {
				certManagerChart.Node().AddDependency(certManagerNamespace)
			}
		}
		// 如果安装了 AWS Load Balancer Controller，确保 cert-manager 在其之后部署
		if eksInstance.AwsLoadBalancerControllerVersion != "" && awsLbControllerChart != nil && certManagerChart != nil {
			certManagerChart.Node().AddDependency(awsLbControllerChart)
		}
	}

	// 部署 Mountpoint S3 CSI Driver（如果指定了版本）
	if eksInstance.MountpointS3CsiDriverVersion != "" {
		s3CsiChart := deployMountpointS3CsiDriverWithStorage(ctx.Stack, cluster, eksInstance)
		if s3CsiChart != nil {
			s3CsiChart.Node().AddDependency(awsAuthConfigMap)
		}
	}

	// 4. 部署 Metrics Server（如果指定了版本）
	// 依赖 Cert Manager，保持简单的线性依赖链
	var metricsServerChart awseks.HelmChart
	if eksInstance.MetricsServerVersion != "" {
		metricsServerChart = deployMetricsServer(ctx.Stack, cluster, eksInstance.MetricsServerVersion, eksInstance.CertManagerVersion != "")
		if metricsServerChart != nil {
			metricsServerChart.Node().AddDependency(awsAuthConfigMap)
		}
		// 如果安装了 cert-manager，确保 metrics-server 在 cert-manager 之后部署
		// 保持简单的依赖链：ALB Controller → Cert Manager → Metrics Server
		if eksInstance.CertManagerVersion != "" && certManagerChart != nil && metricsServerChart != nil {
			metricsServerChart.Node().AddDependency(certManagerChart)
		} else if eksInstance.AwsLoadBalancerControllerVersion != "" && awsLbControllerChart != nil && metricsServerChart != nil {
			// 如果没有 Cert Manager 但有 ALB Controller，直接依赖 ALB Controller
			metricsServerChart.Node().AddDependency(awsLbControllerChart)
		}
	}

	// 部署 Training Operator
	if types.GetBoolValue(eksInstance.DeployTrainingOperator, false) {
		// 使用默认版本，如果未指定
		trainingOperatorVersion := eksInstance.TrainingOperatorVersion
		if trainingOperatorVersion == "" {
			trainingOperatorVersion = "1.9.3" // 默认版本
		}
		
		// 根据配置选择部署方式
		if types.GetBoolValue(eksInstance.UseModernTrainingOperator, false) {
			// 部署新版 Training Operator，传入版本参数
			modernTrainingOp := deployModernTrainingOperator(ctx.Stack, cluster, trainingOperatorVersion, eksAdminSA, eksAdminCRB)
			if modernTrainingOp != nil {
				modernTrainingOp.Node().AddDependency(awsAuthConfigMap)
			}
		} else {
			// 部署 Legacy Training Operator
			legacyTrainingOp := deployLegacyTrainingOperator(ctx.Stack, cluster, trainingOperatorVersion, eksAdminSA, eksAdminCRB)
			if legacyTrainingOp != nil {
				legacyTrainingOp.Node().AddDependency(awsAuthConfigMap)
			}
		}
	}

	// 处理存储依赖关系（Lustre、EFS 等）
	if eksInstance.DependsOn != "" {
		// 获取依赖信息
		magicToken, err := dependency.GetDependencyInfo(eksInstance.DependsOn)
		if err != nil {
			fmt.Printf("Error getting dependency info: %v\n", err)
		} else {
			// 如果需要部署 CSI 驱动
			if types.GetBoolValue(eksInstance.DeployCsiDriver, false) {
				csiCharts := deployStorageCsiDriver(ctx.Stack, cluster, magicToken, eksInstance)
				// 为所有 CSI Charts 添加 mastersRole 依赖
				for _, chart := range csiCharts {
					if chart != nil {
						chart.Node().AddDependency(awsAuthConfigMap)
					}
				}
			}
		}
	}

	// 部署 MLflow（如果指定了版本）
	if eksInstance.MlflowVersion != "" {
		mlflowChart := deployMlflow(ctx.Stack, cluster, eksInstance.MlflowVersion)
		if mlflowChart != nil {
			mlflowChart.Node().AddDependency(awsAuthConfigMap)
		}
	}

	// 如果启用 HyperPod 组件
	var hyperPodJob awseks.KubernetesManifest
	if types.GetBoolValue(eksInstance.EnableHyperPodComponents, false) {
		hyperPodJob = deployHyperPodComponents(ctx.Stack, cluster, eksInstance, mastersRole)
		
		// 确保 HyperPod 在系统组件之后部署
		if eksInstance.MetricsServerVersion != "" && metricsServerChart != nil && hyperPodJob != nil {
			hyperPodJob.Node().AddDependency(metricsServerChart)
		}
		if eksInstance.CertManagerVersion != "" && certManagerChart != nil && hyperPodJob != nil {
			hyperPodJob.Node().AddDependency(certManagerChart)
		}
		// 添加 awsAuthConfigMap 依赖确保权限可用
		if hyperPodJob != nil {
			hyperPodJob.Node().AddDependency(awsAuthConfigMap)
		}
		
		// 部署 AWS Neuron Device Plugin (HyperPod 必需)
		neuronPlugin := deployNeuronDevicePlugin(ctx.Stack, cluster, eksInstance.NeuronDevicePluginVersion)
		if neuronPlugin != nil {
			neuronPlugin.Node().AddDependency(awsAuthConfigMap)
		}
	}

	e.eks = cluster
	
	// 保存 EKS 属性
	if e.properties == nil {
		e.properties = make(map[string]interface{})
	}
	e.properties["clusterName"] = cluster.ClusterName()
	e.properties["clusterArn"] = cluster.ClusterArn()
	e.properties["clusterEndpoint"] = cluster.ClusterEndpoint()
	e.properties["eksVersion"] = eksInstance.EksVersion
	
	// 保存 HyperPod 组件用于依赖
	if hyperPodJob != nil {
		e.properties["hyperPodJob"] = hyperPodJob
	}
	
	// 保存 Karpenter Helm chart 用于依赖
	if karpenterChart != nil {
		e.properties["karpenterChart"] = karpenterChart
	}
	
	return e
}

// 其余方法保持不变...
func (e *EksForge) CreateOutputs(ctx *interfaces.ForgeContext) {
	// 添加集群端点到输出
	awscdk.NewCfnOutput(ctx.Stack, jsii.String("ClusterEndpoint"), &awscdk.CfnOutputProps{
		Value: e.eks.ClusterEndpoint(),
		Description: jsii.String("EKS cluster API server endpoint"),
	})
	// 添加集群 ARN 到输出
	awscdk.NewCfnOutput(ctx.Stack, jsii.String("ClusterArn"), &awscdk.CfnOutputProps{
		Value: e.eks.ClusterArn(),
		Description: jsii.String("EKS cluster ARN"),
	})
	// 如果启用了 OpenID Connect 提供商，也可以输出其 URL
	if e.eks.ClusterOpenIdConnectIssuerUrl() != nil {
		awscdk.NewCfnOutput(ctx.Stack, jsii.String("OidcProviderUrl"), &awscdk.CfnOutputProps{
			Value: e.eks.ClusterOpenIdConnectIssuerUrl(),
			Description: jsii.String("OpenID Connect Provider URL"),
		})
	}
}

func (e *EksForge) ConfigureRules(ctx *interfaces.ForgeContext) {
	// 为 EC2 配置特定的入站规则
	security.AddTcpIngressRule(ctx.SecurityGroups.Default, ctx.SecurityGroups.Public, 22, "Allow EC2 SSH access from public subnet")
	security.AddTcpIngressRule(ctx.SecurityGroups.Default, ctx.SecurityGroups.Private, 22, "Allow EC2 SSH access from private subnet")

	security.AddTcpIngressRuleFromAnyIp(ctx.SecurityGroups.Public, 22, "Allow EC2 SSH access from 0.0.0.0/0")
	if ctx.DualStack {
		security.AddTcpIngressRuleFromAnyIpv6(ctx.SecurityGroups.Public, 22, "Allow EC2 SSH access from 0.0.0.0/0")
	}

	// 配置 EFA 安全组规则
	// EFA 需要允许所有流量在安全组内部通信，以支持 OS-bypass 功能
	security.ConfigureEFASecurityRules(ctx.SecurityGroups.Default, "eks")
}
func (e *EksForge) MergeConfigs(defaults config.InstanceConfig, instance config.InstanceConfig) config.InstanceConfig {
	// 从默认配置中复制基本字段
	merged := defaults.(*EksInstanceConfig)

	// 从实例配置中覆盖基本字段
	eksInstance := instance.(*EksInstanceConfig)
	if instance != nil {
		if eksInstance.GetID() != "" {
			merged.ID = eksInstance.GetID()
		}
		if eksInstance.Type != "" {
			merged.Type = eksInstance.GetType()
		}
		if eksInstance.Subnet != "" {
			merged.Subnet = eksInstance.GetSubnet()
		}
		if eksInstance.SecurityGroup != "" {
			merged.SecurityGroup = eksInstance.GetSecurityGroup()
		}
	}

	// Merge EKS 自己的字段
	// 移除 SharedKeyName 的合并逻辑
	if eksInstance.EksVersion != "" {
		merged.EksVersion = eksInstance.EksVersion
	}
	if eksInstance.KarpenterVersion != "" {
		merged.KarpenterVersion = eksInstance.KarpenterVersion
	}
	if eksInstance.KarpenterNodePools != "" {
		merged.KarpenterNodePools = eksInstance.KarpenterNodePools
	}
	if eksInstance.KarpenterOsType != "" {
		merged.KarpenterOsType = eksInstance.KarpenterOsType
	}
	if eksInstance.InstanceTypes != "" {
		merged.InstanceTypes = eksInstance.InstanceTypes
	}
	if eksInstance.GpuType != "" {
		merged.GpuType = eksInstance.GpuType
	}
	if eksInstance.OsType != "" {
		merged.OsType = eksInstance.OsType
	}
	if eksInstance.OsArch != "" {
		merged.OsArch = eksInstance.OsArch
	}
	if eksInstance.WindowsType != "" {
		merged.WindowsType = eksInstance.WindowsType
	}
	if eksInstance.MinSize != 0 {
		merged.MinSize = eksInstance.MinSize
	}
	if eksInstance.MaxSize != 0 {
		merged.MaxSize = eksInstance.MaxSize
	}
	if eksInstance.DiskSize != 0 {
		merged.DiskSize = eksInstance.DiskSize
	}
	if eksInstance.NvidiaPluginVersion != "" {
		merged.NvidiaPluginVersion = eksInstance.NvidiaPluginVersion
	}
	if eksInstance.EfaPluginVersion != "" {
		merged.EfaPluginVersion = eksInstance.EfaPluginVersion
	}
	if eksInstance.AdminUsers != "" {
		merged.AdminUsers = eksInstance.AdminUsers
	}
	if eksInstance.ControlPlaneAzIndices != "" {
		merged.ControlPlaneAzIndices = eksInstance.ControlPlaneAzIndices
	}

	// 合并 Storage 相关字段
	if eksInstance.DependsOn != "" {
		merged.DependsOn = eksInstance.DependsOn
	}
	if eksInstance.DeployCsiDriver != nil {
		merged.DeployCsiDriver = eksInstance.DeployCsiDriver
	}
	if eksInstance.CreateStorageClass != nil {
		merged.CreateStorageClass = eksInstance.CreateStorageClass
	}
	if eksInstance.StorageClassName != "" {
		merged.StorageClassName = eksInstance.StorageClassName
	}
	if eksInstance.CreateStaticPV != nil {
		merged.CreateStaticPV = eksInstance.CreateStaticPV
	}
	if eksInstance.CreateDefaultPVC != nil {
		merged.CreateDefaultPVC = eksInstance.CreateDefaultPVC
	}
	if eksInstance.DefaultPVCNamespace != "" {
		merged.DefaultPVCNamespace = eksInstance.DefaultPVCNamespace
	}

	// 合并 Metrics Server 相关字段
	if eksInstance.MetricsServerVersion != "" {
		merged.MetricsServerVersion = eksInstance.MetricsServerVersion
	}

	// 合并 Cert Manager 相关字段
	if eksInstance.CertManagerVersion != "" {
		merged.CertManagerVersion = eksInstance.CertManagerVersion
	}

	// 合并 AWS Load Balancer Controller 相关字段
	if eksInstance.AwsLoadBalancerControllerVersion != "" {
		merged.AwsLoadBalancerControllerVersion = eksInstance.AwsLoadBalancerControllerVersion
	}

	// 合并 Mountpoint S3 CSI Driver 相关字段
	if eksInstance.MountpointS3CsiDriverVersion != "" {
		merged.MountpointS3CsiDriverVersion = eksInstance.MountpointS3CsiDriverVersion
	}
	if eksInstance.S3BucketName != "" {
		merged.S3BucketName = eksInstance.S3BucketName
	}

	
	// 合并 Training Operator 相关字段
	if eksInstance.DeployTrainingOperator != nil {
		merged.DeployTrainingOperator = eksInstance.DeployTrainingOperator
	}
	if eksInstance.TrainingOperatorVersion != "" {
		merged.TrainingOperatorVersion = eksInstance.TrainingOperatorVersion
	}
	if eksInstance.UseModernTrainingOperator != nil {
		merged.UseModernTrainingOperator = eksInstance.UseModernTrainingOperator
	}

	// 合并 CPU 节点池配置
	if eksInstance.KarpenterCpuInstanceTypes != "" {
		merged.KarpenterCpuInstanceTypes = eksInstance.KarpenterCpuInstanceTypes
	}
	if eksInstance.KarpenterCpuInstanceFamilies != "" {
		merged.KarpenterCpuInstanceFamilies = eksInstance.KarpenterCpuInstanceFamilies
	}
	if eksInstance.KarpenterCpuInstanceCategories != "" {
		merged.KarpenterCpuInstanceCategories = eksInstance.KarpenterCpuInstanceCategories
	}
	if eksInstance.KarpenterCpuInstanceGenerations != "" {
		merged.KarpenterCpuInstanceGenerations = eksInstance.KarpenterCpuInstanceGenerations
	}
	if eksInstance.KarpenterCpuCapacityTypes != "" {
		merged.KarpenterCpuCapacityTypes = eksInstance.KarpenterCpuCapacityTypes
	}
	if eksInstance.KarpenterCpuArchitectures != "" {
		merged.KarpenterCpuArchitectures = eksInstance.KarpenterCpuArchitectures
	}
	if eksInstance.KarpenterCpuDiskSize != 0 {
		merged.KarpenterCpuDiskSize = eksInstance.KarpenterCpuDiskSize
	}
	if eksInstance.KarpenterCpuDiskType != "" {
		merged.KarpenterCpuDiskType = eksInstance.KarpenterCpuDiskType
	}
	if eksInstance.KarpenterCpuDiskIops != 0 {
		merged.KarpenterCpuDiskIops = eksInstance.KarpenterCpuDiskIops
	}
	if eksInstance.KarpenterCpuDiskThroughput != 0 {
		merged.KarpenterCpuDiskThroughput = eksInstance.KarpenterCpuDiskThroughput
	}
	if eksInstance.KarpenterCpuUseInstanceStore {
		merged.KarpenterCpuUseInstanceStore = eksInstance.KarpenterCpuUseInstanceStore
	}
	if eksInstance.KarpenterCpuLabels != "" {
		merged.KarpenterCpuLabels = eksInstance.KarpenterCpuLabels
	}
	if eksInstance.KarpenterCpuTaints != "" {
		merged.KarpenterCpuTaints = eksInstance.KarpenterCpuTaints
	}

	// 合并 GPU 节点池配置
	if eksInstance.KarpenterGpuInstanceTypes != "" {
		merged.KarpenterGpuInstanceTypes = eksInstance.KarpenterGpuInstanceTypes
	}
	if eksInstance.KarpenterGpuInstanceFamilies != "" {
		merged.KarpenterGpuInstanceFamilies = eksInstance.KarpenterGpuInstanceFamilies
	}
	if eksInstance.KarpenterGpuInstanceCategories != "" {
		merged.KarpenterGpuInstanceCategories = eksInstance.KarpenterGpuInstanceCategories
	}
	if eksInstance.KarpenterGpuInstanceGenerations != "" {
		merged.KarpenterGpuInstanceGenerations = eksInstance.KarpenterGpuInstanceGenerations
	}
	if eksInstance.KarpenterGpuCapacityTypes != "" {
		merged.KarpenterGpuCapacityTypes = eksInstance.KarpenterGpuCapacityTypes
	}
	if eksInstance.KarpenterGpuArchitectures != "" {
		merged.KarpenterGpuArchitectures = eksInstance.KarpenterGpuArchitectures
	}
	if eksInstance.KarpenterGpuDiskSize != 0 {
		merged.KarpenterGpuDiskSize = eksInstance.KarpenterGpuDiskSize
	}
	if eksInstance.KarpenterGpuDiskType != "" {
		merged.KarpenterGpuDiskType = eksInstance.KarpenterGpuDiskType
	}
	if eksInstance.KarpenterGpuDiskIops != 0 {
		merged.KarpenterGpuDiskIops = eksInstance.KarpenterGpuDiskIops
	}
	if eksInstance.KarpenterGpuDiskThroughput != 0 {
		merged.KarpenterGpuDiskThroughput = eksInstance.KarpenterGpuDiskThroughput
	}
	if eksInstance.KarpenterGpuUseInstanceStore {
		merged.KarpenterGpuUseInstanceStore = eksInstance.KarpenterGpuUseInstanceStore
	}
	if eksInstance.KarpenterGpuLabels != "" {
		merged.KarpenterGpuLabels = eksInstance.KarpenterGpuLabels
	}
	if eksInstance.KarpenterGpuTaints != "" {
		merged.KarpenterGpuTaints = eksInstance.KarpenterGpuTaints
	}

	// 合并 Neuron 节点池配置
	if eksInstance.KarpenterNeuronInstanceTypes != "" {
		merged.KarpenterNeuronInstanceTypes = eksInstance.KarpenterNeuronInstanceTypes
	}
	if eksInstance.KarpenterNeuronInstanceFamilies != "" {
		merged.KarpenterNeuronInstanceFamilies = eksInstance.KarpenterNeuronInstanceFamilies
	}
	if eksInstance.KarpenterNeuronInstanceCategories != "" {
		merged.KarpenterNeuronInstanceCategories = eksInstance.KarpenterNeuronInstanceCategories
	}
	if eksInstance.KarpenterNeuronInstanceGenerations != "" {
		merged.KarpenterNeuronInstanceGenerations = eksInstance.KarpenterNeuronInstanceGenerations
	}
	if eksInstance.KarpenterNeuronCapacityTypes != "" {
		merged.KarpenterNeuronCapacityTypes = eksInstance.KarpenterNeuronCapacityTypes
	}
	if eksInstance.KarpenterNeuronArchitectures != "" {
		merged.KarpenterNeuronArchitectures = eksInstance.KarpenterNeuronArchitectures
	}
	if eksInstance.KarpenterNeuronDiskSize != 0 {
		merged.KarpenterNeuronDiskSize = eksInstance.KarpenterNeuronDiskSize
	}
	if eksInstance.KarpenterNeuronDiskType != "" {
		merged.KarpenterNeuronDiskType = eksInstance.KarpenterNeuronDiskType
	}
	if eksInstance.KarpenterNeuronDiskIops != 0 {
		merged.KarpenterNeuronDiskIops = eksInstance.KarpenterNeuronDiskIops
	}
	if eksInstance.KarpenterNeuronDiskThroughput != 0 {
		merged.KarpenterNeuronDiskThroughput = eksInstance.KarpenterNeuronDiskThroughput
	}
	if eksInstance.KarpenterNeuronUseInstanceStore {
		merged.KarpenterNeuronUseInstanceStore = eksInstance.KarpenterNeuronUseInstanceStore
	}
	if eksInstance.KarpenterNeuronLabels != "" {
		merged.KarpenterNeuronLabels = eksInstance.KarpenterNeuronLabels
	}
	if eksInstance.KarpenterNeuronTaints != "" {
		merged.KarpenterNeuronTaints = eksInstance.KarpenterNeuronTaints
	}

	// 合并 MLflow 和 HyperPod 相关字段
	if eksInstance.MlflowVersion != "" {
		merged.MlflowVersion = eksInstance.MlflowVersion
	}
	if eksInstance.EnableHyperPodComponents != nil {
		merged.EnableHyperPodComponents = eksInstance.EnableHyperPodComponents
	}
	if eksInstance.NeuronDevicePluginVersion != "" {
		merged.NeuronDevicePluginVersion = eksInstance.NeuronDevicePluginVersion
	}

	return merged
}

func (e *EksForge) GetProperties() map[string]interface{} {
	return e.properties
}

// GetCluster 返回 EKS 集群对象
func (e *EksForge) GetCluster() awseks.Cluster {
	return e.eks
}

// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package eks

import (
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-cdk-go/awscdk/v2/awseks"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
	"gopkg.in/yaml.v3"
)

// KarpenterNodePoolProps - 扁平化配置
type KarpenterNodePoolProps struct {
	Cluster           awseks.Cluster
	ClusterName       string
	IamResources      *KarpenterIamResources
	KubernetesVersion string
	KarpenterOsType   string
	NodePoolTypes     string // "cpu,gpu,neuron" - 决定启用哪些节点池
	
	// CPU 配置
	CpuInstanceTypes       string
	CpuInstanceFamilies    string
	CpuInstanceCategories  string
	CpuInstanceGenerations string
	CpuCapacityTypes       string
	CpuArchitectures       string
	CpuDiskSize            int
	CpuDiskType            string
	CpuDiskIops            int
	CpuDiskThroughput      int
	CpuLabels              string
	CpuTaints              string
	
	// GPU 配置
	GpuInstanceTypes       string
	GpuInstanceFamilies    string
	GpuInstanceCategories  string
	GpuInstanceGenerations string
	GpuCapacityTypes       string
	GpuArchitectures       string
	GpuDiskSize            int
	GpuDiskType            string
	GpuDiskIops            int
	GpuDiskThroughput      int
	GpuLabels              string
	GpuTaints              string
	
	// Neuron 配置
	NeuronInstanceTypes       string
	NeuronInstanceFamilies    string
	NeuronInstanceCategories  string
	NeuronInstanceGenerations string
	NeuronCapacityTypes       string
	NeuronArchitectures       string
	NeuronDiskSize            int
	NeuronDiskType            string
	NeuronDiskIops            int
	NeuronDiskThroughput      int
	NeuronLabels              string
	NeuronTaints              string
}

// 节点池配置结构
type NodePoolConfig struct {
	InstanceTypes       string
	InstanceFamilies    string
	InstanceCategories  string
	InstanceGenerations string
	CapacityTypes       string
	Architectures       string
	DiskSize            int
	DiskType            string
	DiskIops            int
	DiskThroughput      int
	Labels              string
	Taints              string
}

// 定义节点池类型常量
const (
	NodePoolTypeCpu    = "cpu"
	NodePoolTypeGpu    = "gpu"
	NodePoolTypeNeuron = "neuron"
)

// 定义操作系统类型常量
const (
	OsTypeLinux       = "linux"
	OsTypeBottlerocket = "bottlerocket"
)

// 创建 Karpenter 节点池
func createKarpenterNodePool(scope constructs.Construct, id string, props *KarpenterNodePoolProps) []awseks.KubernetesManifest {
	var kubeManifests []awseks.KubernetesManifest
	
	// 使用提供的 Kubernetes 版本，如果未提供则默认为 "1.33"
	kubeVersion := props.KubernetesVersion
	if kubeVersion == "" {
		kubeVersion = "1.33"
	}
	
	// 解析 NodePoolTypes 参数，决定启用哪些节点池
	enabledNodePools := parseCommaSeparatedString(props.NodePoolTypes)
	
	for _, nodePoolType := range enabledNodePools {
		nodePoolType = strings.ToLower(strings.TrimSpace(nodePoolType))
		
		switch nodePoolType {
		case NodePoolTypeCpu, "standard": // 支持 "standard" 作为 "cpu" 的别名
			config := getNodePoolConfig(props, NodePoolTypeCpu)
			kubeManifest := createTypedNodePool(scope, fmt.Sprintf("%s-cpu", id), props, config, kubeVersion, NodePoolTypeCpu)
			kubeManifests = append(kubeManifests, kubeManifest)
			
		case NodePoolTypeGpu, "nvidia": // 支持 "nvidia" 作为 "gpu" 的别名
			config := getNodePoolConfig(props, NodePoolTypeGpu)
			kubeManifest := createTypedNodePool(scope, fmt.Sprintf("%s-gpu", id), props, config, kubeVersion, NodePoolTypeGpu)
			kubeManifests = append(kubeManifests, kubeManifest)
			
		case NodePoolTypeNeuron:
			config := getNodePoolConfig(props, NodePoolTypeNeuron)
			kubeManifest := createTypedNodePool(scope, fmt.Sprintf("%s-neuron", id), props, config, kubeVersion, NodePoolTypeNeuron)
			kubeManifests = append(kubeManifests, kubeManifest)
			
		default:
			fmt.Printf("Warning: Unknown node pool type '%s', skipping\n", nodePoolType)
		}
	}
	
	return kubeManifests
}

// 根据节点类型获取配置
func getNodePoolConfig(props *KarpenterNodePoolProps, nodeType string) NodePoolConfig {
	switch nodeType {
	case NodePoolTypeCpu:
		return NodePoolConfig{
			InstanceTypes:       props.CpuInstanceTypes,
			InstanceFamilies:    props.CpuInstanceFamilies,
			InstanceCategories:  props.CpuInstanceCategories,
			InstanceGenerations: props.CpuInstanceGenerations,
			CapacityTypes:       props.CpuCapacityTypes,
			Architectures:       props.CpuArchitectures,
			DiskSize:            props.CpuDiskSize,
			DiskType:            props.CpuDiskType,
			DiskIops:            props.CpuDiskIops,
			DiskThroughput:      props.CpuDiskThroughput,
			Labels:              props.CpuLabels,
			Taints:              props.CpuTaints,
		}
	case NodePoolTypeGpu:
		return NodePoolConfig{
			InstanceTypes:       props.GpuInstanceTypes,
			InstanceFamilies:    props.GpuInstanceFamilies,
			InstanceCategories:  props.GpuInstanceCategories,
			InstanceGenerations: props.GpuInstanceGenerations,
			CapacityTypes:       props.GpuCapacityTypes,
			Architectures:       props.GpuArchitectures,
			DiskSize:            props.GpuDiskSize,
			DiskType:            props.GpuDiskType,
			DiskIops:            props.GpuDiskIops,
			DiskThroughput:      props.GpuDiskThroughput,
			Labels:              props.GpuLabels,
			Taints:              props.GpuTaints,
		}
	case NodePoolTypeNeuron:
		return NodePoolConfig{
			InstanceTypes:       props.NeuronInstanceTypes,
			InstanceFamilies:    props.NeuronInstanceFamilies,
			InstanceCategories:  props.NeuronInstanceCategories,
			InstanceGenerations: props.NeuronInstanceGenerations,
			CapacityTypes:       props.NeuronCapacityTypes,
			Architectures:       props.NeuronArchitectures,
			DiskSize:            props.NeuronDiskSize,
			DiskType:            props.NeuronDiskType,
			DiskIops:            props.NeuronDiskIops,
			DiskThroughput:      props.NeuronDiskThroughput,
			Labels:              props.NeuronLabels,
			Taints:              props.NeuronTaints,
		}
	default:
		return NodePoolConfig{}
	}
}
// 统一的节点池创建函数
func createTypedNodePool(scope constructs.Construct, id string, props *KarpenterNodePoolProps, config NodePoolConfig, kubeVersion string, nodeType string) awseks.KubernetesManifest {
	resourceName := fmt.Sprintf("karpenter-%s", nodeType)
	roleName := *props.IamResources.NodeRole.RoleName()
	
	// 构建实例要求
	instanceRequirements := buildInstanceRequirements(config)
	
	// 构建标签和污点
	nodeLabels := buildNodeLabels(config, nodeType)
	nodeTaints := buildNodeTaints(config)
	
	// 构建 AMI 选择器
	amiSelectorTerms := buildAmiSelectorTerms(nodeType, kubeVersion, props.KarpenterOsType)
	
	// 构建磁盘配置
	blockDeviceMappings := buildBlockDeviceMappings(config)
	
	// 设置资源限制 - 移除硬编码限制，允许无限制扩展
	// cpuLimit := 1000
	// memoryLimit := "1000Gi"
	// gpuLimit := ""
	
	// 为 GPU 节点池添加 GPU 限制
	// if nodeType == NodePoolTypeGpu {
	//	gpuLimit = `
    // nvidia.com/gpu: 8`
	// }
	
	// Launch Template 配置已移除（Karpenter v1 不支持）
	launchTemplateConfig := ""

	// 创建 NodePool 和 EC2NodeClass 的 Kubernetes 清单
	nodePoolManifest := fmt.Sprintf(`
apiVersion: karpenter.sh/v1
kind: NodePool
metadata:
  name: %s
spec:
  template:
    metadata:
      labels:%s
    spec:
      requirements:%s%s
      nodeClassRef:
        group: karpenter.k8s.aws
        kind: EC2NodeClass
        name: %s
      expireAfter: 720h
  disruption:
    consolidationPolicy: WhenEmptyOrUnderutilized
    consolidateAfter: 30s
---
apiVersion: karpenter.k8s.aws/v1
kind: EC2NodeClass
metadata:
  name: %s
spec:
  instanceProfile: %s
  amiFamily: %s
  associatePublicIPAddress: false%s
  amiSelectorTerms:%s%s
  subnetSelectorTerms:
  - tags:
      karpenter.sh/discovery: %s
  securityGroupSelectorTerms:
  - tags:
      karpenter.sh/discovery: %s
  tags:
    karpenter.sh/discovery: %s
    nodepool-type: "%s"
    kubernetes-version: "%s"
`,
		resourceName,
		nodeLabels,
		instanceRequirements,
		nodeTaints,
		resourceName,
		resourceName,
		roleName,
		getAmiFamily(props.KarpenterOsType),
		launchTemplateConfig,
		amiSelectorTerms,
		blockDeviceMappings,
		props.ClusterName,
		props.ClusterName,
		props.ClusterName,
		nodeType,
		kubeVersion)
	
	return createKubernetesManifest(scope, id, props.Cluster, props.IamResources, nodePoolManifest, resourceName)
}

// 构建实例要求
func buildInstanceRequirements(config NodePoolConfig) string {
	var requirements []string
	
	// 具体实例类型要求 (最高优先级)
	if config.InstanceTypes != "" {
		instanceTypes := parseCommaSeparatedString(config.InstanceTypes)
		if len(instanceTypes) > 0 {
			values := fmt.Sprintf(`["%s"]`, strings.Join(instanceTypes, `", "`))
			requirements = append(requirements, fmt.Sprintf(`
      - key: node.kubernetes.io/instance-type
        operator: In
        values: %s`, values))
		}
	}
	
	// 实例家族要求
	if config.InstanceFamilies != "" {
		instanceFamilies := parseCommaSeparatedString(config.InstanceFamilies)
		if len(instanceFamilies) > 0 {
			values := fmt.Sprintf(`["%s"]`, strings.Join(instanceFamilies, `", "`))
			requirements = append(requirements, fmt.Sprintf(`
      - key: karpenter.k8s.aws/instance-family
        operator: In
        values: %s`, values))
		}
	}
	
	// 实例类别要求
	if config.InstanceCategories != "" {
		instanceCategories := parseCommaSeparatedString(config.InstanceCategories)
		if len(instanceCategories) > 0 {
			values := fmt.Sprintf(`["%s"]`, strings.Join(instanceCategories, `", "`))
			requirements = append(requirements, fmt.Sprintf(`
      - key: karpenter.k8s.aws/instance-category
        operator: In
        values: %s`, values))
		}
	}
	
	// 实例代数要求
	if config.InstanceGenerations != "" {
		instanceGenerations := parseCommaSeparatedString(config.InstanceGenerations)
		if len(instanceGenerations) > 0 {
			values := fmt.Sprintf(`["%s"]`, strings.Join(instanceGenerations, `", "`))
			requirements = append(requirements, fmt.Sprintf(`
      - key: karpenter.k8s.aws/instance-generation
        operator: In
        values: %s`, values))
		}
	} else {
		// 默认使用较新的代数
		requirements = append(requirements, `
      - key: karpenter.k8s.aws/instance-generation
        operator: Gt
        values: ["2"]`)
	}
	
	// 容量类型要求
	if config.CapacityTypes != "" {
		capacityTypes := parseCommaSeparatedString(config.CapacityTypes)
		if len(capacityTypes) > 0 {
			values := fmt.Sprintf(`["%s"]`, strings.Join(capacityTypes, `", "`))
			requirements = append(requirements, fmt.Sprintf(`
      - key: karpenter.sh/capacity-type
        operator: In
        values: %s`, values))
		}
	} else {
		// 默认支持 spot 和 on-demand
		requirements = append(requirements, `
      - key: karpenter.sh/capacity-type
        operator: In
        values: ["spot", "on-demand"]`)
	}
	
	// 架构要求
	if config.Architectures != "" {
		architectures := parseCommaSeparatedString(config.Architectures)
		if len(architectures) > 0 {
			values := fmt.Sprintf(`["%s"]`, strings.Join(architectures, `", "`))
			requirements = append(requirements, fmt.Sprintf(`
      - key: kubernetes.io/arch
        operator: In
        values: %s`, values))
		}
	} else {
		// 默认支持 amd64 和 arm64
		requirements = append(requirements, `
      - key: kubernetes.io/arch
        operator: In
        values: ["amd64", "arm64"]`)
	}
	
	// 操作系统要求
	requirements = append(requirements, `
      - key: kubernetes.io/os
        operator: In
        values: ["linux"]`)
	
	return strings.Join(requirements, "")
}

// 构建节点标签
func buildNodeLabels(config NodePoolConfig, nodeType string) string {
	labels := make(map[string]string)
	
	// 添加默认标签
	labels["nodepool-type"] = nodeType
	
	// 解析用户自定义标签 (格式: "key1=value1,key2=value2")
	if config.Labels != "" {
		labelPairs := parseCommaSeparatedString(config.Labels)
		for _, pair := range labelPairs {
			if strings.Contains(pair, "=") {
				parts := strings.SplitN(pair, "=", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					value := strings.TrimSpace(parts[1])
					if key != "" && value != "" {
						labels[key] = value
					}
				}
			}
		}
	}
	
	var labelStrings []string
	for k, v := range labels {
		labelStrings = append(labelStrings, fmt.Sprintf(`
        %s: "%s"`, k, v))
	}
	
	if len(labelStrings) == 0 {
		return ""
	}
	
	return strings.Join(labelStrings, "")
}

// 构建节点污点
func buildNodeTaints(config NodePoolConfig) string {
	if config.Taints == "" {
		return ""
	}
	
	// 解析污点 (格式: "key1=value1:Effect1,key2=value2:Effect2")
	taintPairs := parseCommaSeparatedString(config.Taints)
	var taintStrings []string
	
	for _, pair := range taintPairs {
		if strings.Contains(pair, "=") && strings.Contains(pair, ":") {
			// 分割 key=value:effect
			parts := strings.SplitN(pair, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				valueEffect := strings.TrimSpace(parts[1])
				
				valueParts := strings.SplitN(valueEffect, ":", 2)
				if len(valueParts) == 2 {
					value := strings.TrimSpace(valueParts[0])
					effect := strings.TrimSpace(valueParts[1])
					
					if key != "" && value != "" && effect != "" {
						taintStrings = append(taintStrings, fmt.Sprintf(`
      - key: %s
        value: "%s"
        effect: %s`, key, value, effect))
					}
				}
			}
		}
	}
	
	if len(taintStrings) == 0 {
		return ""
	}
	
	return fmt.Sprintf(`
      taints:%s`, strings.Join(taintStrings, ""))
}
// 构建 AMI 选择器
func buildAmiSelectorTerms(nodeType string, kubeVersion string, osType string) string {
	amiFamily := getAmiFamily(osType)
	
	var amiSelectorTerms []string
	
	switch amiFamily {
	case "Bottlerocket":
		switch nodeType {
		case NodePoolTypeGpu:
			amiSelectorTerms = []string{
				fmt.Sprintf("/aws/service/bottlerocket/aws-k8s-%s-nvidia/x86_64/latest/image_id", kubeVersion),
				fmt.Sprintf("/aws/service/bottlerocket/aws-k8s-%s-nvidia/arm64/latest/image_id", kubeVersion),
			}
		case NodePoolTypeNeuron:
			// Bottlerocket 可能不支持 Neuron，使用标准版本
			fmt.Printf("Warning: Bottlerocket may not support Neuron, using standard AMI\n")
			amiSelectorTerms = []string{
				fmt.Sprintf("/aws/service/bottlerocket/aws-k8s-%s/x86_64/latest/image_id", kubeVersion),
			}
		default: // CPU
			amiSelectorTerms = []string{
				fmt.Sprintf("/aws/service/bottlerocket/aws-k8s-%s/x86_64/latest/image_id", kubeVersion),
				fmt.Sprintf("/aws/service/bottlerocket/aws-k8s-%s/arm64/latest/image_id", kubeVersion),
			}
		}
	case "AL2023":
		switch nodeType {
		case NodePoolTypeGpu:
			amiSelectorTerms = []string{
				fmt.Sprintf("/aws/service/eks/optimized-ami/%s/amazon-linux-2023/x86_64/nvidia/recommended/image_id", kubeVersion),
				fmt.Sprintf("/aws/service/eks/optimized-ami/%s/amazon-linux-2023/arm64/nvidia/recommended/image_id", kubeVersion),
			}
		case NodePoolTypeNeuron:
			amiSelectorTerms = []string{
				fmt.Sprintf("/aws/service/eks/optimized-ami/%s/amazon-linux-2023/x86_64/neuron/recommended/image_id", kubeVersion),
			}
		default: // CPU
			amiSelectorTerms = []string{
				fmt.Sprintf("/aws/service/eks/optimized-ami/%s/amazon-linux-2023/x86_64/standard/recommended/image_id", kubeVersion),
				fmt.Sprintf("/aws/service/eks/optimized-ami/%s/amazon-linux-2023/arm64/standard/recommended/image_id", kubeVersion),
			}
		}
	default:
		// 默认使用 AL2023 标准版本
		amiSelectorTerms = []string{
			fmt.Sprintf("/aws/service/eks/optimized-ami/%s/amazon-linux-2023/x86_64/standard/recommended/image_id", kubeVersion),
			fmt.Sprintf("/aws/service/eks/optimized-ami/%s/amazon-linux-2023/arm64/standard/recommended/image_id", kubeVersion),
		}
	}
	
	var amiSelectorTermsYaml string
	for _, ssmParam := range amiSelectorTerms {
		amiSelectorTermsYaml += fmt.Sprintf(`
    - ssmParameter: %s`, ssmParam)
	}
	
	return amiSelectorTermsYaml
}

// 构建块设备映射
func buildBlockDeviceMappings(config NodePoolConfig) string {
	if config.DiskSize <= 0 {
		return ""
	}
	
	// 默认值
	volumeType := "gp3"
	if config.DiskType != "" {
		volumeType = config.DiskType
	}
	
	// 构建基本的 EBS 配置
	ebsConfig := fmt.Sprintf(`
        volumeSize: %dGi
        volumeType: %s
        deleteOnTermination: true`, config.DiskSize, volumeType)
	
	// 添加 IOPS 配置（仅对支持的卷类型）
	iops := config.DiskIops
	if iops == 0 && (volumeType == "gp3" || volumeType == "io1" || volumeType == "io2") {
		// 默认 IOPS 设置为 3000
		iops = 3000
	}
	if iops > 0 && (volumeType == "gp3" || volumeType == "io1" || volumeType == "io2") {
		ebsConfig += fmt.Sprintf(`
        iops: %d`, iops)
	}
	
	// 添加吞吐量配置（仅对 gp3 卷类型）
	if config.DiskThroughput > 0 && volumeType == "gp3" {
		ebsConfig += fmt.Sprintf(`
        throughput: %d`, config.DiskThroughput)
	}
	
	return fmt.Sprintf(`
  blockDeviceMappings:
    - deviceName: /dev/xvda
      ebs:%s`, ebsConfig)
}

// 获取 AMI 家族
func getAmiFamily(osType string) string {
	if osType == OsTypeBottlerocket {
		return "Bottlerocket"
	}
	return "AL2023" // 默认使用 Amazon Linux 2023
}

// 辅助函数：解析逗号分隔的字符串
func parseCommaSeparatedString(input string) []string {
	if input == "" {
		return nil
	}
	
	var result []string
	for _, item := range strings.Split(input, ",") {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// 辅助函数：解析 YAML 并创建 Kubernetes 清单
func createKubernetesManifest(scope constructs.Construct, id string, cluster awseks.Cluster, iamResources *KarpenterIamResources, nodePoolManifest string, resourceName string) awseks.KubernetesManifest {
	// 解析 YAML 文档
	var manifests []*map[string]interface{}
	decoder := yaml.NewDecoder(strings.NewReader(nodePoolManifest))
	for {
		var manifest map[string]interface{}
		if err := decoder.Decode(&manifest); err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
		if len(manifest) > 0 {
			// 确保 metadata.name 字段存在且不为空
			if metadata, ok := manifest["metadata"].(map[string]interface{}); ok {
				if name, ok := metadata["name"].(string); ok && name == "" {
					// 如果名称为空，设置一个默认名称
					metadata["name"] = resourceName
				}
			}
			manifestCopy := manifest
			manifests = append(manifests, &manifestCopy)
		}
	}
	
	// 使用解析后的对象创建资源
	kubeManifest := awseks.NewKubernetesManifest(scope, jsii.String(id), &awseks.KubernetesManifestProps{
		Cluster:   cluster,
		Manifest:  &manifests,
		Overwrite: jsii.Bool(false), // 改为 false，避免不必要的更新
		Prune:     jsii.Bool(true),  // 添加 prune 参数，清理不再需要的资源
	})
	
	// 添加依赖关系，确保 IAM 资源先创建
	kubeManifest.Node().AddDependency(iamResources.InstanceProfile)
	kubeManifest.Node().AddDependency(iamResources.NodeRole)
	
	return kubeManifest
}

// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package eks

import (
	"strings"
	
	"github.com/aws/aws-cdk-go/awscdk/v2/awseks"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

// VpcCniUpgradeProps VPC CNI 升级配置
type VpcCniUpgradeProps struct {
	Cluster          awseks.Cluster
	ClusterName      string
	VpcCniVersion    string
	EnableMultiNic   bool
}

// upgradeVpcCniAndConfigureMultiNic 升级 VPC CNI 到 1.20+ 并配置 Multi-NIC
func upgradeVpcCniAndConfigureMultiNic(scope constructs.Construct, id string, props *VpcCniUpgradeProps, eksAdminSA awseks.KubernetesManifest, eksAdminCRB awseks.KubernetesManifest) awseks.KubernetesManifest {
	// 设置默认值
	if props.VpcCniVersion == "" {
		props.VpcCniVersion = "v1.20.1"
	}

	manifestUrl := "https://raw.githubusercontent.com/aws/amazon-vpc-cni-k8s/" + props.VpcCniVersion + "/config/master/aws-k8s-cni.yaml"

	// 直接使用 Job，依赖传入的 eks-admin 资源
	vpcCniJob := awseks.NewKubernetesManifest(scope, jsii.String(id), &awseks.KubernetesManifestProps{
		Cluster: props.Cluster,
		Manifest: &[]*map[string]interface{}{
			{
				"apiVersion": "batch/v1",
				"kind":       "Job",
				"metadata": map[string]interface{}{
					"name":      "upgrade-vpc-cni",
					"namespace": "kube-system",
				},
				"spec": map[string]interface{}{
					"template": map[string]interface{}{
						"spec": map[string]interface{}{
							"restartPolicy":      "Never",
							"serviceAccountName": "eks-admin",
							"containers": []*map[string]interface{}{
								{
									"name":  "upgrade-vpc-cni",
									"image": "public.ecr.aws/amazonlinux/amazonlinux:latest",
									"command": []*string{
										jsii.String("/bin/sh"),
										jsii.String("-c"),
										jsii.String(`
											# Install kubectl and apply VPC CNI manifest
											dnf install -y --allowerasing curl
											curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
											chmod +x kubectl
											mv kubectl /usr/local/bin/
											
											# Apply VPC CNI manifest and enable Multi-NIC
											/usr/local/bin/kubectl apply --server-side --force-conflicts -f ` + manifestUrl + `
											/usr/local/bin/kubectl set env daemonset aws-node -n kube-system ENABLE_MULTI_NIC=true
										`),
									},
								},
							},
						},
					},
				},
			},
		},
	})

	// 设置依赖关系
	vpcCniJob.Node().AddDependency(eksAdminSA)
	vpcCniJob.Node().AddDependency(eksAdminCRB)

	return vpcCniJob
}

// needsMultiNicSupport 检查是否需要 Multi-NIC 支持
func needsMultiNicSupport(nodePoolProps *KarpenterNodePoolProps) bool {
	instanceTypesConfigs := []string{
		nodePoolProps.GpuInstanceTypes,
		nodePoolProps.CpuInstanceTypes,
		nodePoolProps.NeuronInstanceTypes,
	}
	
	// 检查是否包含支持多网卡的实例类型
	multiNicInstances := []string{
		"g6e.24xlarge", "c6in.32xlarge", "c6in.metal", "c8gn.48xlarge", 
		"hpc6id.32xlarge", "hpc7a.12xlarge", "hpc7a.24xlarge", "hpc7a.48xlarge", "hpc7a.96xlarge",
		"m6idn.32xlarge", "m6idn.metal", "m6in.32xlarge", "m6in.metal",
		"r8gn.48xlarge", "r8gn.metal-48xl", "r6idn.32xlarge", "r6idn.metal",
	}
	
	for _, instanceTypes := range instanceTypesConfigs {
		for _, multiNicInstance := range multiNicInstances {
			if strings.Contains(instanceTypes, multiNicInstance) {
				return true
			}
		}
	}
	
	return false
}

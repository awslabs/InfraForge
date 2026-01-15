// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package eks

import (
	"fmt"
	"strings"
	
	"github.com/awslabs/InfraForge/core/utils/types"
	"github.com/awslabs/InfraForge/core/utils/aws"
	"github.com/awslabs/InfraForge/core/dependency"
	"github.com/awslabs/InfraForge/core/partition"
	
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awseks"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/jsii-runtime-go"
)

// csiDriverConfig CSI Driver通用配置
type csiDriverConfig struct {
	name               string
	chartId            string // Helm Chart ID，用于CloudFormation资源命名
	serviceAccountName string
	chartName          string
	repository         string
	managedPolicyName  string        // 如果不为空，使用managed policy
	customPolicy       awsiam.Policy // 如果managedPolicyName为空，使用custom policy
	version            string
}

// deployGenericCsiDriver 部署通用CSI Driver（EBS/EFS/FSx）
func deployGenericCsiDriver(stack awscdk.Stack, cluster awseks.Cluster, config *csiDriverConfig) awseks.HelmChart {
	// 创建ServiceAccount
	saId := jsii.String(config.name + "-csi-controller-sa")
	csiServiceAccount := cluster.AddServiceAccount(saId, &awseks.ServiceAccountOptions{
		Name:      jsii.String(config.serviceAccountName),
		Namespace: jsii.String("kube-system"),
	})

	// 附加IAM权限
	if config.managedPolicyName != "" {
		// 使用AWS managed policy
		managedPolicy := awsiam.ManagedPolicy_FromAwsManagedPolicyName(jsii.String(config.managedPolicyName))
		csiServiceAccount.Role().AddManagedPolicy(managedPolicy)
	} else if config.customPolicy != nil {
		// 使用自定义policy
		csiServiceAccount.Role().AttachInlinePolicy(config.customPolicy)
	}

	// 部署Helm Chart
	helmOptions := &awseks.HelmChartOptions{
		Chart:      jsii.String(config.chartName),
		Repository: jsii.String(config.repository),
		Namespace:  jsii.String("kube-system"),
		Values: &map[string]interface{}{
			"controller": map[string]interface{}{
				"serviceAccount": map[string]interface{}{
					"create": false,
					"name":   config.serviceAccountName,
				},
			},
		},
	}
	// 只有指定了版本且不是latest才设置Version
	if config.version != "" && config.version != "latest" {
		helmOptions.Version = jsii.String(config.version)
	}
	
	csiChart := cluster.AddHelmChart(jsii.String(config.chartId), helmOptions)
	csiChart.Node().AddDependency(csiServiceAccount)
	
	return csiChart
}

// deployEbsCsiDriver 部署 EBS CSI Driver
func deployEbsCsiDriver(stack awscdk.Stack, cluster awseks.Cluster, version string) awseks.HelmChart {
	return deployGenericCsiDriver(stack, cluster, &csiDriverConfig{
		name:               "ebs",
		chartId:            "ebs-csi-driver",
		serviceAccountName: "ebs-csi-controller-sa",
		chartName:          "aws-ebs-csi-driver",
		repository:         "https://kubernetes-sigs.github.io/aws-ebs-csi-driver",
		managedPolicyName:  "service-role/AmazonEBSCSIDriverPolicy",
		version:            version,
	})
}

// deployMountpointS3CsiDriver 部署 Mountpoint S3 CSI Driver
func deployMountpointS3CsiDriver(stack awscdk.Stack, cluster awseks.Cluster, version string, bucketName string) awseks.HelmChart {
	// 创建 IAM 角色用于 S3 CSI Driver（使用 CDK 的 ServiceAccount 方法）
	s3CsiServiceAccount := cluster.AddServiceAccount(jsii.String("s3-csi-sa"), &awseks.ServiceAccountOptions{
		Name:      jsii.String("s3-csi-driver-sa"),
		Namespace: jsii.String("kube-system"),
	})

	// 创建 S3 访问策略并附加到角色
	s3CsiPolicy := awsiam.NewPolicy(stack, jsii.String("MountpointS3CsiDriverPolicy"), &awsiam.PolicyProps{
		Statements: &[]awsiam.PolicyStatement{
			awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
				Effect: awsiam.Effect_ALLOW,
				Actions: jsii.Strings(
					"s3:ListBucket",
					"s3:GetObject",
					"s3:PutObject",
					"s3:DeleteObject",
					"s3:AbortMultipartUpload",
					"s3:ListMultipartUploadParts",
				),
				Resources: func() *[]*string {
					if bucketName != "" {
						return jsii.Strings(
							fmt.Sprintf("arn:%s:s3:::%s", partition.DefaultPartition, bucketName),
							fmt.Sprintf("arn:%s:s3:::%s/*", partition.DefaultPartition, bucketName),
						)
					}
					// 如果没有指定bucket，不给任何S3权限
					return jsii.Strings()
				}(),
			}),
		},
	})

	// 将策略附加到 ServiceAccount 的角色
	s3CsiServiceAccount.Role().AttachInlinePolicy(s3CsiPolicy)

	// 创建额外的 RBAC 权限，S3 CSI Driver 需要这些权限来管理 mountpoint pods
	s3CsiClusterRole := awseks.NewKubernetesManifest(stack, jsii.String("S3CsiAdditionalClusterRole"), &awseks.KubernetesManifestProps{
		Cluster: cluster,
		Manifest: &[]*map[string]interface{}{
			{
				"apiVersion": "rbac.authorization.k8s.io/v1",
				"kind":       "ClusterRole",
				"metadata": map[string]interface{}{
					"name": "s3-csi-driver-additional-permissions",
				},
				"rules": []map[string]interface{}{
					{
						"apiGroups": []string{""},
						"resources": []string{"pods"},
						"verbs":     []string{"get", "list", "watch", "create", "delete", "update", "patch"},
					},
					{
						"apiGroups": []string{""},
						"resources": []string{"nodes"},
						"verbs":     []string{"get", "list", "watch"},
					},
					{
						"apiGroups": []string{""},
						"resources": []string{"serviceaccounts"},
						"verbs":     []string{"get", "list", "watch"},
					},
					{
						"apiGroups": []string{""},
						"resources": []string{"namespaces"},
						"verbs":     []string{"get", "list", "watch", "create"},
					},
					{
						"apiGroups": []string{"storage.k8s.io"},
						"resources": []string{"volumeattachments"},
						"verbs":     []string{"get", "list", "watch", "update", "patch"},
					},
					{
						"apiGroups": []string{""},
						"resources": []string{"persistentvolumes"},
						"verbs":     []string{"get", "list", "watch", "update", "patch"},
					},
					{
						"apiGroups": []string{""},
						"resources": []string{"persistentvolumeclaims"},
						"verbs":     []string{"get", "list", "watch"},
					},
					{
						"apiGroups": []string{"s3.csi.aws.com"},
						"resources": []string{"mountpoints3podattachments"},
						"verbs":     []string{"get", "list", "watch", "create", "delete", "update", "patch"},
					},
					{
						"apiGroups": []string{"apiextensions.k8s.io"},
						"resources": []string{"customresourcedefinitions"},
						"verbs":     []string{"get", "list", "watch"},
					},
				},
			},
		},
	})

	// 创建 ClusterRoleBinding
	s3CsiClusterRoleBinding := awseks.NewKubernetesManifest(stack, jsii.String("S3CsiAdditionalClusterRoleBinding"), &awseks.KubernetesManifestProps{
		Cluster: cluster,
		Manifest: &[]*map[string]interface{}{
			{
				"apiVersion": "rbac.authorization.k8s.io/v1",
				"kind":       "ClusterRoleBinding",
				"metadata": map[string]interface{}{
					"name": "s3-csi-driver-additional-binding",
				},
				"roleRef": map[string]interface{}{
					"apiGroup": "rbac.authorization.k8s.io",
					"kind":     "ClusterRole",
					"name":     "s3-csi-driver-additional-permissions",
				},
				"subjects": []map[string]interface{}{
					{
						"kind":      "ServiceAccount",
						"name":      "s3-csi-driver-sa",
						"namespace": "kube-system",
					},
				},
			},
		},
	})

	// 部署 Mountpoint S3 CSI Driver Helm Chart
	// 让 CDK 管理 ServiceAccount，Helm 不创建 ServiceAccount
	helmOptions := &awseks.HelmChartOptions{
		Chart:      jsii.String("aws-mountpoint-s3-csi-driver"),
		Repository: jsii.String("https://awslabs.github.io/mountpoint-s3-csi-driver"),
		Namespace:  jsii.String("kube-system"),
		Values: &map[string]interface{}{
			"node": map[string]interface{}{
				"serviceAccount": map[string]interface{}{
					"create": false,
					"name":   "s3-csi-driver-sa",
				},
			},
			"controller": map[string]interface{}{
				"serviceAccount": map[string]interface{}{
					"create": false,
					"name":   "s3-csi-driver-sa",
				},
			},
		},
	}
	// 只有指定了版本且不是latest才设置Version
	if version != "" && version != "latest" {
		helmOptions.Version = jsii.String(version)
	}
	s3CsiChart := cluster.AddHelmChart(jsii.String("s3-csi"), helmOptions)

	// 添加依赖关系，确保资源按正确顺序创建
	s3CsiChart.Node().AddDependency(s3CsiServiceAccount)
	s3CsiChart.Node().AddDependency(s3CsiClusterRole)
	s3CsiChart.Node().AddDependency(s3CsiClusterRoleBinding)
	s3CsiClusterRoleBinding.Node().AddDependency(s3CsiClusterRole)

	return s3CsiChart
}

// 部署 Mountpoint S3 CSI 驱动和 StorageClass
func deployMountpointS3CsiDriverWithStorage(stack awscdk.Stack, cluster awseks.Cluster, eksInstance *EksInstanceConfig) awseks.HelmChart {
	// 如果没有指定S3 bucket，不部署S3 CSI Driver
	if eksInstance.S3BucketName == "" {
		fmt.Printf("Warning: S3BucketName is required for Mountpoint S3 CSI Driver\n")
		return nil
	}
	
	// 部署 CSI 驱动
	s3CsiChart := deployMountpointS3CsiDriver(stack, cluster, eksInstance.MountpointS3CsiDriverVersion, eksInstance.S3BucketName)

	// 如果需要创建 StorageClass
	if types.GetBoolValue(eksInstance.CreateStorageClass, false) {
		// 检查必需的 S3 配置
		if eksInstance.S3BucketName == "" {
			fmt.Printf("Warning: s3BucketName is required when creating StorageClass for Mountpoint S3 CSI Driver\n")
			return nil
		}

		// 通过bucket名称自动获取区域
		s3Region := partition.DefaultPartition
		if eksInstance.S3BucketName != "" {
			if region, err := aws.GetBucketRegion(eksInstance.S3BucketName); err == nil {
				s3Region = region
			}
		}

		var storageClassName string
		if eksInstance.StorageClassName == "" {
			storageClassName = "s3-csi"
		} else {
			// 确保 storageClassName 是小写的，并且符合 Kubernetes 命名规范
			storageClassName = strings.ToLower(eksInstance.StorageClassName)
			// 替换任何不符合规范的字符为连字符
			storageClassName = strings.ReplaceAll(storageClassName, "_", "-")
			// 确保以字母或数字开头和结尾
			if !strings.HasPrefix(storageClassName, "s3-") {
				storageClassName = "s3-" + storageClassName
			}
		}

		// 创建 StorageClass 使用 map 结构
		storageClassObj := map[string]interface{}{
			"apiVersion": "storage.k8s.io/v1",
			"kind": "StorageClass",
			"metadata": map[string]interface{}{
				"name": storageClassName,
			},
			"provisioner": "s3.csi.aws.com",
			"parameters": map[string]interface{}{
				"bucketName": eksInstance.S3BucketName,
				"region":     s3Region,
			},
			"reclaimPolicy": "Delete",
			"volumeBindingMode": "Immediate",
		}

		// 添加 StorageClass 清单
		scManifest := cluster.AddManifest(jsii.String(fmt.Sprintf("%s-storage-class", storageClassName)), &storageClassObj)

		// 确保 StorageClass 在 CSI 驱动部署后创建
		scManifest.Node().AddDependency(s3CsiChart)

		// 如果需要创建静态 PV
		if types.GetBoolValue(eksInstance.CreateStaticPV, false) {
			pvName := fmt.Sprintf("%s-pv", storageClassName)
			pvObj := map[string]interface{}{
				"apiVersion": "v1",
				"kind": "PersistentVolume",
				"metadata": map[string]interface{}{
					"name": pvName,
				},
				"spec": map[string]interface{}{
					"capacity": map[string]interface{}{
						"storage": "1200Gi", // S3 存储容量（理论上无限）
					},
					"accessModes": []string{"ReadWriteMany"},
					"persistentVolumeReclaimPolicy": "Retain",
					"storageClassName": storageClassName,
					"claimRef": map[string]interface{}{
						"namespace": "default",
						"name": fmt.Sprintf("%s-pvc", storageClassName),
					},
					"mountOptions": []string{
						"allow-delete",
						fmt.Sprintf("region %s", s3Region),
					},
					"csi": map[string]interface{}{
						"driver": "s3.csi.aws.com",
						"volumeHandle": fmt.Sprintf("%s-%s", eksInstance.S3BucketName, s3Region),
						"volumeAttributes": map[string]interface{}{
							"bucketName": eksInstance.S3BucketName,
						},
					},
				},
			}

			// 添加 PV 清单
			pvManifest := cluster.AddManifest(jsii.String(pvName), &pvObj)

			// 确保 PV 在 StorageClass 创建后创建
			pvManifest.Node().AddDependency(scManifest)

			// 如果需要创建默认 PVC
			if types.GetBoolValue(eksInstance.CreateDefaultPVC, false) {
				pvcName := fmt.Sprintf("%s-pvc", storageClassName)
				pvcObj := map[string]interface{}{
					"apiVersion": "v1",
					"kind": "PersistentVolumeClaim",
					"metadata": map[string]interface{}{
						"name": pvcName,
						"namespace": "default",
					},
					"spec": map[string]interface{}{
						"accessModes": []string{"ReadWriteMany"},
						"storageClassName": storageClassName,
						"resources": map[string]interface{}{
							"requests": map[string]interface{}{
								"storage": "100Gi", // PVC 请求的存储大小
							},
						},
					},
				}

				// 添加 PVC 清单
				pvcManifest := cluster.AddManifest(jsii.String(pvcName), &pvcObj)

				// 确保 PVC 在 PV 创建后创建
				pvcManifest.Node().AddDependency(pvManifest)
			}
		}
	}

	return s3CsiChart
}

// deployDependentStorageCsiDrivers 部署依赖于预创建文件系统的CSI驱动
// 支持: EFS, FSx Lustre (未来可扩展: FSx ONTAP, FSx OpenZFS, FSx Windows)
func deployDependentStorageCsiDrivers(stack awscdk.Stack, cluster awseks.Cluster, magicTokenStr string, eksInstance *EksInstanceConfig) []awseks.HelmChart {
    // 直接从 eksInstance.DependsOn 解析存储类型
    dependsOn := strings.ToUpper(eksInstance.DependsOn)
    var charts []awseks.HelmChart
    
    // 检查是否包含 LUSTRE 前缀
    if strings.Contains(dependsOn, "LUSTRE:") {
        chart := deployLustreCsiDriver(stack, cluster, magicTokenStr, eksInstance)
        if chart != nil {
            charts = append(charts, chart)
        }
    }
    
    // 检查是否包含 EFS 前缀
    if strings.Contains(dependsOn, "EFS:") {
        chart := deployEfsCsiDriver(stack, cluster, magicTokenStr, eksInstance)
        if chart != nil {
            charts = append(charts, chart)
        }
    }
    
    return charts
}

// 部署 FSx Lustre CSI 驱动和 StorageClass
func deployLustreCsiDriver(stack awscdk.Stack, cluster awseks.Cluster, magicTokenStr string, eksInstance *EksInstanceConfig) awseks.HelmChart {
	// 如果未指定版本，不部署
	if eksInstance.FsxCsiDriverVersion == "" {
		return nil
	}

	// 使用通用函数获取Lustre依赖
	lustreProperties, err := dependency.ExtractDependencyProperties(magicTokenStr, "LUSTRE")
	if err != nil {
		fmt.Printf("Warning: failed to extract Lustre dependencies: %v\n", err)
		return nil
	}

	// 提取Lustre配置信息
	lustreFileSystemId, _ := lustreProperties["fileSystemId"].(string)
	lustreMountName, _ := lustreProperties["mountName"].(string)
	lustreDnsName, _ := lustreProperties["dnsName"].(string)
	
	var lustreStorageCapacityGiB int = 1200 // 默认值
	if storageCapacity, ok := lustreProperties["storageCapacityGiB"].(float64); ok {
		lustreStorageCapacityGiB = int(storageCapacity)
	}

	// 如果没有提供 dnsName，构造一个
	if lustreDnsName == "" {
		lustreDnsName = fmt.Sprintf("%s.fsx.%s.amazonaws.com", lustreFileSystemId, *stack.Region())
	}

	if lustreFileSystemId == "" {
		fmt.Printf("Warning: Lustre file system ID not found\n")
		return nil
	}

	// 设置版本（留空或latest使用最新版）
	version := eksInstance.FsxCsiDriverVersion

	// 创建 IAM 自定义策略
	csiPolicy := awsiam.NewPolicy(stack, jsii.String("FsxLustreCsiPolicy"), &awsiam.PolicyProps{
		Statements: &[]awsiam.PolicyStatement{
			awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
				Effect: awsiam.Effect_ALLOW,
				Actions: jsii.Strings(
					"fsx:CreateBackup",
					"fsx:DeleteBackup",
					"fsx:DescribeBackups",
					"fsx:ListTagsForResource",
					"fsx:DescribeFileSystems",
					"ec2:DescribeSubnets",
					"ec2:DescribeSecurityGroups",
					"ec2:DescribeNetworkInterfaces",
				),
				Resources: jsii.Strings("*"),
			}),
		},
	})

	// 使用通用函数部署CSI Driver
	csiChart := deployGenericCsiDriver(stack, cluster, &csiDriverConfig{
		name:               "fsx",
		chartId:            "fsx-csi-driver",
		serviceAccountName: "fsx-csi-controller-sa",
		chartName:          "aws-fsx-csi-driver",
		repository:         "https://kubernetes-sigs.github.io/aws-fsx-csi-driver",
		customPolicy:       csiPolicy,
		version:            version,
	})

	// 如果需要创建 StorageClass
	if types.GetBoolValue(eksInstance.CreateStorageClass, false) {
		var storageClassName string
		if eksInstance.StorageClassName == "" {
			storageClassName = "fsx-lustre"
		} else {
			// 确保 storageClassName 是小写的，并且符合 Kubernetes 命名规范
			storageClassName = strings.ToLower(eksInstance.StorageClassName)
			// 替换任何不符合规范的字符为连字符
			storageClassName = strings.ReplaceAll(storageClassName, "_", "-")
			// 确保以字母或数字开头和结尾
			if !strings.HasPrefix(storageClassName, "fsx-") {
				storageClassName = "fsx-" + storageClassName
			}
		}

		// 创建 StorageClass 使用 map 结构
		storageClassObj := map[string]interface{}{
			"apiVersion": "storage.k8s.io/v1",
			"kind": "StorageClass",
			"metadata": map[string]interface{}{
				"name": storageClassName,
			},
			"provisioner": "fsx.csi.aws.com",
			"parameters": map[string]interface{}{
				"fileSystemId": lustreFileSystemId,
				"mountname": lustreMountName,  // 注意：使用小写的 mountname
			},
			"reclaimPolicy": "Delete",
			"volumeBindingMode": "Immediate",
			"mountOptions": []string{"flock"},
		}

		// 添加 StorageClass 清单
		scManifest := cluster.AddManifest(jsii.String(fmt.Sprintf("%s-storage-class", storageClassName)), &storageClassObj)

		// 确保 StorageClass 在 CSI 驱动部署后创建
		scManifest.Node().AddDependency(csiChart)

		// 如果需要创建静态 PV
		if types.GetBoolValue(eksInstance.CreateStaticPV, false) {
			pvName := fmt.Sprintf("%s-pv", storageClassName)
			pvObj := map[string]interface{}{
				"apiVersion": "v1",
				"kind": "PersistentVolume",
				"metadata": map[string]interface{}{
					"name": pvName,
				},
				"spec": map[string]interface{}{
					"capacity": map[string]interface{}{
						"storage": fmt.Sprintf("%dGi", lustreStorageCapacityGiB),
					},
					"volumeMode": "Filesystem",
					"accessModes": []string{"ReadWriteMany"},
					"persistentVolumeReclaimPolicy": "Retain",
					"storageClassName": storageClassName,
					"mountOptions": []string{"flock"},
					"csi": map[string]interface{}{
						"driver": "fsx.csi.aws.com",
						"volumeHandle": lustreFileSystemId,
						"volumeAttributes": map[string]interface{}{
							"fileSystemId": lustreFileSystemId,
							"mountname": lustreMountName,  // 注意：使用小写的 mountname
							"dnsname": lustreDnsName,
						},
					},
				},
			}

			// 添加 PV 清单
			pvManifest := cluster.AddManifest(jsii.String(pvName), &pvObj)

			// 确保 PV 在 StorageClass 创建后创建
			pvManifest.Node().AddDependency(scManifest)

			// 如果需要创建默认 PVC
			if types.GetBoolValue(eksInstance.CreateDefaultPVC, false) {
				// 设置默认命名空间
				pvcNamespace := eksInstance.DefaultPVCNamespace
				if pvcNamespace == "" {
					pvcNamespace = "default"
				}

				pvcName := fmt.Sprintf("%s-pvc", storageClassName)
				pvcObj := map[string]interface{}{
					"apiVersion": "v1",
					"kind": "PersistentVolumeClaim",
					"metadata": map[string]interface{}{
						"name": pvcName,
						"namespace": pvcNamespace,
					},
					"spec": map[string]interface{}{
						"accessModes": []string{"ReadWriteMany"},
						"storageClassName": storageClassName,
						"resources": map[string]interface{}{
							"requests": map[string]interface{}{
								"storage": fmt.Sprintf("%dGi", lustreStorageCapacityGiB),
							},
						},
						"volumeName": pvName,  // 显式指定要使用的 PV
					},
				}

				// 添加 PVC 清单
				pvcManifest := cluster.AddManifest(jsii.String(fmt.Sprintf("%s-%s", pvcName, pvcNamespace)), &pvcObj)

				// 确保 PVC 在 PV 创建后创建
				pvcManifest.Node().AddDependency(pvManifest)
			}
		}
	}

	return csiChart
}


// 部署 EFS CSI 驱动和 StorageClass
func deployEfsCsiDriver(stack awscdk.Stack, cluster awseks.Cluster, magicTokenStr string, eksInstance *EksInstanceConfig) awseks.HelmChart {
	// 如果未指定版本，不部署
	if eksInstance.EfsCsiDriverVersion == "" {
		return nil
	}

	// 使用通用函数获取EFS依赖
	efsProperties, err := dependency.ExtractDependencyProperties(magicTokenStr, "EFS")
	if err != nil {
		fmt.Printf("Warning: failed to extract EFS dependencies: %v\n", err)
		return nil
	}

	// 提取EFS配置信息
	efsFileSystemId, _ := efsProperties["fileSystemId"].(string)

	if efsFileSystemId == "" {
		fmt.Printf("Warning: EFS file system ID not found\n")
		return nil
	}

	// 设置版本（留空或latest使用最新版）
	version := eksInstance.EfsCsiDriverVersion

	// 使用通用函数部署CSI Driver
	csiChart := deployGenericCsiDriver(stack, cluster, &csiDriverConfig{
		name:               "efs",
		chartId:            "efs-csi-driver",
		serviceAccountName: "efs-csi-controller-sa",
		chartName:          "aws-efs-csi-driver",
		repository:         "https://kubernetes-sigs.github.io/aws-efs-csi-driver",
		managedPolicyName:  "service-role/AmazonEFSCSIDriverPolicy",
		version:            version,
	})

	// 如果需要创建 StorageClass
	if types.GetBoolValue(eksInstance.CreateStorageClass, false) {
		var storageClassName string
		if eksInstance.StorageClassName == "" {
			storageClassName = "efs-sc"
		} else {
			// 确保 storageClassName 是小写的，并且符合 Kubernetes 命名规范
			storageClassName = strings.ToLower(eksInstance.StorageClassName)
			// 替换任何不符合规范的字符为连字符
			storageClassName = strings.ReplaceAll(storageClassName, "_", "-")
			// 确保以字母或数字开头和结尾
			if !strings.HasPrefix(storageClassName, "efs-") {
				storageClassName = "efs-" + storageClassName
			}
		}

		// 创建 StorageClass 使用 map 结构
		storageClassObj := map[string]interface{}{
			"apiVersion": "storage.k8s.io/v1",
			"kind": "StorageClass",
			"metadata": map[string]interface{}{
				"name": storageClassName,
			},
			"provisioner": "efs.csi.aws.com",
			"parameters": map[string]interface{}{
				"fileSystemId": efsFileSystemId,
				"provisioningMode": "efs-ap",  // 使用 EFS 访问点模式
				"directoryPerms": "700",  // 目录权限
			},
			"reclaimPolicy": "Delete",
			"volumeBindingMode": "Immediate",
		}

		// 添加 StorageClass 清单
		scManifest := cluster.AddManifest(jsii.String(fmt.Sprintf("%s-storage-class", storageClassName)), &storageClassObj)

		// 确保 StorageClass 在 CSI 驱动部署后创建
		scManifest.Node().AddDependency(csiChart)

		// 如果需要创建静态 PV
		if types.GetBoolValue(eksInstance.CreateStaticPV, false) {
			pvName := fmt.Sprintf("%s-pv", storageClassName)
			pvObj := map[string]interface{}{
				"apiVersion": "v1",
				"kind": "PersistentVolume",
				"metadata": map[string]interface{}{
					"name": pvName,
				},
				"spec": map[string]interface{}{
					"capacity": map[string]interface{}{
						"storage": "5Gi",  // EFS 是弹性的，这个值只是象征性的
					},
					"volumeMode": "Filesystem",
					"accessModes": []string{"ReadWriteMany"},
					"persistentVolumeReclaimPolicy": "Retain",
					"storageClassName": storageClassName,
					"csi": map[string]interface{}{
						"driver": "efs.csi.aws.com",
						"volumeHandle": efsFileSystemId,
					},
				},
			}

			// 添加 PV 清单
			pvManifest := cluster.AddManifest(jsii.String(pvName), &pvObj)

			// 确保 PV 在 StorageClass 创建后创建
			pvManifest.Node().AddDependency(scManifest)

			// 如果需要创建默认 PVC
			if types.GetBoolValue(eksInstance.CreateDefaultPVC, false) {
				// 设置默认命名空间
				pvcNamespace := eksInstance.DefaultPVCNamespace
				if pvcNamespace == "" {
					pvcNamespace = "default"
				}

				pvcName := fmt.Sprintf("%s-pvc", storageClassName)
				pvcObj := map[string]interface{}{
					"apiVersion": "v1",
					"kind": "PersistentVolumeClaim",
					"metadata": map[string]interface{}{
						"name": pvcName,
						"namespace": pvcNamespace,
					},
					"spec": map[string]interface{}{
						"accessModes": []string{"ReadWriteMany"},
						"storageClassName": storageClassName,
						"resources": map[string]interface{}{
							"requests": map[string]interface{}{
								"storage": "5Gi",  // EFS 是弹性的，这个值只是象征性的
							},
						},
						"volumeName": pvName,  // 显式指定要使用的 PV
					},
				}

				// 添加 PVC 清单
				pvcManifest := cluster.AddManifest(jsii.String(fmt.Sprintf("%s-%s", pvcName, pvcNamespace)), &pvcObj)

				// 确保 PVC 在 PV 创建后创建
				pvcManifest.Node().AddDependency(pvManifest)
			}
		}
	}

	return csiChart
}

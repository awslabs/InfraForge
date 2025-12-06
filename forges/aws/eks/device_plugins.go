// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package eks

import (
	"fmt"
	
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awseks"
	"github.com/aws/jsii-runtime-go"
	"gopkg.in/yaml.v3"
)

// deployEfaDevicePlugin 部署EFA Kubernetes Device Plugin
func deployEfaDevicePlugin(stack awscdk.Stack, cluster awseks.Cluster, version string) awseks.HelmChart {
	// 如果未指定版本，使用默认版本0.5.19
	if version == "" {
		version = "0.5.19"
	}

	// 为Helm Chart添加v前缀（因为Helm Chart的tag需要v前缀）
	helmVersion := "v" + version

	// 部署EFA Device Plugin Helm Chart
	efaChart := cluster.AddHelmChart(jsii.String("aws-efa-k8s-device-plugin"), &awseks.HelmChartOptions{
		Chart:      jsii.String("aws-efa-k8s-device-plugin"),
		Repository: jsii.String("https://aws.github.io/eks-charts"),
		Namespace:  jsii.String("kube-system"),
		Version:    jsii.String(helmVersion),
		Values: &map[string]interface{}{
			"tolerations": []map[string]interface{}{
				{
					"key":      "nvidia.com/gpu",
					"operator": "Equal",
					"value":    "true",
					"effect":   "NoSchedule",
				},
				{
					"key":      "node.kubernetes.io/not-ready",
					"operator": "Exists",
					"effect":   "NoSchedule",
				},
			},
		},
	})

	return efaChart
}

// deployNvidiaDevicePlugin 部署 NVIDIA Device Plugin
func deployNvidiaDevicePlugin(stack awscdk.Stack, cluster awseks.Cluster, version string) awseks.KubernetesManifest {
	// 使用默认版本，如果未指定
	if version == "" {
		version = "0.18.1" // 默认版本，不带 v 前缀
	}
	
	// NVIDIA Device Plugin 的 YAML 清单
	nvidiaDevicePluginManifest := fmt.Sprintf(`
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: nvidia-device-plugin-daemonset
  namespace: kube-system
  labels:
    name: nvidia-device-plugin-ds
    app.kubernetes.io/version: %s
spec:
  selector:
    matchLabels:
      name: nvidia-device-plugin-ds
  updateStrategy:
    type: RollingUpdate
  template:
    metadata:
      labels:
        name: nvidia-device-plugin-ds
        app.kubernetes.io/version: %s
    spec:
      tolerations:
      - key: nvidia.com/gpu
        operator: Exists
        effect: NoSchedule
      - key: karpenter.sh/unregistered
        operator: Exists
      - key: node.kubernetes.io/not-ready
        operator: Exists
      priorityClassName: system-node-critical
      containers:
      - image: nvcr.io/nvidia/k8s-device-plugin:v%s
        name: nvidia-device-plugin-ctr
        args: ["--fail-on-init-error=false"]
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            drop: ["ALL"]
        volumeMounts:
          - name: device-plugin
            mountPath: /var/lib/kubelet/device-plugins
      volumes:
        - name: device-plugin
          hostPath:
            path: /var/lib/kubelet/device-plugins
`, version, version, version)

	// 解析 YAML 清单
	var manifests []*map[string]interface{}
	var manifest map[string]interface{}
	if err := yaml.Unmarshal([]byte(nvidiaDevicePluginManifest), &manifest); err != nil {
		panic(err)
	}
	manifestCopy := manifest
	manifests = append(manifests, &manifestCopy)

	// 创建 Kubernetes 清单资源
	return awseks.NewKubernetesManifest(stack, jsii.String("nvidia-device-plugin"), &awseks.KubernetesManifestProps{
		Cluster:   cluster,
		Manifest:  &manifests,
		Overwrite: jsii.Bool(false),
		Prune:     jsii.Bool(true),
	})
}


// deployNeuronDevicePlugin 部署 AWS Neuron Device Plugin (HyperPod 必需)
func deployNeuronDevicePlugin(stack awscdk.Stack, cluster awseks.Cluster, version string) awseks.KubernetesManifest {
	// 如果未指定版本，使用最新版本
	if version == "" {
		version = "2.28.27.0"
	}

	// AWS Neuron Device Plugin 的 YAML 清单（基于官方配置）
	neuronDevicePluginManifest := fmt.Sprintf(`
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: neuron-device-plugin-daemonset
  namespace: kube-system
spec:
  selector:
    matchLabels:
      name: neuron-device-plugin-ds
  updateStrategy:
    type: RollingUpdate
  template:
    metadata:
      labels:
        name: neuron-device-plugin-ds
    spec:
      tolerations:
      - key: aws.amazon.com/neuron
        operator: Exists
        effect: NoSchedule
      - key: "node.kubernetes.io/not-ready"
        operator: "Exists"
        effect: "NoExecute"
        tolerationSeconds: 300
      - key: "node.kubernetes.io/unreachable"
        operator: "Exists"
        effect: "NoExecute"
        tolerationSeconds: 300
      priorityClassName: "system-node-critical"
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
              - matchExpressions:
                  - key: "node.kubernetes.io/instance-type"
                    operator: In
                    values:
                      - inf1.xlarge
                      - inf1.2xlarge
                      - inf1.6xlarge
                      - inf1.24xlarge
                      - inf2.xlarge
                      - inf2.8xlarge
                      - inf2.24xlarge
                      - inf2.48xlarge
                      - trn1.2xlarge
                      - trn1.32xlarge
                      - trn1n.32xlarge
      containers:
      - image: public.ecr.aws/neuron/neuron-device-plugin:%s
        imagePullPolicy: Always
        name: neuron-device-plugin
        env:
        - name: KUBECONFIG
          value: /etc/kubernetes/kubelet.conf
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            drop: ["ALL"]
        volumeMounts:
        - name: device-plugin
          mountPath: /var/lib/kubelet/device-plugins
        - name: infa-map
          mountPath: /tmp
      volumes:
      - name: device-plugin
        hostPath:
          path: /var/lib/kubelet/device-plugins
      - name: infa-map
        hostPath:
          path: /tmp
`, version)

	// 解析 YAML 清单
	var manifests []*map[string]interface{}
	var manifest map[string]interface{}
	if err := yaml.Unmarshal([]byte(neuronDevicePluginManifest), &manifest); err != nil {
		panic(err)
	}
	manifestCopy := manifest
	manifests = append(manifests, &manifestCopy)

	// 创建 Kubernetes 清单资源
	return awseks.NewKubernetesManifest(stack, jsii.String("neuron-device-plugin"), &awseks.KubernetesManifestProps{
		Cluster:   cluster,
		Manifest:  &manifests,
		Overwrite: jsii.Bool(false),
		Prune:     jsii.Bool(true),
	})
}

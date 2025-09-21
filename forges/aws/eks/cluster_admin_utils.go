// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package eks

import (
	"github.com/aws/aws-cdk-go/awscdk/v2/awseks"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

// createEksAdminResources 创建具有 cluster-admin 权限的 ServiceAccount 和 ClusterRoleBinding
func createEksAdminResources(scope constructs.Construct, id string, cluster awseks.Cluster, saName string) (awseks.KubernetesManifest, awseks.KubernetesManifest) {
	// 创建 ServiceAccount
	sa := awseks.NewKubernetesManifest(scope, jsii.String(id+"SA"), &awseks.KubernetesManifestProps{
		Cluster: cluster,
		Manifest: &[]*map[string]interface{}{
			{
				"apiVersion": "v1",
				"kind":       "ServiceAccount",
				"metadata": map[string]interface{}{
					"name":      saName,
					"namespace": "kube-system",
				},
			},
		},
	})

	// 创建 ClusterRoleBinding
	crb := awseks.NewKubernetesManifest(scope, jsii.String(id+"CRB"), &awseks.KubernetesManifestProps{
		Cluster: cluster,
		Manifest: &[]*map[string]interface{}{
			{
				"apiVersion": "rbac.authorization.k8s.io/v1",
				"kind":       "ClusterRoleBinding",
				"metadata": map[string]interface{}{
					"name": saName,
				},
				"roleRef": map[string]interface{}{
					"apiGroup": "rbac.authorization.k8s.io",
					"kind":     "ClusterRole",
					"name":     "cluster-admin",
				},
				"subjects": []*map[string]interface{}{
					{
						"kind":      "ServiceAccount",
						"name":      saName,
						"namespace": "kube-system",
					},
				},
			},
		},
	})

	return sa, crb
}

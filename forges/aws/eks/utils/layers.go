// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package utils

import (
    "github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
    "github.com/aws/constructs-go/constructs/v10"
    "github.com/aws/jsii-runtime-go"

    kubectlv33 "github.com/cdklabs/awscdk-kubectl-go/kubectlv33/v2"
    kubectlv34 "github.com/cdklabs/awscdk-kubectl-go/kubectlv34/v2"
    kubectlv35 "github.com/cdklabs/awscdk-kubectl-go/kubectlv35/v2"
)

// GetKubectlLayer 根据 EKS 版本返回合适的 KubectlLayer
func GetKubectlLayer(scope constructs.Construct, id string, eksVersion string) awslambda.ILayerVersion {
    switch eksVersion {
    case "1.33":
        return kubectlv33.NewKubectlV33Layer(scope, jsii.String(id))
    case "1.34":
        return kubectlv34.NewKubectlV34Layer(scope, jsii.String(id))
    case "1.35", "latest":
        return kubectlv35.NewKubectlV35Layer(scope, jsii.String(id))
    default:
        return kubectlv35.NewKubectlV35Layer(scope, jsii.String(id))
    }
}


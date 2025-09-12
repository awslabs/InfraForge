// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package utils

import (
    "github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
    "github.com/aws/constructs-go/constructs/v10"
    "github.com/aws/jsii-runtime-go"
    
    // 使用正确的导入路径
    kubectlv21 "github.com/cdklabs/awscdk-kubectl-go/kubectlv21/v2"
    kubectlv22 "github.com/cdklabs/awscdk-kubectl-go/kubectlv22/v2"
    kubectlv23 "github.com/cdklabs/awscdk-kubectl-go/kubectlv23/v2"
    kubectlv24 "github.com/cdklabs/awscdk-kubectl-go/kubectlv24/v2"
    kubectlv25 "github.com/cdklabs/awscdk-kubectl-go/kubectlv25/v2"
    kubectlv26 "github.com/cdklabs/awscdk-kubectl-go/kubectlv26/v2"
    kubectlv27 "github.com/cdklabs/awscdk-kubectl-go/kubectlv27/v2"
    kubectlv28 "github.com/cdklabs/awscdk-kubectl-go/kubectlv28/v2"
    kubectlv29 "github.com/cdklabs/awscdk-kubectl-go/kubectlv29/v2"
    kubectlv30 "github.com/cdklabs/awscdk-kubectl-go/kubectlv30/v2"
    kubectlv31 "github.com/cdklabs/awscdk-kubectl-go/kubectlv31/v2"
    kubectlv32 "github.com/cdklabs/awscdk-kubectl-go/kubectlv32/v2"
    kubectlv33 "github.com/cdklabs/awscdk-kubectl-go/kubectlv33/v2"
)

// GetKubectlLayer 根据 EKS 版本返回合适的 KubectlLayer
func GetKubectlLayer(scope constructs.Construct, id string, eksVersion string) awslambda.ILayerVersion {
    // 根据版本选择合适的 KubectlLayer
    switch eksVersion {
    case "1.21":
        return kubectlv21.NewKubectlLayer(scope, jsii.String(id))
    case "1.22":
        return kubectlv22.NewKubectlV22Layer(scope, jsii.String(id))
    case "1.23":
        return kubectlv23.NewKubectlV23Layer(scope, jsii.String(id))
    case "1.24":
        return kubectlv24.NewKubectlV24Layer(scope, jsii.String(id))
    case "1.25":
        return kubectlv25.NewKubectlV25Layer(scope, jsii.String(id))
    case "1.26":
        return kubectlv26.NewKubectlV26Layer(scope, jsii.String(id))
    case "1.27":
        return kubectlv27.NewKubectlV27Layer(scope, jsii.String(id))
    case "1.28":
        return kubectlv28.NewKubectlV28Layer(scope, jsii.String(id))
    case "1.29":
        return kubectlv29.NewKubectlV29Layer(scope, jsii.String(id))
    case "1.30":
        return kubectlv30.NewKubectlV30Layer(scope, jsii.String(id))
    case "1.31":
        return kubectlv31.NewKubectlV31Layer(scope, jsii.String(id))
    case "1.32":
        return kubectlv32.NewKubectlV32Layer(scope, jsii.String(id))
    case "1.33", "latest":
        return kubectlv33.NewKubectlV33Layer(scope, jsii.String(id))
    default:
        // 默认使用最新版本
        return kubectlv33.NewKubectlV33Layer(scope, jsii.String(id))
    }
}


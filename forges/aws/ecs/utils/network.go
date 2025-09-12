// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"strings"
	
	"github.com/aws/aws-cdk-go/awscdk/v2/awsecs"
)

func ParseNetworkMode(input string) awsecs.NetworkMode{
    // 将输入转换为小写
    lowered := strings.ToLower(input)

    // 移除所有非字母数字字符
    cleaned := strings.ReplaceAll(lowered, "_", "")
    cleaned = strings.ReplaceAll(cleaned, "-", "")

    // 将已知的变体转换为目标格式
    switch cleaned {
    case "awsvpc":
        return awsecs.NetworkMode_AWS_VPC
    case "bridge":
        return awsecs.NetworkMode_BRIDGE
    case "nat":
        return awsecs.NetworkMode_NAT
    case "none":
        return awsecs.NetworkMode_NONE
    case "host":
        return awsecs.NetworkMode_HOST
    default:
        return awsecs.NetworkMode_BRIDGE
    }
}

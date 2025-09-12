// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"strings"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsfsx"
)

func ParseLustreVersion(input string) awsfsx.FileSystemTypeVersion {
        // 将输入转换为小写
        lowered := strings.ToLower(input)

        // 移除所有非字母数字字符
        cleaned := strings.ReplaceAll(lowered, "_", "")
        cleaned = strings.ReplaceAll(cleaned, "-", "")

        // 将已知的变体转换为目标格式
        switch cleaned {
        case "2.10":
                return awsfsx.FileSystemTypeVersion_V_2_10
        case "2.12":
                return awsfsx.FileSystemTypeVersion_V_2_12
        case "2.15":
                return awsfsx.FileSystemTypeVersion_V_2_15
        default:
                return awsfsx.FileSystemTypeVersion_V_2_15
        }
}



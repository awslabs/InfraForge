// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"regexp"
	"strings"
	"strconv"

	"github.com/aws/aws-cdk-go/awscdk/v2/awsautoscaling"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/jsii-runtime-go"
)

// ParseInstanceTypeOverrides 解析实例类型字符串为 LaunchTemplateOverrides 数组
// instanceTypesStr 格式为: "type1:weight1,type2:weight2" 
// 例如: "c7g.xlarge:2,c7g.2xlarge:1,c7g.4xlarge:1"
func ParseInstanceTypeOverrides(instanceTypesStr string) []*awsautoscaling.LaunchTemplateOverrides {
	instanceConfigs := strings.Split(instanceTypesStr, ",")
	overrides := make([]*awsautoscaling.LaunchTemplateOverrides, 0, len(instanceConfigs))

	for _, config := range instanceConfigs {
		parts := strings.Split(strings.TrimSpace(config), ":")
		if len(parts) != 2 {
			continue
		}

		weight, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			continue
		}

		override := &awsautoscaling.LaunchTemplateOverrides{
			InstanceType: awsec2.NewInstanceType(jsii.String(parts[0])),
			WeightedCapacity: jsii.Number(weight),
		}
		overrides = append(overrides, override)
	}

	return overrides
}

func ParsePlacementGroupStrategy(input string) awsec2.PlacementGroupStrategy{
	// 将输入转换为小写
	lowered := strings.ToLower(input)

	// 移除所有非字母数字字符
	cleaned := strings.ReplaceAll(lowered, "_", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")

	// 将已知的变体转换为目标格式
	switch cleaned {
	case "cluster":
		return awsec2.PlacementGroupStrategy_CLUSTER
	case "spread":
		return awsec2.PlacementGroupStrategy_SPREAD
	case "partition":
		return awsec2.PlacementGroupStrategy_PARTITION
	default:
		return awsec2.PlacementGroupStrategy_SPREAD
	}
}

func GetOriginalID(id string) string {
	return regexp.MustCompile(`\.\d+$`).ReplaceAllString(id, "")
}

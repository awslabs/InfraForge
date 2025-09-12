// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"fmt"
	"strings"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/jsii-runtime-go"
)

// EbsConfig 包含所有 EBS 相关的配置参数
type EbsConfig struct {
    VolumeType    string  // EBS 卷类型
    Iops          int     // IOPS
    Size          int     // 卷大小
    Throughput    int     // 吞吐量
    Optimized     bool    // 是否启用 EBS 优化
    DeviceName    string  // 设备名称
}

func CreateEbsBlockDevices(config *EbsConfig) ([]*awsec2.BlockDevice, error) {
    // 参数验证
    if config == nil {
        return nil, fmt.Errorf("EbsConfig cannot be nil")
    }

    // 解析 EBS 卷类型
    ebsVolumeType := parseEbsVolumeType(config.VolumeType)

    // 复制配置以避免修改原始值
    throughput := config.Throughput
    iops := config.Iops

    // GP3 以外的类型不支持 Throughput
    if ebsVolumeType != awsec2.EbsDeviceVolumeType_GP3 {
        throughput = 0
    }

    // 只有 GP3、IO1 和 IO2 支持 IOPS
    if ebsVolumeType != awsec2.EbsDeviceVolumeType_GP3 &&
       ebsVolumeType != awsec2.EbsDeviceVolumeType_IO1 &&
       ebsVolumeType != awsec2.EbsDeviceVolumeType_IO2 {
        iops = 0
    }

    // 创建 EBS 选项
    ebsOptions := &awsec2.EbsDeviceOptions{
        VolumeType: ebsVolumeType,
        Throughput: jsii.Number(throughput),
        Iops:      jsii.Number(iops),
    }

    // 创建块设备
    blockDevice := &awsec2.BlockDevice{
        DeviceName:     jsii.String(config.DeviceName),
        Volume:         awsec2.BlockDeviceVolume_Ebs(jsii.Number(config.Size), ebsOptions),
        MappingEnabled: jsii.Bool(false),
    }


    return []*awsec2.BlockDevice{blockDevice}, nil
}


func parseEbsVolumeType(input string) awsec2.EbsDeviceVolumeType {
        // 将输入转换为小写
        lowered := strings.ToLower(input)

        // 移除所有非字母数字字符
        cleaned := strings.ReplaceAll(lowered, "_", "")
        cleaned = strings.ReplaceAll(cleaned, "-", "")

        // 将已知的变体转换为目标格式
        switch cleaned {
        case "io1":
                return awsec2.EbsDeviceVolumeType_IO1
        case "io2":
                return awsec2.EbsDeviceVolumeType_IO2
        case "gp2":
                return awsec2.EbsDeviceVolumeType_GP2
        case "gp3":
                return awsec2.EbsDeviceVolumeType_GP3
        case "st1":
                return awsec2.EbsDeviceVolumeType_ST1
        case "sc1":
                return awsec2.EbsDeviceVolumeType_SC1
        default:
                return awsec2.EbsDeviceVolumeType_GP3
        }
}



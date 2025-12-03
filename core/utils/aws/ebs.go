// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"fmt"
	"strconv"
	"strings"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/jsii-runtime-go"
)

// EbsConfig 包含所有 EBS 相关的配置参数（支持多磁盘）
type EbsConfig struct {
	VolumeTypes  string // 逗号分隔的卷类型，如 "gp3,io2,gp3"
	Iops         string // 逗号分隔的 IOPS，如 "3000,16000,3000"
	Sizes        string // 逗号分隔的大小，如 "100,512,200"
	Throughputs  string // 逗号分隔的吞吐量，如 "125,1000,125"
	Optimized    bool   // 是否启用 EBS 优化
	RootDevice   string // Root 设备名称
}

// CreateEbsBlockDevices 创建多个 EBS 块设备
func CreateEbsBlockDevices(config *EbsConfig) ([]*awsec2.BlockDevice, error) {
	if config == nil {
		return nil, fmt.Errorf("EbsConfig cannot be nil")
	}

	// 解析配置
	volumeTypes := parseStringList(config.VolumeTypes, "gp3")
	sizes := parseIntList(config.Sizes, 30)
	iops := parseIntList(config.Iops, 3000)
	throughputs := parseIntList(config.Throughputs, 125)

	// 确定磁盘数量（取最大长度）
	diskCount := max(len(volumeTypes), len(sizes), len(iops), len(throughputs))
	if diskCount == 0 {
		diskCount = 1
	}

	// 生成设备名
	deviceNames := generateDeviceNames(config.RootDevice, diskCount)

	// 创建块设备数组
	blockDevices := make([]*awsec2.BlockDevice, diskCount)

	for i := 0; i < diskCount; i++ {
		volType := getStringValue(volumeTypes, i, "gp3")
		size := getIntValue(sizes, i, 30)
		iop := getIntValue(iops, i, 3000)
		throughput := getIntValue(throughputs, i, 125)

		blockDevices[i] = createSingleBlockDevice(volType, size, iop, throughput, deviceNames[i])
	}

	return blockDevices, nil
}

// createSingleBlockDevice 创建单个块设备
func createSingleBlockDevice(volumeType string, size, iops, throughput int, deviceName string) *awsec2.BlockDevice {
	ebsVolumeType := parseEbsVolumeType(volumeType)

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

	ebsOptions := &awsec2.EbsDeviceOptions{
		VolumeType: ebsVolumeType,
		Throughput: jsii.Number(throughput),
		Iops:       jsii.Number(iops),
	}

	return &awsec2.BlockDevice{
		DeviceName:     jsii.String(deviceName),
		Volume:         awsec2.BlockDeviceVolume_Ebs(jsii.Number(size), ebsOptions),
		MappingEnabled: jsii.Bool(false),
	}
}

// NeedsLaunchTemplateForThroughput 检查是否需要 LaunchTemplate（GP3 且 throughput > 125）
func NeedsLaunchTemplateForThroughput(volumeTypes, throughputs string) bool {
	volTypes := parseStringList(volumeTypes, "gp3")
	tputs := parseIntList(throughputs, 125)

	for i := 0; i < len(volTypes) && i < len(tputs); i++ {
		if strings.ToLower(strings.TrimSpace(volTypes[i])) == "gp3" && tputs[i] > 125 {
			return true
		}
	}
	return false
}

// CreateLaunchTemplateBlockDeviceMappings 为 LaunchTemplate 创建 BlockDeviceMappings
func CreateLaunchTemplateBlockDeviceMappings(volumeTypes, throughputs, rootDevice string) []interface{} {
	volTypes := parseStringList(volumeTypes, "gp3")
	tputs := parseIntList(throughputs, 125)
	
	diskCount := max(len(volTypes), len(tputs))
	if diskCount == 0 {
		return []interface{}{}
	}

	deviceNames := generateDeviceNames(rootDevice, diskCount)
	mappings := make([]interface{}, 0)

	for i := 0; i < diskCount; i++ {
		volType := getStringValue(volTypes, i, "gp3")
		throughput := getIntValue(tputs, i, 125)

		// 只为 GP3 且 throughput > 125 的磁盘创建映射
		if strings.ToLower(strings.TrimSpace(volType)) == "gp3" && throughput > 125 {
			mappings = append(mappings, &awsec2.CfnLaunchTemplate_BlockDeviceMappingProperty{
				DeviceName: jsii.String(deviceNames[i]),
				Ebs: &awsec2.CfnLaunchTemplate_EbsProperty{
					Throughput: jsii.Number(throughput),
				},
			})
		}
	}

	return mappings
}

// parseStringList 解析逗号分隔的字符串
func parseStringList(input, defaultValue string) []string {
	if input == "" {
		return []string{}
	}

	parts := strings.Split(input, ",")
	result := make([]string, len(parts))

	for i, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed == "" {
			result[i] = defaultValue
		} else {
			result[i] = trimmed
		}
	}

	return result
}

// parseIntList 解析逗号分隔的整数字符串
func parseIntList(input string, defaultValue int) []int {
	if input == "" {
		return []int{}
	}

	parts := strings.Split(input, ",")
	result := make([]int, len(parts))

	for i, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed == "" {
			result[i] = defaultValue
		} else {
			if val, err := strconv.Atoi(trimmed); err == nil {
				result[i] = val
			} else {
				result[i] = defaultValue
			}
		}
	}

	return result
}

// generateDeviceNames 生成设备名数组
// 第一块：rootDevice (如 /dev/sda1 或 /dev/xvda)
// 第二块：/dev/sdb
// 第三块：/dev/sdc
// ...
func generateDeviceNames(rootDevice string, count int) []string {
	if count <= 1 {
		return []string{rootDevice}
	}

	names := make([]string, count)
	names[0] = rootDevice

	// 从 'b' 开始（第二块磁盘）
	for i := 1; i < count; i++ {
		names[i] = fmt.Sprintf("/dev/sd%c", 'b'+i-1)
	}

	return names
}

// getStringValue 安全获取字符串数组的值
func getStringValue(arr []string, index int, defaultValue string) string {
	if index < len(arr) {
		return arr[index]
	}
	if len(arr) > 0 {
		return arr[0]
	}
	return defaultValue
}

// getIntValue 安全获取整数数组的值
func getIntValue(arr []int, index int, defaultValue int) int {
	if index < len(arr) {
		return arr[index]
	}
	if len(arr) > 0 {
		return arr[0]
	}
	return defaultValue
}

// max 返回多个整数的最大值
func max(nums ...int) int {
	if len(nums) == 0 {
		return 0
	}
	maxVal := nums[0]
	for _, n := range nums[1:] {
		if n > maxVal {
			maxVal = n
		}
	}
	return maxVal
}

// parseEbsVolumeType 解析 EBS 卷类型
func parseEbsVolumeType(input string) awsec2.EbsDeviceVolumeType {
	if input == "" {
		return awsec2.EbsDeviceVolumeType_GP3
	}

	lowered := strings.ToLower(input)
	cleaned := strings.ReplaceAll(lowered, "_", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")

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

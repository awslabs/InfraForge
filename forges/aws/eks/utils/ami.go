// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"log"

	"github.com/aws/aws-cdk-go/awscdk/v2/awseks"
)

// SelectEksAmiType 根据提供的操作系统类型、架构、GPU类型和Windows类型选择适当的AMI类型
// 如果输入无效，将使用默认值：Linux、ARM64、Standard，并输出警告信息
// 所有参数都使用字符串类型，方便直接调用
func SelectEksAmiType(osType string, osArch string, gpuType string, windowsType string) awseks.NodegroupAmiType {
	// 定义有效的值
	validGpuTypes := map[string]bool{
		"standard": true,
		"nvidia":   true,
		"neuron":   true,
	}

	validOsArches := map[string]bool{
		"x86_64":  true,
		"amd64":   true, // 别名，等同于 x86_64
		"arm64":   true,
		"aarch64": true, // 别名，等同于 arm64
	}

	validOsTypes := map[string]bool{
		"bottlerocket": true,
		"windows":      true,
		"linux":        true,
	}

	validWindowsTypes := map[string]bool{
		"core_2019": true,
		"core_2022": true,
		"full_2019": true,
		"full_2022": true,
	}

	// 标准化架构名称
	if osArch == "amd64" {
		osArch = "x86_64"
	}
	if osArch == "aarch64" {
		osArch = "arm64"
	}
	if !validGpuTypes[gpuType] {
		log.Printf("WARNING: Unsupported GPU type '%s', using default 'standard'", gpuType)
		gpuType = "standard"
	}

	if !validOsArches[osArch] {
		log.Printf("WARNING: Unsupported architecture '%s', using default 'arm64'", osArch)
		osArch = "arm64"
	}

	if !validOsTypes[osType] {
		log.Printf("WARNING: Unsupported OS type '%s', using default 'linux'", osType)
		osType = "linux"
	}

	switch osType {
	case "bottlerocket":
		switch osArch {
		case "x86_64":
			switch gpuType {
			case "standard":
				return awseks.NodegroupAmiType_BOTTLEROCKET_X86_64
			case "nvidia":
				return awseks.NodegroupAmiType_BOTTLEROCKET_X86_64_NVIDIA
			default:
				log.Printf("WARNING: Unsupported GPU type '%s' for Bottlerocket x86_64, using standard", gpuType)
				return awseks.NodegroupAmiType_BOTTLEROCKET_X86_64
			}
		case "arm64":
			switch gpuType {
			case "standard":
				return awseks.NodegroupAmiType_BOTTLEROCKET_ARM_64
			case "nvidia":
				return awseks.NodegroupAmiType_BOTTLEROCKET_ARM_64_NVIDIA
			default:
				log.Printf("WARNING: Unsupported GPU type '%s' for Bottlerocket ARM64, using standard", gpuType)
				return awseks.NodegroupAmiType_BOTTLEROCKET_ARM_64
			}
		default:
			// 不应该到达这里，因为我们已经设置了默认值
			return awseks.NodegroupAmiType_BOTTLEROCKET_ARM_64
		}

	case "windows":
		// Windows 只支持 x86_64 架构
		if osArch != "x86_64" {
			log.Printf("WARNING: Windows only supports x86_64 architecture, not '%s'. Falling back to 'core_2022'", osArch)
			return awseks.NodegroupAmiType_WINDOWS_CORE_2022_X86_64
		}

		// Windows 不支持特殊 GPU 类型
		if gpuType != "standard" {
			log.Printf("WARNING: Windows doesn't support GPU type '%s'. Falling back to standard", gpuType)
		}

		// 如果未指定 Windows 类型或类型无效，默认使用 Core 2022
		if windowsType == "" || !validWindowsTypes[windowsType] {
			if windowsType != "" {
				log.Printf("WARNING: Unsupported Windows type '%s', using default 'core_2022'", windowsType)
			}
			windowsType = "core_2022"
		}

		switch windowsType {
		case "core_2019":
			return awseks.NodegroupAmiType_WINDOWS_CORE_2019_X86_64
		case "core_2022":
			return awseks.NodegroupAmiType_WINDOWS_CORE_2022_X86_64
		case "full_2019":
			return awseks.NodegroupAmiType_WINDOWS_FULL_2019_X86_64
		case "full_2022":
			return awseks.NodegroupAmiType_WINDOWS_FULL_2022_X86_64
		default:
			// 不应该到达这里，因为我们已经验证了windowsType
			return awseks.NodegroupAmiType_WINDOWS_CORE_2022_X86_64
		}

	case "linux": // OsTypeLinux 默认使用 AL2023
		switch osArch {
		case "x86_64":
			switch gpuType {
			case "standard":
				return awseks.NodegroupAmiType_AL2023_X86_64_STANDARD
			case "nvidia":
				return awseks.NodegroupAmiType_AL2023_X86_64_NVIDIA
			case "neuron":
				return awseks.NodegroupAmiType_AL2023_X86_64_NEURON
			default:
				log.Printf("WARNING: Unsupported GPU type '%s' for Linux x86_64, using standard", gpuType)
				return awseks.NodegroupAmiType_AL2023_X86_64_STANDARD
			}
		case "arm64":
			if gpuType != "standard" {
				log.Printf("WARNING: Linux ARM64 only supports standard GPU type, not '%s'", gpuType)
			}
			return awseks.NodegroupAmiType_AL2023_ARM_64_STANDARD
		default:
			// 不应该到达这里，因为我们已经设置了默认值
			return awseks.NodegroupAmiType_AL2023_ARM_64_STANDARD
		}

	default:
		// 不应该到达这里，因为我们已经设置了默认值
		return awseks.NodegroupAmiType_AL2023_ARM_64_STANDARD
	}
}

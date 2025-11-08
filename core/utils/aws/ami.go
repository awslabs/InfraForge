// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"fmt"
	"errors"
	"context"
	"strings"
	"time"
	
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsecs"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

type ForgeAMIConfig struct {
	OsImage           string
	OsType            string
	UserDataToken     string
	UserDataScriptPath string
	MagicToken        string
	S3Location        string
}

func GetAMIInfo(partition, osType, osVersion, instanceArch string) (string, string) {
	amiOwners := map[string]map[string]string{
		"aws": {
			"amazon": "amazon",
			"ubuntu": "099720109477",
			"debian": "136693071363",
			"centos": "125523088429",
			"redhat": "309956199498",
			"suse":   "013907871322",
//			"rocky":  "792107900819",
			"rocky":  "679593333241",
			"windows": "801119661308",
		},
		"aws-cn": {
			"amazon": "amazon",
			"ubuntu": "837727238323",
			"debian": "336777782633",
			"redhat": "841258680906",
			"centos": "336777782633",
			"suse":   "841869936221",
			"rocky":  "336777782633",
			"windows": "016951021795",
		},
	}

	
	amiNames := map[string]map[string]map[string]map[string]string{
		"aws": {
			"amazon": {
				"2": {
					"aarch64": "amzn2-ami-kernel-5.10-hvm-*-arm64-gp2",
					"x86_64":  "amzn2-ami-kernel-5.10-hvm-*-x86_64-gp2",
				},
				"2023": {
					"aarch64": "al2023-ami-2023*-kernel-6.1-arm64",
					"x86_64":  "al2023-ami-2023*-kernel-6.1-x86_64",
				},
			},
			"ubuntu": {
				"18.04": {
					"aarch64": "ubuntu/images/hvm-ssd/ubuntu-bionic-18.04-arm64-server-*",
					"x86_64":  "ubuntu/images/hvm-ssd/ubuntu-bionic-18.04-amd64-server-*",
				},
				"20.04": {
					"aarch64": "ubuntu/images/hvm-ssd/ubuntu-focal-20.04-arm64-server-*",
					"x86_64":  "ubuntu/images/hvm-ssd/ubuntu-focal-20.04-amd64-server-*",
				},
				"22.04": {
					"aarch64": "ubuntu/images/hvm-ssd/ubuntu-jammy-22.04-arm64-server-*",
					"x86_64":  "ubuntu/images/hvm-ssd/ubuntu-jammy-22.04-amd64-server-*",
				},
				"24.04": {
					"aarch64": "ubuntu/images/hvm-ssd-gp3/ubuntu-noble-24.04-arm64-server-*",
					"x86_64":  "ubuntu/images/hvm-ssd-gp3/ubuntu-noble-24.04-amd64-server-*",
				},
			},
			"debian": {
				"10": {
					"aarch64": "debian-10-arm64-*",
					"x86_64":  "debian-10-amd64-*",
				},
				"11": {
					"aarch64": "debian-11-arm64-*",
					"x86_64":  "debian-11-amd64-*",
				},
				"12": {
					"aarch64": "debian-12-arm64-*",
					"x86_64":  "debian-12-amd64-*",
				},
				"13": {
					"aarch64": "debian-13-arm64-*",
					"x86_64":  "debian-13-amd64-*",
				},
			},
			"centos": {
				"9": {
					"aarch64": "CentOS Stream 9 aarch64 *",
					"x86_64":  "CentOS Stream 9 x86_64 *",
				},
				"10": {
					"aarch64": "CentOS Stream 10 aarch64 *",
					"x86_64":  "CentOS Stream 10 x86_64 *",
				},
			},
			"suse": {
				"12": {
					"x86_64": "suse-sles-12-sp5-*-hvm-ssd-x86_64",
				},
				"15": {
					"aarch64": "suse-sles-15-sp5-*-hvm-ssd-arm64",
					"x86_64":  "suse-sles-15-sp5-*-hvm-ssd-x86_64",
				},
				"16": {
					"aarch64": "suse-sles-16-0-*-hvm-ssd-arm64",
					"x86_64":  "suse-sles-16-0-*-hvm-ssd-x86_64",
				},
			},
			"rocky": {
				"8": {
					"x86_64": "Rocky-8-EC2-LVM-8*.x86_64*",
					"aarch64": "Rocky-8-EC2-LVM-8*.aarch64*",
				},
				"9": {
					"x86_64": "Rocky-9-EC2-LVM-9*.x86_64*",
					"aarch64": "Rocky-9-EC2-LVM-9*.aarch64*",
				},
				"10": {
					"x86_64": "Rocky-10-EC2-Base-10*.x86_64*",
					"aarch64": "Rocky-10-EC2-Base-10*.aarch64*",
				},
			},
			"windows": {
				"2025": {
					"x86_64": "Windows_Server-2025-English-Full-Base*",
				},
				"2022": {
					"x86_64": "Windows_Server-2022-English-Full-Base*",
				},
				"2019": {
					"x86_64": "Windows_Server-2019-English-Full-Base*",
				},
				"2016": {
					"x86_64": "Windows_Server-2019-English-Full-Base*",
				},
			},
		},
		"aws-cn": {
			"amazon": {
				"2": {
					"aarch64": "amzn2-ami-kernel-5.10-hvm-*-arm64-gp2",
					"x86_64":  "amzn2-ami-kernel-5.10-hvm-*-x86_64-gp2",
				},
				"2023": {
					"aarch64": "al2023-ami-2023*-kernel-6.1-arm64",
					"x86_64":  "al2023-ami-2023*-kernel-6.1-x86_64",
				},
			},
			"ubuntu": {
				"18.04": {
					"aarch64": "ubuntu-pro-server/images/hvm-ssd/ubuntu-bionic-18.04-arm64-pro-server-*",
					"x86_64":  "ubuntu-pro-server/images/hvm-ssd/ubuntu-bionic-18.04-amd64-pro-server-*",
				},
				"20.04": {
					"aarch64": "ubuntu/images/hvm-ssd/ubuntu-focal-20.04-arm64-server-*",
					"x86_64":  "ubuntu/images/hvm-ssd/ubuntu-focal-20.04-amd64-server-*",
				},
				"22.04": {
					"aarch64": "ubuntu/images/hvm-ssd/ubuntu-jammy-22.04-arm64-server-*",
					"x86_64":  "ubuntu/images/hvm-ssd/ubuntu-jammy-22.04-amd64-server-*",
				},
				"24.04": {
					"aarch64": "ubuntu/images/hvm-ssd-gp3/ubuntu-noble-24.04-arm64-server-*",
					"x86_64":  "ubuntu/images/hvm-ssd-gp3/ubuntu-noble-24.04-amd64-server-*",
				},
			},
			"debian": {
				"10": {
					"x86_64":  "debian-10-final-*",
				},
				"11": {
					"x86_64":  "debian-11-final-*",
				},
				"12": {
					"x86_64":  "debian-12-final-*",
				},
			},
			"suse": {
				"12": {
					"x86_64": "suse-sles-12-sp5-*-hvm-ssd-x86_64",
				},
				"15": {
					"aarch64": "suse-sles-15-sp5-*-hvm-ssd-arm64",
					"x86_64":  "suse-sles-15-sp5-*-hvm-ssd-x86_64",
				},
				"16": {
					"aarch64": "suse-sles-16-0-*-hvm-ssd-arm64",
					"x86_64":  "suse-sles-16-0-*-hvm-ssd-x86_64",
				},
			},
			"rocky": {
				"8": {
					"x86_64": "Rocky8-final-*",
				},
				"9": {
					"x86_64": "Rocky9-final-*",
				},
			},
			"windows": {
				"2025": {
					"x86_64": "Windows_Server-2025-Chinese_Simplified-Full-Base-*",
				},
				"2022": {
					"x86_64": "Windows_Server-2022-Chinese_Simplified-Full-Base-*",
				},
				"2019": {
					"x86_64": "Windows_Server-2019-Chinese_Simplified-Full-Base-*",
				},
				"2016": {
					"x86_64": "Windows_Server-2016-Chinese_Simplified-Full-Base-*",
				},
			},
		},
	}

	amiOwner, ok := amiOwners[partition][osType]
	if !ok {
		return "", ""
	}


	amiName, ok := amiNames[partition][osType][osVersion][instanceArch]
	if !ok {
		return "", ""
	}

	return amiOwner, amiName

}

type ForgeAMILookup struct {
	AmiOwner  string
	AmiName    string
	AmiArch    string
}

func (l *ForgeAMILookup) FindAMI() (osImage string, err error) {

	// 加载 AWS 配置
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return "", fmt.Errorf("failed to load AWS configuration: %w", err)
	}

	// 创建 EC2 服务客户端
	ec2Client := ec2.NewFromConfig(cfg)

	input := &ec2.DescribeImagesInput{
		Owners: []string{l.AmiOwner},
		Filters: []types.Filter{
			{
				Name:   aws.String("name"),
				Values: []string{l.AmiName},
			},
			{
				Name:   aws.String("architecture"),
				Values: []string{l.AmiArch},
			},
			{
				Name:   aws.String("state"),
				Values: []string{"available"},
			},
		},
	}
	// 发送查询请求
	result, err := ec2Client.DescribeImages(context.TODO(), input)
	if err != nil {
		return "", nil
	}

	// 找到最新的 AMI
	var latestAmi *types.Image
	latestTime := time.Time{}
	for _, image := range result.Images {
		if image.CreationDate != nil {
			creationDate, err := time.Parse(time.RFC3339, *image.CreationDate)
			if err == nil && creationDate.After(latestTime) {
				latestTime = creationDate
				latestAmi = &image
			}
		}
	}

	// 返回最新 AMI ID
	if latestAmi != nil {
		return *latestAmi.ImageId, nil
	} else {
		return "", nil
	}

}

// DescribeAMI 函数用于描述给定的 AMI 并返回其根设备名称
func DescribeAMI(AMIID string) (string, error) {
	// 加载 AWS 配置
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return "", fmt.Errorf("failed to load AWS configuration: %w", err)
	}

	// 创建 EC2 服务客户端
	ec2Client := ec2.NewFromConfig(cfg)

	input := &ec2.DescribeImagesInput{
		ImageIds: []string{AMIID},
	}
	result, err := ec2Client.DescribeImages(context.TODO(), input)
	if err != nil {
		return "", err
	}
	if len(result.Images) == 0 {
		return "", errors.New("AMI not found")
	}
	return *result.Images[0].RootDeviceName, nil
}

func (f *ForgeAMIConfig) GetImage(constructs.Construct) *awsec2.MachineImageConfig {
	// 根据操作系统名称设置 OsType
	
	var instanceOsType awsec2.OperatingSystemType

	var scriptPath string

	switch strings.ToLower(string(f.OsType)) {
	case "windows":
		instanceOsType = awsec2.OperatingSystemType_WINDOWS
	case "linux":
		instanceOsType = awsec2.OperatingSystemType_LINUX
	case "unknown":
		instanceOsType = awsec2.OperatingSystemType_UNKNOWN
	default:
		instanceOsType = awsec2.OperatingSystemType_LINUX
	}

	amiID := f.OsImage

	switch instanceOsType {
	case awsec2.OperatingSystemType_LINUX:
		scriptPath = "./userdata.sh"
	case awsec2.OperatingSystemType_WINDOWS:
		scriptPath = "./userdata.ps1"
	default:
		scriptPath = "./userdata.sh"
	}

	userDataGenerator := &UserDataGenerator{
		OsType:             instanceOsType,
		ScriptPath:         scriptPath,
		UserDataToken:      f.UserDataToken,
		UserDataScriptPath: f.UserDataScriptPath,
		MagicToken:         f.MagicToken,
		S3Location:         f.S3Location,
	}

	userData, err := userDataGenerator.GenerateUserData()
	if err != nil {
		// 处理错误
		fmt.Errorf("Generate user data: %v", err)
	}


	return &awsec2.MachineImageConfig{
		ImageId:  &amiID,
		OsType:   instanceOsType,
		UserData: userData,
	}
}


func ParseAmiHardwareType(input string) awsecs.AmiHardwareType {
    // 将输入转换为小写
    lowered := strings.ToLower(input)

    // 移除所有非字母数字字符
    cleaned := strings.ReplaceAll(lowered, "_", "")
    cleaned = strings.ReplaceAll(cleaned, "-", "")

    // 将已知的变体转换为目标格式
    switch cleaned {
    case "gpu":
        return awsecs.AmiHardwareType_GPU
    case "arm", "arm64", "aarch64":
        return awsecs.AmiHardwareType_ARM
    case "neuron":
        return awsecs.AmiHardwareType_NEURON
    case "amd64", "x8664", "x86", "x64", "standard":
        return awsecs.AmiHardwareType_STANDARD
    default:
        return awsecs.AmiHardwareType_ARM
    }
}

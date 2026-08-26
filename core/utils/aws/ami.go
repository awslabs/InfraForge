// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"fmt"
	"errors"
	"context"
	"regexp"
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
			"rocky":  "792107900819",
//			"rocky":  "679593333241",
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
					"aarch64": "al2023-ami-2023*-kernel-*-arm64",
					"x86_64":  "al2023-ami-2023*-kernel-*-x86_64",
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
				"26.04": {
					"aarch64": "ubuntu/images/hvm-ssd-gp3/ubuntu-resolute-26.04-arm64-server-*",
					"x86_64":  "ubuntu/images/hvm-ssd-gp3/ubuntu-resolute-26.04-amd64-server-*",
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
				"7": {
					"aarch64": "CentOS Linux 7 aarch64 *",
					"x86_64":  "CentOS Linux 7 x86_64 *",
				},
				"9": {
					"aarch64": "CentOS Stream 9 aarch64 *",
					"x86_64":  "CentOS Stream 9 x86_64 *",
				},
				"10": {
					"aarch64": "CentOS Stream 10 aarch64 *",
					"x86_64":  "CentOS Stream 10 x86_64 *",
				},
			},
			// Red Hat 官方镜像（owner 309956199498）。只有 Hourly2（含订阅、按小时计费）
			// 是公开发布的；Access2/BYOS 不作为公共 AMI 提供，因此这里不做区分。
			// RHEL_HA（高可用附加组件）用的是 "RHEL_HA-" 前缀，天然被下面的 "RHEL-" 排除。
			"redhat": {
				"7": {
					"x86_64": "RHEL-7*_HVM-*-x86_64-*-Hourly2-GP3",
				},
				"8": {
					"aarch64": "RHEL-8*_HVM-*-arm64-*-Hourly2-GP3",
					"x86_64":  "RHEL-8*_HVM-*-x86_64-*-Hourly2-GP3",
				},
				"9": {
					"aarch64": "RHEL-9*_HVM-*-arm64-*-Hourly2-GP3",
					"x86_64":  "RHEL-9*_HVM-*-x86_64-*-Hourly2-GP3",
				},
				"10": {
					"aarch64": "RHEL-10*_HVM-*-arm64-*-Hourly2-GP3",
					"x86_64":  "RHEL-10*_HVM-*-x86_64-*-Hourly2-GP3",
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
					"aarch64": "al2023-ami-2023*-kernel-*-arm64",
					"x86_64":  "al2023-ami-2023*-kernel-*-x86_64",
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
			"centos": {
				"7": {
					"x86_64": "centos7.5-*",
				},
				"8": {
					"x86_64": "CentOS-8-ec2-*",
				},
			},
			// 中国区 Red Hat 官方镜像（owner 841258680906）。实测 cn-north-1 / cn-northwest-1
			// 的镜像名与商业区完全一致（RHEL 8/9/10，x86_64 + arm64，全部 Hourly2-GP3），
			// 区别只有 owner 账号；中国区同样有一张 RHEL-7.9 x86_64。
			"redhat": {
				"7": {
					"x86_64": "RHEL-7*_HVM-*-x86_64-*-Hourly2-GP3",
				},
				"8": {
					"aarch64": "RHEL-8*_HVM-*-arm64-*-Hourly2-GP3",
					"x86_64":  "RHEL-8*_HVM-*-x86_64-*-Hourly2-GP3",
				},
				"9": {
					"aarch64": "RHEL-9*_HVM-*-arm64-*-Hourly2-GP3",
					"x86_64":  "RHEL-9*_HVM-*-x86_64-*-Hourly2-GP3",
				},
				"10": {
					"aarch64": "RHEL-10*_HVM-*-arm64-*-Hourly2-GP3",
					"x86_64":  "RHEL-10*_HVM-*-x86_64-*-Hourly2-GP3",
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
	AmiName   string
	AmiArch   string
	Region    string
	Profile   string
}

func (l *ForgeAMILookup) FindAMI() (osImage string, err error) {

	// 加载 AWS 配置
	var opts []func(*config.LoadOptions) error
	if l.Region != "" {
		opts = append(opts, config.WithRegion(l.Region))
	}
	if l.Profile != "" {
		opts = append(opts, config.WithSharedConfigProfile(l.Profile))
	}
	cfg, err := config.LoadDefaultConfig(context.TODO(), opts...)
	if err != nil {
		return "", fmt.Errorf("failed to load AWS configuration: %w", err)
	}

	// 创建 EC2 服务客户端
	ec2Client := ec2.NewFromConfig(cfg)

	input := &ec2.DescribeImagesInput{
		Owners: []string{l.AmiOwner},
		IncludeDeprecated: aws.Bool(true),
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
		// 之前这里返回 (nil error)，导致 AuthFailure / 无凭证 / 区域不对等真实错误
		// 被伪装成「没找到镜像」，排查时看不到原因。
		return "", fmt.Errorf("DescribeImages failed (owner=%s, name=%s, arch=%s): %w",
			l.AmiOwner, l.AmiName, l.AmiArch, err)
	}

	// 选出「主版本下的最新版本」：先比镜像名里的版本号，版本相同再比发布时间。
	//
	// 只按发布时间取最新是不够的：当名称模式跨小版本时（如 redhat 的
	// "RHEL-9*_HVM-*"），Red Hat 会回头重新发布旧的小版本（EUS 流），
	// 其发布时间反而比新小版本更晚。实测 2026-08：
	//   RHEL-9.8.0_HVM-20260728  发布于 2026-07-29
	//   RHEL-9.6.0_HVM-20260811  发布于 2026-08-11   ← 只按时间会选中它
	// 结果是「要 RHEL 9」拿到 9.6 而不是 9.8，而且随时会来回跳。
	//
	// 对于名称里已经写死版本的发行版（ubuntu 22.04、CentOS Stream 9、
	// Windows_Server-2022 等），所有候选的版本号相同，行为与只按时间取最新一致。
	var best *types.Image
	var bestVer []int
	var bestTime time.Time
	for i := range result.Images {
		image := &result.Images[i]
		if image.Name == nil {
			continue
		}
		ver := amiNameVersion(*image.Name)
		var t time.Time
		if image.CreationDate != nil {
			if parsed, perr := time.Parse(time.RFC3339, *image.CreationDate); perr == nil {
				t = parsed
			}
		}
		if best == nil {
			best, bestVer, bestTime = image, ver, t
			continue
		}
		switch cmpVersion(ver, bestVer) {
		case 1:
			best, bestVer, bestTime = image, ver, t
		case 0:
			if t.After(bestTime) {
				best, bestVer, bestTime = image, ver, t
			}
		}
	}

	if best != nil && best.ImageId != nil {
		return *best.ImageId, nil
	}
	return "", nil

}

// amiNameVersion 把镜像名里出现的所有数字按顺序取出来，用于版本比较：
//	RHEL-9.8.0_HVM-20260728-x86_64-0-Hourly2-GP3    -> [9 8 0 20260728 86 64 0 2 3]
//	RHEL-9.6.0_HVM-20260811-x86_64-0-Hourly2-GP3    -> [9 6 0 20260811 86 64 0 2 3]
//	                                                      ↑ 第二位就能分出 9.8 > 9.6
//
// 刻意不去「只取第一个版本号」：镜像名里第一个数字未必是版本号，例如
// ubuntu 的 hvm-ssd-gp3 会让 gp3 的 3 被误当成版本。逐位比较所有数字则天然稳健——
// 同一族镜像的名字结构一致，前面几位相同，后面自然落到构建日期那一位上，
// 效果等同于按发布时间排序（也就是改动前的行为）。
//
// 取不到数字时返回 nil，视为「无版本信息」，完全退化为按发布时间比较。
var amiVersionRe = regexp.MustCompile(`[0-9]+`)

func amiNameVersion(name string) []int {
	matches := amiVersionRe.FindAllString(name, -1)
	if len(matches) == 0 {
		return nil
	}
	nums := make([]int, 0, len(matches))
	for _, m := range matches {
		n := 0
		overflow := false
		for _, c := range m {
			n = n*10 + int(c-'0')
			if n > 1<<50 { // 防御异常长数字串
				overflow = true
				break
			}
		}
		if overflow {
			continue
		}
		nums = append(nums, n)
	}
	return nums
}

// cmpVersion 比较两个版本号切片：a>b 返回 1，a<b 返回 -1，相等返回 0。
// 长度不同时，缺少的位视为 0（8.10 > 8.8，9.6 < 9.6.1）。
func cmpVersion(a, b []int) int {
	n := len(a)
	if len(b) > n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		var x, y int
		if i < len(a) {
			x = a[i]
		}
		if i < len(b) {
			y = b[i]
		}
		if x > y {
			return 1
		}
		if x < y {
			return -1
		}
	}
	return 0
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

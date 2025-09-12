// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package vpc

import (
	"strings"

	"github.com/awslabs/InfraForge/core/config"
	"github.com/awslabs/InfraForge/core/interfaces"
	"github.com/awslabs/InfraForge/core/utils/aws"
	"github.com/awslabs/InfraForge/core/utils/types"
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/jsii-runtime-go"
)

type VpcInstanceConfig struct {
        config.BaseInstanceConfig
	VpcId		 string `json:"vpcId"`
	CidrBlock        string `json:"cidrBlock"`
	NatGatewayPerAZ  *bool  `json:"natGatewayPerAZ,omitempty"`
}

type VpcForge struct {
        vpc      awsec2.IVpc
        properties map[string]interface{}
}

func (v *VpcForge) Create(ctx *interfaces.ForgeContext) interface{} {
	vpcInstance, ok := (*ctx.Instance).(*VpcInstanceConfig)
	if !ok {
		// 处理类型断言失败的情况
		return nil
	}

	// 新增逻辑：优先使用现有 VPC
	if vpcInstance.VpcId != "" {  // 假设 VpcInstanceConfig 新增了 VpcId 字段
		// 通过 VPC ID 查找已有 VPC
		existingVpc := awsec2.Vpc_FromLookup(ctx.Stack, jsii.String("ExistingVPC"), &awsec2.VpcLookupOptions{
			VpcId: jsii.String(vpcInstance.VpcId),
		})
		v.vpc = existingVpc
		
		// 保存 VPC 属性
		if v.properties == nil {
			v.properties = make(map[string]interface{})
		}
		v.properties["vpcId"] = vpcInstance.VpcId
		v.properties["isExisting"] = true
		
		return v  // 返回 VpcForge 自身
	}

	availabilityZones := aws.GetAvailabilityZones()
	
	// 转换为 []*string 类型
	var azPointers []*string
	for _, az := range availabilityZones {
		azPointers = append(azPointers, jsii.String(az))
	}
	
	var azCount int
	if types.GetBoolValue(vpcInstance.NatGatewayPerAZ, false) {
		azCount = len(availabilityZones)
	} else {
		azCount = 1
	}

	var IpProtocol awsec2.IpProtocol
	if ctx.DualStack {
		IpProtocol = awsec2.IpProtocol_DUAL_STACK
	} else {
		IpProtocol = awsec2.IpProtocol_IPV4_ONLY
	}

	newVpc := awsec2.NewVpc(ctx.Stack, jsii.String("VPC"), &awsec2.VpcProps{
                IpProtocol: IpProtocol,
                IpAddresses: awsec2.IpAddresses_Cidr(jsii.String(vpcInstance.CidrBlock)),
                AvailabilityZones: &azPointers,

                //MaxAzs:      jsii.Number(3),
                //AvailabilityZones: &[]*string{
                //    jsii.String("us-east-1a"),
                //    jsii.String("us-east-1b"),
                //    jsii.String("us-east-1c"),
                //},
                NatGateways: jsii.Number(azCount),
                SubnetConfiguration: &[]*awsec2.SubnetConfiguration{
                        {
                                CidrMask:   jsii.Number(24),
                                Name:       jsii.String("Public"),
                                SubnetType: awsec2.SubnetType_PUBLIC,
                                MapPublicIpOnLaunch:   jsii.Bool(true), // 允许在公共子网中启动具有公共 IP 地址的实例
                        },
                        {
                                CidrMask:   jsii.Number(24),
                                Name:       jsii.String("Private"),
                                SubnetType: awsec2.SubnetType_PRIVATE_WITH_EGRESS,
                        },
                        {
                                CidrMask:   jsii.Number(24),
                                Name:       jsii.String("Isolated"),
                                SubnetType: awsec2.SubnetType_PRIVATE_ISOLATED,
                        },
                },
        })

	v.vpc = newVpc
	
	// 保存 VPC 属性
	if v.properties == nil {
		v.properties = make(map[string]interface{})
	}
	v.properties["vpcId"] = newVpc.VpcId()
	v.properties["cidrBlock"] = vpcInstance.CidrBlock
	v.properties["isExisting"] = false
	
        return v
}

func (v *VpcForge) CreateOutputs(ctx *interfaces.ForgeContext) {
	awscdk.NewCfnOutput(ctx.Stack, jsii.String("VPCId"), &awscdk.CfnOutputProps{
		Value:       v.vpc.VpcId(),
		Description: jsii.String("VPC ID"),
	})

	awscdk.NewCfnOutput(ctx.Stack, jsii.String("VPCCidr"), &awscdk.CfnOutputProps{
		Value:       v.vpc.VpcCidrBlock(),
		Description: jsii.String("VPC CIDR Block"),
	})

	publicSubnets := v.vpc.PublicSubnets()
	publicSubnetIds := make([]string, 0, len(*publicSubnets))
	for _, subnet := range *publicSubnets {
		publicSubnetIds = append(publicSubnetIds, *subnet.SubnetId())
	}

	awscdk.NewCfnOutput(ctx.Stack, jsii.String("PublicSubnets"), &awscdk.CfnOutputProps{
		Value:       jsii.String(strings.Join(publicSubnetIds, ",")),
		Description: jsii.String("Public Subnet IDs"),
	})

	publicSubnetCidrs := make([]string, 0, len(*publicSubnets))
	for _, subnet := range *publicSubnets {
		publicSubnetCidrs = append(publicSubnetCidrs, *subnet.Ipv4CidrBlock())
	}

	awscdk.NewCfnOutput(ctx.Stack, jsii.String("PublicSubnetsCidrs"), &awscdk.CfnOutputProps{
		Value:       jsii.String(strings.Join(publicSubnetCidrs, ",")),
		Description: jsii.String("Public Subnet CIDR Blocks"),
	})

	privateSubnets := v.vpc.PrivateSubnets()
	privateSubnetIds := make([]string, 0, len(*privateSubnets))
	for _, subnet := range *privateSubnets {
		privateSubnetIds = append(privateSubnetIds, *subnet.SubnetId())
	}

	awscdk.NewCfnOutput(ctx.Stack, jsii.String("PrivateSubnets"), &awscdk.CfnOutputProps{
		Value:       jsii.String(strings.Join(privateSubnetIds, ",")),
		Description: jsii.String("Private Subnet IDs"),
	})

	privateSubnetCidrs := make([]string, 0, len(*privateSubnets))
	for _, subnet := range *privateSubnets {
		privateSubnetCidrs = append(privateSubnetCidrs, *subnet.Ipv4CidrBlock())
	}

	awscdk.NewCfnOutput(ctx.Stack, jsii.String("PrivateSubnetsCidrs"), &awscdk.CfnOutputProps{
		Value:       jsii.String(strings.Join(privateSubnetCidrs, ",")),
		Description: jsii.String("Private Subnet CIDR Blocks"),
	})

	isolatedSubnets := v.vpc.IsolatedSubnets()
	isolatedSubnetIds := make([]string, 0, len(*isolatedSubnets))
	for _, subnet := range *isolatedSubnets {
		isolatedSubnetIds = append(isolatedSubnetIds, *subnet.SubnetId())
	}

	awscdk.NewCfnOutput(ctx.Stack, jsii.String("IsolatedSubnets"), &awscdk.CfnOutputProps{
		Value:       jsii.String(strings.Join(isolatedSubnetIds, ",")),
		Description: jsii.String("Isolated Subnet IDs"),
	})

	isolatedSubnetCidrs := make([]string, 0, len(*isolatedSubnets))
	for _, subnet := range *isolatedSubnets {
		isolatedSubnetCidrs = append(isolatedSubnetCidrs, *subnet.Ipv4CidrBlock())
	}

	awscdk.NewCfnOutput(ctx.Stack, jsii.String("IsolatedSubnetsCidrs"), &awscdk.CfnOutputProps{
		Value:       jsii.String(strings.Join(isolatedSubnetCidrs, ",")),
		Description: jsii.String("Isolated Subnet CIDR Blocks"),
	})
}

func (v *VpcForge) ConfigureRules(ctx *interfaces.ForgeContext) {
	// 为 Public subnet 配置特定的入站规则
	// 打开 Public Subnet 打开端口 22, 80, 443, 8443
	return
	if ctx.DualStack {
		ctx.SecurityGroups.Public.AddIngressRule(awsec2.Peer_AnyIpv6(), awsec2.Port_Tcp(jsii.Number(8443)), jsii.String("Allow HTTP access"), nil)
		ctx.SecurityGroups.Public.AddIngressRule(awsec2.Peer_AnyIpv6(), awsec2.Port_Tcp(jsii.Number(443)), jsii.String("Allow HTTP access"), nil)
		ctx.SecurityGroups.Public.AddIngressRule(awsec2.Peer_AnyIpv6(), awsec2.Port_Tcp(jsii.Number(80)), jsii.String("Allow HTTP access"), nil)
		//publicSG.AddIngressRule(awsec2.Peer_AnyIpv6(), awsec2.Port_Tcp(jsii.Number(22)), jsii.String("Allow SSH access"), nil)
	}

	ctx.SecurityGroups.Public.AddIngressRule(awsec2.Peer_AnyIpv4(), awsec2.Port_Tcp(jsii.Number(8443)), jsii.String("Allow HTTP access"), nil)
	ctx.SecurityGroups.Public.AddIngressRule(awsec2.Peer_AnyIpv4(), awsec2.Port_Tcp(jsii.Number(443)), jsii.String("Allow HTTP access"), nil)
	ctx.SecurityGroups.Public.AddIngressRule(awsec2.Peer_AnyIpv4(), awsec2.Port_Tcp(jsii.Number(80)), jsii.String("Allow HTTP access"), nil)
	//ctx.SecurityGroups.Public.AddIngressRule(awsec2.Peer_AnyIpv4(), awsec2.Port_Tcp(jsii.Number(22)), jsii.String("Allow SSH access"), nil)
	//ctx.SecurityGroups.Private.AddIngressRule(ctx.SecurityGroups.Public, awsec2.Port_TcpRange(jsii.Number(22), jsii.Number(22)), jsii.String("Allow SSH access from public subnet"), nil)

}

func CreateSecurityGroups(stack awscdk.Stack, vpc awsec2.IVpc, dualStack bool) (publicSG, privateSG, isolatedSG awsec2.SecurityGroup) {
	publicSG = awsec2.NewSecurityGroup(stack, jsii.String("PublicSG"), &awsec2.SecurityGroupProps{
		Vpc:               vpc,
		Description:       jsii.String("Allow HTTP and SSH access"),
		AllowAllIpv6Outbound: jsii.Bool(dualStack),
		AllowAllOutbound:  jsii.Bool(true),
	})

	privateSG = awsec2.NewSecurityGroup(stack, jsii.String("PrivateSG"), &awsec2.SecurityGroupProps{
		Vpc:              vpc,
		Description:      jsii.String("Allow access from public subnet"),
		AllowAllIpv6Outbound: jsii.Bool(dualStack),
		AllowAllOutbound: jsii.Bool(true),
	})

	privateSG.AddIngressRule(publicSG, awsec2.Port_AllTraffic(), jsii.String("Allow access from public subnet"), nil)
	privateSG.AddIngressRule(privateSG, awsec2.Port_AllTraffic(), jsii.String("Allow access within private subnet"), nil)

	isolatedSG = awsec2.NewSecurityGroup(stack, jsii.String("IsolatedSG"), &awsec2.SecurityGroupProps{
		Vpc:              vpc,
		Description:      jsii.String("Allow access from private subnet"),
		AllowAllIpv6Outbound: jsii.Bool(dualStack),
		AllowAllOutbound: jsii.Bool(true),
	})

	return publicSG, privateSG, isolatedSG
}

func (v *VpcForge) MergeConfigs(defaults config.InstanceConfig, instance config.InstanceConfig) config.InstanceConfig {
	// 从默认配置中复制基本字段
	merged := defaults.(*VpcInstanceConfig)

	// 从实例配置中覆盖基本字段
        vpcInstance := instance.(*VpcInstanceConfig)
	if instance != nil {
		if vpcInstance.GetID() != "" {
			merged.ID = vpcInstance.GetID()
		}
		if vpcInstance.Type != "" {
			merged.Type = vpcInstance.GetType()
		}
		if vpcInstance.Subnet != "" {
			merged.Subnet = vpcInstance.GetSubnet()
		}
		if vpcInstance.SecurityGroup != "" {
			merged.SecurityGroup = vpcInstance.GetSecurityGroup()
		}
	}

	// 处理 RemovePolicy 字段
	if vpcInstance.CidrBlock != "" {
		merged.CidrBlock = vpcInstance.CidrBlock
	}

	return merged
}

func (v *VpcForge) GetProperties() map[string]interface{} {
	return v.properties
}

// GetVpc 返回实际的 VPC 资源
func (v *VpcForge) GetVpc() awsec2.IVpc {
	return v.vpc
}

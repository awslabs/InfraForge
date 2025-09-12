// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package ds

import (
        "fmt"

        "github.com/awslabs/InfraForge/core/config"
        "github.com/awslabs/InfraForge/core/interfaces"
        "github.com/awslabs/InfraForge/core/utils/types"
        "github.com/awslabs/InfraForge/core/utils/security"
        "github.com/aws/aws-cdk-go/awscdk/v2"
        "github.com/aws/aws-cdk-go/awscdk/v2/awsdirectoryservice"
        "github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
        "github.com/aws/aws-cdk-go/awscdk/v2/awssecretsmanager"
        "github.com/aws/jsii-runtime-go"
)

// DsInstanceConfig 配置结构体
type DsInstanceConfig struct {
        config.BaseInstanceConfig
        DomainName string `json:"domainName"`
        ShortName  string `json:"shortName"`
        Edition    string `json:"edition"`
        EnableSso  *bool  `json:"enableSso,omitempty"`
        UnixHome   string `json:"unixHome"`
}

// DsForge 结构体
type DsForge struct {
        directory  awsdirectoryservice.CfnMicrosoftAD
        secret     awssecretsmanager.Secret
        properties map[string]interface{}
}

// GetSecret 返回创建的密码
func (d *DsForge) GetSecret() awssecretsmanager.Secret {
        return d.secret
}

// GetSecretARN 返回Secret的ARN
func (d *DsForge) GetSecretARN() *string {
        return d.secret.SecretArn()
}

// GetSecretName 返回Secret的名称
func (d *DsForge) GetSecretName() *string {
        return d.secret.SecretName()
}

// GetDirectory 返回创建的 Directory Service 对象
func (d *DsForge) GetDirectory() awsdirectoryservice.CfnMicrosoftAD {
	return d.directory
}

// GetDirectoryId 返回Directory Service的ID
func (d *DsForge) GetDirectoryId() *string {
        return d.directory.Ref()
}

// Create 实现创建接口
func (d *DsForge) Create(ctx *interfaces.ForgeContext) interface{} {
        dsInstance, ok := (*ctx.Instance).(*DsInstanceConfig)
        if !ok {
                return nil
        }

        // 获取两个私有子网
        subnets := ctx.VPC.SelectSubnets(&awsec2.SubnetSelection{
                SubnetType: awsec2.SubnetType_PRIVATE_WITH_EGRESS,
        })

        // 获取子网切片
        subnetSlice := *subnets.Subnets

        if len(subnetSlice) < 2 {
                panic("Need at least 2 private subnets for Directory Service")
        }

        // 创建密码Secret
        secretName := fmt.Sprintf("%s-%s-DirectoryPassword", *ctx.Stack.StackName(), dsInstance.GetID())

	// 获取或创建密码和Secret对象
	dsPassId := fmt.Sprintf("%s-DSPassword", dsInstance.GetID())
	password, secret := security.GetOrCreateSecretPassword(ctx.Stack, dsPassId, secretName, 30)

        // 保存密码引用
        d.secret = secret
	
        // 创建Directory Service属性
        props := &awsdirectoryservice.CfnMicrosoftADProps{
                Name:        jsii.String(dsInstance.DomainName),
                CreateAlias: jsii.Bool(types.GetBoolValue(dsInstance.EnableSso, false)),
                ShortName:   jsii.String(dsInstance.ShortName),
                Password:    jsii.String(password),
                Edition:     jsii.String(dsInstance.Edition), // Standard or Enterprise
                EnableSso:   jsii.Bool(types.GetBoolValue(dsInstance.EnableSso, false)),
                VpcSettings: &awsdirectoryservice.CfnMicrosoftAD_VpcSettingsProperty{
                        VpcId: ctx.VPC.VpcId(),
                        SubnetIds: &[]*string{
                                subnetSlice[0].SubnetId(),
                                subnetSlice[1].SubnetId(),
                        },
                },
        }
        

        // 创建 Microsoft AD
        directory := awsdirectoryservice.NewCfnMicrosoftAD(ctx.Stack, jsii.String(dsInstance.GetID()), props)

        unixHomeKey := "unixHome"
        directory.AddMetadata(&unixHomeKey, dsInstance.UnixHome)

        d.directory = directory
        
        // 保存 DS 属性
        if d.properties == nil {
                d.properties = make(map[string]interface{})
        }
        d.properties["attrId"] = directory.Ref()
        d.properties["domainName"] = dsInstance.DomainName
        d.properties["shortName"] = dsInstance.ShortName
        d.properties["edition"] = dsInstance.Edition
        d.properties["name"] = directory.Name()
        d.properties["password"] = directory.Password()
        d.properties["secretARN"] = secret.SecretArn()
        
        // DNS IP 地址
        d.properties["attrDnsIpAddresses"] = []interface{}{
                awscdk.Fn_Select(jsii.Number(0), directory.AttrDnsIpAddresses()),
                awscdk.Fn_Select(jsii.Number(1), directory.AttrDnsIpAddresses()),
        }
        
        // Unix Home (重用之前定义的 unixHomeKey)
        d.properties["unixHome"] = directory.GetMetadata(&unixHomeKey)

        //return directory
        return d
}

// CreateOutputs 实现输出接口
func (d *DsForge) CreateOutputs(ctx *interfaces.ForgeContext) {
        dsInstance, ok := (*ctx.Instance).(*DsInstanceConfig)
        if !ok {
                return
        }

        awscdk.NewCfnOutput(ctx.Stack, jsii.String("DirectoryService"+dsInstance.GetID()), &awscdk.CfnOutputProps{
                Value:       d.directory.ShortName(),
                Description: jsii.String("Directory Service DNS Name"),
        })

        awscdk.NewCfnOutput(ctx.Stack, jsii.String("DirectoryServiceId"+dsInstance.GetID()), &awscdk.CfnOutputProps{
                Value:       d.directory.Ref(),
                Description: jsii.String("Directory Service ID"),
        })

        // 输出密码的 ARN，以便后续查找
        awscdk.NewCfnOutput(ctx.Stack, jsii.String(fmt.Sprintf("%sPasswordARN", dsInstance.GetID())), &awscdk.CfnOutputProps{
                Value:       d.secret.SecretArn(),
                Description: jsii.String(fmt.Sprintf("ARN of the secret containing the password for Directory Service %s", dsInstance.DomainName)),
        })
        
        // 添加密码存储位置的输出
        awscdk.NewCfnOutput(ctx.Stack, jsii.String("DirectoryServicePasswordInfo"+dsInstance.GetID()), &awscdk.CfnOutputProps{
                Value:       jsii.String("Directory Service password is stored in Secrets Manager"),
                Description: jsii.String("Directory Service Password Information"),
        })
}

// ConfigureRules 实现安全组规则配置接口
func (d *DsForge) ConfigureRules(ctx *interfaces.ForgeContext) {
        // AWS Directory Service 自动创建和管理自己的安全组
        // 不需要手动配置安全组规则
        
        /*
        // Directory Service 所需端口
        ports := []int{53, 88, 389, 445, 464, 636, 3268, 3269, 5985, 9389}
        for _, port := range ports {
                security.AddTcpIngressRule(
                        ctx.SecurityGroups.Default,
                        privateSG,
			port,
                        fmt.Sprintf("Allow AD TCP port %d from private subnet", port),
                )
                security.AddTcpIngressRule(
                        defaultSG,
                        publicSG,
			port,
                        fmt.Sprintf("Allow AD TCP port %d from public subnet", port),
                )
        }

        // UDP ports
        udpPorts := []int{53, 88, 123, 389, 464}
        for _, udpPort := range udpPorts {
                security.AddUdpIngressRule(
                        defaultSG,
                        privateSG,
			udpPort,
                        fmt.Sprintf("Allow AD UDP port %d from private subnet", udpPort),
                )
                security.AddUdpIngressRule(
                        defaultSG,
                        publicSG,
			udpPort,
                        fmt.Sprintf("Allow AD UDP port %d from public subnet", udpPort),
		)
        }
        */

	/*
        ports := []float64{53, 88, 389, 445, 464, 636, 3268, 3269, 5985, 9389}

        for _, port := range ports {
                defaultSG.AddIngressRule(
                        privateSG,
                        awsec2.Port_Tcp(jsii.Number(port)),
                        jsii.String(fmt.Sprintf("Allow AD port %f from private subnet", port)),
                        nil,
                )
        }

        // UDP ports
        udpPorts := []float64{53, 88, 123, 389, 464}
        for _, port := range udpPorts {
                defaultSG.AddIngressRule(
                        privateSG,
                        awsec2.Port_Udp(jsii.Number(port)),
                        jsii.String(fmt.Sprintf("Allow AD UDP port %f from private subnet", port)),
                        nil,
                )
        }
	*/
}

// MergeConfigs 实现配置合并接口
func (d *DsForge) MergeConfigs(defaults config.InstanceConfig, instance config.InstanceConfig) config.InstanceConfig {
        merged := defaults.(*DsInstanceConfig)
        dsInstance := instance.(*DsInstanceConfig)

        if instance != nil {
                // 合并基本配置
                if dsInstance.GetID() != "" {
                        merged.ID = dsInstance.GetID()
                }
                if dsInstance.Type != "" {
                        merged.Type = dsInstance.GetType()
                }
                if dsInstance.Subnet != "" {
                        merged.Subnet = dsInstance.GetSubnet()
                }
                if dsInstance.SecurityGroup != "" {
                        merged.SecurityGroup = dsInstance.GetSecurityGroup()
                }

                // 合并特定配置
                if dsInstance.EnableSso != nil {
                        merged.EnableSso = dsInstance.EnableSso
                }
                if dsInstance.DomainName != "" {
                        merged.DomainName = dsInstance.DomainName
                }
                if dsInstance.ShortName != "" {
                        merged.ShortName = dsInstance.ShortName
                }
                if dsInstance.Edition != "" {
                        merged.Edition = dsInstance.Edition
                }
                if dsInstance.UnixHome != "" {
                        merged.UnixHome = dsInstance.UnixHome
                }
        }

        // 设置默认值
        if merged.Edition == "" {
                merged.Edition = "Standard"
        }

        if merged.UnixHome == "" {
                merged.UnixHome = "/home"
        }

        return merged
}
func (d *DsForge) GetProperties() map[string]interface{} {
	return d.properties
}

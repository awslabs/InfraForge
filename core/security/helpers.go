// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package security

import (
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
)

// 安全组到安全组的规则

// AddTcpIngressRule 是一个辅助函数，用于添加 TCP 入站规则
func AddTcpIngressRule(targetSG, sourceSG awsec2.SecurityGroup, port int, description string) {
	GlobalRuleRegistry.AddTcpIngressRuleSafely(targetSG, sourceSG, port, description)
}

// AddTcpRangeIngressRule 是一个辅助函数，用于添加 TCP 端口范围入站规则
func AddTcpRangeIngressRule(targetSG, sourceSG awsec2.SecurityGroup, fromPort, toPort int, description string) {
	GlobalRuleRegistry.AddTcpRangeIngressRuleSafely(targetSG, sourceSG, fromPort, toPort, description)
}

// AddUdpIngressRule 是一个辅助函数，用于添加 UDP 入站规则
func AddUdpIngressRule(targetSG, sourceSG awsec2.SecurityGroup, port int, description string) {
	GlobalRuleRegistry.AddUdpIngressRuleSafely(targetSG, sourceSG, port, description)
}

// AddUdpRangeIngressRule 是一个辅助函数，用于添加 UDP 端口范围入站规则
func AddUdpRangeIngressRule(targetSG, sourceSG awsec2.SecurityGroup, fromPort, toPort int, description string) {
	GlobalRuleRegistry.AddUdpRangeIngressRuleSafely(targetSG, sourceSG, fromPort, toPort, description)
}

// AddAllTrafficIngressRule 是一个辅助函数，用于添加允许所有流量的入站规则
func AddAllTrafficIngressRule(targetSG, sourceSG awsec2.SecurityGroup, description string) {
	GlobalRuleRegistry.AddAllTrafficIngressRuleSafely(targetSG, sourceSG, description)
}

// IPv4 CIDR 到安全组的规则

// AddTcpIngressRuleFromCidr 是一个辅助函数，用于添加来自特定 IPv4 CIDR 的 TCP 入站规则
func AddTcpIngressRuleFromCidr(targetSG awsec2.SecurityGroup, cidr string, port int, description string) {
	GlobalRuleRegistry.AddTcpIngressRuleFromCidrSafely(targetSG, cidr, port, description)
}

// AddTcpRangeIngressRuleFromCidr 是一个辅助函数，用于添加来自特定 IPv4 CIDR 的 TCP 端口范围入站规则
func AddTcpRangeIngressRuleFromCidr(targetSG awsec2.SecurityGroup, cidr string, fromPort, toPort int, description string) {
	GlobalRuleRegistry.AddTcpRangeIngressRuleFromCidrSafely(targetSG, cidr, fromPort, toPort, description)
}

// AddUdpIngressRuleFromCidr 是一个辅助函数，用于添加来自特定 IPv4 CIDR 的 UDP 入站规则
func AddUdpIngressRuleFromCidr(targetSG awsec2.SecurityGroup, cidr string, port int, description string) {
	GlobalRuleRegistry.AddUdpIngressRuleFromCidrSafely(targetSG, cidr, port, description)
}

// AddUdpRangeIngressRuleFromCidr 是一个辅助函数，用于添加来自特定 IPv4 CIDR 的 UDP 端口范围入站规则
func AddUdpRangeIngressRuleFromCidr(targetSG awsec2.SecurityGroup, cidr string, fromPort, toPort int, description string) {
	GlobalRuleRegistry.AddUdpRangeIngressRuleFromCidrSafely(targetSG, cidr, fromPort, toPort, description)
}

// AddAllTrafficIngressRuleFromCidr 是一个辅助函数，用于添加允许所有流量的入站规则
func AddAllTrafficIngressRuleFromCidr(targetSG awsec2.SecurityGroup, cidr string, description string) {
	GlobalRuleRegistry.AddAllTrafficIngressRuleFromCidrSafely(targetSG, cidr, description)
}

// IPv6 CIDR 到安全组的规则

// AddTcpIngressRuleFromCidrIpv6 是一个辅助函数，用于添加来自特定 IPv6 CIDR 的 TCP 入站规则
func AddTcpIngressRuleFromCidrIpv6(targetSG awsec2.SecurityGroup, cidr string, port int, description string) {
	GlobalRuleRegistry.AddTcpIngressRuleFromCidrIpv6Safely(targetSG, cidr, port, description)
}

// AddTcpRangeIngressRuleFromCidrIpv6 是一个辅助函数，用于添加来自特定 IPv6 CIDR 的 TCP 端口范围入站规则
func AddTcpRangeIngressRuleFromCidrIpv6(targetSG awsec2.SecurityGroup, cidr string, fromPort, toPort int, description string) {
	GlobalRuleRegistry.AddTcpRangeIngressRuleFromCidrIpv6Safely(targetSG, cidr, fromPort, toPort, description)
}

// AddUdpIngressRuleFromCidrIpv6 是一个辅助函数，用于添加来自特定 IPv6 CIDR 的 UDP 入站规则
func AddUdpIngressRuleFromCidrIpv6(targetSG awsec2.SecurityGroup, cidr string, port int, description string) {
	GlobalRuleRegistry.AddUdpIngressRuleFromCidrIpv6Safely(targetSG, cidr, port, description)
}

// AddUdpRangeIngressRuleFromCidrIpv6 是一个辅助函数，用于添加来自特定 IPv6 CIDR 的 UDP 端口范围入站规则
func AddUdpRangeIngressRuleFromCidrIpv6(targetSG awsec2.SecurityGroup, cidr string, fromPort, toPort int, description string) {
	GlobalRuleRegistry.AddUdpRangeIngressRuleFromCidrIpv6Safely(targetSG, cidr, fromPort, toPort, description)
}

// AddAllTrafficIngressRuleFromCidrIpv6 是一个辅助函数，用于添加允许所有流量的入站规则
func AddAllTrafficIngressRuleFromCidrIpv6(targetSG awsec2.SecurityGroup, cidr string, description string) {
	GlobalRuleRegistry.AddAllTrafficIngressRuleFromCidrIpv6Safely(targetSG, cidr, description)
}

// 便捷函数 - 任意 IP 地址

// AddTcpIngressRuleFromAnyIp 是一个辅助函数，用于添加来自任意 IPv4 地址的 TCP 入站规则
func AddTcpIngressRuleFromAnyIp(targetSG awsec2.SecurityGroup, port int, description string) {
	AddTcpIngressRuleFromCidr(targetSG, "0.0.0.0/0", port, description)
}

// AddTcpIngressRuleFromAnyIpv6 是一个辅助函数，用于添加来自任意 IPv6 地址的 TCP 入站规则
func AddTcpIngressRuleFromAnyIpv6(targetSG awsec2.SecurityGroup, port int, description string) {
	AddTcpIngressRuleFromCidrIpv6(targetSG, "::/0", port, description)
}

// AddAllTrafficIngressRuleFromAnyIp 是一个辅助函数，用于添加允许来自任意 IPv4 地址的所有流量的入站规则
func AddAllTrafficIngressRuleFromAnyIp(targetSG awsec2.SecurityGroup, description string) {
	AddAllTrafficIngressRuleFromCidr(targetSG, "0.0.0.0/0", description)
}

// AddAllTrafficIngressRuleFromAnyIpv6 是一个辅助函数，用于添加允许来自任意 IPv6 地址的所有流量的入站规则
func AddAllTrafficIngressRuleFromAnyIpv6(targetSG awsec2.SecurityGroup, description string) {
	AddAllTrafficIngressRuleFromCidrIpv6(targetSG, "::/0", description)
}

// 出站规则 (Egress Rules)

// AddAllTrafficEgressRule 是一个辅助函数，用于添加允许所有出站流量的规则
func AddAllTrafficEgressRule(sourceGroup, targetGroup awsec2.SecurityGroup, description string) {
	GlobalRuleRegistry.AddAllTrafficEgressRuleSafely(sourceGroup, targetGroup, description)
}

// 特殊场景辅助函数

// AddL1EgressRule 是一个辅助函数，用于添加使用 L1 构造函数的出站规则
func AddEfaEgressRule(sourceGroup awsec2.SecurityGroup, description string) {
	GlobalRuleRegistry.AddEfaEgressRuleSafely(sourceGroup, description)
}

// ConfigureEFASecurityRules 配置 EFA 所需的安全组规则
func ConfigureEFASecurityRules(securityGroup awsec2.SecurityGroup, description string) {
	// EFA 需要配置安全组到自己的入站和出站规则
	AddAllTrafficIngressRule(securityGroup, securityGroup, "Allow all inbound traffic for EFA from self - " + description)
	
	// 使用 L1 构造函数直接添加出站规则，绕过 allowAllOutbound 设置，并检查重复
	AddEfaEgressRule(securityGroup, "Allow all outbound traffic for EFA from self - " + description)
}

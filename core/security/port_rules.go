// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package security

import (
	"fmt"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	utilsSecurity "github.com/awslabs/InfraForge/core/utils/security"
)

// ApplyPortRules 应用端口规则到安全组
func ApplyPortRules(targetSG awsec2.SecurityGroup, allowedPorts, allowedPortsIpv6 string, dualStack bool) {
	// 处理IPv4规则
	if allowedPorts != "" {
		rules := utilsSecurity.ParseAllowedPorts(allowedPorts)
		for _, rule := range rules {
			if rule.Port > 0 {
				if rule.Protocol == "udp" {
					AddUdpIngressRuleFromCidr(targetSG, rule.Cidr, rule.Port, fmt.Sprintf("Allow port %d UDP", rule.Port))
				} else if rule.Protocol == "both" {
					AddTcpIngressRuleFromCidr(targetSG, rule.Cidr, rule.Port, fmt.Sprintf("Allow port %d TCP", rule.Port))
					AddUdpIngressRuleFromCidr(targetSG, rule.Cidr, rule.Port, fmt.Sprintf("Allow port %d UDP", rule.Port))
				} else {
					AddTcpIngressRuleFromCidr(targetSG, rule.Cidr, rule.Port, fmt.Sprintf("Allow port %d TCP", rule.Port))
				}
			} else if rule.FromPort > 0 && rule.ToPort > 0 {
				if rule.Protocol == "udp" {
					AddUdpRangeIngressRuleFromCidr(targetSG, rule.Cidr, rule.FromPort, rule.ToPort, fmt.Sprintf("Allow ports %d-%d UDP", rule.FromPort, rule.ToPort))
				} else if rule.Protocol == "both" {
					AddTcpRangeIngressRuleFromCidr(targetSG, rule.Cidr, rule.FromPort, rule.ToPort, fmt.Sprintf("Allow ports %d-%d TCP", rule.FromPort, rule.ToPort))
					AddUdpRangeIngressRuleFromCidr(targetSG, rule.Cidr, rule.FromPort, rule.ToPort, fmt.Sprintf("Allow ports %d-%d UDP", rule.FromPort, rule.ToPort))
				} else {
					AddTcpRangeIngressRuleFromCidr(targetSG, rule.Cidr, rule.FromPort, rule.ToPort, fmt.Sprintf("Allow ports %d-%d TCP", rule.FromPort, rule.ToPort))
				}
			}
		}
	}
	
	// 处理IPv6规则
	if dualStack && allowedPortsIpv6 != "" {
		ipv6Rules := utilsSecurity.ParseAllowedPorts(allowedPortsIpv6)
		for _, rule := range ipv6Rules {
			if rule.Port > 0 {
				if rule.Protocol == "udp" {
					AddUdpIngressRuleFromCidrIpv6(targetSG, rule.Cidr, rule.Port, fmt.Sprintf("Allow port %d UDP IPv6", rule.Port))
				} else if rule.Protocol == "both" {
					AddTcpIngressRuleFromCidrIpv6(targetSG, rule.Cidr, rule.Port, fmt.Sprintf("Allow port %d TCP IPv6", rule.Port))
					AddUdpIngressRuleFromCidrIpv6(targetSG, rule.Cidr, rule.Port, fmt.Sprintf("Allow port %d UDP IPv6", rule.Port))
				} else {
					AddTcpIngressRuleFromCidrIpv6(targetSG, rule.Cidr, rule.Port, fmt.Sprintf("Allow port %d TCP IPv6", rule.Port))
				}
			} else if rule.FromPort > 0 && rule.ToPort > 0 {
				if rule.Protocol == "udp" {
					AddUdpRangeIngressRuleFromCidrIpv6(targetSG, rule.Cidr, rule.FromPort, rule.ToPort, fmt.Sprintf("Allow ports %d-%d UDP IPv6", rule.FromPort, rule.ToPort))
				} else if rule.Protocol == "both" {
					AddTcpRangeIngressRuleFromCidrIpv6(targetSG, rule.Cidr, rule.FromPort, rule.ToPort, fmt.Sprintf("Allow ports %d-%d TCP IPv6", rule.FromPort, rule.ToPort))
					AddUdpRangeIngressRuleFromCidrIpv6(targetSG, rule.Cidr, rule.FromPort, rule.ToPort, fmt.Sprintf("Allow ports %d-%d UDP IPv6", rule.FromPort, rule.ToPort))
				} else {
					AddTcpRangeIngressRuleFromCidrIpv6(targetSG, rule.Cidr, rule.FromPort, rule.ToPort, fmt.Sprintf("Allow ports %d-%d TCP IPv6", rule.FromPort, rule.ToPort))
				}
			}
		}
	}
}

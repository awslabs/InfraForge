// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package security

import (
	"testing"

	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/jsii-runtime-go"
)

// 模拟安全组
type mockSecurityGroup struct {
	id            string
	ingressRules  []string
	securityGroup awsec2.SecurityGroup
}

func (m *mockSecurityGroup) SecurityGroupId() *string {
	return jsii.String(m.id)
}

func (m *mockSecurityGroup) AddIngressRule(peer awsec2.IPeer, connection awsec2.Port, description *string, remoteRule *bool) awsec2.CfnIngress {
	// 记录规则
	m.ingressRules = append(m.ingressRules, *description)
	return nil
}

func TestSecurityGroupToSecurityGroupRules(t *testing.T) {
	// 重置全局注册表
	GlobalRuleRegistry.Reset()

	// 创建模拟安全组
	targetSG := &mockSecurityGroup{id: "sg-target"}
	sourceSG := &mockSecurityGroup{id: "sg-source"}

	// 测试 TCP 规则
	GlobalRuleRegistry.AddTcpIngressRuleSafely(targetSG, sourceSG, 22, "SSH")
	if len(targetSG.ingressRules) != 1 {
		t.Errorf("Expected 1 rule, got %d", len(targetSG.ingressRules))
	}

	// 再次添加相同的规则
	GlobalRuleRegistry.AddTcpIngressRuleSafely(targetSG, sourceSG, 22, "SSH")
	if len(targetSG.ingressRules) != 1 {
		t.Errorf("Expected still 1 rule, got %d", len(targetSG.ingressRules))
	}

	// 添加不同端口的规则
	GlobalRuleRegistry.AddTcpIngressRuleSafely(targetSG, sourceSG, 80, "HTTP")
	if len(targetSG.ingressRules) != 2 {
		t.Errorf("Expected 2 rules, got %d", len(targetSG.ingressRules))
	}

	// 测试 UDP 规则
	GlobalRuleRegistry.AddUdpIngressRuleSafely(targetSG, sourceSG, 53, "DNS")
	if len(targetSG.ingressRules) != 3 {
		t.Errorf("Expected 3 rules, got %d", len(targetSG.ingressRules))
	}

	// 测试端口范围规则
	GlobalRuleRegistry.AddTcpRangeIngressRuleSafely(targetSG, sourceSG, 1024, 2048, "TCP Range")
	if len(targetSG.ingressRules) != 4 {
		t.Errorf("Expected 4 rules, got %d", len(targetSG.ingressRules))
	}

	// 测试 AllTraffic 规则
	GlobalRuleRegistry.AddAllTrafficIngressRuleSafely(targetSG, sourceSG, "All Traffic")
	if len(targetSG.ingressRules) != 5 {
		t.Errorf("Expected 5 rules, got %d", len(targetSG.ingressRules))
	}
}

func TestCidrRules(t *testing.T) {
	// 重置全局注册表
	GlobalRuleRegistry.Reset()

	// 创建模拟安全组
	targetSG := &mockSecurityGroup{id: "sg-target"}

	// 测试 IPv4 CIDR 规则
	GlobalRuleRegistry.AddTcpIngressRuleFromCidrSafely(targetSG, "192.168.1.0/24", 22, "IPv4 SSH")
	if len(targetSG.ingressRules) != 1 {
		t.Errorf("Expected 1 rule, got %d", len(targetSG.ingressRules))
	}

	// 再次添加相同的规则
	GlobalRuleRegistry.AddTcpIngressRuleFromCidrSafely(targetSG, "192.168.1.0/24", 22, "IPv4 SSH")
	if len(targetSG.ingressRules) != 1 {
		t.Errorf("Expected still 1 rule, got %d", len(targetSG.ingressRules))
	}

	// 测试 IPv6 CIDR 规则
	GlobalRuleRegistry.AddTcpIngressRuleFromCidrIpv6Safely(targetSG, "2001:db8::/32", 22, "IPv6 SSH")
	if len(targetSG.ingressRules) != 2 {
		t.Errorf("Expected 2 rules, got %d", len(targetSG.ingressRules))
	}

	// 再次添加相同的 IPv6 规则
	GlobalRuleRegistry.AddTcpIngressRuleFromCidrIpv6Safely(targetSG, "2001:db8::/32", 22, "IPv6 SSH")
	if len(targetSG.ingressRules) != 2 {
		t.Errorf("Expected still 2 rules, got %d", len(targetSG.ingressRules))
	}

	// 添加不同的 IPv6 规则
	GlobalRuleRegistry.AddTcpIngressRuleFromCidrIpv6Safely(targetSG, "2001:db8::/32", 80, "IPv6 HTTP")
	if len(targetSG.ingressRules) != 3 {
		t.Errorf("Expected 3 rules, got %d", len(targetSG.ingressRules))
	}

	// 测试 AllTraffic IPv4 规则
	GlobalRuleRegistry.AddAllTrafficIngressRuleFromCidrSafely(targetSG, "0.0.0.0/0", "All Traffic IPv4")
	if len(targetSG.ingressRules) != 4 {
		t.Errorf("Expected 4 rules, got %d", len(targetSG.ingressRules))
	}

	// 测试 AllTraffic IPv6 规则
	GlobalRuleRegistry.AddAllTrafficIngressRuleFromCidrIpv6Safely(targetSG, "::/0", "All Traffic IPv6")
	if len(targetSG.ingressRules) != 5 {
		t.Errorf("Expected 5 rules, got %d", len(targetSG.ingressRules))
	}
}

func TestHelperFunctions(t *testing.T) {
	// 重置全局注册表
	GlobalRuleRegistry.Reset()

	// 创建模拟安全组
	targetSG := &mockSecurityGroup{id: "sg-target"}
	sourceSG := &mockSecurityGroup{id: "sg-source"}

	// 测试安全组到安全组的辅助函数
	AddTcpIngressRule(targetSG, sourceSG, 22, "SG to SG SSH")
	if len(targetSG.ingressRules) != 1 {
		t.Errorf("Expected 1 rule, got %d", len(targetSG.ingressRules))
	}

	// 测试 IPv4 CIDR 辅助函数
	AddTcpIngressRuleFromCidr(targetSG, "192.168.1.0/24", 80, "IPv4 HTTP")
	if len(targetSG.ingressRules) != 2 {
		t.Errorf("Expected 2 rules, got %d", len(targetSG.ingressRules))
	}

	// 测试 IPv6 CIDR 辅助函数
	AddTcpIngressRuleFromCidrIpv6(targetSG, "2001:db8::/32", 443, "IPv6 HTTPS")
	if len(targetSG.ingressRules) != 3 {
		t.Errorf("Expected 3 rules, got %d", len(targetSG.ingressRules))
	}

	// 测试任意 IP 辅助函数
	AddTcpIngressRuleFromAnyIp(targetSG, 8080, "Any IPv4 to 8080")
	if len(targetSG.ingressRules) != 4 {
		t.Errorf("Expected 4 rules, got %d", len(targetSG.ingressRules))
	}

	// 测试任意 IPv6 辅助函数
	AddTcpIngressRuleFromAnyIpv6(targetSG, 8080, "Any IPv6 to 8080")
	if len(targetSG.ingressRules) != 5 {
		t.Errorf("Expected 5 rules, got %d", len(targetSG.ingressRules))
	}

	// 测试 AllTraffic 辅助函数
	AddAllTrafficIngressRule(targetSG, sourceSG, "All traffic from SG")
	if len(targetSG.ingressRules) != 6 {
		t.Errorf("Expected 6 rules, got %d", len(targetSG.ingressRules))
	}

	// 测试 AllTraffic IPv4 辅助函数
	AddAllTrafficIngressRuleFromAnyIp(targetSG, "All traffic from any IPv4")
	if len(targetSG.ingressRules) != 7 {
		t.Errorf("Expected 7 rules, got %d", len(targetSG.ingressRules))
	}

	// 测试 AllTraffic IPv6 辅助函数
	AddAllTrafficIngressRuleFromAnyIpv6(targetSG, "All traffic from any IPv6")
	if len(targetSG.ingressRules) != 8 {
		t.Errorf("Expected 8 rules, got %d", len(targetSG.ingressRules))
	}
}

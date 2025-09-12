// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package security

import (
	"fmt"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/jsii-runtime-go"
	"sync"
)

// SecurityGroupRuleRegistry 用于跟踪已添加的安全组规则
type SecurityGroupRuleRegistry struct {
	rules map[string]bool
	mutex sync.RWMutex
}

// 全局单例
var GlobalRuleRegistry = &SecurityGroupRuleRegistry{
	rules: make(map[string]bool),
}

// 生成安全组到安全组规则的唯一标识符
func (r *SecurityGroupRuleRegistry) generateSGRuleID(targetSG, sourceSG awsec2.SecurityGroup, protocol string, port int) string {
	targetID := *targetSG.SecurityGroupId()
	sourceID := *sourceSG.SecurityGroupId()
	return fmt.Sprintf("%s-%s-%s-%d", targetID, sourceID, protocol, port)
}

// 生成安全组到安全组规则的唯一标识符（端口范围）
func (r *SecurityGroupRuleRegistry) generateSGRangeRuleID(targetSG, sourceSG awsec2.SecurityGroup, protocol string, fromPort, toPort int) string {
	targetID := *targetSG.SecurityGroupId()
	sourceID := *sourceSG.SecurityGroupId()
	return fmt.Sprintf("%s-%s-%s-%d-%d", targetID, sourceID, protocol, fromPort, toPort)
}

// 生成 IPv4 CIDR 规则的唯一标识符
func (r *SecurityGroupRuleRegistry) generateCidrRuleID(targetSG awsec2.SecurityGroup, cidr string, protocol string, port int) string {
	targetID := *targetSG.SecurityGroupId()
	return fmt.Sprintf("%s-cidr:%s-%s-%d", targetID, cidr, protocol, port)
}

// 生成 IPv4 CIDR 规则的唯一标识符（端口范围）
func (r *SecurityGroupRuleRegistry) generateCidrRangeRuleID(targetSG awsec2.SecurityGroup, cidr string, protocol string, fromPort, toPort int) string {
	targetID := *targetSG.SecurityGroupId()
	return fmt.Sprintf("%s-cidr:%s-%s-%d-%d", targetID, cidr, protocol, fromPort, toPort)
}

// 生成 IPv6 CIDR 规则的唯一标识符
func (r *SecurityGroupRuleRegistry) generateCidrIpv6RuleID(targetSG awsec2.SecurityGroup, cidr string, protocol string, port int) string {
	targetID := *targetSG.SecurityGroupId()
	return fmt.Sprintf("%s-cidrv6:%s-%s-%d", targetID, cidr, protocol, port)
}

// 生成 IPv6 CIDR 规则的唯一标识符（端口范围）
func (r *SecurityGroupRuleRegistry) generateCidrIpv6RangeRuleID(targetSG awsec2.SecurityGroup, cidr string, protocol string, fromPort, toPort int) string {
	targetID := *targetSG.SecurityGroupId()
	return fmt.Sprintf("%s-cidrv6:%s-%s-%d-%d", targetID, cidr, protocol, fromPort, toPort)
}

// AddTcpIngressRuleSafely 安全地添加 TCP 入站规则，避免重复
func (r *SecurityGroupRuleRegistry) AddTcpIngressRuleSafely(targetSG, sourceSG awsec2.SecurityGroup, port int, description string) {
	// 生成规则 ID
	ruleID := r.generateSGRuleID(targetSG, sourceSG, "tcp", port)
	
	// 检查规则是否已存在
	r.mutex.RLock()
	exists := r.rules[ruleID]
	r.mutex.RUnlock()
	
	if !exists {
		// 添加规则
		targetSG.AddIngressRule(sourceSG, awsec2.Port_Tcp(jsii.Number(port)), jsii.String(description), nil)
		
		// 注册规则
		r.mutex.Lock()
		r.rules[ruleID] = true
		r.mutex.Unlock()
	}
}

// AddTcpRangeIngressRuleSafely 安全地添加 TCP 端口范围入站规则，避免重复
func (r *SecurityGroupRuleRegistry) AddTcpRangeIngressRuleSafely(targetSG, sourceSG awsec2.SecurityGroup, fromPort, toPort int, description string) {
	// 生成规则 ID
	ruleID := r.generateSGRangeRuleID(targetSG, sourceSG, "tcp", fromPort, toPort)
	
	// 检查规则是否已存在
	r.mutex.RLock()
	exists := r.rules[ruleID]
	r.mutex.RUnlock()
	
	if !exists {
		// 添加规则
		targetSG.AddIngressRule(sourceSG, awsec2.Port_TcpRange(jsii.Number(fromPort), jsii.Number(toPort)), jsii.String(description), nil)
		
		// 注册规则
		r.mutex.Lock()
		r.rules[ruleID] = true
		r.mutex.Unlock()
	}
}

// AddUdpIngressRuleSafely 安全地添加 UDP 入站规则，避免重复
func (r *SecurityGroupRuleRegistry) AddUdpIngressRuleSafely(targetSG, sourceSG awsec2.SecurityGroup, port int, description string) {
	// 生成规则 ID
	ruleID := r.generateSGRuleID(targetSG, sourceSG, "udp", port)
	
	// 检查规则是否已存在
	r.mutex.RLock()
	exists := r.rules[ruleID]
	r.mutex.RUnlock()
	
	if !exists {
		// 添加规则
		targetSG.AddIngressRule(sourceSG, awsec2.Port_Udp(jsii.Number(port)), jsii.String(description), nil)
		
		// 注册规则
		r.mutex.Lock()
		r.rules[ruleID] = true
		r.mutex.Unlock()
	}
}

// AddUdpRangeIngressRuleSafely 安全地添加 UDP 端口范围入站规则，避免重复
func (r *SecurityGroupRuleRegistry) AddUdpRangeIngressRuleSafely(targetSG, sourceSG awsec2.SecurityGroup, fromPort, toPort int, description string) {
	// 生成规则 ID
	ruleID := r.generateSGRangeRuleID(targetSG, sourceSG, "udp", fromPort, toPort)
	
	// 检查规则是否已存在
	r.mutex.RLock()
	exists := r.rules[ruleID]
	r.mutex.RUnlock()
	
	if !exists {
		// 添加规则
		targetSG.AddIngressRule(sourceSG, awsec2.Port_UdpRange(jsii.Number(fromPort), jsii.Number(toPort)), jsii.String(description), nil)
		
		// 注册规则
		r.mutex.Lock()
		r.rules[ruleID] = true
		r.mutex.Unlock()
	}
}

// AddAllTrafficIngressRuleSafely 安全地添加允许所有流量的入站规则，避免重复
func (r *SecurityGroupRuleRegistry) AddAllTrafficIngressRuleSafely(targetSG, sourceSG awsec2.SecurityGroup, description string) {
	// 生成规则 ID
	ruleID := r.generateSGRuleID(targetSG, sourceSG, "all", 0)
	
	// 检查规则是否已存在
	r.mutex.RLock()
	exists := r.rules[ruleID]
	r.mutex.RUnlock()
	
	if !exists {
		// 添加规则
		targetSG.AddIngressRule(sourceSG, awsec2.Port_AllTraffic(), jsii.String(description), nil)
		
		// 注册规则
		r.mutex.Lock()
		r.rules[ruleID] = true
		r.mutex.Unlock()
	}
}

// AddTcpIngressRuleFromCidrSafely 安全地添加来自 IPv4 CIDR 的 TCP 入站规则，避免重复
func (r *SecurityGroupRuleRegistry) AddTcpIngressRuleFromCidrSafely(targetSG awsec2.SecurityGroup, cidr string, port int, description string) {
	// 生成规则 ID
	ruleID := r.generateCidrRuleID(targetSG, cidr, "tcp", port)
	
	// 检查规则是否已存在
	r.mutex.RLock()
	exists := r.rules[ruleID]
	r.mutex.RUnlock()
	
	if !exists {
		// 添加规则
		targetSG.AddIngressRule(awsec2.Peer_Ipv4(jsii.String(cidr)), awsec2.Port_Tcp(jsii.Number(port)), jsii.String(description), nil)
		
		// 注册规则
		r.mutex.Lock()
		r.rules[ruleID] = true
		r.mutex.Unlock()
	}
}

// AddTcpRangeIngressRuleFromCidrSafely 安全地添加来自 IPv4 CIDR 的 TCP 端口范围入站规则，避免重复
func (r *SecurityGroupRuleRegistry) AddTcpRangeIngressRuleFromCidrSafely(targetSG awsec2.SecurityGroup, cidr string, fromPort, toPort int, description string) {
	// 生成规则 ID
	ruleID := r.generateCidrRangeRuleID(targetSG, cidr, "tcp", fromPort, toPort)
	
	// 检查规则是否已存在
	r.mutex.RLock()
	exists := r.rules[ruleID]
	r.mutex.RUnlock()
	
	if !exists {
		// 添加规则
		targetSG.AddIngressRule(awsec2.Peer_Ipv4(jsii.String(cidr)), awsec2.Port_TcpRange(jsii.Number(fromPort), jsii.Number(toPort)), jsii.String(description), nil)
		
		// 注册规则
		r.mutex.Lock()
		r.rules[ruleID] = true
		r.mutex.Unlock()
	}
}

// AddUdpIngressRuleFromCidrSafely 安全地添加来自 IPv4 CIDR 的 UDP 入站规则，避免重复
func (r *SecurityGroupRuleRegistry) AddUdpIngressRuleFromCidrSafely(targetSG awsec2.SecurityGroup, cidr string, port int, description string) {
	// 生成规则 ID
	ruleID := r.generateCidrRuleID(targetSG, cidr, "udp", port)
	
	// 检查规则是否已存在
	r.mutex.RLock()
	exists := r.rules[ruleID]
	r.mutex.RUnlock()
	
	if !exists {
		// 添加规则
		targetSG.AddIngressRule(awsec2.Peer_Ipv4(jsii.String(cidr)), awsec2.Port_Udp(jsii.Number(port)), jsii.String(description), nil)
		
		// 注册规则
		r.mutex.Lock()
		r.rules[ruleID] = true
		r.mutex.Unlock()
	}
}

// AddUdpRangeIngressRuleFromCidrSafely 安全地添加来自 IPv4 CIDR 的 UDP 端口范围入站规则，避免重复
func (r *SecurityGroupRuleRegistry) AddUdpRangeIngressRuleFromCidrSafely(targetSG awsec2.SecurityGroup, cidr string, fromPort, toPort int, description string) {
	// 生成规则 ID
	ruleID := r.generateCidrRangeRuleID(targetSG, cidr, "udp", fromPort, toPort)
	
	// 检查规则是否已存在
	r.mutex.RLock()
	exists := r.rules[ruleID]
	r.mutex.RUnlock()
	
	if !exists {
		// 添加规则
		targetSG.AddIngressRule(awsec2.Peer_Ipv4(jsii.String(cidr)), awsec2.Port_UdpRange(jsii.Number(fromPort), jsii.Number(toPort)), jsii.String(description), nil)
		
		// 注册规则
		r.mutex.Lock()
		r.rules[ruleID] = true
		r.mutex.Unlock()
	}
}

// AddAllTrafficIngressRuleFromCidrSafely 安全地添加来自 IPv4 CIDR 的所有流量入站规则，避免重复
func (r *SecurityGroupRuleRegistry) AddAllTrafficIngressRuleFromCidrSafely(targetSG awsec2.SecurityGroup, cidr string, description string) {
	// 生成规则 ID
	ruleID := r.generateCidrRuleID(targetSG, cidr, "all", 0)
	
	// 检查规则是否已存在
	r.mutex.RLock()
	exists := r.rules[ruleID]
	r.mutex.RUnlock()
	
	if !exists {
		// 添加规则
		targetSG.AddIngressRule(awsec2.Peer_Ipv4(jsii.String(cidr)), awsec2.Port_AllTraffic(), jsii.String(description), nil)
		
		// 注册规则
		r.mutex.Lock()
		r.rules[ruleID] = true
		r.mutex.Unlock()
	}
}

// AddTcpIngressRuleFromCidrIpv6Safely 安全地添加来自 IPv6 CIDR 的 TCP 入站规则，避免重复
func (r *SecurityGroupRuleRegistry) AddTcpIngressRuleFromCidrIpv6Safely(targetSG awsec2.SecurityGroup, cidr string, port int, description string) {
	// 生成规则 ID
	ruleID := r.generateCidrIpv6RuleID(targetSG, cidr, "tcp", port)
	
	// 检查规则是否已存在
	r.mutex.RLock()
	exists := r.rules[ruleID]
	r.mutex.RUnlock()
	
	if !exists {
		// 添加规则
		targetSG.AddIngressRule(awsec2.Peer_Ipv6(jsii.String(cidr)), awsec2.Port_Tcp(jsii.Number(port)), jsii.String(description), nil)
		
		// 注册规则
		r.mutex.Lock()
		r.rules[ruleID] = true
		r.mutex.Unlock()
	}
}

// AddTcpRangeIngressRuleFromCidrIpv6Safely 安全地添加来自 IPv6 CIDR 的 TCP 端口范围入站规则，避免重复
func (r *SecurityGroupRuleRegistry) AddTcpRangeIngressRuleFromCidrIpv6Safely(targetSG awsec2.SecurityGroup, cidr string, fromPort, toPort int, description string) {
	// 生成规则 ID
	ruleID := r.generateCidrIpv6RangeRuleID(targetSG, cidr, "tcp", fromPort, toPort)
	
	// 检查规则是否已存在
	r.mutex.RLock()
	exists := r.rules[ruleID]
	r.mutex.RUnlock()
	
	if !exists {
		// 添加规则
		targetSG.AddIngressRule(awsec2.Peer_Ipv6(jsii.String(cidr)), awsec2.Port_TcpRange(jsii.Number(fromPort), jsii.Number(toPort)), jsii.String(description), nil)
		
		// 注册规则
		r.mutex.Lock()
		r.rules[ruleID] = true
		r.mutex.Unlock()
	}
}

// AddUdpIngressRuleFromCidrIpv6Safely 安全地添加来自 IPv6 CIDR 的 UDP 入站规则，避免重复
func (r *SecurityGroupRuleRegistry) AddUdpIngressRuleFromCidrIpv6Safely(targetSG awsec2.SecurityGroup, cidr string, port int, description string) {
	// 生成规则 ID
	ruleID := r.generateCidrIpv6RuleID(targetSG, cidr, "udp", port)
	
	// 检查规则是否已存在
	r.mutex.RLock()
	exists := r.rules[ruleID]
	r.mutex.RUnlock()
	
	if !exists {
		// 添加规则
		targetSG.AddIngressRule(awsec2.Peer_Ipv6(jsii.String(cidr)), awsec2.Port_Udp(jsii.Number(port)), jsii.String(description), nil)
		
		// 注册规则
		r.mutex.Lock()
		r.rules[ruleID] = true
		r.mutex.Unlock()
	}
}

// AddUdpRangeIngressRuleFromCidrIpv6Safely 安全地添加来自 IPv6 CIDR 的 UDP 端口范围入站规则，避免重复
func (r *SecurityGroupRuleRegistry) AddUdpRangeIngressRuleFromCidrIpv6Safely(targetSG awsec2.SecurityGroup, cidr string, fromPort, toPort int, description string) {
	// 生成规则 ID
	ruleID := r.generateCidrIpv6RangeRuleID(targetSG, cidr, "udp", fromPort, toPort)
	
	// 检查规则是否已存在
	r.mutex.RLock()
	exists := r.rules[ruleID]
	r.mutex.RUnlock()
	
	if !exists {
		// 添加规则
		targetSG.AddIngressRule(awsec2.Peer_Ipv6(jsii.String(cidr)), awsec2.Port_UdpRange(jsii.Number(fromPort), jsii.Number(toPort)), jsii.String(description), nil)
		
		// 注册规则
		r.mutex.Lock()
		r.rules[ruleID] = true
		r.mutex.Unlock()
	}
}

// AddAllTrafficIngressRuleFromCidrIpv6Safely 安全地添加来自 IPv6 CIDR 的所有流量入站规则，避免重复
func (r *SecurityGroupRuleRegistry) AddAllTrafficIngressRuleFromCidrIpv6Safely(targetSG awsec2.SecurityGroup, cidr string, description string) {
	// 生成规则 ID
	ruleID := r.generateCidrIpv6RuleID(targetSG, cidr, "all", 0)
	
	// 检查规则是否已存在
	r.mutex.RLock()
	exists := r.rules[ruleID]
	r.mutex.RUnlock()
	
	if !exists {
		// 添加规则
		targetSG.AddIngressRule(awsec2.Peer_Ipv6(jsii.String(cidr)), awsec2.Port_AllTraffic(), jsii.String(description), nil)
		
		// 注册规则
		r.mutex.Lock()
		r.rules[ruleID] = true
		r.mutex.Unlock()
	}
}

// 出站规则 (Egress Rules)

// AddAllTrafficEgressRuleSafely 安全地添加允许所有出站流量的规则，避免重复
func (r *SecurityGroupRuleRegistry) AddAllTrafficEgressRuleSafely(sourceGroup, targetGroup awsec2.SecurityGroup, description string) {
	// 生成规则 ID
	ruleID := r.generateSGRuleID(sourceGroup, targetGroup, "egress-all", 0)
	
	// 检查规则是否已存在
	r.mutex.RLock()
	exists := r.rules[ruleID]
	r.mutex.RUnlock()
	
	if !exists {
		// 添加规则
		sourceGroup.AddEgressRule(targetGroup, awsec2.Port_AllTraffic(), jsii.String(description), nil)
		
		// 注册规则
		r.mutex.Lock()
		r.rules[ruleID] = true
		r.mutex.Unlock()
	}
}

// AddL1EgressRuleSafely 安全地添加使用 L1 构造函数的出站规则，避免重复
func (r *SecurityGroupRuleRegistry) AddEfaEgressRuleSafely(sourceGroup awsec2.SecurityGroup, description string) {
	// 生成规则 ID（自引用出站规则）
	ruleID := r.generateSGRuleID(sourceGroup, sourceGroup, "efa-l1-egress-all", 0)
	
	// 检查规则是否已存在
	r.mutex.RLock()
	exists := r.rules[ruleID]
	r.mutex.RUnlock()
	
	if !exists {
		// 使用 L1 构造函数添加出站规则
		stack := sourceGroup.Stack()
		sgId := sourceGroup.SecurityGroupId()
		
		awsec2.NewCfnSecurityGroupEgress(stack, jsii.String("EfaL1EgressRule-"+description), &awsec2.CfnSecurityGroupEgressProps{
			GroupId: sgId,
			IpProtocol: jsii.String("-1"),  // 所有协议
			DestinationSecurityGroupId: sgId,
			Description: jsii.String(description),
		})
		
		// 注册规则
		r.mutex.Lock()
		r.rules[ruleID] = true
		r.mutex.Unlock()
	}
}

func (r *SecurityGroupRuleRegistry) Reset() {
        r.mutex.Lock()
        defer r.mutex.Unlock()
        r.rules = make(map[string]bool)
}

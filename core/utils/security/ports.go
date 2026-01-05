// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package security

import (
	"strconv"
	"strings"
)

type PortRule struct {
	Port      int    `json:"port,omitempty"`
	FromPort  int    `json:"fromPort,omitempty"`
	ToPort    int    `json:"toPort,omitempty"`
	Cidr      string `json:"cidr"`
	Protocol  string `json:"protocol,omitempty"`
}

// ParseAllowedPorts 解析端口配置字符串
// 格式: "22@10.0.0.0/8;80,443@0.0.0.0/0;8000-8999@10.69.0.0/16"
// IPv6格式: "22@2406:da18::/32;80,443@::/0"
func ParseAllowedPorts(allowedPorts string) []PortRule {
	var rules []PortRule
	if allowedPorts == "" {
		return rules
	}
	
	for _, rule := range strings.Split(allowedPorts, ";") {
		parts := strings.Split(rule, "@")  // 改用@分隔符
		if len(parts) != 2 {
			continue
		}
		
		ports := strings.TrimSpace(parts[0])
		cidr := strings.TrimSpace(parts[1])
		
		for _, portStr := range strings.Split(ports, ",") {
			portStr = strings.TrimSpace(portStr)
			
			// 解析协议，默认为 tcp
			protocol := "tcp"
			if strings.Contains(portStr, "/") {
				parts := strings.Split(portStr, "/")
				if len(parts) == 2 {
					portStr = parts[0]
					protocol = strings.ToLower(strings.TrimSpace(parts[1]))
				}
			}
			
			if strings.Contains(portStr, "-") {
				// 端口范围
				rangeParts := strings.Split(portStr, "-")
				if len(rangeParts) == 2 {
					if fromPort, err := strconv.Atoi(strings.TrimSpace(rangeParts[0])); err == nil {
						if toPort, err := strconv.Atoi(strings.TrimSpace(rangeParts[1])); err == nil {
							rules = append(rules, PortRule{
								FromPort: fromPort,
								ToPort:   toPort,
								Cidr:     cidr,
								Protocol: protocol,
							})
						}
					}
				}
			} else {
				// 单个端口
				if port, err := strconv.Atoi(portStr); err == nil {
					rules = append(rules, PortRule{
						Port:     port,
						Cidr:     cidr,
						Protocol: protocol,
					})
				}
			}
		}
	}
	return rules
}

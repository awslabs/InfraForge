// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package list

import (
	"strconv"
	"strings"
)

// SplitGet 按分隔符拆分字符串，按索引取值，越界则复用最后一个值，空字符串返回空
func SplitGet(s, sep string, idx int) string {
	if s == "" {
		return ""
	}
	parts := strings.Split(s, sep)
	if idx < len(parts) {
		return strings.TrimSpace(parts[idx])
	}
	return strings.TrimSpace(parts[len(parts)-1])
}

// GetString 按逗号分隔取 string 值，越界广播最后一个值
func GetString(s string, idx int, defaultVal string) string {
	v := SplitGet(s, ",", idx)
	if v == "" {
		return defaultVal
	}
	return v
}

// GetInt 按逗号分隔取 int 值，越界广播最后一个值
func GetInt(s string, idx int, defaultVal int) int {
	v := SplitGet(s, ",", idx)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	return n
}

// GetBool 按逗号分隔取 bool 值，越界广播最后一个值
func GetBool(s string, idx int, defaultVal bool) bool {
	v := SplitGet(s, ",", idx)
	if v == "" {
		return defaultVal
	}
	return strings.EqualFold(v, "true")
}

// GetGroup 按分号分隔取值（用于 group/partition 级别），越界广播最后一段
func GetGroup(s string, idx int) string {
	return SplitGet(s, ";", idx)
}

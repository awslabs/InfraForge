// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package types

// GetIntValue 获取指针int值，如果为nil或0则返回默认值
func GetIntValue(ptr *int, defaultVal int) int {
	if ptr != nil && *ptr != 0 {
		return *ptr
	}
	return defaultVal
}

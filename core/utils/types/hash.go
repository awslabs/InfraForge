// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package types

import "fmt"

// HashString 创建字符串的简单哈希，用于资源名称
func HashString(s string) string {
	var hash uint32 = 0
	for i := 0; i < len(s); i++ {
		hash = hash*31 + uint32(s[i])
	}
	return fmt.Sprintf("%x", hash)
}

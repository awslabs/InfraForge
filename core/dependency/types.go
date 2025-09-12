// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package dependency

import "errors"

// 定义错误
var (
	ErrResourceNotFound = errors.New("resource not found")
	ErrInvalidFormat   = errors.New("invalid format")
)

// 定义一个新的结构体来解析返回的 JSON
type DependenciesResponse struct {
	Dependencies map[string]*ResourceInfo `json:"dependencies"`
}

// 通用资源信息结构
type ResourceInfo struct {
	Type       string                 `json:"type"`
	Id         string                 `json:"id"`
	Properties map[string]interface{} `json:"properties"`
}


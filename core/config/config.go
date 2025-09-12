// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"encoding/json"
)

type Config struct {
	Global        GlobalConfig   `json:"global"`
	EnabledForges []string       `json:"enabledForges"`
	Forges        map[string]ForgeConfig `json:"forges"`
}

type GlobalConfig struct {
	StackName	string `json:"stackName"`
	Description	string `json:"description"`
	DualStack	bool   `json:"dualStack"`
}

type ForgeConfig struct {
	Defaults json.RawMessage    `json:"defaults"`
	Instances []json.RawMessage `json:"instances"`
}

type InstanceConfig interface {
	GetID() string
	GetType() string
	GetSubnet() string
	GetSecurityGroup() string
}

type BaseInstanceConfig struct {
	ID            string `json:"id"`
	Type          string `json:"type"`
	Subnet        string `json:"subnet"`
	SecurityGroup string `json:"security"`
}

func (c *BaseInstanceConfig) GetID() string {
	return c.ID
}

func (c *BaseInstanceConfig) SetID(id string) string {
	c.ID = id
	return c.ID
}

func (c *BaseInstanceConfig) GetType() string {
	return c.Type
}

func (c *BaseInstanceConfig) GetSubnet() string {
	return c.Subnet
}

func (c *BaseInstanceConfig) GetSecurityGroup() string {
	return c.SecurityGroup
}

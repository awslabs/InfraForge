// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/jsii-runtime-go"
)

type UserDataGenerator struct {
	OsType             awsec2.OperatingSystemType
	ScriptPath         string
	UserDataToken      string
	UserDataScriptPath string
	MagicToken         string
	S3Location         string
}

func (g *UserDataGenerator) GenerateUserData() (awsec2.UserData, error) {
	var replacedScript string

	scriptContent, err := ioutil.ReadFile(g.ScriptPath)
	if err != nil {
		replacedScript = "echo Welcome to Infra Forge"
		fmt.Printf("Warning: Failed to read script file %s: %v\n", g.ScriptPath, err)
	} else {
		// 使用 map 处理替换
		replacedScript = string(scriptContent)
		replacements := map[string]string{
			"{{userDataToken}}": g.UserDataToken,
			"{{magicToken}}":    g.MagicToken,
			"{{s3Location}}":    g.S3Location,
			"{{customUserDataLocation}}": g.UserDataScriptPath,
		}
		
		for old, new := range replacements {
			if new != "" {
				replacedScript = strings.ReplaceAll(replacedScript, old, new)
			}
		}
	}

	var userData awsec2.UserData
	switch g.OsType {
	case awsec2.OperatingSystemType_LINUX:
		linuxUserDataOptions := &awsec2.LinuxUserDataOptions{
			Shebang: jsii.String("#!/bin/bash"),
		}
		userData = awsec2.UserData_ForLinux(linuxUserDataOptions)
		userData.AddCommands(jsii.String(replacedScript))
	case awsec2.OperatingSystemType_WINDOWS:
		windowsUserDataOptions := &awsec2.WindowsUserDataOptions{
			Persist: jsii.Bool(false),
		}
		userData = awsec2.UserData_ForWindows(windowsUserDataOptions)
		userData.AddCommands(jsii.String(replacedScript))
	default:
		return nil, fmt.Errorf("unsupported operating system type: %v", g.OsType)
	}

	return userData, nil
}

// GenerateMimeMultipartUserData 生成 MIME multipart 格式的 UserData（用于 Batch Launch Template）
func (g *UserDataGenerator) GenerateMimeMultipartUserData() (awsec2.UserData, error) {
	var replacedScript string

	scriptContent, err := ioutil.ReadFile(g.ScriptPath)
	if err != nil {
		replacedScript = "echo Welcome to Infra Forge"
		fmt.Printf("Warning: Failed to read script file %s: %v\n", g.ScriptPath, err)
	} else {
		replacedScript = string(scriptContent)
		replacements := map[string]string{
			"{{userDataToken}}": g.UserDataToken,
			"{{magicToken}}":    g.MagicToken,
			"{{s3Location}}":    g.S3Location,
			"{{customUserDataLocation}}": g.UserDataScriptPath,
		}
		
		for old, new := range replacements {
			if new != "" {
				replacedScript = strings.ReplaceAll(replacedScript, old, new)
			}
		}
	}

	// 生成 MIME multipart 格式
	boundary := "==BOUNDARY=="
	mimeContent := fmt.Sprintf(`Content-Type: multipart/mixed; boundary="%s"

--%s
Content-Type: text/x-shellscript

%s
--%s--`, boundary, boundary, replacedScript, boundary)

	return awsec2.UserData_Custom(jsii.String(mimeContent)), nil
}

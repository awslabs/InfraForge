// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"fmt"
	"os"
	
	"github.com/awslabs/InfraForge/core/config"
	"github.com/awslabs/InfraForge/core/manager"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/jsii-runtime-go"
)

var (
	Version   = "dev"
	BuildDate = "unknown"
	GitCommit = "unknown"
)

func main() {
	defer jsii.Close()

	// Parse command line flags
	var showVersion = flag.Bool("version", false, "Show version information")
	flag.Parse()

	if *showVersion {
		fmt.Printf("InfraForge %s\n", Version)
		fmt.Printf("Build Date: %s\n", BuildDate)
		fmt.Printf("Git Commit: %s\n", GitCommit)
		os.Exit(0)
	}

	/*
	// 防止 CDK 重复执行，虽可以提速，但有可能导致资源被清理
	if os.Getenv("CDK_CONTEXT_JSON") != "" {
		return
	}
	*/

	// 加载配置
	infraConfig, err := config.LoadConfig("config.json")
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	// 创建 CDK 应用和堆栈
	app := awscdk.NewApp(nil)
	stack := awscdk.NewStack(app, jsii.String(infraConfig.Global.StackName), &awscdk.StackProps{})

	// 创建 ForgeManager
	forgeManager := manager.NewForgeManager(stack, infraConfig.Global.DualStack)

	// 创建 VPC
	if err := forgeManager.CreateVPC(infraConfig); err != nil {
		fmt.Printf("Error creating VPC: %v\n", err)
		os.Exit(1)
	}

	// 处理所有启用的 forges
	for _, instanceId := range infraConfig.EnabledForges {
		if err := forgeManager.CreateForge(instanceId, infraConfig); err != nil {
			fmt.Printf("Error creating forge %s: %v\n", instanceId, err)
			os.Exit(1)
		}
	}

	// 合成 CloudFormation 模板
	app.Synth(nil)
}

// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"fmt"
	"context"
	"strings"

	"github.com/awslabs/InfraForge/core/interfaces"
	"github.com/awslabs/InfraForge/core/config"
	"github.com/awslabs/InfraForge/core/partition"
	"github.com/awslabs/InfraForge/forges/aws/storage/efs"
	"github.com/awslabs/InfraForge/forges/aws/storage/lustre"
	"github.com/awslabs/InfraForge/forges/aws/ec2"
	"github.com/awslabs/InfraForge/forges/aws/ecs"
	"github.com/awslabs/InfraForge/forges/aws/eks"
	"github.com/awslabs/InfraForge/forges/aws/ds"
	"github.com/awslabs/InfraForge/forges/aws/rds"
	"github.com/awslabs/InfraForge/forges/aws/vpc"
	"github.com/awslabs/InfraForge/forges/aws/parallelcluster"
	"github.com/awslabs/InfraForge/forges/aws/batch"
	"github.com/awslabs/InfraForge/forges/aws/hyperpod"
	iconfig "github.com/aws/aws-sdk-go-v2/config"
)

var ForgeConstructors = make(map[string]func() interfaces.Forge)

func RegisterForge(name string, constructor func() interfaces.Forge) {
        if constructor == nil {
		panic(fmt.Sprintf("Constructor for forge type %q is nil", name))
	}
	ForgeConstructors[name] = constructor
}


var instanceCreators = make(map[string]func() config.InstanceConfig)

func RegisterInstanceCreator(typ string, creator func() config.InstanceConfig) {
	if _, exists := instanceCreators[typ]; !exists {
		instanceCreators[typ] = creator
	} else {
		panic(fmt.Sprintf("Instance creator for type %q already registered", typ))
	}
}

func CreateInstance(typ string) config.InstanceConfig {
	creator, ok := instanceCreators[typ]
	if !ok {
		return nil
	}
	return creator()
}


func init() {
	cfg, err := iconfig.LoadDefaultConfig(context.TODO())
	if err != nil {
		panic(err)
	}

	// 获取当前的 AWS 区域
	currentRegion := cfg.Region
	partition.DefaultRegion = currentRegion

	// 根据区域前缀设置 DefaultPartition
	if strings.HasPrefix(currentRegion, "cn-") {
		partition.DefaultPartition = "aws-cn"
	} else {
		partition.DefaultPartition = "aws"
	}
	

        RegisterInstanceCreator("vpc", func() config.InstanceConfig {
                return &vpc.VpcInstanceConfig{}
        })
        RegisterInstanceCreator("efs", func() config.InstanceConfig {
                return &efs.EfsInstanceConfig{}
        })
        RegisterInstanceCreator("lustre", func() config.InstanceConfig {
                return &lustre.LustreInstanceConfig{}
        })
        RegisterInstanceCreator("ec2", func() config.InstanceConfig {
                return &ec2.Ec2InstanceConfig{}
        })
        RegisterInstanceCreator("ecs", func() config.InstanceConfig {
                return &ecs.EcsInstanceConfig{}
        })
        RegisterInstanceCreator("eks", func() config.InstanceConfig {
                return &eks.EksInstanceConfig{}
        })
        RegisterInstanceCreator("parallelcluster", func() config.InstanceConfig {
                return &parallelcluster.ParallelClusterInstanceConfig{}
        })
        RegisterInstanceCreator("ds", func() config.InstanceConfig {
                return &ds.DsInstanceConfig{}
        })
        RegisterInstanceCreator("rds", func() config.InstanceConfig {
                return &rds.RdsInstanceConfig{}
        })
        RegisterInstanceCreator("batch", func() config.InstanceConfig {
                return &batch.BatchInstanceConfig{}
        })
        RegisterInstanceCreator("hyperpod", func() config.InstanceConfig {
                return &hyperpod.HyperPodInstanceConfig{}
        })


        RegisterForge("vpc", func() interfaces.Forge { return &vpc.VpcForge{} })
        RegisterForge("efs", func() interfaces.Forge { return &efs.EfsForge{} })
        RegisterForge("lustre", func() interfaces.Forge { return &lustre.LustreForge{} })
        RegisterForge("ec2", func() interfaces.Forge { return &ec2.Ec2Forge{} })
        RegisterForge("ecs", func() interfaces.Forge { return &ecs.EcsForge{} })
        RegisterForge("eks", func() interfaces.Forge { return &eks.EksForge{} })
        RegisterForge("parallelcluster", func() interfaces.Forge { return &parallelcluster.ParallelClusterForge{} })
        RegisterForge("ds", func() interfaces.Forge { return &ds.DsForge{} })
        RegisterForge("rds", func() interfaces.Forge { return &rds.RdsForge{} })
        RegisterForge("batch", func() interfaces.Forge { return &batch.BatchForge{} })
        RegisterForge("hyperpod", func() interfaces.Forge { return &hyperpod.HyperPodForge{} })

	
	// 资源处理器已移除 - 直接使用 Forge.GetProperties()

}

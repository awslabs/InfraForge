// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package iam

import (
	"fmt"
	"github.com/awslabs/InfraForge/core/partition"
        "github.com/aws/aws-cdk-go/awscdk/v2"
        "github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
        "github.com/aws/jsii-runtime-go"
)

func CreateDCVLicensingPolicy(stack awscdk.Stack) awsiam.ManagedPolicy {
        policyName := jsii.String(fmt.Sprintf("%s-DCVLicensingPolicy-%s", *stack.StackName(), partition.DefaultRegion))
        policyStatement := awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
                Effect: awsiam.Effect_ALLOW,
                Actions: &[]*string{
                        jsii.String("s3:GetObject"),
                },
                Resources: &[]*string{
                        jsii.String(fmt.Sprintf("arn:%s:s3:::dcv-license.%s/*", *awscdk.Aws_PARTITION(), *awscdk.Aws_REGION())),
                        /*
                        awscdk.Fn_Sub(
                                jsii.String("arn:${Partition}:s3:::dcv-license.${Region}/*"),
                                &map[string]*string{
                                        "Partition": stack.Partition(),
                                        "Region":    stack.Region(),
                                },
                        ),
                        */
                },
        })

        policy := awsiam.NewManagedPolicy(stack, policyName, &awsiam.ManagedPolicyProps{
                ManagedPolicyName: jsii.String(fmt.Sprintf("%s-DCVLicensingPolicy-%s", *stack.StackName(), partition.DefaultRegion)),
                Description:      jsii.String("Policy for accessing DCV license bucket"),
                Statements:       &[]awsiam.PolicyStatement{policyStatement},
        })

        return policy
}

func CreateDCVOutputs(stack awscdk.Stack, policy awsiam.ManagedPolicy) {
	awscdk.NewCfnOutput(stack, jsii.String("DCVLicensingPolicy" + "-" + partition.DefaultRegion), &awscdk.CfnOutputProps{
                Value:       policy.ManagedPolicyArn(),
                Description: jsii.String("A reference to the created DCVLicensingPolicy" + "-" + partition.DefaultRegion),
        })
}


func CreateInstanceRole(stack awscdk.Stack) awsiam.IRole {
	instanceRole := awsiam.NewRole(stack, jsii.String(fmt.Sprintf("%s-instance-role-%s", *stack.StackName(), partition.DefaultRegion)), &awsiam.RoleProps{
		AssumedBy: awsiam.NewServicePrincipal(jsii.String("ec2.amazonaws.com"), nil),
	})
	return instanceRole
}

func CreateInstanceProfile(stack awscdk.Stack, iRole awsiam.IRole) awsiam.InstanceProfile {
	instanceProfile := awsiam.NewInstanceProfile(stack, jsii.String(fmt.Sprintf("%s-instance-profile-%s", *stack.StackName(), partition.DefaultRegion)), &awsiam.InstanceProfileProps{
		Role: iRole,
		InstanceProfileName: jsii.String(fmt.Sprintf("%s-instance-profile-%s", *stack.StackName(), partition.DefaultRegion)),
	})

	return instanceProfile
}

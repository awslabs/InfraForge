package eks

import (
	"encoding/json"
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awseks"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/aws-cdk-go/awscdk/v2/customresources"
	"github.com/awslabs/InfraForge/forges/aws/eks/utils"
	"github.com/aws/jsii-runtime-go"
)

// KubectlExecutorProps kubectl执行器配置
type KubectlExecutorProps struct {
	Cluster        awseks.ICluster
	EksVersion     string
	OnCreateCmds   []string // 创建时执行的命令
	OnUpdateCmds   []string // 更新时执行的命令  
	OnDeleteCmds   []string // 删除时执行的命令
}

// CreateKubectlExecutor 创建可复用的kubectl执行器
func CreateKubectlExecutor(stack awscdk.Stack, id string, props *KubectlExecutorProps) customresources.AwsCustomResource {
	// 为Lambda创建独立的执行角色，包含所有必要权限
	lambdaRole := awsiam.NewRole(stack, jsii.String(id+"LambdaRole"), &awsiam.RoleProps{
		AssumedBy: awsiam.NewServicePrincipal(jsii.String("lambda.amazonaws.com"), nil),
		ManagedPolicies: &[]awsiam.IManagedPolicy{
			awsiam.ManagedPolicy_FromAwsManagedPolicyName(jsii.String("service-role/AWSLambdaBasicExecutionRole")),
		},
		InlinePolicies: &map[string]awsiam.PolicyDocument{
			"EKSKubectlPolicy": awsiam.NewPolicyDocument(&awsiam.PolicyDocumentProps{
				Statements: &[]awsiam.PolicyStatement{
					awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
						Effect: awsiam.Effect_ALLOW,
						Actions: jsii.Strings(
							"eks:DescribeCluster",
							"eks:ListClusters",
							"sts:GetCallerIdentity",
						),
						Resources: jsii.Strings("*"),
					}),
				},
			}),
		},
	})

	// 创建kubectl执行Lambda
	kubectlLambda := awslambda.NewFunction(stack, jsii.String(id+"Lambda"), &awslambda.FunctionProps{
		Runtime: awslambda.Runtime_PYTHON_3_13(),
		Handler: jsii.String("index.handler"),
		Code: awslambda.Code_FromInline(jsii.String(`
import subprocess
import json
import logging
import os

logger = logging.getLogger()
logger.setLevel(logging.INFO)

def handler(event, context):
	logger.info(f"Event: {json.dumps(event)}")
	
	request_type = event.get('RequestType', 'Create')
	resource_props = event.get('ResourceProperties', {})
	
	try:
		import subprocess
		import os
		
		# 安装 AWS CLI 到 /tmp 目录
		install_cmd = "pip install awscli --target /tmp/awscli --quiet"
		install_result = subprocess.run(install_cmd, shell=True, capture_output=True, text=True, timeout=120)
		if install_result.returncode != 0:
			raise Exception(f"AWS CLI installation failed: {install_result.stderr}")
		
		# 添加 AWS CLI 到 PATH
		os.environ['PATH'] = '/tmp/awscli/bin:' + os.environ.get('PATH', '')
		os.environ['PYTHONPATH'] = '/tmp/awscli:' + os.environ.get('PYTHONPATH', '')
		
		cluster_name = os.environ.get('CLUSTER_NAME')
		region = os.environ.get('AWS_REGION', 'us-east-1')
		
		# 直接使用 Lambda 自己的权限配置 kubectl（不使用 mastersRole）
		config_cmd = f"aws eks update-kubeconfig --name {cluster_name} --region {region} --kubeconfig /tmp/kubeconfig"
		config_result = subprocess.run(config_cmd, shell=True, capture_output=True, text=True, timeout=60)
		
		if config_result.returncode != 0:
			logger.warning(f"kubectl configuration failed: {config_result.stderr}")
			# 如果配置失败，直接返回成功（可能是权限已被删除）
			return {'Status': 'SUCCESS', 'PhysicalResourceId': f'kubectl-executor-{id}'}
		
		os.environ['KUBECONFIG'] = '/tmp/kubeconfig'
		os.environ['PATH'] = '/opt/kubectl:/opt/helm:' + os.environ.get('PATH', '')
		
		# 根据请求类型获取命令
		if request_type == 'Create':
			commands = resource_props.get('OnCreateCmds', [])
		elif request_type == 'Update':
			commands = resource_props.get('OnUpdateCmds', [])
		elif request_type == 'Delete':
			commands = resource_props.get('OnDeleteCmds', [])
		else:
			commands = []
		
		logger.info(f"Executing {len(commands)} commands for {request_type}")
		
		# 执行命令
		for i, cmd in enumerate(commands):
			logger.info(f"Executing command {i+1}/{len(commands)}: {cmd}")
			result = subprocess.run(cmd, shell=True, capture_output=True, text=True, timeout=300)
			
			if result.stdout:
				logger.info(f"Command {i+1} output: {result.stdout}")
			if result.stderr:
				logger.warning(f"Command {i+1} stderr: {result.stderr}")
			if result.returncode != 0:
				logger.warning(f"Command {i+1} failed with return code {result.returncode}")
		
		return {
			'Status': 'SUCCESS',
			'PhysicalResourceId': f'kubectl-executor-{id}'
		}
		
	except Exception as e:
		logger.error(f"Error: {str(e)}")
		return {
			'Status': 'FAILED',
			'Reason': str(e),
			'PhysicalResourceId': f'kubectl-executor-{id}'
		}
		`)),
		Role: lambdaRole, // 使用独立的Lambda执行角色
		Environment: &map[string]*string{
			"CLUSTER_NAME":      props.Cluster.ClusterName(),
			"CLUSTER_ENDPOINT":  props.Cluster.ClusterEndpoint(),
		},
		Layers: &[]awslambda.ILayerVersion{
			utils.GetKubectlLayer(stack, id+"KubectlLayer", props.EksVersion),
		},
		Timeout: awscdk.Duration_Minutes(jsii.Number(10)),
	})

	// 将 Lambda 角色添加到 EKS 集群访问控制
	if cluster, ok := props.Cluster.(awseks.Cluster); ok {
		cluster.AwsAuth().AddRoleMapping(lambdaRole, &awseks.AwsAuthMapping{
			Groups: jsii.Strings("system:masters"),
		})
	}

	// 构建自定义资源配置
	customResourceProps := &customresources.AwsCustomResourceProps{
		Policy: customresources.AwsCustomResourcePolicy_FromStatements(&[]awsiam.PolicyStatement{
			awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
				Effect: awsiam.Effect_ALLOW,
				Actions: jsii.Strings("lambda:InvokeFunction"),
				Resources: jsii.Strings(*kubectlLambda.FunctionArn()),
			}),
		}),
		Timeout: awscdk.Duration_Minutes(jsii.Number(15)),
		InstallLatestAwsSdk: jsii.Bool(false), // 使用Lambda内置SDK，启动更快更稳定
	}

	// 添加onCreate配置
	if len(props.OnCreateCmds) > 0 {
		payloadData := map[string]interface{}{
			"RequestType": "Create",
			"ResourceProperties": map[string]interface{}{
				"OnCreateCmds": props.OnCreateCmds,
			},
		}
		payloadBytes, _ := json.Marshal(payloadData)
		
		customResourceProps.OnCreate = &customresources.AwsSdkCall{
			Service: jsii.String("Lambda"),
			Action:  jsii.String("Invoke"),
			Parameters: &map[string]interface{}{
				"FunctionName": kubectlLambda.FunctionArn(),
				"Payload":      jsii.String(string(payloadBytes)),
			},
		}
	}

	// 添加onUpdate配置
	if len(props.OnUpdateCmds) > 0 {
		payloadData := map[string]interface{}{
			"RequestType": "Update",
			"ResourceProperties": map[string]interface{}{
				"OnUpdateCmds": props.OnUpdateCmds,
			},
		}
		payloadBytes, _ := json.Marshal(payloadData)
		
		customResourceProps.OnUpdate = &customresources.AwsSdkCall{
			Service: jsii.String("Lambda"),
			Action:  jsii.String("Invoke"),
			Parameters: &map[string]interface{}{
				"FunctionName": kubectlLambda.FunctionArn(),
				"Payload":      jsii.String(string(payloadBytes)),
			},
		}
	}

	// 添加onDelete配置
	if len(props.OnDeleteCmds) > 0 {
		payloadData := map[string]interface{}{
			"RequestType": "Delete",
			"ResourceProperties": map[string]interface{}{
				"OnDeleteCmds": props.OnDeleteCmds,
			},
		}
		payloadBytes, _ := json.Marshal(payloadData)
		
		customResourceProps.OnDelete = &customresources.AwsSdkCall{
			Service: jsii.String("Lambda"),
			Action:  jsii.String("Invoke"),
			Parameters: &map[string]interface{}{
				"FunctionName": kubectlLambda.FunctionArn(),
				"Payload":      jsii.String(string(payloadBytes)),
			},
		}
	}

	return customresources.NewAwsCustomResource(stack, jsii.String(id), customResourceProps)
}

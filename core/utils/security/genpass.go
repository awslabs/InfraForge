// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package security

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"sync"

	//"github.com/awslabs/InfraForge/core/partition"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsssm"
	"github.com/aws/aws-cdk-go/awscdk/v2/awssecretsmanager"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/jsii-runtime-go"
)

// 密码缓存和互斥锁
var passwordCache = make(map[string]string)
var passwordMutex sync.Mutex

// generateSecurePassword 生成一个符合常见密码复杂性要求的随机密码
// length: 密码长度
// 返回: 随机生成的密码字符串
func generateSecurePassword(length int) string {
	const (
		lowerChars   = "abcdefghijklmnopqrstuvwxyz"
		upperChars   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		digitChars   = "0123456789"

		// slurmdbd 密码中不能有 #
		// sed 中不能有 & < > 
		// 不含转义字符和引号 \ ' "
		// 减去号在最前可能导致识别为命令行参数 -
		// RDS 不允许 / @ " 空格
		// 不包含 '"#&-\<>/@
		specialChars = "`~!$%^*()_=+[]{}|;:,.?"
	)

	// 合并所有字符集
	allChars := lowerChars + upperChars + digitChars + specialChars

	// 创建密码构建器
	var password strings.Builder
	password.Grow(length)

	// 确保密码包含至少一个字符从每个字符集
	must := []string{
		getRandomChar(lowerChars),
		getRandomChar(upperChars),
		getRandomChar(digitChars),
		getRandomChar(specialChars),
	}

	// 添加必需的字符
	for _, ch := range must {
		password.WriteString(ch)
	}

	// 填充剩余长度的随机字符
	for i := len(must); i < length; i++ {
		password.WriteString(getRandomChar(allChars))
	}

	// 将密码转换为字节切片以便洗牌
	result := []rune(password.String())

	// Fisher-Yates 洗牌算法打乱密码字符顺序
	for i := len(result) - 1; i > 0; i-- {
		j, _ := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		result[i], result[j.Int64()] = result[j.Int64()], result[i]
	}

	return string(result)
}

// getRandomChar 从给定字符串中随机选择一个字符
func getRandomChar(charset string) string {
	if len(charset) == 0 {
		return ""
	}

	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
	if err != nil {
		panic(err) // 在生产环境中应适当处理错误
	}

	return string(charset[n.Int64()])
}

// 检查 SSM Parameter 是否存在
func checkParameterExists(paramName string) bool {
	// 创建 AWS 配置
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return false
	}

	// 创建 SSM 客户端
	ssmClient := ssm.NewFromConfig(cfg)

	// 检查参数是否存在
	_, err = ssmClient.GetParameter(context.TODO(), &ssm.GetParameterInput{
		Name: &paramName,
	})

	if err != nil {
		// 参数不存在或其他错误
		return false
	}

	// 参数存在
	return true
}

// 检查 Secret 是否存在
func checkSecretExists(secretName string) bool {
	// 创建 AWS 配置
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return false
	}

	// 创建 Secrets Manager 客户端
	secretsClient := secretsmanager.NewFromConfig(cfg)

	// 尝试描述 Secret
	_, err = secretsClient.DescribeSecret(context.TODO(), &secretsmanager.DescribeSecretInput{
		SecretId: &secretName,
	})

	if err != nil {
		// Secret 不存在或其他错误
		return false
	}

	// Secret 存在
	return true
}

// GetOrCreateSecretPassword 创建或获取一个 Secret，并直接返回密码和 Secret 对象
// 
// 重要说明：此函数使用UnsafePlainText存储密码，适用于需要明文密码的场景
// 
// 典型使用场景：
// 1. AWS ParallelCluster: 要求密码必须以明文形式存储在Secrets Manager中
//    参考文档：https://docs.aws.amazon.com/parallelcluster/latest/ug/Scheduling-v3.html#yaml-Scheduling-SlurmSettings-Database-PasswordSecretArn
//    "When you create a secret using the AWS Secrets Manager console be sure to select 
//    'Other type of secret', select plaintext, and only include the password text in the secret."
// 2. Grafana等应用：需要在Helm配置中直接使用明文密码
//
// 参数:
//   - stack: CDK 堆栈
//   - id: 构造 ID
//   - secretName: Secret 名称
//   - description: Secret 描述信息
//   - length: 如果需要创建新密码，指定密码长度
// 返回:
//   - string: 密码明文
//   - awssecretsmanager.Secret: Secret 对象
func GetOrCreateSecretPassword(stack awscdk.Stack, id string, secretName string, description string, length int) (string, awssecretsmanager.Secret) {
	var password string
	var secret awssecretsmanager.Secret

	// 检查Secret是否存在
	secretExists := checkSecretExists(secretName)

	if secretExists {
		// 使用AWS SDK直接获取密码值
		cfg, err := config.LoadDefaultConfig(context.TODO())
		if err == nil {
			/* 

			当Secret已存在时，使用 awssecretsmanager.Secret_FromSecretNameV2 获取的 Secret 对象在CDK 合成阶段
			并不会包含实际的密码值。 这是因为CDK合成阶段只是生成CloudFormation模板，而不会实际访问 AWS 服务来
			获取密码值。

			secretValue.ToString()在合成阶段不会返回实际的密码，而是一个引用或占位符。这会导致两个问题：
			1. 返回的密码值可能不是实际的密码
			2. 在代码中使用这个值可能会导致意外行为
			
			解决方案是，当Secret已存在时，使用AWS SDK直接获取密码值，而不是依赖CDK的SecretValue。

			所以不能使用 Secret_FromSecretNameV2 或者类似方式，因为第一次通过 generateSecurePassword 获得真实密码, 而
			第二次获得的 通过 Secret_FromSecretNameV2 在合成阶段并不会包含实际的密码值。

			下面是个错误的示例:
			iSecret := awssecretsmanager.Secret_FromSecretNameV2(stack, jsii.String(id+"Ref"), jsii.String(secretName))
			secretValue := iSecret.SecretValue()
			password = *secretValue.ToString()

			*/

			secretsClient := secretsmanager.NewFromConfig(cfg)
			result, err := secretsClient.GetSecretValue(context.TODO(), &secretsmanager.GetSecretValueInput{
				SecretId: &secretName,
			})
			if err == nil && result.SecretString != nil {
				password = *result.SecretString
			} else {
				// 如果无法获取密码，生成一个新的
				password = generateSecurePassword(length)
			}
		} else {
			// 如果无法访问AWS，生成一个新的密码
			password = generateSecurePassword(length)
		}
	} else {
		// 生成新密码
		password = generateSecurePassword(length)
	}

	// 如果通过条件判断是否创建 NewSecret
	// 1. 第一次运行时，由于密码不存在，代码会创建 NewSecret，这个参数会被记录为 CloudFormation 堆栈的一部分
	// 2. 第二次运行时，检测到密码已存在，代码如果不调用 NewSecret 而引用现有参数 CloudFormation 会认为这个参数不再是堆栈创建的资源
	//（因为现在是引用而不是创建），所以会尝试删除它
	// 这是 CDK 和 CloudFormation 资源管理的一个常见陷阱。要解决这个问题，我们需要确保无论参数是否存在，都使用相同的 CDK 构造方式。
	// 无论参数是否存在，都使用 NewSecret 在 CloudFormation 模板中创建参数引用

	// 创建或更新 SSM 参数, 这样保证第二次运行，还是认为这个属于此 stack
	// 无论Secret是否存在，都创建一个新的Secret构造
	// 这样可以确保它属于当前堆栈
	// 创建Secret - 使用UnsafePlainText存储明文密码
	secret = awssecretsmanager.NewSecret(stack, jsii.String(id), &awssecretsmanager.SecretProps{
		SecretName: jsii.String(secretName),
		Description: jsii.String(description),
		SecretStringValue: awscdk.SecretValue_UnsafePlainText(jsii.String(password)),
	})

	return password, secret
}

func GetOrCreatePassword(stack awscdk.Stack, id string, paramName string, length int) string {
	// 构建缓存键
	cacheKey := paramName

	// 获取锁
	passwordMutex.Lock()
	defer passwordMutex.Unlock()

	// 检查缓存中是否已经存在该密码
	if cachedPassword, ok := passwordCache[cacheKey]; ok {
		return cachedPassword
	}

	// 检查 SSM Parameter Store 中是否已存在该参数
	paramExists := checkParameterExists(paramName)

	var password string
	if paramExists {
		// 如果参数已存在，获取其值
		cfg, err := config.LoadDefaultConfig(context.TODO())
		if err == nil {
			ssmClient := ssm.NewFromConfig(cfg)
			result, err := ssmClient.GetParameter(context.TODO(), &ssm.GetParameterInput{
				Name: &paramName,
				WithDecryption: aws.Bool(true),
			})
			if err == nil && result.Parameter != nil && result.Parameter.Value != nil {
				password = *result.Parameter.Value
			}
		}
	}

	// 如果无法获取现有密码，生成新密码
	if password == "" {
		password = generateSecurePassword(length)
	}

	// 如果通过条件判断是否创建 NewParameter
	// 1. 第一次运行时，由于密码不存在，代码会创建 NewStringParameter，这个参数会被记录为 CloudFormation 堆栈的一部分
	// 2. 第二次运行时，检测到密码已存在，代码会使用 StringParameter_FromStringParameterName 引用现有参数
	// 3. 但是，CloudFormation 会认为这个参数不再是堆栈创建的资源（因为现在是引用而不是创建），所以会尝试删除它
	// 这是 CDK 和 CloudFormation 资源管理的一个常见陷阱。要解决这个问题，我们需要确保无论参数是否存在，都使用相同的 CDK 构造方式。
	// 无论参数是否存在，都使用 CfnParameter 在 CloudFormation 模板中创建参数引用


	// 创建或更新 SSM 参数, 这样保证第二次运行，还是认为这个属于此 stack
	awsssm.NewStringParameter(stack, jsii.String(id), &awsssm.StringParameterProps{
		ParameterName: jsii.String(paramName),
		StringValue: jsii.String(password),
		Description: jsii.String(fmt.Sprintf("Secure password for %s", paramName)),
		AllowedPattern: jsii.String(".*"),
		Tier: awsssm.ParameterTier_STANDARD,
	})
	/*
	awscdk.NewCfnParameter(stack, jsii.String(id), &awscdk.CfnParameterProps{
		Type: jsii.String("String"),
		Default: jsii.String(password),
		NoEcho: jsii.Bool(true),
		Description: jsii.String(fmt.Sprintf("Password for %s", paramName)),
	})
	*/

	// 存储到缓存
	passwordCache[cacheKey] = password

	// 返回密码值
	return password
}

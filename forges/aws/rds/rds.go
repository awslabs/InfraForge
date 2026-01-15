// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package rds

import (
	"fmt"
	"strings"

	"github.com/awslabs/InfraForge/core/config"
	"github.com/awslabs/InfraForge/core/interfaces"
	"github.com/awslabs/InfraForge/core/security"
	"github.com/awslabs/InfraForge/core/utils/aws"
	"github.com/awslabs/InfraForge/core/utils/types"
	utilsSecurity "github.com/awslabs/InfraForge/core/utils/security"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsrds"
	"github.com/aws/aws-cdk-go/awscdk/v2/awssecretsmanager"
	"github.com/aws/jsii-runtime-go"
)

// RDS引擎默认版本常量
// 查询最新版本的命令：
// MySQL: aws rds describe-db-engine-versions --engine mysql --query 'DBEngineVersions[*].EngineVersion' --output text | awk '{print $NF}'
// PostgreSQL: aws rds describe-db-engine-versions --engine postgres --query 'DBEngineVersions[*].EngineVersion' --output text | awk '{print $NF}'
// MariaDB: aws rds describe-db-engine-versions --engine mariadb --query 'DBEngineVersions[*].EngineVersion' --output text | awk '{print $NF}'
// SQL Server EE: aws rds describe-db-engine-versions --engine sqlserver-ee --query 'DBEngineVersions[*].EngineVersion' --output text | awk '{print $NF}'
// SQL Server EX: aws rds describe-db-engine-versions --engine sqlserver-ex --query 'DBEngineVersions[*].EngineVersion' --output text | awk '{print $NF}'
// SQL Server SE: aws rds describe-db-engine-versions --engine sqlserver-se --query 'DBEngineVersions[*].EngineVersion' --output text | awk '{print $NF}'
// SQL Server Web: aws rds describe-db-engine-versions --engine sqlserver-web --query 'DBEngineVersions[*].EngineVersion' --output text | awk '{print $NF}'
// Aurora MySQL: aws rds describe-db-engine-versions --engine aurora-mysql --query 'DBEngineVersions[*].EngineVersion' --output text | awk '{print $NF}'
// Aurora PostgreSQL: aws rds describe-db-engine-versions --engine aurora-postgresql --query 'DBEngineVersions[*].EngineVersion' --output text | awk '{print $NF}'
// 查看所有可用引擎: aws rds describe-db-engine-versions --query 'DBEngineVersions[*].Engine' | sort | uniq
const (
	DefaultMysqlVersion      = "8.4.6"
	DefaultPostgresVersion   = "16.4"
	DefaultMariaDBVersion    = "10.11.9"
	DefaultSqlServerVersion  = "16.00.4210.1.v1"
	DefaultAuroraMysqlVersion = "8.0.mysql_aurora.3.10.0"
	DefaultAuroraPostgresVersion = "17.5"
)

type RdsInstanceConfig struct {
	config.BaseInstanceConfig
	Engine              string `json:"engine"`
	EngineVersion       string `json:"engineVersion,omitempty"`
	InstanceType        string `json:"instanceType"`
	DatabaseName        string `json:"databaseName"`
	Username            string `json:"username"`
	AllocatedStorage    int    `json:"allocatedStorage,omitempty"`
	StorageType         string `json:"storageType,omitempty"`
	StorageEncrypted    *bool  `json:"storageEncrypted,omitempty"`
	MultiAZ             *bool  `json:"multiAZ,omitempty"`
	PubliclyAccessible  *bool  `json:"publiclyAccessible,omitempty"`
	BackupRetentionDays int    `json:"backupRetentionDays,omitempty"`
	DeletionProtection  *bool  `json:"deletionProtection,omitempty"`
	Port                int    `json:"port,omitempty"`
	DependsOn           string `json:"dependsOn,omitempty"`
	// Aurora支持
	ClusterMode         *bool  `json:"clusterMode,omitempty"`
	ReaderInstances     int    `json:"readerInstances,omitempty"`
	ServerlessV2        *bool  `json:"serverlessV2,omitempty"`
	// 密码管理
	UseManagedPassword  *bool  `json:"useManagedPassword,omitempty"`
}

type RdsForge struct {
	rdsInstances []awsrds.DatabaseInstance
	rdsClusters  []awsrds.DatabaseCluster
	secrets      []awssecretsmanager.ISecret
	properties   map[string]interface{}  // 存储连接属性
}

func (r *RdsForge) Create(ctx *interfaces.ForgeContext) interface{} {
	rdsInstance, ok := (*ctx.Instance).(*RdsInstanceConfig)
	if !ok {
		return nil
	}

	// RDS 目前不需要依赖信息，如果将来需要可以取消注释
	// magicToken, err := dependency.GetDependencyInfo(rdsInstance.DependsOn)
	// if err != nil {
	//	fmt.Printf("Error getting dependency info: %v\n", err)
	// }

	// 设置默认用户名
	username := rdsInstance.Username
	if username == "" {
		if strings.Contains(strings.ToLower(rdsInstance.Engine), "postgres") {
			username = "postgres"
		} else {
			username = "admin"
		}
	}

	// 存储连接属性
	if r.properties == nil {
		r.properties = make(map[string]interface{})
	}
	r.properties["username"] = username
	r.properties["engine"] = rdsInstance.Engine
	r.properties["port"] = getDefaultPort(rdsInstance.Engine, rdsInstance.Port)
	r.properties["useManagedPassword"] = types.GetBoolValue(rdsInstance.UseManagedPassword, true)
	if rdsInstance.DatabaseName != "" {
		r.properties["databaseName"] = rdsInstance.DatabaseName
	}

	clusterMode := types.GetBoolValue(rdsInstance.ClusterMode, false)

	if clusterMode && isAuroraEngine(rdsInstance.Engine) {
		r.createCluster(ctx.Stack, rdsInstance, ctx.VPC, ctx.SubnetType, ctx.SecurityGroups.Default)
	} else {
		r.createInstance(ctx.Stack, rdsInstance, ctx.VPC, ctx.SubnetType, ctx.SecurityGroups.Default)
	}

	// 添加运行时属性
	if len(r.secrets) > 0 {
		r.properties["secretArn"] = r.secrets[0].SecretArn()
	}
	
	// 添加连接端点
	if len(r.rdsClusters) > 0 {
		cluster := r.rdsClusters[0]
		r.properties["endpoint"] = cluster.ClusterEndpoint().Hostname()
		r.properties["readEndpoint"] = cluster.ClusterReadEndpoint().Hostname()
	} else if len(r.rdsInstances) > 0 {
		instance := r.rdsInstances[0]
		r.properties["endpoint"] = instance.DbInstanceEndpointAddress()
		r.properties["port"] = instance.DbInstanceEndpointPort()
	}

	return r
}

func (r *RdsForge) createInstance(stack awscdk.Stack, rdsInstance *RdsInstanceConfig, vpc awsec2.IVpc, subnetType awsec2.SubnetType, defaultSG awsec2.SecurityGroup) awsrds.DatabaseInstance {
	subnetGroup := awsrds.NewSubnetGroup(stack, jsii.String(rdsInstance.GetID()+"-subnet-group"), &awsrds.SubnetGroupProps{
		Description: jsii.String(fmt.Sprintf("Subnet group for %s", rdsInstance.GetID())),
		Vpc:         vpc,
		VpcSubnets:  &awsec2.SubnetSelection{SubnetType: subnetType},
	})

	var credentials awsrds.Credentials
	var secret awssecretsmanager.ISecret

	useManagedPassword := types.GetBoolValue(rdsInstance.UseManagedPassword, false)

	if useManagedPassword {
		// RDS托管密码
		credentials = awsrds.Credentials_FromGeneratedSecret(jsii.String(rdsInstance.Username), &awsrds.CredentialsBaseOptions{})
	} else {
		// 非托管密码 - 使用幂等性密码生成
		secretName := fmt.Sprintf("%s-%s-password", *stack.StackName(), rdsInstance.GetID())
		password, customSecret := utilsSecurity.GetOrCreateSecretPassword(
			stack, 
			rdsInstance.GetID()+"-password", 
			secretName,
			fmt.Sprintf("RDS password for %s", rdsInstance.GetID()),
			30,
		)
		secret = customSecret
		
		credentials = awsrds.Credentials_FromPassword(
			jsii.String(rdsInstance.Username),
			awscdk.SecretValue_UnsafePlainText(jsii.String(password)),
		)
	}

	instanceProps := &awsrds.DatabaseInstanceProps{
		Engine:             getInstanceEngineFromString(rdsInstance.Engine, rdsInstance.EngineVersion),
		InstanceType:       awsec2.NewInstanceType(jsii.String(rdsInstance.InstanceType)),
		Vpc:                vpc,
		VpcSubnets:         &awsec2.SubnetSelection{SubnetType: subnetType},
		SecurityGroups:     &[]awsec2.ISecurityGroup{defaultSG},
		Credentials:        credentials,
		Port:               jsii.Number(getDefaultPort(rdsInstance.Engine, rdsInstance.Port)),
		MultiAz:            jsii.Bool(types.GetBoolValue(rdsInstance.MultiAZ, false)),
		PubliclyAccessible: jsii.Bool(types.GetBoolValue(rdsInstance.PubliclyAccessible, false)),
		StorageEncrypted:   jsii.Bool(types.GetBoolValue(rdsInstance.StorageEncrypted, true)),
		DeletionProtection: jsii.Bool(types.GetBoolValue(rdsInstance.DeletionProtection, false)),
		SubnetGroup:        subnetGroup,
	}

	// 只在指定了数据库名时才设置
	if rdsInstance.DatabaseName != "" {
		instanceProps.DatabaseName = jsii.String(rdsInstance.DatabaseName)
	}

	if rdsInstance.AllocatedStorage > 0 {
		instanceProps.AllocatedStorage = jsii.Number(rdsInstance.AllocatedStorage)
	}
	if rdsInstance.StorageType != "" {
		instanceProps.StorageType = getStorageTypeFromString(rdsInstance.StorageType)
	}
	if rdsInstance.BackupRetentionDays > 0 {
		instanceProps.BackupRetention = awscdk.Duration_Days(jsii.Number(rdsInstance.BackupRetentionDays))
	}

	dbInstance := awsrds.NewDatabaseInstance(stack, jsii.String(rdsInstance.GetID()), instanceProps)
	
	// 获取Secret引用
	if useManagedPassword {
		secret = dbInstance.Secret()
	}
	
	// 存储Secret引用
	if secret != nil {
		r.secrets = append(r.secrets, secret)
	}
	
	r.rdsInstances = append(r.rdsInstances, dbInstance)
	return dbInstance
}

func (r *RdsForge) createCluster(stack awscdk.Stack, rdsInstance *RdsInstanceConfig, vpc awsec2.IVpc, subnetType awsec2.SubnetType, defaultSG awsec2.SecurityGroup) awsrds.DatabaseCluster {
	subnetGroup := awsrds.NewSubnetGroup(stack, jsii.String(rdsInstance.GetID()+"-subnet-group"), &awsrds.SubnetGroupProps{
		Description: jsii.String(fmt.Sprintf("Subnet group for %s", rdsInstance.GetID())),
		Vpc:         vpc,
		VpcSubnets:  &awsec2.SubnetSelection{SubnetType: subnetType},
	})

	// 设置默认用户名
	username := rdsInstance.Username
	if username == "" {
		if strings.Contains(strings.ToLower(rdsInstance.Engine), "postgres") {
			username = "postgres"
		} else {
			username = "admin"
		}
	}
	
	var credentials awsrds.Credentials
	var secret awssecretsmanager.ISecret
	
	if types.GetBoolValue(rdsInstance.UseManagedPassword, true) {
		// 使用AWS管理的密码（JSON格式）
		credentials = awsrds.Credentials_FromGeneratedSecret(jsii.String(username), &awsrds.CredentialsBaseOptions{})
	} else {
		// 使用纯文本密码
		secretName := fmt.Sprintf("%s-%s-password", *stack.StackName(), rdsInstance.GetID())
		passwordSecret, customSecret := utilsSecurity.GetOrCreateSecretPassword(
			stack, 
			rdsInstance.GetID()+"-password", 
			secretName,
			fmt.Sprintf("RDS Aurora password for %s", rdsInstance.GetID()),
			30,
		)
		secret = customSecret
		credentials = awsrds.Credentials_FromPassword(jsii.String(username), awscdk.SecretValue_UnsafePlainText(jsii.String(passwordSecret)))
	}

	clusterProps := &awsrds.DatabaseClusterProps{
		Engine:             getClusterEngineFromString(rdsInstance.Engine, rdsInstance.EngineVersion),
		Credentials:        credentials,
		Vpc:                vpc,
		VpcSubnets:         &awsec2.SubnetSelection{SubnetType: subnetType},
		SecurityGroups:     &[]awsec2.ISecurityGroup{defaultSG},
		Port:               jsii.Number(getDefaultPort(rdsInstance.Engine, rdsInstance.Port)),
		StorageEncrypted:   jsii.Bool(types.GetBoolValue(rdsInstance.StorageEncrypted, true)),
		DeletionProtection: jsii.Bool(types.GetBoolValue(rdsInstance.DeletionProtection, false)),
		SubnetGroup:        subnetGroup,
		Writer: awsrds.ClusterInstance_Provisioned(jsii.String("writer"), &awsrds.ProvisionedClusterInstanceProps{
			InstanceType:       awsec2.NewInstanceType(jsii.String(rdsInstance.InstanceType)),
			PubliclyAccessible: jsii.Bool(types.GetBoolValue(rdsInstance.PubliclyAccessible, false)),
		}),
	}

	// 只在指定了数据库名时才设置
	if rdsInstance.DatabaseName != "" {
		clusterProps.DefaultDatabaseName = jsii.String(rdsInstance.DatabaseName)
	}

	if rdsInstance.BackupRetentionDays > 0 {
		clusterProps.Backup = &awsrds.BackupProps{
			Retention: awscdk.Duration_Days(jsii.Number(rdsInstance.BackupRetentionDays)),
		}
	}

	dbCluster := awsrds.NewDatabaseCluster(stack, jsii.String(rdsInstance.GetID()), clusterProps)
	r.rdsClusters = append(r.rdsClusters, dbCluster)
	
	// 保存Secret引用
	if secret != nil {
		r.secrets = append(r.secrets, secret)
	} else if types.GetBoolValue(rdsInstance.UseManagedPassword, true) {
		// 对于托管密码，获取自动创建的secret
		if clusterSecret := dbCluster.Secret(); clusterSecret != nil {
			r.secrets = append(r.secrets, clusterSecret)
		}
	}
	
	return dbCluster
}

func (r *RdsForge) ConfigureRules(ctx *interfaces.ForgeContext) {
	if ctx.Instance == nil {
		return
	}

	rdsInstance, ok := (*ctx.Instance).(*RdsInstanceConfig)
	if !ok {
		return
	}

	port := getDefaultPort(rdsInstance.Engine, rdsInstance.Port)
	engineName := strings.ToUpper(rdsInstance.Engine)
	
	security.AddTcpIngressRule(ctx.SecurityGroups.Default, ctx.SecurityGroups.Public, port, fmt.Sprintf("Allow %s access from public subnet", engineName))
	security.AddTcpIngressRule(ctx.SecurityGroups.Default, ctx.SecurityGroups.Private, port, fmt.Sprintf("Allow %s access from private subnet", engineName))
}

func (r *RdsForge) CreateOutputs(ctx *interfaces.ForgeContext) {
	rdsInstance, ok := (*ctx.Instance).(*RdsInstanceConfig)
	if !ok {
		return
	}

	// 输出实例信息
	for i, dbInstance := range r.rdsInstances {
		suffix := ""
		if len(r.rdsInstances) > 1 {
			suffix = fmt.Sprintf("-%d", i+1)
		}

		awscdk.NewCfnOutput(ctx.Stack, jsii.String(fmt.Sprintf("RdsEndpoint%s%s", aws.GetOriginalID(rdsInstance.GetID()), suffix)), &awscdk.CfnOutputProps{
			Value:       dbInstance.DbInstanceEndpointAddress(),
			Description: jsii.String(fmt.Sprintf("RDS Endpoint for %s", rdsInstance.GetID())),
		})
	}

	// 输出集群信息
	for i, dbCluster := range r.rdsClusters {
		suffix := ""
		if len(r.rdsClusters) > 1 {
			suffix = fmt.Sprintf("-%d", i+1)
		}

		awscdk.NewCfnOutput(ctx.Stack, jsii.String(fmt.Sprintf("RdsClusterEndpoint%s%s", aws.GetOriginalID(rdsInstance.GetID()), suffix)), &awscdk.CfnOutputProps{
			Value:       dbCluster.ClusterEndpoint().Hostname(),
			Description: jsii.String(fmt.Sprintf("RDS Cluster Endpoint for %s", rdsInstance.GetID())),
		})

		awscdk.NewCfnOutput(ctx.Stack, jsii.String(fmt.Sprintf("RdsClusterReadEndpoint%s%s", aws.GetOriginalID(rdsInstance.GetID()), suffix)), &awscdk.CfnOutputProps{
			Value:       dbCluster.ClusterReadEndpoint().Hostname(),
			Description: jsii.String(fmt.Sprintf("RDS Cluster Read Endpoint for %s", rdsInstance.GetID())),
		})
	}

	// 输出Secret ARN
	if len(r.secrets) > 0 {
		awscdk.NewCfnOutput(ctx.Stack, jsii.String(fmt.Sprintf("RdsSecretArn%s", aws.GetOriginalID(rdsInstance.GetID()))), &awscdk.CfnOutputProps{
			Value:       r.secrets[0].SecretArn(),
			Description: jsii.String(fmt.Sprintf("RDS Secret ARN for %s", rdsInstance.GetID())),
		})
	}
}

func (r *RdsForge) MergeConfigs(defaults config.InstanceConfig, instance config.InstanceConfig) config.InstanceConfig {
	merged := defaults.(*RdsInstanceConfig)
	rdsInstance := instance.(*RdsInstanceConfig)

	if rdsInstance.GetID() != "" {
		merged.ID = rdsInstance.GetID()
	}
	if rdsInstance.Type != "" {
		merged.Type = rdsInstance.GetType()
	}
	if rdsInstance.Subnet != "" {
		merged.Subnet = rdsInstance.GetSubnet()
	}
	if rdsInstance.SecurityGroup != "" {
		merged.SecurityGroup = rdsInstance.GetSecurityGroup()
	}
	if rdsInstance.Engine != "" {
		merged.Engine = rdsInstance.Engine
	}
	if rdsInstance.EngineVersion != "" {
		merged.EngineVersion = rdsInstance.EngineVersion
	}
	if rdsInstance.InstanceType != "" {
		merged.InstanceType = rdsInstance.InstanceType
	}
	if rdsInstance.DatabaseName != "" {
		merged.DatabaseName = rdsInstance.DatabaseName
	}
	if rdsInstance.Username != "" {
		merged.Username = rdsInstance.Username
	}
	if rdsInstance.AllocatedStorage > 0 {
		merged.AllocatedStorage = rdsInstance.AllocatedStorage
	}
	if rdsInstance.StorageType != "" {
		merged.StorageType = rdsInstance.StorageType
	}
	if rdsInstance.StorageEncrypted != nil {
		merged.StorageEncrypted = rdsInstance.StorageEncrypted
	}
	if rdsInstance.MultiAZ != nil {
		merged.MultiAZ = rdsInstance.MultiAZ
	}
	if rdsInstance.PubliclyAccessible != nil {
		merged.PubliclyAccessible = rdsInstance.PubliclyAccessible
	}
	if rdsInstance.BackupRetentionDays > 0 {
		merged.BackupRetentionDays = rdsInstance.BackupRetentionDays
	}
	if rdsInstance.DeletionProtection != nil {
		merged.DeletionProtection = rdsInstance.DeletionProtection
	}
	if rdsInstance.Port > 0 {
		merged.Port = rdsInstance.Port
	}
	if rdsInstance.DependsOn != "" {
		merged.DependsOn = rdsInstance.DependsOn
	}
	if rdsInstance.ClusterMode != nil {
		merged.ClusterMode = rdsInstance.ClusterMode
	}
	if rdsInstance.ReaderInstances > 0 {
		merged.ReaderInstances = rdsInstance.ReaderInstances
	}
	if rdsInstance.ServerlessV2 != nil {
		merged.ServerlessV2 = rdsInstance.ServerlessV2
	}

	if rdsInstance.UseManagedPassword != nil {
		merged.UseManagedPassword = rdsInstance.UseManagedPassword
	}
	if rdsInstance.StorageEncrypted != nil {
		merged.StorageEncrypted = rdsInstance.StorageEncrypted
	}
	if rdsInstance.MultiAZ != nil {
		merged.MultiAZ = rdsInstance.MultiAZ
	}
	if rdsInstance.PubliclyAccessible != nil {
		merged.PubliclyAccessible = rdsInstance.PubliclyAccessible
	}
	if rdsInstance.DeletionProtection != nil {
		merged.DeletionProtection = rdsInstance.DeletionProtection
	}
	if rdsInstance.ClusterMode != nil {
		merged.ClusterMode = rdsInstance.ClusterMode
	}
	if rdsInstance.ServerlessV2 != nil {
		merged.ServerlessV2 = rdsInstance.ServerlessV2
	}

	return merged
}

func (r *RdsForge) GetSecretArn() *string {
	if len(r.secrets) > 0 {
		return r.secrets[0].SecretArn()
	}
	return nil
}

func (r *RdsForge) GetProperties() map[string]interface{} {
	return r.properties
}

func (r *RdsForge) GetInstances() []awsrds.DatabaseInstance {
	return r.rdsInstances
}

func (r *RdsForge) GetClusters() []awsrds.DatabaseCluster {
	return r.rdsClusters
}

func isAuroraEngine(engine string) bool {
	engine = strings.ToLower(engine)
	return engine == "aurora-mysql" || engine == "aurora-postgresql"
}

// getInstanceEngineFromString 根据引擎类型和版本返回实例引擎
func getInstanceEngineFromString(engine string, version string) awsrds.IInstanceEngine {
	switch strings.ToLower(engine) {
	case "mysql":
		if version == "" {
			version = DefaultMysqlVersion
		}
		// MySQL: 8.4.6 -> 8.4
		majorVersion := strings.Split(version, ".")[0] + "." + strings.Split(version, ".")[1]
		return awsrds.DatabaseInstanceEngine_Mysql(&awsrds.MySqlInstanceEngineProps{
			Version: awsrds.MysqlEngineVersion_Of(jsii.String(version), jsii.String(majorVersion)),
		})
	case "postgres", "postgresql":
		if version == "" {
			version = DefaultPostgresVersion
		}
		// PostgreSQL: 16.4 -> 16
		majorVersion := strings.Split(version, ".")[0]
		return awsrds.DatabaseInstanceEngine_Postgres(&awsrds.PostgresInstanceEngineProps{
			Version: awsrds.PostgresEngineVersion_Of(jsii.String(version), jsii.String(majorVersion), nil),
		})
	case "mariadb":
		if version == "" {
			version = DefaultMariaDBVersion
		}
		// MariaDB: 10.11.9 -> 10.11
		majorVersion := strings.Split(version, ".")[0] + "." + strings.Split(version, ".")[1]
		return awsrds.DatabaseInstanceEngine_MariaDb(&awsrds.MariaDbInstanceEngineProps{
			Version: awsrds.MariaDbEngineVersion_Of(jsii.String(version), jsii.String(majorVersion)),
		})
	case "sqlserver-ee":
		if version == "" {
			version = DefaultSqlServerVersion
		}
		return awsrds.DatabaseInstanceEngine_SqlServerEe(&awsrds.SqlServerEeInstanceEngineProps{
			Version: awsrds.SqlServerEngineVersion_Of(jsii.String(version), nil),
		})
	case "sqlserver-ex":
		if version == "" {
			version = DefaultSqlServerVersion
		}
		return awsrds.DatabaseInstanceEngine_SqlServerEx(&awsrds.SqlServerExInstanceEngineProps{
			Version: awsrds.SqlServerEngineVersion_Of(jsii.String(version), nil),
		})
	case "sqlserver-se":
		if version == "" {
			version = DefaultSqlServerVersion
		}
		return awsrds.DatabaseInstanceEngine_SqlServerSe(&awsrds.SqlServerSeInstanceEngineProps{
			Version: awsrds.SqlServerEngineVersion_Of(jsii.String(version), nil),
		})
	case "sqlserver-web":
		if version == "" {
			version = DefaultSqlServerVersion
		}
		return awsrds.DatabaseInstanceEngine_SqlServerWeb(&awsrds.SqlServerWebInstanceEngineProps{
			Version: awsrds.SqlServerEngineVersion_Of(jsii.String(version), nil),
		})
	default:
		if version == "" {
			version = DefaultMysqlVersion
		}
		return awsrds.DatabaseInstanceEngine_Mysql(&awsrds.MySqlInstanceEngineProps{
			Version: awsrds.MysqlEngineVersion_Of(jsii.String(version), nil),
		})
	}
}

// getClusterEngineFromString 根据引擎类型和版本返回集群引擎
func getClusterEngineFromString(engine string, version string) awsrds.IClusterEngine {
	switch strings.ToLower(engine) {
	case "aurora-mysql":
		if version == "" {
			version = DefaultAuroraMysqlVersion
		}
		// 从完整版本中提取主版本号 (如 "8.0.mysql_aurora.3.07.0" -> "8.0")
		majorVersion := strings.Split(version, ".")[0] + "." + strings.Split(version, ".")[1]
		return awsrds.DatabaseClusterEngine_AuroraMysql(&awsrds.AuroraMysqlClusterEngineProps{
			Version: awsrds.AuroraMysqlEngineVersion_Of(jsii.String(version), jsii.String(majorVersion)),
		})
	case "aurora-postgresql":
		if version == "" {
			version = DefaultAuroraPostgresVersion
		}
		return awsrds.DatabaseClusterEngine_AuroraPostgres(&awsrds.AuroraPostgresClusterEngineProps{
			Version: awsrds.AuroraPostgresEngineVersion_Of(jsii.String(version), nil, nil),
		})
	default:
		if version == "" {
			version = DefaultAuroraMysqlVersion
		}
		// 从完整版本中提取主版本号
		majorVersion := strings.Split(version, ".")[0] + "." + strings.Split(version, ".")[1]
		return awsrds.DatabaseClusterEngine_AuroraMysql(&awsrds.AuroraMysqlClusterEngineProps{
			Version: awsrds.AuroraMysqlEngineVersion_Of(jsii.String(version), jsii.String(majorVersion)),
		})
	}
}

func getStorageTypeFromString(storageType string) awsrds.StorageType {
	switch strings.ToLower(storageType) {
	case "gp2":
		return awsrds.StorageType_GP2
	case "gp3":
		return awsrds.StorageType_GP3
	case "io1":
		return awsrds.StorageType_IO1
	case "io2":
		return awsrds.StorageType_IO2
	default:
		return awsrds.StorageType_GP2
	}
}

func getDefaultPort(engine string, configuredPort int) int {
	if configuredPort > 0 {
		return configuredPort
	}

	switch strings.ToLower(engine) {
	case "mysql", "mariadb", "aurora-mysql":
		return 3306
	case "postgres", "postgresql", "aurora-postgresql":
		return 5432
	case "sqlserver", "mssql":
		return 1433
	case "oracle":
		return 1521
	default:
		return 3306
	}
}

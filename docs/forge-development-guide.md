# InfraForge New Forge Development Guide

## 🎯 Overview

This guide provides step-by-step instructions for developing new Forge components in the InfraForge framework. A Forge is a modular infrastructure component that manages specific AWS services.

## 📁 Project Structure

### 1. Create Forge Directory Structure
```
forges/aws/<service>/
├── <service>.go          # Main implementation file
├── <service>_test.go     # Unit tests
└── README.md            # Documentation
```

## 🏗️ Implementation Steps

### 2. Define Configuration Structure

```go
package <service>

import (
    "github.com/aws-samples/infraforge/core/config"
)

type <Service>InstanceConfig struct {
    config.BaseInstanceConfig
    
    // Service-specific required fields
    RequiredField    string `json:"requiredField"`           // Required field
    
    // Service-specific optional fields
    OptionalField    string `json:"optionalField,omitempty"` // Optional string field
    BoolField        *bool  `json:"boolField,omitempty"`     // Boolean field (use pointer)
    IntField         int    `json:"intField,omitempty"`      // Integer field
}
```

### 3. Research and Refine with AWS CDK Documentation

After initial design, refine your implementation using AWS CDK Go documentation:

1. **Visit CDK Documentation:**  https://pkg.go.dev/github.com/aws/aws-cdk-go/awscdk/v2/
2. **Find Service Package:**  Navigate to the specific AWS service (e.g., `awsbatch`, `awsrds`, `awsec2`)
3. **Download Documentation:**  Save relevant documentation pages locally for reference
4. **Note Package Naming:**  CDK packages use `aws<service>` format (e.g., `batch.xxx` becomes `awsbatch.xxx`)
5. **Study Resource Properties:**  Review available properties, methods, and configuration options
6. **Refine Configuration:**  Add missing fields, correct property names, and enhance functionality

**Example CDK Package URLs:**
- AWS Batch: https://pkg.go.dev/github.com/aws/aws-cdk-go/awscdk/v2/awsbatch
- AWS RDS: https://pkg.go.dev/github.com/aws/aws-cdk-go/awscdk/v2/awsrds
- AWS EC2: https://pkg.go.dev/github.com/aws/aws-cdk-go/awscdk/v2/awsec2

**Why Design First?**
- Initial design captures business requirements and use cases
- Documentation review adds technical depth and completeness
- Prevents over-simplification that comes from documentation-first approach
- Ensures the solution addresses real-world scenarios

### 4. Implement Forge Interface

```go
type <Service>Forge struct {
    // Store created resource references
    resource    <AwsResource>
    properties  map[string]interface{}  // Store resource properties for dependencies
}

// Required interface methods
func (f *<Service>Forge) Create(ctx *interfaces.ForgeContext) error
func (f *<Service>Forge) MergeConfigs(defaults, instance config.InstanceConfig) config.InstanceConfig
func (f *<Service>Forge) ConfigureRules(ctx *interfaces.ForgeContext)
func (f *<Service>Forge) CreateOutputs(ctx *interfaces.ForgeContext)

// If resource will be used as dependency, implement this method
func (f *<Service>Forge) GetProperties() map[string]interface{}
```

### 4. Implement Create Method

```go
func (f *<Service>Forge) Create(ctx *interfaces.ForgeContext) error {
    instance := ctx.Instance.(*<Service>InstanceConfig)
    
    // 1. Merge configurations
    merged := f.MergeConfigs(ctx.DefaultConfig, instance).(*<Service>InstanceConfig)
    
    // 2. Initialize properties map
    if f.properties == nil {
        f.properties = make(map[string]interface{})
    }
    
    // 3. Create AWS resource
    resourceId := fmt.Sprintf("%s-%s", merged.GetID(), "<service>")
    props := &aws<Service>.<Resource>Props{
        // Configure resource properties
        RequiredProperty: jsii.String(merged.RequiredField),
        OptionalProperty: jsii.String(utils.GetStringValue(&merged.OptionalField, "default")),
        BoolProperty:     jsii.Bool(utils.GetBoolValue(merged.BoolField, false)),
        IntProperty:      jsii.Number(utils.GetIntValue(&merged.IntField, 0)),
    }
    
    // Handle IAM roles if needed
    if merged.InstanceRolePolicies != "" {
        roleId := fmt.Sprintf("%s-role", merged.GetID())
        role := aws.CreateRole(ctx.Stack, roleId, merged.InstanceRolePolicies, "<service-principal>")
        props.Role = role
    }
    
    // Handle dependencies
    if merged.DependsOn != "" {
        magicToken, err := dependency.GetDependencyInfo(merged.DependsOn)
        if err != nil {
            return fmt.Errorf("error getting dependency info: %v", err)
        }
        // Process dependency information as needed
    }
    
    f.resource = aws<Service>.New<Resource>(ctx.Stack, jsii.String(resourceId), props)
    
    // 4. Store resource properties for other resources to depend on
    f.properties["resourceId"] = *f.resource.ResourceId()
    f.properties["endpoint"] = *f.resource.Endpoint()
    // Add other relevant properties based on service type
    
    // 5. Register with dependency manager
    dependency.GlobalManager.SetProperties(merged.GetID(), f.properties)
    
    return nil
}
```

### 5. Implement MergeConfigs Method

```go
func (f *<Service>Forge) MergeConfigs(defaults, instance config.InstanceConfig) config.InstanceConfig {
    merged := defaults.(*<Service>InstanceConfig)  // Start with defaults (direct reference)
    serviceInstance := instance.(*<Service>InstanceConfig)
    
    // Override with instance configuration values
    if serviceInstance.GetID() != "" {
        merged.ID = serviceInstance.GetID()
    }
    if serviceInstance.GetType() != "" {
        merged.Type = serviceInstance.GetType()
    }
    if serviceInstance.GetSubnet() != "" {
        merged.Subnet = serviceInstance.GetSubnet()
    }
    if serviceInstance.GetSecurityGroup() != "" {
        merged.SecurityGroup = serviceInstance.GetSecurityGroup()
    }
    
    // String fields - override if not empty
    if serviceInstance.OptionalField != "" {
        merged.OptionalField = serviceInstance.OptionalField
    }
    if serviceInstance.InstanceRolePolicies != "" {
        merged.InstanceRolePolicies = serviceInstance.InstanceRolePolicies
    }
    if serviceInstance.UserDataToken != "" {
    
    // Boolean pointer fields - override if set (not nil)
    if serviceInstance.BoolField != nil {
        merged.BoolField = serviceInstance.BoolField
    }
    
    // Integer fields - override if greater than 0
    if serviceInstance.IntField > 0 {
        merged.IntField = serviceInstance.IntField
    }
    
    return merged  // Return direct reference, not &merged
}
```

### 6. Implement Other Required Methods

```go
func (f *<Service>Forge) ConfigureRules(ctx *interfaces.ForgeContext) {
    // Configure security group rules if needed
    // Example:
    // ctx.SecurityGroups.Default.AddIngressRule(
    //     awsec2.Peer_AnyIpv4(),
    //     awsec2.Port_Tcp(jsii.Number(80)),
    //     jsii.String("Allow HTTP"),
    // )
}

func (f *<Service>Forge) CreateOutputs(ctx *interfaces.ForgeContext) {
    // Create CloudFormation outputs
    // Example:
    // awscdk.NewCfnOutput(ctx.Stack, jsii.String("ResourceEndpoint"), &awscdk.CfnOutputProps{
    //     Value: f.resource.Endpoint(),
    //     Description: jsii.String("Resource endpoint"),
    // })
}

// Implement if resource will be used as dependency
func (f *<Service>Forge) GetProperties() map[string]interface{} {
    return f.properties
}
```

### 7. Register Forge

Add to `registry/registry.go`:

```go
import (
    "<service>" "github.com/aws-samples/infraforge/forges/aws/<service>"
)

func init() {
    RegisterForge("<service>", func() interfaces.Forge {
        return &<service>.<Service>Forge{}
    })
}
```

## 📋 Configuration Examples

### 8. Configuration Structure Best Practices

InfraForge uses a structured configuration format with `defaults` and `instances` sections to minimize duplication:

```json
{
  "global": {
    "stackName": "my-infrastructure",
    "description": "Infrastructure deployment description"
  },
  "enabledForges": ["instance1", "instance2"],
  "forges": {
    "<service>": {
      "defaults": {
        "type": "SERVICE_TYPE",
        "subnet": "private",
        "security": "private",
        "commonField1": "defaultValue1",
        "commonField2": "defaultValue2",
        "booleanField": true,
        "numericField": 100
      },
      "instances": [
        {
          "id": "instance1"
        },
        {
          "id": "instance2",
          "commonField1": "overrideValue1"
        }
      ]
    }
  }
}
```

**Key Principles:**
- **Defaults Section:**  Contains all common configuration parameters
- **Minimal Instances:**  Instances only need an `id` and any overrides
- **Type Convention:**  Use UPPERCASE for service types (e.g., "RDS", "HYPERPOD", "EC2")
- **Standard Values:**  Use "private", "public", "isolated" for subnet/security

**Important Notes:**
- **enabledForges:**  List of specific instance IDs to deploy (e.g., `["instance1", "instance2"]`)
- **VPC:**  Always created automatically as the foundation layer - no need to specify in enabledForges
- **Instance IDs:**  Must match between `enabledForges` and the actual instance definitions in `forges`

## 🔧 Utility Functions

### 9. Common Utility Functions

```go
// Handle default values
utils.GetStringValue(&config.Field, "default")
utils.GetIntValue(&config.Field, 0)
utils.GetBoolValue(config.BoolField, false)

// Create IAM roles
role := aws.CreateRole(ctx.Stack, roleId, policies, servicePrincipal)

// Handle dependencies
if config.DependsOn != "" {
    magicToken, err := dependency.GetDependencyInfo(config.DependsOn)
    mountPoint, err := dependency.GetMountPoint(config.DependsOn)
    properties, err := dependency.ExtractDependencyProperties(magicToken, resourceType)
}
```

## 📊 Common Properties Examples

### Storage Services (EFS, FSx)
```go
f.properties["fileSystemId"] = *f.fileSystem.FileSystemId()
f.properties["mountPoint"] = "/mnt/efs"
```

### Database Services (RDS, ElastiCache)
```go
f.properties["endpoint"] = *f.database.Endpoint()
f.properties["port"] = *f.database.Port()
f.properties["username"] = username
f.properties["databaseName"] = databaseName
```

### Compute Services (EC2, ECS)
```go
f.properties["instanceId"] = *f.instance.InstanceId()
f.properties["privateIp"] = *f.instance.InstancePrivateIp()
f.properties["publicIp"] = *f.instance.InstancePublicIp()
```

### Network Services (VPC, SecurityGroup)
```go
f.properties["vpcId"] = *f.vpc.VpcId()
f.properties["securityGroupId"] = *f.securityGroup.SecurityGroupId()
```

## 🧪 Testing

### 10. Unit Test Template

```go
package <service>

import (
    "testing"
    "github.com/aws-samples/infraforge/core/config"
)

func TestCreate<Service>(t *testing.T) {
    config := &<Service>InstanceConfig{
        BaseInstanceConfig: config.BaseInstanceConfig{
            ID: "test-<service>",
            Type: "<service>",
        },
        RequiredField: "test-value",
        OptionalField: "test-optional",
    }
    
    forge := &<Service>Forge{}
    
    // Test configuration merging
    defaultConfig := &<Service>InstanceConfig{
        BaseInstanceConfig: config.BaseInstanceConfig{
            ID: "default-<service>",
            Type: "<service>",
        },
        OptionalField: "default-optional",
    }
    
    merged := forge.MergeConfigs(defaultConfig, config).(*<Service>InstanceConfig)
    
    if merged.RequiredField != "test-value" {
        t.Errorf("Expected RequiredField 'test-value', got '%s'", merged.RequiredField)
    }
    
    if merged.OptionalField != "test-optional" {
        t.Errorf("Expected OptionalField 'test-optional', got '%s'", merged.OptionalField)
    }
}

func TestMergeConfigs(t *testing.T) {
    // Test configuration merging logic
    // Add comprehensive test cases
}
```

## 🎯 Best Practices

### 11. Development Guidelines

1. **Field Types:**  Use `*bool` for boolean fields to enable proper configuration merging
2. **Default Values:**  Use `utils.Get*Value()` functions for handling default values
3. **Dependency Management:**  Use `dependency` package for resource dependencies
4. **Error Handling:**  Return meaningful error messages with context
5. **Testing:**  Write comprehensive unit tests for each Forge
6. **Documentation:**  Add clear documentation for configuration fields and usage examples
7. **Resource Naming:**  Use consistent naming patterns for AWS resources
8. **Security:**  Follow AWS security best practices and least privilege principles

### 12. Common Patterns

- **IAM Roles:**  Create service-specific IAM roles with minimal required permissions
- **Security Groups:**  Configure only necessary ingress/egress rules
- **Dependencies:**  Properly handle resource dependencies and circular references
- **Properties:**  Store relevant resource properties for other resources to consume
- **Configuration:**  Support both required and optional configuration fields
- **Validation:**  Validate configuration parameters before resource creation

## 🚀 Deployment

### 13. Testing Your Forge

1. Add your forge to `enabledForges` in configuration
2. Run `go build` to compile
3. Deploy with `./deploy.sh`
4. Verify resources are created correctly
5. Test dependency resolution with other forges

### 14. Integration

Once your forge is complete:
- Add comprehensive documentation
- Create example configurations
- Add integration tests
- Submit for code review
- Update main README with new forge capabilities

Following this guide ensures your new Forge integrates seamlessly with the InfraForge framework and follows established patterns and best practices.

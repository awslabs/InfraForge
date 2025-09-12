# InfraForge Dependency Management System

This directory contains the core code for the InfraForge dependency management system. The system is responsible for tracking AWS resources created during CDK deployment and providing a standardized way to access their properties across different Forge services.

## System Overview

The dependency management system consists of the following components:

1. **ForgeManager:** A central registry that stores references to Forge instances
2. **GlobalManager:** A global instance for cross-forge communication

The system enables seamless communication between different Forge services during the CDK deployment process by providing a standardized way to access resource properties directly from Forge instances.

## Key Components

### ForgeManager

The `ForgeManager` is a central registry that stores references to Forge instances. It provides methods for storing and retrieving Forge instances by their ID.

```go
type ForgeManager struct {
    forges map[string]interface{} // 存储 Forge 实例
    mutex  sync.RWMutex
}
```

### Methods

- `Store(key string, forge interface{})`: Store a Forge instance with the given key
- `Get(key string) (interface{}, bool)`: Retrieve a Forge instance by key

### GlobalManager

A global instance of ForgeManager is available for cross-forge communication:

```go
var GlobalManager *ForgeManager
```

## Usage Example

```go
import "github.com/awslabs/InfraForge/core/dependency"

// Store a forge instance
dependency.GlobalManager.Store("vpc:main", vpcForge)

// Retrieve a forge instance
if forge, exists := dependency.GlobalManager.Get("vpc:main"); exists {
    vpcForge := forge.(*vpc.VpcForge)
    // Use the VPC forge
}
```

## Key Features

- Thread-safe operations with mutex protection
- Support for any forge type through interface{} storage
- Global accessibility across all forge implementations
- Simple key-value storage mechanism
// Store a Forge instance
dependency.GlobalManager.Store("my-resource-id", forgeInstance)

// Retrieve a Forge instance
forge, exists := dependency.GlobalManager.Get("my-resource-id")

// Get properties directly from Forge
properties, exists := dependency.GlobalManager.GetProperties("my-resource-id")
```

### Forge Properties

Each Forge instance implements a `GetProperties()` method that returns a map of its resource properties. Properties are saved during Forge creation and include all relevant resource information.

```go
type Forge interface {
    GetProperties() map[string]interface{}
}
```

### Dependency Resolution

The system provides utilities for resolving dependencies between resources. The `GetDependencyInfo` function takes a dependency string in the format `<ResourceType>:<ResourceID>` and returns a JSON representation of the resource's properties:

```go
dependencyInfo, err := dependency.GetDependencyInfo("EFS:my-efs-resource")
```

For multiple dependencies, you can provide a comma-separated list:

```go
dependencyInfo, err := dependency.GetDependencyInfo("EFS:my-efs-resource,EC2:my-ec2-instance")
```

## Supported Resource Types

The system supports the following AWS resource types:

- **VPC**: Virtual Private Cloud resources
- **EC2**: EC2 instances with detailed instance information
- **ECS**: Elastic Container Service clusters
- **EKS**: Elastic Kubernetes Service clusters
- **EFS**: Elastic File System resources
- **Lustre**: FSx for Lustre file systems
- **DS**: Directory Service (Microsoft Active Directory)
- **RDS**: Relational Database Service instances and clusters

## Integration with CDK Deployment

During CDK deployment, the DependencyManager is populated with Forge instances as they are created. Each Forge saves its resource properties during the creation process. When a Forge service needs to access properties of another resource, it can use the dependency resolution utilities to obtain the required information.

This approach decouples the services and allows them to interact without direct dependencies, making the system more maintainable and flexible.

## Example Usage

```go
// Forge instances are automatically stored during creation
// Get dependency information for use in other Forges
dependencyInfo, err := dependency.GetDependencyInfo("EFS:my-efs")
if err != nil {
    return err
}

// Extract specific resource properties
efsProperties, err := dependency.ExtractDependencyProperties(dependencyInfo, "EFS")
if err != nil {
    return err
}

// Use the properties
fileSystemId := efsProperties["fileSystemId"].(string)
mountPoint := efsProperties["mountPoint"].(string)
```

## Architecture Benefits

The current architecture provides several advantages:

1. **Simplified Code**: No need for separate resource handlers
2. **Better Performance**: Properties are computed once during creation
3. **Centralized Logic**: All resource property logic is in the Forge creation methods
4. **Easy Maintenance**: Single source of truth for resource properties
5. **Type Safety**: Direct access to Forge methods and properties

# ForgeManager Dependency Management

## Overview

ForgeManager uses GlobalManager to manage dependencies between forge instances. This document explains the storage, lookup, and usage mechanisms for dependencies.

## Dependency Storage Mechanism

### Storage Location
- **File:**  `core/manager/forgemanager.go`
- **Method:**  `processForge()` lines 141-143

### Storage Logic
```go
if trimIndex {
    dependency.GlobalManager.Store(aws.GetOriginalID(merged.GetID()), iforge)
} else {
    dependency.GlobalManager.Store(merged.GetID(), iforge)
}
```

### Storage Key
- **Normal case:**  Uses the `id` field from configuration
- **EC2 multi-instance:**  Uses ID processed by `aws.GetOriginalID()`

### Storage Value
- Stores the **forge object itself** (e.g., `*EksForge`), not the resource object

## Dependency Lookup Mechanism

### Configuration Format
```json
{
    "dependsOn": "EKS:eks"
}
```

### Parsing Logic
```go
// Parse "EKS:eks" -> "eks"
parts := strings.Split(dependsOn, ":")
dependencyID := dependsOn
if len(parts) == 2 {
    dependencyID = parts[1]  // Take the part after colon
}
```

### Lookup Steps
1. Parse dependency string to extract actual ID
2. Look up corresponding forge object from GlobalManager
3. Type cast to get specific forge type
4. Get actual resource object from forge object

## Real Case: HyperPod Depends on EKS

### Configuration Example
```json
{
    "eks": {
        "instances": [
            {
                "id": "eks"
            }
        ]
    },
    "hyperpod": {
        "instances": [
            {
                "id": "hyperpod",
                "dependsOn": "EKS:eks"
            }
        ]
    }
}
```

### Dependency Resolution Process
1. **EKS Creation:**  `GlobalManager.Store("eks", eksForgeObject)`
2. **HyperPod Lookup:**  
   - Parse `"EKS:eks"` -> `"eks"`
   - Call `GlobalManager.Get("eks")`
   - Get `*EksForge` object
   - Call `eksForge.GetCluster()` to get actual EKS cluster

### Code Example
```go
// Dependency lookup in HyperPod
parts := strings.Split(hyperPodInstance.DependsOn, ":")
dependencyID := parts[1] // "eks"

if eksForge, exists := dependency.GlobalManager.Get(dependencyID); exists {
    if eksForgeObj, ok := eksForge.(*eksforge.EksForge); ok {
        eksCluster := eksForgeObj.GetCluster()
        // Use eksCluster for subsequent operations
    }
}
```

## Key Points

1. **Stores forge objects**, not resource objects
2. **Dependency format:**  `"ServiceType:InstanceID"`, only use InstanceID part for lookup
3. **Type conversion:**  Need to convert `interface{}` to specific forge type
4. **Resource access:**  Get actual resources through forge object methods

## Extending New Dependencies

### Steps
1. Ensure forge object is properly stored in GlobalManager
2. Parse `dependsOn` string in dependent forge
3. Get forge object from GlobalManager
4. Type cast and get required resource
5. Add getter methods to forge if needed (e.g., `GetCluster()`)

### Considerations
- Dependency resolution should happen at appropriate time during resource creation
- Ensure dependent forge has completed creation
- Handle cases where dependency doesn't exist

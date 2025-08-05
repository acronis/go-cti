# Metadata Library <!-- omit in toc -->

This document provides a comprehensive guide on how to use the go-cti (Cross-domain Typed Identifiers) metadata library.
The library provides tools for parsing, validating, and managing CTI entities and packages.

## Table of Contents <!-- omit in toc -->

- [Overview](#overview)
- [Installation](#installation)
- [Core Concepts](#core-concepts)
  - [Identifiers](#identifiers)
  - [CTI Entities](#cti-entities)
- [Basic usage scenarios](#basic-usage-scenarios)
  - [Working with entity types](#working-with-entity-types)
  - [Validating the data against the schema](#validating-the-data-against-the-schema)
  - [Getting CTI type traits](#getting-cti-type-traits)
  - [Getting CTI instance values](#getting-cti-instance-values)
- [Advanced usage scenarios](#advanced-usage-scenarios)
  - [Getting CTI parent](#getting-cti-parent)

## Overview

The CTI metadata library is part of the Go CTI ecosystem that implements the [Cross-domain Typed Identifiers (CTI) version 1.0 Specification](../cti-spec/SPEC.md). It provides:

- **CTI Entity Management**: Create, parse, and manage the metadata of CTI types and instances.
- **Package Management**: Handle CTI packages with dependencies.
- **Validation**: Validate CTI entities against their definitions.
- **Collectors**: Reference implementation of metadata transformations from other sources (RAMLx, CTI metadata files).

## Installation

```bash
go get -u github.com/acronis/go-cti/metadata
```

## Core Concepts

### Identifiers

CTIs follow the format: `cti.<vendor>.<package>.<entity_name>.v<major>.<minor>[~<extension>]`

Examples:

- `cti.a.p.alert.v1.0` - Base alert type.
- `cti.a.p.alert.v1.0~a.p.user.v1.0` - User-related alert extending base alert.

### CTI Entities

By design, CTIs are just unique identifiers that do not carry the information about the underlying metadata.
Methods, structure members that can be used are not known without looking at the definition.

The library makes it easy to build more predictable code using strict data types for CTI entities,
allowing for the compile-time type checking and convenient interface.

A CTI entity can be either type or instance. The library provides an object model around these CTI concepts
expressed by the following two structs:

1. `EntityType`: Represents a CTI type with schema, traits schema, and traits.
2. `EntityInstance`: Represents a CTI instance with values conforming to a parent type.

All CTI entities are composed from the base `entity` struct that implements the `Entity` interface
for all common operations on CTI entities. Additionally, `EntityType` and `EntityInstance` implement
entity-specific methods which will be also covered in usage scenarios.

## Basic usage scenarios

In basic usage scenarios, it is assumed that you have a client that works with already processed CTI metadata.
This section will demonstrate how your implementation can work with CTIs.

Note that examples may be truncated for brevity.

### Working with entity types

Let's assume that we want to get an alert type by `cti.a.p.alert.v1.0~a.p.user.v1.0` with the following definition:

```yaml
#%CTI Type 1.0
cti: cti.a.p.alert.v1.0~a.p.user.v1.0
# ...
traits:
  severity: MEDIUM
```

And we have a function that gets a CTI type from a storage:

```go
import "github.com/acronis/go-cti/metadata"

func getCTIType(cti string) *metadata.EntityType {
  // An example implementation that takes *metadata.EntityType
  // from the storage and returns to the caller.
  return &metadata.EntityType{
    CTI: "cti.a.p.alert.v1.0~a.p.user.v1.0",
    // ...
    Traits: map[string]any{"severity": "MEDIUM"}
  }
}
```

With the following processing function, we can get the metadata associated with this CTI, access its `CTI` field
and read its traits using the `GetMergedTraits` method:

```go
func processAlert() {
  // Get the CTI type by identifier from the storage
  userAlertType := getCTIType("cti.a.p.alert.v1.0~a.p.user.v1.0")
  // Will print: "Alert type 'CTI' is cti.a.p.alert.v1.0~a.p.user.v1.0"
  fmt.Printf("Alert type 'CTI' is %s", userAlertType.CTI)

  // Using EntityType.GetMergedTraits() to get the type traits.
  traits = userAlertType.GetMergedTraits()
  // Will print: "Alert type 'severity' is MEDIUM"
  fmt.Printf("Alert type 'severity' is %v", traits["severity"])
}
```

In cases where the received metadata is unknown and just implements the `metadata.Entity` interface,
you can identify the type using type `switch` and then pass this entity to the corresponding processing function.
For example:

```go
import (
    "fmt"

    "github.com/acronis/go-cti/metadata"
)

func processEntity(entity metadata.Entity) error {
  switch e := entity.(type) {
  case *metadata.EntityType:
  case *metadata.EntityInstance:
  default:
    return fmt.Errorf("invalid entity: %s", entity.GetCTI())
  }
}
```

### Validating the data against the schema

Let's assume we have the following definitions with type schemas:

```yaml
#%CTI Type 1.0
cti: cti.a.p.alert.v1.0
# ...
schema:
  $schema: http://json-schema.org/draft-07/schema#
  $ref: "#/definitions/Alert"
  definitions:
    Alert:
      properties:
        id:
          type: string
          description: A unique identifier of the alert.
          format: uuid
        data:
          type: object
          description: An alert payload.
          x-cti.overridable: true # This property can be specialized by other vendors.
      required: [ id, type ]

--

#%CTI Type 1.0
cti: cti.a.p.alert.v1.0~a.p.user.v1.0
# ...
schema:
  $schema: http://json-schema.org/draft-07/schema#
  $ref: "#/definitions/UserLoginAttemptAlert"
  definitions:
    UserLoginAttemptAlert:
      type: object
      properties:
        data:
          type: object
          description: An alert payload.
          properties:
            user_agent:
              type: string
              description: A User-Agent of the browser that was used in log in attempt.
          required: [ user_agent ]
      required: [ id, type, data ]
```

In order to validate against the `cti.a.p.alert.v1.0~a.p.user.v1.0` schema, you can use the `Validate` or `ValidateBytes`
method. These methods take care of making a complete schema and caching, making subsequent calls fast and simple.

```go
import (
  "fmt"
  "encoding/json"

  "github.com/acronis/go-cti/metadata"
)

// AlertRequest is a payload wrapper that provides the information
// about the underlying payload type.
type AlertRequest struct {
  Type    string          `json:"type"`
  Payload json.RawMessage `json:"payload"`
}

// AlertBase is the alert type as defined by cti.a.p.alert.v1.0.
type AlertBase struct {
  ID    string  `json:"id"`
  Data  any     `json:"data"`
}

/*
Assuming that payload is the following JSON text:
{
  "type": "cti.a.p.alert.v1.0~a.p.user.v1.0",
  "payload": {
    "id": "71128772-75ce-46fd-ae22-251503a17961",
    "data": { "user_agent": "MyUserAgent/1.0.0" }
  }
}
*/
func validateData(payload []byte) error {
  var req AlertRequest
  if err := json.Unmarshal(payload, &req) {
    return err
  }
  // Get the CTI type by identifier from the storage
  userAlertType := getCTIType(req.Type)
  // Validate the payload against the type
  err := userAlertType.ValidateBytes([]byte(req.Payload))
  if err != nil {
    return fmt.Errorf("validation failed: %w", err)
  }
}
```

In case your base type provides information about the type within the payload, you can use third-party library
like [tidwall/gjson](https://github.com/tidwall/gjson) to extract the type and then validate against it like
in the following example:

```go
import (
  "fmt"
  "encoding/json"

  "github.com/tidwall/gjson"
  "github.com/acronis/go-cti/metadata"
)

// AlertBase is the alert type as defined by cti.a.p.alert.v1.0.
type AlertBase struct {
  ID    string   `json:"id"`
  Type  string   `json:"type"`
  Data  any      `json:"data"`
}

/*
Assuming that payload is the following JSON text:
{
  "id": "71128772-75ce-46fd-ae22-251503a17961",
  "type": "cti.a.p.alert.v1.0~a.p.user.v1.0",
  "data": { "user_agent": "MyUserAgent/1.0.0" }
}
*/
func validateData(payload []byte) error {
  typ := gjson.Get(payload, "type")
  // Get the CTI type by identifier from the storage
  userAlertType := getCTIType(typ.String())
  // Validate the payload against the type
  err := userAlertType.ValidateBytes(payload)
  if err != nil {
    return fmt.Errorf("validation failed: %w", err)
  }
}
```

### Getting CTI type traits

In order to enforce type-specific behavior, you can get CTI traits of the type. Recommended method is using
the `GetMergedTraits` method that will collect all traits in the CTI chain. Let's assume the following
chain of CTIs where a base type defines `severity` and `expiry_duration` traits:

```yaml
#%CTI Type 1.0
cti: cti.a.p.alert.v1.0
# ...
traits_schema:
  $schema: http://json-schema.org/draft-07/schema#
  $ref: "#/definitions/cti-traits"
  definitions:
    cti-traits:
      type: object
      properties:
        severity:
          type: string
          description: A severity of the alert.
          enum: [ LOW, MEDIUM, HIGH, CRITICAL ]
        expiry_duration:
          type: string
          description: Whether to send a notification for this alert.
      required: [ severity ]
```

A derived type specifies the alert severity for all user-related alerts, which is mandatory:

```yaml
#%CTI Type 1.0
cti: cti.a.p.alert.v1.0~a.p.user.v1.0
# ...
traits:
  severity: MEDIUM
  expiry_duration: 1h
```

And second derived type that specifies the expiry duration trait for a specific user alert:

```yaml
#%CTI Type 1.0
cti: cti.a.p.alert.v1.0~a.p.user.v1.0~a.p.login_attempt.v1.0
# ...
traits:
  severity: MEDIUM # Also specifies it since "severity" is mandatory, even though parent specifies it.
  # expiry_duration: 1h # Inherits "expiry_duration" since it's optional, but specified by parent.
```

The following example will demonstrate the outcomes:

```go
func getAlertTraits() {
  // Get the first derived CTI type by identifier from the storage
  userAlertType := getCTIType("cti.a.p.alert.v1.0~a.p.user.v1.0")
  traits := userAlertType.GetMergedTraits()
  // Will print: "Alert type 'severity' is MEDIUM"
  fmt.Printf("Alert type 'severity' is %v", traits["severity"])
  // Will print: "Alert type 'expiry_duration' is 1h"
  fmt.Printf("Alert type 'expiry_duration' is %v", traits["expiry_duration"])

  // Get the second derived CTI type by identifier from the storage
  userAlertType := getCTIType("cti.a.p.alert.v1.0~a.p.user.v1.0~a.p.login_attempt.v1.0")
  traits = userAlertType.GetMergedTraits()
  // Will print: "Alert type 'severity' is MEDIUM"
  fmt.Printf("Alert type 'severity' is %v", traits["severity"])
  // Will print: "Alert type 'expiry_duration' is 1h"
  fmt.Printf("Alert type 'expiry_duration' is %v", traits["expiry_duration"])
}
```

### Getting CTI instance values

CTI instances are particularly useful for defining static configuration, such as predefined service configuration,
intermediate mapping, templates, rules, etc. Let's assume the following event topic definition that defines
the `name` and `retention` of the topic:

```yaml
#%CTI Type 1.0
cti: cti.a.p.topic.v1.0
# ...
schema:
  $schema: http://json-schema.org/draft-07/schema#
  $ref: "#/definitions/Topic"
  definitions:
    Topic:
      type: object
      properties:
        name:
          type: string
          description: A name of the topic.
        retention:
          type: string
          description: A retention duration of the events created in this topic.
      required: [ name, retention ]
```

A topic configuration instance may then be created to represent a specific topic in the system,
such as a topic for user-related events:

```yaml
#%CTI Instance 1.0
cti: cti.a.p.topic.v1.0~a.p.user.v1.0
final: true
access: public
display_name: User-related events topic
description: A topic for user-related events.
values:
  name: User-related events topic.
  retention: 30d
```

With the following example, you can obtain values of CTI instance:

```go
func getTopicConfiguration() {
  // Get the first derived CTI type by identifier from the storage
  topicInstance := getCTIInstance("cti.a.p.topic.v1.0~a.p.user.v1.0")
  // Cast to map[string]any since Topic is object
  values := topicInstance.Values.(map[string]any)
  // Will print: "Topic 'name' is 'User-related events topic.'"
  fmt.Printf("Topic 'name' is '%v'", traits["name"])
  // Will print: "Topic 'retention' is 30d"
  fmt.Printf("Topic 'retention' is %v", traits["retention"])
}
```

## Advanced usage scenarios

Advanced usage scenarios cover scenarios

### Getting CTI parent

Any entity that implements `metadata.Entity` provides a reference to the parent, if present. The parent is
**always** of the `metadata.EntityType` type. For example, given the following two definitions of parent and child:

```yaml
#%CTI Type 1.0
cti: cti.a.p.alert.v1.0
# ...

--

#%CTI Type 1.0
cti: cti.a.p.alert.v1.0~a.p.user.v1.0
# ...
```

If we receive the `cti.a.p.alert.v1.0~a.p.user.v1.0` type, we can get its parent using the `Parent` method:

```go
import (
  "fmt"

  "github.com/acronis/go-cti/metadata"
)

func processEntity(entity *metadata.EntityType) error {
  // Assuming that received type is "cti.a.p.alert.v1.0~a.p.user.v1.0"
  // Will print: "received entity: cti.a.p.alert.v1.0~a.p.user.v1.0"
  fmt.Printf("received entity: %s", entity.CTI)
  parent := entity.Parent()
  // Parent will not be nil in this case
  if parent == nil {
    fmt.Printf("entity %s has no parent", entity.CTI)
  } else {
      // Will print: "entity parent is cti.a.p.alert.v1.0"
    fmt.Printf("entity parent is %s", parent.CTI)
  }
}
```

# CTI Types & Instances management tool and library

## What is CTI Types & Instances?

**CTI Types and Instances (CTI)** is a technology that provides a unified, vendor-agnostic way to define and uniquely identify types and instances, extend and package them. With CTI, types and instances are identified by CTI Typed Identifier that is associated with a particular entity.

## What does this project provide?

The project provides the following:

* An extensible library that provides interfaces for:
  * A parser for RAMLx files that are extended with CTI specification.
  * CTI package management to work with dependent packages in other Github repositories.
  * A validator for compiled CTI entities.
* A CLI tool that is ready to use with CTI packages and implements functionality according to the interface.

## How the technology is used

CTI Types and Instances (CTI) technology is utilized by Acronis Cyber Application technology that allows third-party ISVs (application vendors) to extend Acronis Cyber Protect Cloud platform (the platform) by:

* Bringing new object types and APIs to the system.
* Extending the platform base domain model types (like types of tenants, alerts, events, protection plans) by new inherited types.
* Enforce granular access to the objects of different types for the API clients.

With CTI Typed Identifier, the following entities become explicitly defined and linked to corresponding entities:

* Domain object types, i.e. object schemas like tenants, alerts, protection plans, etc.
* Well-known object instances, like event topics, namespaces, groups.

To describe types and instances that are associated with the CTI identifiers, RAMLx is used.

## Installation

### Library

```
go get -u github.com/acronis/go-cti
```

### CLI

```
go install github.com/acronis/go-cti/cmd/cti@latest
```

#### Example usage

> [!NOTE]
> By default, all commands are executed in the current working directory.
> You can use the global `--working-dir` argument to specify the working directory if necessary.

##### cti init

Initializes a CTI package. Writes `index.json` and `.ramlx` folder with CTI specification files for RAMLx.

Example:

```
cti init
```

##### cti pkg get \<git_remote\>

Fetches the package from the specified git remote.

Example:

```
cti pkg get github.com/acronis/sample-package
```

##### cti validate

Parses and validates the package against RAMLx.

Example:

```
cti validate
```

##### cti pack

Packs the package into a bundle.

Example:

```
cti pack
```

# Cross-domain Typed Identifiers (CTI) management tool and library

- [Cross-domain Typed Identifiers (CTI)?](#what-is-cross-domain-typed-identifiers-cti)
- [What does this project provide?](#what-does-this-project-provide)
- [How the technology is used](#how-the-technology-is-used)
- [Installation](#installation)
  - [Library](#library)
  - [CLI](#cli)
- [CLI Reference](#cli-reference)
  - [cti init](#cti-init)
  - [cti pkg get](#cti-pkg-get)
  - [cti validate](#cti-validate)
  - [cti pack](#cti-pack)
    - [--include-source](#--include-source)
    - [--format](#--format)
    - [--prefix](#--prefix)
    - [--output](#--output)


## What is Cross-domain Typed Identifiers (CTI)?

**Cross-domain Typed Identifiers (CTI)** provide a structured, standardized approach for uniquely identifying data types, instances, and their relationships across multi-service, multi-vendor, multi-platform, and multi-application environments. This system enables the unique definition of resources (both data types and instances) throughout the ecosystem, embedding vendor, package, and extension information within each identifier. Using CTI, each type and instance is represented by a **CTI Typed Identifier** associated with a specific entity, ensuring clear, consistent identification across diverse systems.

> [!NOTE]
> For more details on CTI specification, see [Cross-domain Typed Identifiers (CTI) version 1.0 Specification](./cti-spec/SPEC.md)

## What does this project provide?

The project provides the following:

* An extensible library that provides interfaces for:
  * A parser for RAMLx files that are extended with CTI specification.
  * CTI package management to work with dependent packages in other Github repositories.
  * A validator for compiled CTI entities.
* A CLI tool that is ready to use with CTI packages and implements functionality according to the interface.

## How the technology is used

Cross-domain Typed Identifiers (CTI) technology is utilized by Acronis Cyber Application technology that allows third-party ISVs (application vendors) to extend Acronis Cyber Protect Cloud platform (the platform) by:

* Bringing new object types and APIs to the system.
* Extending the platform base domain model types (like types of tenants, alerts, events, protection plans) by new inherited types.
* Enforce granular access to the objects of different types for the API clients.

With CTI identifiers, the following entities become explicitly defined and linked to corresponding entities:

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

## CLI Reference

> [!NOTE]
> By default, all commands are executed in the current working directory.
> You can use the global `--working-dir` argument to specify the working directory if necessary.

### cti init

Initializes a CTI package. Writes `index.json` and `.ramlx` folder with CTI specification files for RAMLx.

Example:

```
cti init
```

### cti pkg get

```
cti pkg get <git_remote>@<git_ref>
```

Fetches the package from the specified git remote and append package in the dependencies list of current component.

Example:

```
cti pkg get github.com/acronis/sample-package@v1
```

### cti validate

Parses and validates the package against RAMLx.

Example:

```
cti validate
```

### cti pack

Packs the package into a bundle. Valid package should be in current working directory (or directory specified by `--working-dir`).

Example:


```shell
> cti pack --include-source --format zip --prefix output --output=sample-package.cti

> ls output
sample-package.cti
```

#### --include-source

Includes the source files into the bundle. By default, the source files are not included.
Hidden files (starting with a dot) are not included in the bundle.

#### --format

The format of the output bundle. Supported formats are `zip` and `tgz`. Default is `tgz`.

#### --prefix

The directory where the output bundle will be saved. Default is `.`.

#### --output

The name of the output bundle. Default is `bundle.cti`. Please note that the extension is not added automatically.

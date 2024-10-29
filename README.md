# CTI Types & Instances management tool and library

- [Problem Statement](#problem-statement)
- [CTI Introduction](#cti-introduction)
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


## Problem Statement

In systems with contributions from multiple independent parties or vendors, unique identification is essential for interoperability, data integrity, and effective management. Outside of software, this need is addressed by common identification patterns across various fields. **[Peripheral Component Interconnect Code (Vendor ID (VID), Device ID (DID), Class Codes) and ID Assignment](https://pcisig.com/sites/default/files/files/PCI_Code-ID_r_1_11__v24_Jan_2019.pdf)** associated to Peripheral Component Interconnect (PCI) devices, **[ISBN-13 codes](https://www.isbn-international.org/content/isbn-bar-coding)** for books, **[GTIN (Global Trade Item Number)](https://www.gtin.info/what-is-a-gtin/)** for products, **[MAC addresses](https://en.wikipedia.org/wiki/MAC_address)** for network devices, and **[Payment Card Numbers](https://www.iso.org/obp/ui/#iso:std:iso-iec:7812:-1:ed-5:v1:en)** for credit cards are examples of conventions that encode essential details about the vendor, category, and instance into an identification code.

In software, this challenge becomes even more complex due to the dynamic, multi-vendor nature of environments like operating systems, cloud platforms, IoT ecosystems, and distributed microservices architectures. Here, unique identifiers must account for multiple vendors contributing different applications, services, data types, or specific identifiable data instances. To prevent conflicts and collisions and to ensure scalability and security, it is critical to have a standardized identification system that distinguishes each data type and instance while encoding vendor, application, and version information.

Programming languages typically define types for basic scalar data, like integers, strings, and booleans. Enums are often used to identify fixed sets of values, such as instance IDs or limited categories. More complex data structures, however, are represented by classes or structures whose names are unique only within the context of a compiled program or shared libraries. This approach limits the reliability of class-based identifiers in systems that exchange data through APIs or shared data storage, especially in multi-vendor environments where global uniqueness and cross-vendor, cross-service consistency are required.

There are several established identification systems and conventions for specific distributed applications or systems, including **[UUID](https://datatracker.ietf.org/doc/html/rfc4122)** for Universally Unique IDentifier, **[MIME types](https://datatracker.ietf.org/doc/html/rfc2045)** for mime formats, **[Java Package Naming Convention](https://docs.oracle.com/javase/tutorial/java/package/namingpkgs.html)** for namespace uniqueness in Java, **[Amazon ARN](https://docs.aws.amazon.com/general/latest/gr/aws-arns-and-namespaces.html)** for AWS resources, **[Apple UTI](https://developer.apple.com/documentation/uniformtypeidentifiers)** for data types on Apple platforms, and **[URL](https://url.spec.whatwg.org)** for web resource locations. While effective within their intended contexts, these systems are generally not suited for broader, generic identification of diverse data types or data objects.

To address these issues, **CTI** (a recursive acronym for Cross-domain Typed Identifiers) notation provides a robust convention for identifying data entities across multi-service, multi-vendor, multi-platform and multi-application environments.

## CTI Introduction

**Cross-domain Typed Identifiers (CTI)** is a conventional system that provides a structured, standardized approach for uniquely identifying data types, instances and their relationships across multi-service, multi-vendor, multi-platform and multi-application environments. By encoding essential information about vendor, package, and version, CTI ensures consistent, scalable identification of data types and instances in shared data storage, API objects, and documentation. Designed to support interoperability, CTI enables cross-platform compatibility and prevents conflicts by assigning each data type and instance a globally unique identifier that retains meaning and structure across contexts.

CTI identifiers are not limited to identifying data structures and instances alone; they also enable several advanced capabilities:

- **Cross-references**: Allow fields in one data object (e.g., type A) to reference objects of a different type (e.g., type B), enabling data linkages and associations across types.
- **Data class inheritance**: Supports type hierarchies, allowing type B to inherit from type A, promoting reusability and consistency in data models.
- **Data structure grouping**: Facilitates grouping by vendor or package, simplifying the organization and management of data structures within multi-vendor environments.

These capabilities enable CTI to support the construction of comprehensive, distributed, multi-vendor, and multi-service data type graphs and domain models. With CTI-based identification, organizations and cross-vendor platforms can manage data structures throughout their lifecycle, including aspects like data object relationships, access control, dependency, and compatibility management—creating a robust framework for scalable and secure data type systems management.

> [!NOTE]
> For more details on CTI specification, see [CTI Types and Instances (CTI) version 1.0 Specification](./cti-spec/SPEC.md)

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

# scip-apex

A [SCIP](https://sourcegraph.com/github.com/sourcegraph/scip) indexer for Apex source code. Generates SCIP index files that enable code navigation features like go-to-definition, find-references, and hover documentation in Sourcegraph and compatible editors.

## Installation

Download the latest binary for your platform from the [Releases](https://github.com/octoberswimmer/scip-apex/releases) page.

## Usage

```bash
scip-apex index [source-dirs...] [flags]
```

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--output` | `-o` | `index.scip` | Output file path |
| `--project-root` | | current directory | Project root for relative paths |
| `--namespace` | `-n` | | Default namespace for loaded code |

### Examples

Index a standard Salesforce project:

```bash
scip-apex index force-app/main/default/classes
```

Index with a custom output path:

```bash
scip-apex index force-app/main/default/classes -o my-index.scip
```

Index with a namespace:

```bash
scip-apex index force-app/main/default/classes -n MyNamespace
```

## What Gets Indexed

- **Definitions**: Classes, interfaces, enums, triggers, methods, fields, properties
- **References**: Type references, method calls, field accesses, constructor invocations
- **Relationships**: Implements and extends relationships between types
- **Documentation**: Doc comments attached to definitions
- **Symbol kinds**: Mapped to SCIP symbol kinds (Class, Interface, Enum, Method, Constructor, Field, Property, Parameter, Variable)

## Supported File Types

- `.cls` - Apex classes, interfaces, and enums
- `.trigger` - Apex triggers

## Verification

Inspect the generated index with the `scip` CLI:

```bash
go install github.com/sourcegraph/scip/cmd/scip@latest
scip print index.scip
scip snapshot --from index.scip
```

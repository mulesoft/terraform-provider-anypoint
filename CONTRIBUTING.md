# Contributing Guide for terraform-provider-anypoint

This page covers the governance model, development workflow, and contribution requirements for the MuleSoft Anypoint Terraform Provider. Thanks for contributing!

## Governance Model

### Salesforce Sponsored

Only Salesforce/MuleSoft employees hold `admin` rights and make final decisions on accepted contributions.

## Getting Started

### Prerequisites

- Go 1.25+
- Terraform CLI
- `golangci-lint` (for linting)
- `terraform-plugin-docs` (auto-installed via `go generate`)

### Development Setup

```bash
git clone https://github.com/mulesoft/terraform-provider-anypoint.git
cd terraform-provider-anypoint
make dev-setup   # runs: deps, fmt, docs
```

To install the provider locally for manual testing:

```bash
make install
```

This builds the binary and places it under `~/.terraform.d/plugins/`.

## Project Structure

```
.
├── internal/
│   ├── provider/          # Provider registration and configuration
│   ├── resource/          # Managed resources
│   │   ├── accessmanagement/
│   │   ├── apimanagement/
│   │   ├── agentstools/
│   │   ├── cloudhub2/
│   │   └── secretsmanagement/
│   ├── datasource/        # Read-only data sources (mirrors resource/ layout)
│   └── acctest/           # Acceptance test helpers
├── examples/              # Per-resource Terraform usage examples
├── docs/                  # Auto-generated Terraform Registry documentation
├── scripts/               # Build and distribution utilities
└── .github/workflows/     # CI (test.yml, release.yml)
```

## Building

```bash
make build        # build for current platform
make build-all    # cross-platform builds (Windows, Linux, macOS x86_64/ARM64)
```

## Testing

### Unit Tests

```bash
make test
```

A minimum **25% code coverage** is enforced. The CI pipeline fails if coverage drops below this threshold.

```bash
make test-coverage   # generate local HTML coverage report
```

### Acceptance Tests

Acceptance tests make real API calls against an Anypoint Platform environment. Set the following environment variables before running:

| Variable | Description |
|---|---|
| `TF_ACC` | Set to `1` to enable acceptance tests |
| `ANYPOINT_CLIENT_ID` | Connected App client ID |
| `ANYPOINT_CLIENT_SECRET` | Connected App client secret |
| `ANYPOINT_BASE_URL` | Anypoint Platform base URL (optional, defaults to production) |

```bash
TF_ACC=1 \
ANYPOINT_CLIENT_ID=... \
ANYPOINT_CLIENT_SECRET=... \
make testacc
```

Acceptance tests have a 120-minute timeout.

## Code Style

Format all Go source and Terraform example files before submitting:

```bash
make fmt
```

Lint:

```bash
make lint
```

## Documentation

Docs under `docs/` are auto-generated from resource schemas and the content in `examples/`. Do not edit files in `docs/` directly — edit the schema descriptions or example `.tf` files instead, then regenerate:

```bash
make docs
```

## Distribution Examples

The `scripts/sanitize_dist_examples.sh` script scrubs hardcoded credentials, UUIDs, and placeholder values from example files before distribution. It replaces sensitive values with `<add-your-value-here>`. This runs automatically as part of the packaging pipeline — do not commit real credentials in `examples/`.

## Issues, Requests & Ideas

Use the [GitHub Issues](https://github.com/mulesoft/terraform-provider-anypoint/issues) page to report bugs, request features, and discuss ideas.

### Bug Reports

- Search existing issues before filing a new one.
- Include steps to reproduce, Terraform version, and provider version.
- Issues confirmed as bugs will be labelled `bug`.

### Feature Requests

- Describe the problem you want to solve in a new issue.
- Wait for maintainer feedback before investing significant implementation time.

## Contribution Checklist

- [ ] Code is formatted (`make fmt`) and lints cleanly (`make lint`)
- [ ] Unit tests pass and coverage stays at or above 25%
- [ ] New resources/data sources include acceptance tests
- [ ] Schema attributes have clear `Description` strings (used for doc generation)
- [ ] Examples under `examples/` do not contain real credentials or UUIDs
- [ ] `make docs` has been run if schemas or examples changed
- [ ] Commits are atomic with descriptive messages referencing the relevant issue

## Creating a Pull Request

1. Fork the repository and create a branch from `main`.
2. Make your changes following the checklist above.
3. Push your branch and open a Pull Request against `main`.
4. Reference any related issues in the PR description.
5. Sign the Salesforce CLA when prompted (required once per contributor).

> **Note:** Sync your fork with `main` before opening a PR to minimize merge conflicts.

## Contributor License Agreement (CLA)

You must sign the Salesforce CLA before we can accept your pull request. You only need to do this once across all Salesforce open source projects.

Sign here: <https://cla.salesforce.com/sign-cla>

## Code of Conduct

Please follow our [Code of Conduct](CODE_OF_CONDUCT.md).

## License

By contributing, you agree to license your code under the project [LICENSE](LICENSE.txt) and to sign the [Salesforce CLA](https://cla.salesforce.com/sign-cla).

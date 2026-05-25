# Contributing

Thanks for your interest. This project is a Kubernetes operator built with
[kubebuilder](https://book.kubebuilder.io/) and [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime).

## Prerequisites

- Go 1.25+
- Docker (or a compatible container runtime)
- `kubectl` 1.28+
- [`kind`](https://kind.sigs.k8s.io/) for e2e tests

## Development workflow

```sh
make generate    # Regenerate DeepCopy methods after editing api/v1alpha1/*_types.go
make manifests   # Regenerate CRDs and RBAC after editing kubebuilder markers
make fmt vet     # Format and vet
make lint        # Run golangci-lint
make test        # Unit + envtest controller tests (downloads envtest binaries on first run)
make test-e2e    # End-to-end tests against a Kind cluster
```

## Pull requests

- Branch off `main` with a descriptive name (`feat/...`, `fix/...`, `chore/...`, `docs/...`).
- Keep PRs focused and small. Mixed concerns slow review.
- Reference any related issue in the PR description.
- CI must pass (lint, test, test-e2e, helm-validate, test-chart).
- At least one approval from a CODEOWNER is required before merging.

## Reporting bugs / proposing features

Use the issue templates under `.github/ISSUE_TEMPLATE/`. For security
vulnerabilities, see [SECURITY.md](SECURITY.md).

## License

By contributing, you agree that your contributions will be licensed under the
[MIT License](LICENSE).

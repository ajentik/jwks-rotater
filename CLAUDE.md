# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

JWKS key rotater — a tool for rotating JSON Web Key Sets.

## Status

This project is in initial development. Update this file as the codebase takes shape.

## Active Technologies
- Go 1.22+ + controller-runtime v0.18+, kubebuilder v4, go-jose/v4, controller-runtime/pkg/metrics (Prometheus) (001-jwks-operator)
- Kubernetes Secrets (native) — no external datastore (001-jwks-operator)

## Recent Changes
- 001-jwks-operator: Added Go 1.22+ + controller-runtime v0.18+, kubebuilder v4, go-jose/v4, controller-runtime/pkg/metrics (Prometheus)

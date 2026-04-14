---
description: 'These instructions apply to all files in the project and should be followed by all developers working on the application.'
applyTo: '**/*'
---

# Testing Instructions

When changes are made it is required to validate the application by running automated and repeatable tests. You will never commit code that is not covered by tests, and all tests must pass before merging your code or returning back to the user. This ensures that the application remains stable and that new changes do not introduce bugs or regressions.

## Makefile

For local testing, we have a `Makefile` that provides convenient commands to run tests. Read this file when you want to know how to run tests locally. The `Makefile` includes commands for running unit tests, integration tests, and end-to-end tests, as well as commands for running tests with code coverage and generating test reports.

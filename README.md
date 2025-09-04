# Paratest

A Go test execution wrapper that parallelizes test execution for improved performance.

## Overview

Paratest is designed to work with `go test -exec` to automatically parallelize test execution when beneficial. It analyzes test packages and only applies parallelization when it will improve performance without breaking existing parallel test patterns. This pattern can be useful for parallelizing tests which are difficult to run in parallel in a single binary (e.g. tests which rely on OS signals). This type of test running parallelism is useful for Delve tests in particular those found in the `./pkg/proc` directory.

## Installation

```bash
go install github.com/go-delve/paratest@latest
```

## Usage

Use paratest as an execution wrapper with `go test`:

```bash
go test -exec paratest
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass: `go test -v`
5. Submit a pull request

## License

MIT

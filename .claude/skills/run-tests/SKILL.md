---
name: run-tests
description: Run Go tests with race detection and verbose output
---

# Run Tests

Run the agentboard test suite.

```bash
go test -race -v ./...
```

If tests fail, analyze the output and report which packages/tests failed with a summary of the errors.

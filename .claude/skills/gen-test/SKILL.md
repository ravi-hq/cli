---
name: gen-test
description: Use when generating a Go test file for a module following Ravi CLI test conventions
---

# Generate Go Test File

Generate a Go test file for a specified module, following all Ravi CLI test conventions.

## Arguments

The user should specify which module to generate tests for (e.g., "gen-test internal/api/secrets.go" or "gen-test pkg/cli/passwords.go").

## Conventions

- **File naming**: `<module>_test.go` in the same package directory
- **Package**: Same package as the module under test (no `_test` suffix — tests are white-box)
- **Table-driven tests**: Use `[]struct{ name string; ... }` with `t.Run(tt.name, ...)`
- **Test helpers**: Always mark with `t.Helper()` as the first line
- **Config isolation**: Use `withTempHome(t)` for any test that touches `config.LoadAuth()`, `config.SaveAuth()`, or `os.UserHomeDir()`
- **API tests**: Use `httptest.NewServer()` + `newTestClient(server.URL)` or `clientFromAuth(server.URL, auth)`
- **Crypto tests**: Use `testKeyPair(t)` (PIN "123456", zero salt) for deterministic keypair
- **Error checking**: Use `t.Fatalf` for setup failures, `t.Errorf` for assertion failures
- **Cleanup**: Use `defer cleanup()` or `defer server.Close()` immediately after creation

## Test Helpers Available

### `internal/api/` package
```go
withTempHome(t)                        // Temp HOME dir, returns (tmpDir, cleanup)
withAPIBaseURL(t, url)                 // Temp API URL override, returns cleanup
setupTestAuth(t, auth)                 // Save auth config to temp home
newTestClient(serverURL)               // Client with test-token, no disk config
clientFromAuth(serverURL, auth)        // Client with specific auth config
```

### `internal/crypto/` package
```go
testKeyPair(t)                         // Deterministic keypair (PIN "123456", zero salt)
testEncrypt(t, plaintext, kp)          // Encrypt with SealAnonymous
```

### `pkg/cli/` package
```go
withTempHome(t)                        // Same pattern as api package (redefined)
```

## Process

1. Read the target module to understand exported and unexported functions/types
2. Read existing `*_test.go` files in the same directory to match local patterns
3. Check if test helpers exist in the package (avoid redefining `withTempHome`, etc.)
4. Generate the test file with:
   - Proper imports (testing, httptest, encoding/json as needed)
   - Table-driven tests for functions with multiple input/output combinations
   - `withTempHome(t)` + `defer cleanup()` for config-dependent tests
   - `httptest.NewServer` mocks for API client tests
   - Both happy path and error/edge cases
5. Write the file to the same directory as the module

## Example Structure

```go
package api

import (
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestMyFunction(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"valid input", "hello", "HELLO", false},
        {"empty string", "", "", false},
        {"special chars", "e2e::data", "E2E::DATA", false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := MyFunction(tt.input)
            if (err != nil) != tt.wantErr {
                t.Fatalf("MyFunction(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
            }
            if got != tt.want {
                t.Errorf("MyFunction(%q) = %q, want %q", tt.input, got, tt.want)
            }
        })
    }
}

func TestMyAPIMethod(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`{"status": "ok"}`))
    }))
    defer server.Close()

    client := newTestClient(server.URL)
    // ... test the API method
}
```

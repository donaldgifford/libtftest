---
name: libtftest:add-sneakystack-service
description: >
  Scaffold a new gap-service handler in sneakystack/services/ for an AWS
  service LocalStack doesn't cover well. Use when adding support for SSO
  Admin, Organizations, Control Tower, or other Pro/edge services.
when_to_use: >
  When the user says "add a sneakystack handler for X", "we need to
  proxy Y in sneakystack", or when a consumer test fails because
  LocalStack doesn't support the service the module touches.
allowed-tools: [Read, Write, Edit, Bash, Grep]
---

# libtftest:add-sneakystack-service

Scaffolds a new service handler in `sneakystack/services/` plus its
routing entry in `sneakystack/proxy.go` and a Store-typed wrapper for the
service's resources.

## Repo conventions

- Handler files: `sneakystack/services/<service>.go` (one per service)
- Each handler implements `sneakystack.ServiceHandler`
  (`Handle(w http.ResponseWriter, r *http.Request)`)
- The handler dispatches by AWS API operation. AWS SDKs use one of three
  patterns to identify the operation:
  - **JSON-RPC** (DynamoDB, SSO Admin, Organizations): `X-Amz-Target`
    header carries `<service>.<Operation>`
  - **REST-XML** (S3, IAM in some cases): URL path + HTTP method
  - **Query** (SQS, SNS): form-encoded `Action=<op>` parameter
- Storage uses the `sneakystack.Store` interface
  (Put/Get/List/Delete by `kind` + `id`). Each handler wraps the
  generic Store with typed accessors for its resources.

## Procedure

### 1. Confirm the service

Ask the user:

- **Service name** (e.g., `sso-admin`, `organizations`, `controltower`)
- **Dispatch protocol**: JSON-RPC / REST-XML / Query. If unsure, find
  the AWS SDK documentation for the service — JSON-RPC services declare
  `X-Amz-Target`, REST services use path matching.
- **Operations to support** (e.g., `CreatePermissionSet`,
  `ListPermissionSets`, `DescribePermissionSet`).
- **Resources to store**: typed structs matching what AWS would return.

### 2. Pick the right template

Based on dispatch protocol, use one of:

- `references/jsonrpc-handler.go.tmpl`
- `references/restxml-handler.go.tmpl`

If the service uses Query (SQS/SNS-style), generate from the JSON-RPC
template and document the mismatch — Query handlers are rare enough that
we ask the user to manually adjust the dispatch instead of maintaining
a third template.

### 3. Create the handler file

Place at `sneakystack/services/<service>.go`. The handler should:

- Be in package `services` (subdirectory)
- Embed or wrap a `sneakystack.Store` reference for typed access
- Dispatch by operation, return JSON or XML matching the AWS SDK's
  response shape
- Use stdlib `encoding/json` / `encoding/xml` — no third-party

### 4. Register in the proxy router

Add an entry to `sneakystack/proxy.go` `NewProxy`:

```go
case "sso-admin":
    handlers[svc] = services.NewSSOAdmin(store)
```

The `case` value matches the service name in `Config.Services`.

### 5. Generate a handler test

Use `httptest.NewRecorder` and `httptest.NewRequest`. Test:

- Each operation produces the expected response shape
- Unknown operations return 404 / "OperationNotImplemented"
- Errors propagate as AWS-shaped error responses

Place the test at `sneakystack/services/<service>_test.go`.

### 6. Run lint and test

```bash
make lint
make test-pkg PKG=./sneakystack/...
```

Common issues:

- `errcheck` on `w.Write` — explicitly check the error or use
  `fmt.Fprint` (which returns errors but is more idiomatic than
  ignoring `Write`'s return)
- `gocritic httpNoBody` — when generating a `*http.Request` with no
  body, use `http.NoBody` instead of `nil`
- `gocritic noctx` — the proxy passes context through; preserve it in
  the handler

### 7. Update docs

If the new service is meant for general use, add it to
`docs/examples/06-sneakystack.md`'s service list. Optional — ask the
user.

## Edge cases

- **AWS SDK is in a different language**: the SDK identifies the
  service via the request format, not language. Test against the Go AWS
  SDK and trust other SDKs use the same wire format.
- **Service has multiple endpoints** (e.g., separate read/write
  endpoints): the proxy doesn't currently distinguish; route both to
  the same handler and let the handler dispatch by operation.
- **Service uses presigned URLs**: very rare for sneakystack-targeted
  services. If needed, the handler must implement the URL-signing
  protocol — out of scope for the standard scaffold.

## References

- `references/jsonrpc-handler.go.tmpl` — JSON-RPC handler template
- `references/restxml-handler.go.tmpl` — REST-XML handler template
- `sneakystack/proxy.go` — router registration site
- `sneakystack/store.go` — Store interface
- `docs/examples/06-sneakystack.md` — consumer-facing usage docs

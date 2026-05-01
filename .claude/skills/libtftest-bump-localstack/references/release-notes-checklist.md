# LocalStack release-notes checklist

Areas to scan in LocalStack release notes when bumping the pinned image.
These are accumulated regressions / behavior changes we've actually hit.

## Where the release notes live

- Official changelog: <https://docs.localstack.cloud/references/changelog/>
- GitHub releases: <https://github.com/localstack/localstack/releases>
- Pro changelog (if applicable): same URL, look for `pro:` sections

## Known regression areas

For each area below, find the relevant section in the new version's
release notes and check whether anything changed.

### S3

- **CreateBucket XML format**: LS 4.4 returned `MalformedXML` with current
  AWS provider. Verify the new version handles the AWS provider's request
  shape correctly.
- **Path-style addressing**: libtftest uses path-style for S3
  (`UsePathStyle=true` in `awsx.NewS3`). Verify this still works.
- **Versioning + lifecycle rules**: cross-cutting features that often
  break.

### Health endpoint (`/_localstack/health`)

- Response shape changes here directly break `AllServicesReady` and
  `DetectEditionFromHealth` in `localstack/health.go`. Always verify
  the JSON structure matches what we parse.

### Edge port behavior

- LocalStack >=4.0 exposes ports starting at 4510, with the edge at 4566.
  Any change here breaks `PortEndpoint(ctx, "4566/tcp", "http")`. Verify
  the edge port is still 4566.

### Pro vs Community split

- Some services move between editions over time. Check whether anything
  in `references/pro-services.md` (in `libtftest:add-assertion` skill)
  needs updating.

### IAM enforcement

- LocalStack Pro IAM behavior shifts between minor versions. Tests that
  rely on policy *evaluation* (not just resource existence) may need
  adjustment.

### Init hooks

- `WriteInitHooks` writes scripts that LocalStack runs at startup. The
  hook directory (`/etc/localstack/init/ready.d/`) and execution order
  occasionally change. Verify in the changelog.

### Docker image

- Base image upgrades (e.g., debian-bookworm → trixie) sometimes break
  bind mounts or networking. Test with the standard
  `WithHostConfigModifier` path libtftest uses.

## Smoke-test recipe

After running `make bump-localstack LS_VERSION=<x>`:

1. `make lint` — catches any string-literal regressions
2. `make test` — unit tests
3. `make test PKG=./localstack/...` — container lifecycle integration
4. `make test PKG=./libtftest_integration_test.go` — full TestCase
   end-to-end (S3 module apply + assert)
5. If sneakystack changed: `make test PKG=./sneakystack/...`

## When to roll back

If steps 4-5 fail with errors that look like LocalStack-side regressions
(rather than libtftest bugs), roll back the bump:

```bash
git checkout -- .
```

Then file an upstream issue at
`https://github.com/localstack/localstack/issues` with a minimal
reproduction. Re-attempt the bump after upstream resolves.

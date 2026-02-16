# 🔐 Permissions (RBAC/ABAC)

Buffalo supports permission annotations and policy/code generation.

## Inspect and audit

```bash
buffalo permissions summary -p ./protos
buffalo permissions matrix -p ./protos
buffalo permissions audit -p ./protos
```

## Generate policies/code

```bash
buffalo permissions generate --framework go --output permissions.go
buffalo permissions generate --framework casbin --output policy.csv
buffalo permissions generate --framework opa --output authz.rego
```

## Recommended flow

1. Annotate service methods in `.proto`
2. Run `summary` and `matrix`
3. Fix `audit` warnings
4. Generate target framework artifacts

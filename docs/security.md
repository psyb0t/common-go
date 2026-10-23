# Security checks

`make sec` runs in the development image without Docker socket access. It scans the working tree and available Git history with Gitleaks, checks dependency manifests and container configuration with Trivy, and checks reachable Go dependency vulnerabilities with govulncheck. Go source checks, including gosec, run through `make lint`.

The command writes `sec.sarif` for GitHub Code Scanning. Secret findings, high or critical Trivy findings, reachable dependency vulnerabilities and scanner execution failures fail the command. govulncheck's report mode and its reachability gate both run, so writing a SARIF report does not replace enforcement. The report is gitignored.

The pipeline uses the same Make targets locally and in CI. Tag publishing waits for the default-branch run of the same commit. Dependency commands check third-party release age before changing pins, regenerate module checksums and vendor, and run the security checks afterward.

## OpenPGP advisory

govulncheck reports [GO-2026-5932](https://pkg.go.dev/vuln/GO-2026-5932) at module level because the module graph includes `golang.org/x/crypto`. The advisory concerns its unmaintained `openpgp` packages and has no fixed version. common-go does not import or vendor those packages, and govulncheck reports no reachable affected symbols. This note is not suppressed from the SARIF report. Do not introduce imports of those packages.

See the [project README](../README.md) for development commands.

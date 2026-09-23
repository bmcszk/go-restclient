# Security Policy

## Supported versions

Only the latest tagged release and master are supported with security fixes.

## Reporting a vulnerability

Please do NOT open a public issue for security reports.

Use GitHub's private vulnerability reporting:
**https://github.com/bmcszk/go-restclient/security/advisories/new**

You can expect an initial response within 7 days.

## Handling of credentials by the tool

- `.env` files and `--env` overlays are read locally; values are never sent anywhere except to the target host you specify in the `.http` request itself.
- OAuth2 `client_credentials` tokens are fetched from the token endpoint you configure and held in memory for the process lifetime; they are not written to disk by this tool.
- Secrets shown in verbose output are not redacted automatically — treat `--verbose` output as sensitive.

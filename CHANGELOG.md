# Changelog

## Unreleased

- fix: the "Create Action Suppression" action now captures the created suppression id, so it is actually deleted again on stop instead of leaving AppDynamics alerting suppressed indefinitely
- fix: guard the health-rule check against missing target attributes instead of panicking, and avoid a possible nil-dereference when an AppDynamics API request fails before a response is received

## v1.1.13

- chore(deps): bump github.com/steadybit/extension-kit

## v1.1.12

- chore(deps): bump golang.org/x/net to v0.55.0 (CVE-2026-39821) (#59)

## v1.1.11

- chore(deps): bump alpine from 3.23 to 3.24
- chore(deps): bump k8s.io/apimachinery from 0.36.1 to 0.36.2

## v1.1.10

- chore: update to go 1.26.4
- feat: add weekly auto patch-release workflow

## v1.1.9

- Support discovery group attribute via `STEADYBIT_EXTENSION_DISCOVERY_GROUP` env var (or `discovery.group` Helm value) — when set, the extension adds `steadybit.group=<value>` to every discovered target
- chart: use shared `extensionlib.deployment.env` helper so standard env vars (logging, TLS, discovery group) flow through consistently
- Update dependencies

## v1.1.8

- Bump Go to 1.26.3
- Update dependencies

## v1.1.7

- Support if-none-match for the extension list endpoint

## v1.1.6

- feat(chart): split image.name into image.registry + image.name
- Support global.priorityClassName
- Update alpine packages in Docker image to address CVEs
- Update dependencies

## v1.1.5

- Update dependencies

## v1.1.4

- Update dependencies

## v1.1.3

- Update dependencies

## v1.1.2

- Update dependencies

## v1.1.1

- Added an option to filter the discovery by a list of application ids.

## v1.1.0

- Breaking change - The access token is a short-lived token - Authentication is now done via OAUTH2.0 client credentials flow
  - Removed support for setting an access token via `STEADYBIT_EXTENSION_ACCESS_TOKEN` or `appdynamics.accessToken`
  - Added parameters client name, client secret and account name to the configuration.

## v1.0.0

 - Initial release

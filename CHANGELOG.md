# Changelog

## (next)

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

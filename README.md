# Steadybit extension-appdynamics

A [Steadybit](https://www.steadybit.com/) extension to integrate [AppDynamics](https://docs.appdynamics.com/) into Steadybit.

Learn about the capabilities of this extension in our [Reliability Hub](https://hub.steadybit.com/extension/com.steadybit.extension_appdynamics).

## Prerequisites

You need to have an [Api Client](https://docs.appdynamics.com/appd/23.x/latest/en/extend-appdynamics/appdynamics-apis/api-clients).

The token must have the following permissions:
- Account Owner

## Configuration

| Environment Variable                                             | Helm value                                | Meaning                                                                                                                                                                                                         | Required | Default |
|------------------------------------------------------------------|-------------------------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|----------|---------|
| `STEADYBIT_EXTENSION_API_BASE_URL`                               | appdynamics.apiBaseUrl                    | The base url for AppDynamics API Calls, for example `https://XXXXXXXXX.saas.appdynamics.com`                                                                                                                    | yes      |         |
| `STEADYBIT_EXTENSION_API_CLIENT_NAME`                            | appdynamics.apiClientName                 | The name of the API client.                                                                                                                                                                                     | yes      |         |
| `STEADYBIT_EXTENSION_API_CLIENT_SECRET`                          | appdynamics.apiClientSecret               | The secret of the API client.                                                                                                                                                                                   | yes      |         |
| `STEADYBIT_EXTENSION_ACCOUNT_NAME`                               | appdynamics.accountName                   | The name of the AppDynamics account, usually the first part of you url.                                                                                                                                         | yes      |         |
| `STEADYBIT_EXTENSION_EVENT_APPLICATION_ID`                       | appdynamics.eventApplicationID            | The extension reports experiment executions to AppDynamics if an Application Event ID (A manually created Steadybit App is sufficient) is given, which helps you to correlate experiments with your dashboards. | no       |         |
| `STEADYBIT_EXTENSION_ACTION_SUPPRESSION_TIMEZONE`                | appdynamics.actionSuppressionTimezone     | The timezone to enforce for the action suppression action in the form "Europe/Paris", if none, the local one will be determined where the extension is deployed (optional).                                     | no       |         |
| `STEADYBIT_EXTENSION_DISCOVERY_ATTRIBUTES_EXCLUDES_APPLICATIONS` | discovery.attributes.excludes.application | List of Application attributes to exclude from discovery.. Checked by key equality and supporting trailing "*"                                                                                                  | no       |         |
| `STEADYBIT_EXTENSION_DISCOVERY_ATTRIBUTES_EXCLUDES_HEALTH_RULES` | discovery.attributes.excludes.healthRule  | List of Health Rule attributes to exclude from discovery.. Checked by key equality and supporting trailing "*"                                                                                                  | no       |         |
| `STEADYBIT_EXTENSION_APPLICATION_FILTER`                         |                                           | List of Application IDs that should be reported by the extension. If not set, all applications will be discovered.                                                                                              | no       |         |

The extension supports all environment variables provided by [steadybit/extension-kit](https://github.com/steadybit/extension-kit#environment-variables).

## Installation

### Kubernetes

Detailed information about agent and extension installation in kubernetes can also be found in
our [documentation](https://docs.steadybit.com/install-and-configure/install-agent/install-on-kubernetes).

#### Recommended (via agent helm chart)

All extensions provide a helm chart that is also integrated in the
[helm-chart](https://github.com/steadybit/helm-charts/tree/main/charts/steadybit-agent) of the agent.

You must provide additional values to activate this extension.

```
--set extension-appdynamics.enabled=true \
--set extension-appdynamics.appdynamics.apiBaseUrl="{{API_BASE_URL}}" \
--set extension-appdynamics.appdynamics.apiClientName="{{API_CLIENT_NAME}}" \
--set extension-appdynamics.appdynamics.apiClientSecret="{{API_CLIENT_SECRET}}" \
--set extension-appdynamics.appdynamics.accountName="{{ACCOUNT_NAME}}" \
```

Additional configuration options can be found in
the [helm-chart](https://github.com/steadybit/extension-appdynamics/blob/main/charts/steadybit-extension-appdynamics/values.yaml) of the
extension.

#### Alternative (via own helm chart)

If you need more control, you can install the extension via its
dedicated [helm-chart](https://github.com/steadybit/extension-appdynamics/blob/main/charts/steadybit-extension-appdynamics).

```bash
helm repo add steadybit-extension-appdynamics https://steadybit.github.io/extension-appdynamics
helm repo update
helm upgrade steadybit-extension-appdynamics \
    --install \
    --wait \
    --timeout 5m0s \
    --create-namespace \
    --namespace steadybit-agent \
    --set appdynamics.apiBaseUrl="{{API_BASE_URL}}" \
    --set appdynamics.apiClientName="{{API_CLIENT_NAME}}" \
    --set appdynamics.apiClientSecret="{{API_CLIENT_SECRET}}" \
    --set appdynamics.accountName="{{ACCOUNT_NAME}}" \
    steadybit-extension-appdynamics/steadybit-extension-appdynamics
```

### Linux Package

Please use
our [agent-linux.sh script](https://docs.steadybit.com/install-and-configure/install-agent/install-on-linux-hosts)
to install the extension on your Linux machine. The script will download the latest version of the extension and install
it using the package manager.

After installing, configure the extension by editing `/etc/steadybit/extension-appdynamics` and then restart the service.

## Extension registration

Make sure that the extension is registered with the agent. In most cases this is done automatically. Please refer to
the [documentation](https://docs.steadybit.com/install-and-configure/install-agent/extension-registration) for more
information about extension registration and how to verify.

## Version and Revision

The version and revision of the extension:
- are printed during the startup of the extension
- are added as a Docker label to the image
- are available via the `version.txt`/`revision.txt` files in the root of the image

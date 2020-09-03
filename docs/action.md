# Faythe action

- [Faythe action](#faythe-action)
  - [1. Action HTTP](#1-action-http)
  - [2. Action Mail](#2-action-mail)
  - [3. Action OpenStack Mistral](#3-action-openstack-mistral)

Faythe supports 3 types of Actions

## 1. Action HTTP

- REST action.
- Can be used in both Scaling & Healing.
- _For authentication, you can enable `cloud_auth_token` option to use Cloud provider authentication service or use additional header `header`._

| Name                    | Type    | Required | Default | Description                                                                                                                                                                                      |
| ----------------------- | ------- | -------- | ------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| action.url              | string  | true     |         | The action URL.                                                                                                                                                                                  |
| action.cloud_auth_token | boolean | false    | false   | If True, this action is called using Cloud provider authentication (Keystone if the provider is OpenStack).                                                                                      |
| action.header           | object  | false    |         | The additional headers for action request.                                                                                                                                                       |
| action.body             | object  | false    |         | The additional body for action request.                                                                                                                                                          |
| action.type             | string  | false    | http    | The type of action.                                                                                                                                                                              |
| action.method           | string  | false    | POST    | The HTTP method                                                                                                                                                                                  |
| action.attempts         | integer | false    | 10      | The count of retry.                                                                                                                                                                              |
| action.delay            | string  | false    | 100ms   | The delay between retries. Please refer [note](./note.md#time-durations) for formats.                                                                                                            |
| action.delay_type       | string  | false    | fixed   | The delay type: `fixed` or `backoff`. BackOffDelay is a DelayType which increases delay between consecutive retries. FixedDelay is a DelayType which keeps delay the same through all iterations |

## 2. Action Mail

- Mail action.
- Can be used only in Healing.
- Notify to users with a fixed email _template_ when alert is firing.

| Name             | Type | Required | Default | Description              |
| ---------------- | ---- | -------- | ------- | ------------------------ |
| action.receivers | list | true     |         | List of mail recipients. |

## 3. Action OpenStack Mistral

- [OpenStack Mistral](https://docs.openstack.org/mistral/latest/) workflow action.
- _Faythe allows to trigger a OpenStack Workflow execution by its ID. Mistral workflow is a definition of a set of tasks and transitions between them. User has to create a Workflow first and pass its ID to Faythe._
- Only supported with `openstack` cloud provider.
- Can be used only in Healing.

| Name               | Type   | Required | Default | Description                                                           |
| ------------------ | ------ | -------- | ------- | --------------------------------------------------------------------- |
| action.workflow_id | string | true     |         | Executing Mistral workflow. Only supported with `openstack` provider. |

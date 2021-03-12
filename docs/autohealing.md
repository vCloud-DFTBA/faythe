# Faythe Autohealing

> **BEFORE YOU READ:** We suggest you to go through [Getting Started](getting-started.md) and [Autoscaling](autoscaling.md) to have understanding of the big picture.

- [Faythe Autohealing](#faythe-autohealing)
  - [1. Name Resolver](#1-name-resolver)
    - [1.1. Overview](#11-overview)
    - [1.2. NResolver API](#12-nresolver-api)
  - [2. Silencer](#2-silencer)
    - [2.1. Overview](#21-overview)
    - [2.2. Sync silences (experimental)](#22-sync-silences-experimental)
      - [2.2.1. How it works](#221-how-it-works)
      - [2.2.2. Conditions](#222-conditions)
    - [2.3. Silencer API](#23-silencer-api)
      - [2.3.1. Create Silencer](#231-create-silencer)
      - [2.3.2. List Silencer](#232-list-silencer)
      - [2.3.3. Delete Silencer/Expire Silencer](#233-delete-silencerexpire-silencer)
  - [3. Healer](#3-healer)
    - [3.1. Overview](#31-overview)
    - [3.2. Healer API](#32-healer-api)
      - [3.2.1. Create healer](#321-create-healer)
      - [3.2.2. List healer](#322-list-healer)
      - [3.2.3. Delete healer](#323-delete-healer)

Faythe autohealing basically does the job that automatically migrate VMs on hosts if predicted problems occur.

Faythe autohealing contains 3 major components:

- Name Resolver
- Silencer
- Healer

## 1. Name Resolver

### 1.1. Overview

Name resolver as the name it tells, it resolve host IP address to host's name.

Some cloud providers, OpenStack for example, execute commands based on host's name not the IP address. On the other hand, some metrics backends only care about host IP address. Hence, faythe needs to store the mapping of host IP address and its name.

### 1.2. NResolver API

You do not need to care much about Name Resolver because it is created along with the cloud provider.

However, you can get all the Name Resolver just the same as you do with cloud provider.

```json
GET /nresolvers
Resp
{
  "Status": "OK",
  "Data": {
    "/nresolvers/848cf56b1fb6641570a824fde994456b/847a20c465fa0131afd3090ec2f6b8e0": {
      "address": {
        "backend": "prometheus",
        "address": "http://127.0.0.1:9091/",
        "metadata": null,
        "username": "admin",
        "password": "supersecretpassword"
      },
      "ID": "847a20c465fa0131afd3090ec2f6b8e0",
      "interval": "30s",
      "cloudid": "848cf56b1fb6641570a824fde994456b"
    },
    "/nresolvers/eb31219d766fde6d8f2d8bcad6269175/dfd8327e456413db7b3b493ef262cf20": {
      "address": {
        "backend": "prometheus",
        "address": "http://192.168.1.1:9091/",
        "metadata": null,
        "username": "admin",
        "password": "topsecretpassword"
      },
      "ID": "dfd8327e456413db7b3b493ef262cf20",
      "interval": "30s",
      "cloudid": "eb31219d766fde6d8f2d8bcad6269175"
    }
  },
  "Err": ""
}
```

## 2. Silencer

### 2.1. Overview

Silencers come in handy if you want to add a set of ignored hosts in case of maintenance.

### 2.2. Sync silences (experimental)

If your metric backend is `Prometheus`, Faythe is able to sync from Prometheus Alertmanager(s) associated with the Prometheus backend. User has to enable this feature when creating Healer, it is disable by default.

#### 2.2.1. How it works

- Faythe retrieves the Prometheus backend's configuration then get Prometheus Alertmanager's urls and setup clients.
- Faythe queries the active silences which satisfy [conditions](#222-conditions) from Prometheus Alertmanagers every **60 seconds** (Yes, this is fixed value, at least by now), converts them to Faythe silence format.
- The Healer's silences dict will be updated with the new silences.
- Note that, to reduce complexity, this is **one-way sync process from Alertmanager to Faythe**.
- The logic is quite simple:

| Alertmanager                  | Faythe                                                                                             |
| ----------------------------- | -------------------------------------------------------------------------------------------------- |
| Create/recreate a new silence | Create a new silence                                                                               |
| Edit a silence                | Find the exist silence, check if pattern/expiration time was updated, force recreate a new silence |
| Expire a silence              | Delete the exist silence                                                                           |

#### 2.2.2. Conditions

Faythe doesn't get all the silences from Prometheus Alertmanager. Here are the restricted conditions user has to follow when creating Prometheus Alertmanager's silence:

- Silence's comment has to start with `[faythe]` prefix and contains all Faythe healer's tags. For example:

```
# Faythe Healer's tags
Tags: ["autohealing", "openstack-production-cluster"]
# Alertmanager silence's comment
Comment: '[faythe][autohealing][openstack-production-cluster] Silence for maintainance'
```

- Silence's matcher has to be on `instance` label.

### 2.3. Silencer API

#### 2.3.1. Create Silencer

Parameter explains:

**PATH**: `/silences/{provider-id}`

**METHOD**: `POST`

| Parameter   | In   | Type   | Required | Default | Description                                                                |
| ----------- | ---- | ------ | -------- | ------- | -------------------------------------------------------------------------- |
| name        | body | string | true     |         | Name of silencer                                                           |
| pattern     | body | string | true     |         | Regex pattern of silencer                                                  |
| ttl         | body | string | true     |         | Time to live, re.the format please refer [note](./note.md#time-durations). |
| tags        | body | list   | false    |         | Silencer tags                                                              |
| description | body | string | false    |         | Silencer description                                                       |

For example:

```json
POST /silencers/848cf56b1fb6641570a824fde994456b
{
    "name": "silence 6f 180",
    "pattern": "10.6.1.18.*",
    "ttl": "2h",
    "tags": ["silence", "6f", "180"]
}
Resp
{
    "Status": "OK",
    "Data": null,
    "Err": ""
}
```

#### 2.3.2. List Silencer

Silencers of a cloud provider can be listed in:

**PATH**: `/silences/{provider-id}`

**METHOD**: `GET`

#### 2.3.3. Delete Silencer/Expire Silencer

Silencer is automatically deleted and expired after reaching TTL duration. However, you can manually delete it by:

**PATH**: `/silences/{provider-id}/{silencer-id}`

**METHOD**: `DELETE`

For example:

```json
DELETE /silences/848cf56b1fb6641570a824fde994456b/e57e170aa842ee5f42ef397b6f0df072
Resp
{
    "Status": "OK",
    "Data": null,
    "Err": ""
}
```

## 3. Healer

### 3.1. Overview

Healer is the core of this module. Healer receives the query from user and evaluate the need of healing for hosts based on that query.

You can also define the level of evaluation, that means healing is only triggered if it meets the required level of evaluation. For example, if the evaluation level is 2 then healer must receives 2 kinds if metric before triggering actions.

### 3.2. Healer API

Healer has 3 APIs as usual: create, list, delete

#### 3.2.1. Create healer

Currently, we only support one healer per cloud provider. For supported actions, please check [here](./action.md)

**PATH**: `/healers/{provider-id}`

**METHOD**: `POST`

| Parameter               | In   | Type    | Required | Default                                                 | Description                                                                                                                                                                                      |                                                                                                                                                                           |
| ----------------------- | ---- | ------- | -------- | ------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| query                   | body | string  | true     | up{job=~\"._compute-cadvisor._\|._compute-node._\"} < 1 | Query that will be executed against the Prometheus API. See [the official documentation](https://prometheus.io/docs/prometheus/latest/querying/basics/) for more details.                        | Query that will be executed against the Prometheus API. See [the official documentation](https://prometheus.io/docs/prometheus/latest/querying/basics/) for more details. |
| action                  | body | object  | true     |                                                         | List of actions when healing is triggered                                                                                                                                                        |
| action.receivers        | body | list    | false    |                                                         | List of receivers in mail action                                                                                                                                                                 |
| action.workflow_id      | body | string  | false    |                                                         | Executing Mistral workflow. Only supported with `openstack` provider. `Required` if `action.type==mistral`                                                                                       |
| action                  | body | object  | true     |                                                         | The defined scale actions.                                                                                                                                                                       |
| action.url              | body | string  | true     |                                                         | The action URL.                                                                                                                                                                                  |
| action.cloud_auth_token | body | boolean | false    | false                                                   | If True, this action is called using Cloud provider authentication (Keystone if the provider is OpenStack).                                                                                      |
| action.header           | body | object  | false    |                                                         | The additional headers for action request.                                                                                                                                                       |
| action.body             | body | object  | false    |                                                         | The additional body for action request.                                                                                                                                                          |
| action.type             | body | string  | false    | http                                                    | The type of action.                                                                                                                                                                              |
| action.method           | body | string  | false    | POST                                                    | The HTTP method                                                                                                                                                                                  |
| action.attempts         | body | integer | false    | 10                                                      | The count of retry.                                                                                                                                                                              |
| action.delay            | body | string  | false    | 100ms                                                   | The delay between retries. Please refer [note](./note.md#time-durations) for formats.                                                                                                            |
| action.delay_type       | body | string  | false    | fixed                                                   | The delay type: `fixed` or `backoff`. BackOffDelay is a DelayType which increases delay between consecutive retries. FixedDelay is a DelayType which keeps delay the same through all iterations |
| interval                | body | string  | true     | 18s                                                     | The time between two continuous evaluate, re.the format please refer [note](./note.md#time-durations).                                                                                           |
| receivers               | body | list    | true     |                                                         | List of email receiving healing notifications                                                                                                                                                    |
| duration                | body | string  | true     | 3m                                                      | The total evaluation time, re.the format please refer [note](./note.md#time-durations).                                                                                                          |
| description             | body | string  | false    |                                                         |                                                                                                                                                                                                  |
| tags                    | body | list    | false    |                                                         |                                                                                                                                                                                                  |
| active                  | body | boolean | true     | false                                                   | Enable the healer or not.                                                                                                                                                                        |
| sync_silences           | body | boolean | false    | false                                                   | Enable sync silences feature (Prometheus backend is the only supported).                                                                                                                         |

For example:

```json
POST /healers/eb31219d766fde6d8f2d8bcad6269175
{
	"actions": {
		"http": {
			"attempts": 4,
			"delay": "50ms",
			"type": "mail",
			"delay_type": "backoff"
		},
		"mail": {
			"url": "https://127.0.0.1/api/v1/webhooks/autohealing",
			"attempts": 4,
			"delay": "50ms",
			"type": "http",
			"delay_type": "backoff",
			"method": "POST"
		}
	},
  "receivers": ["cloud@example.com"],
	"duration": "2m",
	"tags": [
		"autohealing",
		"5f"
	],
  "active": true,
  "sync_silences": true
}
Resp
{
    "Status": "OK",
    "Data": null,
    "Err": ""
}
```

#### 3.2.2. List healer

**PATH**: `/healers/{provider-id}`

**METHOD**: `GET`

#### 3.2.3. Delete healer

**PATH**: `/healers/{provider-id}/{healer-id}`

**METHOD**: `DELETE`

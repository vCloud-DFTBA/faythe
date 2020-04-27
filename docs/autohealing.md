# Faythe Autohealing

> **BEFORE YOU READ:** We suggest you to go through [Getting Started](getting-started.md) and [Autoscaling](autoscaling.md) to have understanding of the big picture.

- [Faythe Autohealing](#faythe-autohealing)
  - [Name Resolver](#name-resolver)
    - [NResolver API](#nresolver-api)
  - [Silencer](#silencer)
    - [Silencer API](#silencer-api)
      - [Create Silencer](#create-silencer)
      - [List Silencer](#list-silencer)
      - [Delete Silencer/Expire Silencer](#delete-silencerexpire-silencer)
  - [Healer](#healer)
    - [Healer API](#healer-api)
      - [Create healer](#create-healer)
      - [List healer](#list-healer)
      - [Delete healer](#delete-healer)

Faythe autohealing basically does the job that automatically migrate VMs on hosts if predicted problems occur.

Faythe autohealing contains 3 major components:

- Name Resolver
- Silencer
- Healer

## Name Resolver

Name resolver as the name it tells, it resolve host IP address to host's name.

Some cloud providers, OpenStack for example, execute commands based on host's name not the IP address. On the other hand, some metrics backends only care about host IP address. Hence, faythe needs to store the mapping of host IP address and its name.

### NResolver API

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

## Silencer

Silencers come in handy if you want to add a set of ignored hosts in case of maintenance.

### Silencer API

#### Create Silencer

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

#### List Silencer

Silencers of a cloud provider can be listed in:

**PATH**: `/silences/{provider-id}`

**METHOD**: `GET`

#### Delete Silencer/Expire Silencer

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

## Healer

Healer is the core of this module. Healer receives the query from user and evaluate the need of healing for hosts based on that query.

You can also define the level of evaluation, that means healing is only triggered if it meets the required level of evaluation. For example, if the evaluation level is 2 then healer must receives 2 kinds if metric before triggering actions.

### Healer API

Healer has 3 APIs as usual: create, list, delete

#### Create healer

Currently, we only support one healer per cloud provider.

**PATH**: `/healers/{provider-id}`

**METHOD**: `POST`

| Parameter          | In   | Type    | Required | Default                                                 | Description                                                                                                                                                                                      |
| ------------------ | ---- | ------- | -------- | ------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| query              | body | string  | true     | up{job=~\"._compute-cadvisor._\|._compute-node._\"} < 1 | Query that will be executed against the Prometheus API. See [the official documentation](https://prometheus.io/docs/prometheus/latest/querying/basics/) for more details.                        | Query that will be executed against the Prometheus API. See [the official documentation](https://prometheus.io/docs/prometheus/latest/querying/basics/) for more details. |
| action             | body | object  | true     |                                                         | List of actions when healing is triggered                                                                                                                                                        |
| action.receivers   | body | list    | false    |                                                         | List of receivers in mail action                                                                                                                                                                 |
| action.url         | body | string  | false    |                                                         | The url that the action will call to                                                                                                                                                             |
| action.workflow_id | body | string  | false    |                                                         | Executing Mistral workflow. Only supported with `openstack` provider. `Required` if `action.type==mistral`                                                                                       |
| actions.type       | body | string  | false    | http                                                    | The type of action. Currently support `mail`, `http`, `mistral`.                                                                                                                                 |
| actions.method     | body | string  | false    | POST                                                    | The HTTP method                                                                                                                                                                                  |
| actions.attempts   | body | integer | false    | 10                                                      | The count of retry.                                                                                                                                                                              |
| actions.delay      | body | string  | false    | 100ms                                                   | The delay between retries, re.the format please refer [note](./note.md#time-durations).                                                                                                          |
| actions.delay_type | body | string  | false    | fixed                                                   | The delay type: `fixed` or `backoff`. BackOffDelay is a DelayType which increases delay between consecutive retries. FixedDelay is a DelayType which keeps delay the same through all iterations |
| interval           | body | string  | true     | 18s                                                     | The time between two continuous evaluate, re.the format please refer [note](./note.md#time-durations).                                                                                           |
| receivers          | body | list    | true     |                                                         | List of email receiving healing notifications                                                                                                                                                    |
| duration           | body | string  | true     | 3m                                                      | The total evaluation time, re.the format please refer [note](./note.md#time-durations).                                                                                                          |
| description        | body | string  | false    |                                                         |                                                                                                                                                                                                  |
| tags               | body | list    | false    |                                                         |                                                                                                                                                                                                  |
| active             | body | boolean | true     | false                                                   | Enable the healer or not.                                                                                                                                                                        |

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
	"active": true
}
Resp
{
    "Status": "OK",
    "Data": null,
    "Err": ""
}
```

#### List healer

**PATH**: `/healers/{provider-id}`

**METHOD**: `GET`

#### Delete healer

**PATH**: `/healers/{provider-id}/{healer-id}`

**METHOD**: `DELETE`

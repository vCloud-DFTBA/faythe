# Faythe

[![license](https://img.shields.io/badge/license-Apache%20v2.0-blue.svg)](LICENSE) [![Go Report Card](https://goreportcard.com/badge/github.com/ntk148v/faythe)](https://goreportcard.com/report/github.com/ntk148v/faythe)
```
  _____                __  .__            
_/ ____\____  ___.__._/  |_|  |__   ____  
\   __\\__  \<   |  |\   __\  |  \_/ __ \ 
 |  |   / __ \\___  | |  | |   Y  \  ___/ 
 |__|  (____  / ____| |__| |___|  /\___  >
            \/\/                \/     \/ 
```

## What is Faythe?

* A ~~simple~~ Golang web api.
* The name is inspired by a character in cryptology.

<details>
    <summary>Who is Faythe</summary>
    <p>
    <b>Faythe</b>: A trusted advisor, courier or intermediary. Faythe is used infrequently, and is associated with Faith and Faithfulness. Faythe may be a repository of key service or courier of shared secrets.)
    </p>
</details>

## Install & Run

1. Use executable file:

```bash
# Modify etc/config.yml file
$ vim etc/config.yml
# Move config file to config directory
$ cp etc/config.yml /path/to/config/dir
# Build it
$ go build -mod vendor -o /path/to/executable/faythe
# Run it
$ /path/to/executable/faythe -conf /path/to/config/dir
```

2. Use Docker

* Build Docker image (use git tag/git branch as Docker image tag):

```bash
$ make build
```

* Run container from built image (For more details & options please check [Makefile](./Makefile)):

```
$ make run
```

## Features

* Trigger Heat AutoscalingPolicy & AutoscalingGroup with information from Prometheus alerts instead of OpenStack Telemetry (Aodh, Gnocchi & Ceilometer).
* Trigger Stackstorm rules with information from Prometheus alerts.
* Basic authentication.
* Restrict domains(s).
* Deduplicate alerts from Prometheus Alertmanager.

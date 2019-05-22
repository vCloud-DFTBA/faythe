# Faythe

```
  _____                __  .__            
_/ ____\____  ___.__._/  |_|  |__   ____  
\   __\\__  \<   |  |\   __\  |  \_/ __ \ 
 |  |   / __ \\___  | |  | |   Y  \  ___/ 
 |__|  (____  / ____| |__| |___|  /\___  >
            \/\/                \/     \/ 
```

## What is Faythe?

* A simple Golang web api.
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
# Run it
$ ./bin/faythe -conf /path/to/config/dir
```

2. Use Docker

* Build Docker image.

```bash
$ docker build -t faythe:latest .
```

* Run container from built image.

```
$ docker run -d --name faythe -p <port>:<port> -v /path/to/config/dir:/etc/faythe/ faythe
```

## Features

* Modify the given request headers then forward it!
* Autoscaling OpenStack instance using Heat features.

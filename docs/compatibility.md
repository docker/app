List of supported features, depending on the Docker Compose version.

```
Legend:
- [*]: Supported everywhere
- [C]: docker-compose up [engine only]
- [c]: docker-compose up [engine only] (partial support or compatibility mode only, see documentation for details)
- [S]: docker stack deploy [Swarm]
- [K]: docker stack deploy [Kubernetes]
- [k]: docker stack deploy [Kubernetes] (supported but behavior might be slightly different)
```

## docker stack deploy

| Features                     | 3.7 | 3.6 | 3.5 | 3.4 | 3.3 | 3.2 | 3.1 | 3.0 | 2.4 | 2.3 | 2.2 | 2.1 | 2.0 |
|------------------------------|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| configs.<name> (long syntax) |  SK |  SK |  SK |  SK |  SK |     |     |     |     |     |     |     |     |
| - external                   |  SK |  SK |  SK |  SK |  SK |     |     |     |     |     |     |     |     |
| - file                       |  SK |  SK |  SK |  SK |  SK |     |     |     |     |     |     |     |     |
| - mode                       |  SK |  SK |  SK |  SK |  SK |     |     |     |     |     |     |     |     |
| - name                       |  SK |  SK |  SK |     |     |     |     |     |     |     |     |     |     |
| configs.<name> (short syntax)|  SK |  SK |  SK |  SK |  SK |     |     |     |     |     |     |     |     |
| - source                     |  SK |  SK |  SK |  SK |  SK |     |     |     |     |     |     |     |     |
| - target                     |  SK |  SK |  SK |  SK |  SK |     |     |     |     |     |     |     |     |
| - uid                        |  SK |  SK |  SK |  SK |  SK |     |     |     |     |     |     |     |     |
| - gid                        |  SK |  SK |  SK |  SK |  SK |     |     |     |     |     |     |     |     |
| - mode                       |  SK |  SK |  SK |  SK |  SK |     |     |     |     |     |     |     |     |
| secrets.<name>               |  *  |  *  |  *  |  *  |  *  |     |     |     |     |     |     |     |     |
| - external                   |  SK |  SK |  SK |  SK |  SK |     |     |     |     |     |     |     |     |
| - file                       |  *  |  *  |  *  |  *  |  *  |     |     |     |     |     |     |     |     |
| - name                       |  *  |  *  |  *  |     |     |     |     |     |     |     |     |     |     |
| services.<name>              |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |
| - build                      |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |
|   - args                     |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |
|   - cache_from               |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |
|   - context                  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |
|   - dockerfile               |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |
|   - extra_hosts              |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |
|   - labels                   |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |
|   - network                  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |
|   - shm_size                 |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |
|   - target                   |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |
| - cap_add                    |  CK |  CK |  CK |  CK |  CK |  CK |  CK |  C  |  C  |  C  |  C  |  C  |  C  |
| - cap_drop                   |  CK |  CK |  CK |  CK |  CK |  CK |  CK |  C  |  C  |  C  |  C  |  C  |  C  |
| - cpu_shares                 |     |     |     |     |     |     |     |     |  C  |  C  |  C  |  C  |  C  |
| - cpu_quota                  |     |     |     |     |     |     |     |     |  C  |  C  |  C  |  C  |  C  |
| - cpuset                     |     |     |     |     |     |     |     |     |  C  |  C  |  C  |  C  |  C  |
| - command                    |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |
| - configs                    |  CK |  CK |  CK |  CK |  CK |     |     |     |     |     |     |     |     |
| - cgroup_parent              |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |
| - container_name             |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |
| - credential_spec            |  CK |  CK |  CK |  CK |  CK |     |     |     |     |     |     |     |     |
|   - file                     |  *  |  *  |  *  |  *  |  *  |     |     |     |     |     |     |     |     |
|   - registry                 |  *  |  *  |  *  |  *  |  *  |     |     |     |     |     |     |     |     |
| - deploy                     |  SK |  SK |  SK |  SK |  SK |  SK |  SK |  SK |     |     |     |     |     |
|   - endpoint_mode            |  S  |  S  |  S  |  S  |  S  |     |     |     |     |     |     |     |     |
|     - dnsrr                  |  S  |  S  |  S  |  S  |  S  |     |     |     |     |     |     |     |     |
|     - vip                    |  S  |  S  |  S  |  S  |  S  |     |     |     |     |     |     |     |     |
|   - labels                   |  SK |  SK |  SK |  SK |  SK |  SK |  SK |  SK |     |     |     |     |     |
|   - mode                     |  SK |  SK |  SK |  SK |  SK |  SK |  SK |  SK |     |     |     |     |     |
|   - placement                |  SK |  SK |  SK |  SK |  SK |  SK |  SK |  SK |     |     |     |     |     |
|   - replicas                 | cSK | cSK | cSK | cSK | cSK | cSK | cSK | cSK |     |     |     |     |     |
|   - resources                | cSK | cSK | cSK | cSK | cSK | cSK | cSK | cSK |     |     |     |     |     |
|     - limits                 |  SK |  SK |  SK |  SK |  SK |  SK |  SK |  SK |     |     |     |     |     |
|     - reservations           |  SK |  SK |  SK |  SK |  SK |  SK |  SK |  SK |     |     |     |     |     |
|   - restart_policy           | cSK | cSK | cSK | cSK | cSK | cSK | cSK | cSK |     |     |     |     |     |
|     - condition: none        |  SK |  SK |  SK |  SK |  SK |  SK |  SK |  SK |     |     |     |     |     |
|     - condition: on-failure  |  SK |  SK |  SK |  SK |  SK |  SK |  SK |  SK |     |     |     |     |     |
|     - condition: any         |  SK |  SK |  SK |  SK |  SK |  SK |  SK |  SK |     |     |     |     |     |
|     - delay                  |  S  |  S  |  S  |  S  |  S  |  S  |  S  |  S  |     |     |     |     |     |
|     - max_attempts           |  S  |  S  |  S  |  S  |  S  |  S  |  S  |  S  |     |     |     |     |     |
|     - window                 |  S  |  S  |  S  |  S  |  S  |  S  |  S  |  S  |     |     |     |     |     |
|   - rollback_config          |  S  |     |     |     |     |     |     |     |     |     |     |     |     |
|     - delay                  |  S  |     |     |     |     |     |     |     |     |     |     |     |     |
|     - failure_action         |  S  |     |     |     |     |     |     |     |     |     |     |     |     |
|     - max_failure_ratio      |  S  |     |     |     |     |     |     |     |     |     |     |     |     |
|     - monitor                |  S  |     |     |     |     |     |     |     |     |     |     |     |     |
|     - order                  |  S  |     |     |     |     |     |     |     |     |     |     |     |     |
|     - parallelism            |  S  |     |     |     |     |     |     |     |     |     |     |     |     |
|   - update_config            |  SK |  SK |  SK |  SK |  SK |  SK |  SK |  SK |     |     |     |     |     |
|     - delay                  |  SK |  SK |  SK |  SK |  SK |  SK |  SK |  SK |     |     |     |     |     |
|     - failure_action         |  SK |  SK |  SK |  SK |  SK |  SK |  SK |  SK |     |     |     |     |     |
|     - max_failure_ratio      |  SK |  SK |  SK |  SK |  SK |  SK |  SK |  SK |     |     |     |     |     |
|     - monitor                |  SK |  SK |  SK |  SK |  SK |  SK |  SK |  SK |     |     |     |     |     |
|     - order                  |  SK |  SK |  SK |  SK |     |     |     |     |     |     |     |     |     |
|     - parallelism            |  SK |  SK |  SK |  SK |  SK |  SK |  SK |  SK |     |     |     |     |     |
| - devices                    |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |
| - depends_on                 |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |
| - dns                        |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |
| - dns_search                 |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |
| - domainname                 |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |
| - entrypoint                 |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |
| - env_file                   |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |
| - environment                |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |
| - expose                     |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |
| - external_links             |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |
| - extra_hosts                |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |
| - healthcheck                |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |     |
|   - disable                  |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |
|   - interval                 |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |
|   - retries                  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |
|   - start_period             |  CS |  CS |  CS |  CS |     |     |     |     |     |     |     |     |     |
|   - test                     |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |
|   - timeout                  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |
| - hostname                   |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |
| - image                      |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |
| - init                       |  C  |     |     |     |     |     |     |     |     |     |     |     |     |
| - ipc                        |  CK |  CK |  CK |  CK |  CK |  CK |  CK |  CK |  C  |  C  |  C  |  C  |  C  |
| - isolation                  |  CS |  CS |  CS |     |     |     |     |     |     |     |     |     |     |
| - labels                     |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |
| - links                      |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |
| - logging                    |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |
|   - driver                   |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |
|   - options                  |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |
| - mac_address                |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |
| - mem_limit                  |     |     |     |     |     |     |     |     |  C  |  C  |  C  |  C  |  C  |
| - mem_swappiness             |     |     |     |     |     |     |     |     |  C  |  C  |  C  |  C  |  C  |
| - memswap_limit              |     |     |     |     |     |     |     |     |  C  |  C  |  C  |  C  |  C  |
| - network_mode               |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |
| - networks                   |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |
|   - <name>                   |  CS |  CS |  CS |  CS |     |     |     |     |     |     |     |     |     |
|     - aliases                |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |
|     - ipv4_address           |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |
|     - ipv6_address           |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |
|     - link_local_ips         |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |     |
|     - priority               |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |
| - pid                        |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |
| - ports                      |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |
|   - mode:                    |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |
|     - host                   | CSk | CSk | CSk | CSk | CSk | CSk | CSk | CSk |  CS |  CS |  CS |  CS |  CS |
|     - ingress                | CSk | CSk | CSk | CSk | CSk | CSk | CSk | CSk |  CS |  CS |  CS |  CS |  CS |
|   - protocol                 |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |
|   - published                |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |
|   - target                   |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |
| - privileged                 |  CK |  CK |  CK |  CK |  CK |  CK |  CK |  CK |  CK |  CK |  CK |  CK |  CK |
| - read_only                  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |
| - restart                    |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |
| - runtime                    |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |     |     |     |
| - secrets                    |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |
|   - gid                      |  SK |  SK |  SK |  SK |  SK |  SK |  SK |  SK |  SK |  SK |  SK |  SK |  SK |
|   - mode                     |  SK |  SK |  SK |  SK |  SK |  SK |  SK |  SK |  SK |  SK |  SK |  SK |  SK |
|   - name                     |  CS |  CS |  CS |  CS |     |     |     |     |     |     |     |     |     |
|   - source                   |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |
|   - target                   |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |
|   - uid                      |  SK |  SK |  SK |  SK |  SK |  SK |  SK |  SK |  SK |  SK |  SK |  SK |  SK |
| - security_opt               |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |
| - shm_size                   |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |
| - stdin_open                 |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |
| - stop_grace_period          |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |
| - stop_signal                |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |
| - sysctls                    |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |
| - tmpfs                      |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |
| - tty                        |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |
| - ulimits                    |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |
| - user                       |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |
|   - numerical                |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |
|   - name                     |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |
| - userns_mode                |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |  C  |
| - volumes                    |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |
| - volumes_from               |     |     |     |     |     |     |     |     |  C  |  C  |  C  |  C  |  C  |
| - working_dir                |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |
| networks.<name>              |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |
| - driver                     |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |
| - driver_opts                |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |
| - enable_ipv6                |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |     |
| - ipam                       |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |
|   - driver                   |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |
|   - config                   |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |
|     - subnet                 |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |
|     - ip_range               |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |
|     - gateway                |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |
|     - aux_addresses          |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |
|   - options                  |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |
| - internal                   |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |
| - labels                     |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |     |
| - external                   |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |
| - name                       |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |  CS |     |
| volumes.<name> (long syntax) |  *  |  *  |  *  |  *  |  *  |  *  |     |     |     |     |     |     |     |
| - read_only                  |  *  |  *  |  *  |  *  |  *  |  *  |     |     |     |     |     |     |     |
| - source                     |  *  |  *  |  *  |  *  |  *  |  *  |     |     |     |     |     |     |     |
| - target                     |  *  |  *  |  *  |  *  |  *  |  *  |     |     |     |     |     |     |     |
| - type                       |  *  |  *  |  *  |  *  |  *  |  *  |     |     |     |     |     |     |     |
|   - volume                   |  *  |  *  |  *  |  *  |  *  |  *  |     |     |     |     |     |     |     |
|   - bind                     |  *  |  *  |  *  |  *  |  *  |  *  |     |     |     |     |     |     |     |
|   - tmpfs                    |  *  |  *  |  *  |  *  |  *  |  *  |     |     |     |     |     |     |     |
| - bind                       |  CS |  CS |  CS |  CS |  CS |  CS |     |     |     |     |     |     |     |
|   - propagation              |  CS |  CS |  CS |  CS |  CS |  CS |     |     |     |     |     |     |     |
| - volume                     |  CS |  CS |  CS |  CS |  CS |  CS |     |     |     |     |     |     |     |
|   - nocopy                   |  CS |  CS |  CS |  CS |  CS |  CS |     |     |     |     |     |     |     |
| - tmpfs                      |  CS |  CS |     |     |     |     |     |     |     |     |     |     |     |
|   - size                     |  CS |  CS |     |     |     |     |     |     |     |     |     |     |     |
| volume.<name> (short syntax) |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |  *  |

## Ratio (not up to date)

| Version | Docker   | %     | % of major |
|---------|----------|-------|------------|
| 3.6     | 18.02.0+ | -     | -          |
| 3.5     | 17.12.0+ | -     | -          |
| 3.4     | 17.09.0+ | 0.58  | 2.51       |
| 3.3     | 17.06.0+ | 1.53  | 6.63       |
| 3.2     | 17.04.0+ | 1.01  | 4.40       |
| 3.1     | 1.13.1+  | 1.48  | 6.41       |
| 3.0     | 1.13.0+  | 18.51 | 80.03      |
| 2.1     | 1.12.0+  | 3.06  | 6.20       |
| 2.0     | 1.10.0+  | 46.40 | 93.80      |

## docker stack deploy

| Features                     | 3.6 | 3.5 | 3.4 | 3.3 | 3.2 | 3.1 | 3.0 |
|------------------------------|-----|-----|-----|-----|-----|-----|-----|
| build                        |     |     |     |     |     |     |     |
|  - args                      |     |     |     |     |     |     |     |
|  - cache_from                |     |     |     |     |     |     |     |
|  - context                   |     |     |     |     |     |     |     |
|  - dockerfile                |     |     |     |     |     |     |     |
|  - labels                    |     |     |     |     |     |     |     |
|  - labels                    |     |     |     |     |     |     |     |
|  - network                   |     |     |     |     |     |     |     |
|  - shm_size                  |     |     |     |     |     |     |     |
|  - target                    |     |     |     |     |     |     |     |
| cap_add                      |     |     |     |     |     |     |     |
| cap_drop                     |     |     |     |     |     |     |     |
| command                      |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| configs                      |  Y  |  Y  |  Y  |  Y  |     |     |     |
| - external                   |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - file                       |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - mode                       |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - name                       |  Y  |  Y  |     |     |     |     |     |
| cgroup_parent                |     |     |     |     |     |     |     |
| container_name               |     |     |     |     |     |     |     |
| credential_spec              |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| deploy                       |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - endpoint_mode: dnsrr       |  Y  |  Y  |  Y  |  Y  |     |     |     |
| - endpoint_mode: vip         |  Y  |  Y  |  Y  |  Y  |     |     |     |
| - labels                     |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - mode                       |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - placement                  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - replicas                   |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - resources                  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- limits                  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- reservations            |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - restart_policy             |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- condition: none         |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- condition: on-failure   |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- condition: any          |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- delay                   |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- max_attempts            |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- window                  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - update_config              |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- delay                   |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- failure_action          |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- max_failure_ratio       |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- monitor                 |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- order                   |  Y  |  Y  |  Y  |     |     |     |     |
|   -- parallelism             |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - devices                    |     |     |     |     |     |     |     |
| - depends_on                 |     |     |     |     |     |     |     |
| - dns                        |     |     |     |     |     |     |     |
| - dns_search                 |     |     |     |     |     |     |     |
| - tmpfs                      |     |     |     |     |     |     |     |
| - entrypoint                 |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - env_file                   |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - environment                |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- key=value               |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- key                     |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - expose                     |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - external_links             |     |     |     |     |     |     |     |
| - extra_hosts                |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - healthcheck                |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- disable                 |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- interval                |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- retries                 |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- start_period            |  Y  |  Y  |  Y  |     |     |     |     |
|   -- test                    |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- test: NONE              |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- timeout                 |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - image                      |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - isolation                  |  Y  |  Y  |     |     |     |     |     |
| - labels                     |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - links                      |     |     |     |     |     |     |     |
| - logging                    |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- driver                  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- options                 |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - network_mode               |     |     |     |     |     |     |     |
| - networks                   |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- aliases                 |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- ipv4_address            |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- ipv6_address            |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- name                    |  Y  |  Y  |  Y  |     |     |     |     |
| - pid: host                  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - ports                      |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- mode: host              |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- mode: ingress           |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- protocol                |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- published               |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- target                  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - secrets                    |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- gid                     |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- mode                    |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- name                    |  Y  |  Y  |  Y  |     |     |     |     |
|   -- source                  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- target                  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- uid                     |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - security_opt               |     |     |     |     |     |     |     |
| - stop_grace_period          |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - stop_signal                |     |     |     |     |     |     |     |
| - sysctls                    |     |     |     |     |     |     |     |
| - ulimits                    |     |     |     |     |     |     |     |
| - userns_mode                |     |     |     |     |     |     |     |
| - volumes                    |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- bind/propagation        |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- name                    |  Y  |  Y  |  Y  |     |     |     |     |
|   -- read_only               |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- source                  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- target                  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- type: bind              |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- type: volume            |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- volume/nocopy           |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - restart                    |     |     |     |     |     |     |     |
| - domainname                 |     |     |     |     |     |     |     |
| - hostname                   |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - ipc                        |     |     |     |     |     |     |     |
| - mac_address                |     |     |     |     |     |     |     |
| - privileged                 |     |     |     |     |     |     |     |
| - read_only                  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - shm_size                   |     |     |     |     |     |     |     |
| - stdin_open                 |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - tty                        |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - user                       |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- numerical               |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- name                    |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - working_dir                |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |

## Kamoulox (and HELM)

| Features                     | 3.6 | 3.5 | 3.4 | 3.3 | 3.2 | 3.1 | 3.0 |
|------------------------------|-----|-----|-----|-----|-----|-----|-----|
| build                        |     |     |     |     |     |     |     |
|  - args                      |     |     |     |     |     |     |     |
|  - cache_from                |     |     |     |     |     |     |     |
|  - context                   |     |     |     |     |     |     |     |
|  - dockerfile                |     |     |     |     |     |     |     |
|  - labels                    |     |     |     |     |     |     |     |
|  - labels                    |     |     |     |     |     |     |     |
|  - network                   |     |     |     |     |     |     |     |
|  - shm_size                  |     |     |     |     |     |     |     |
|  - target                    |     |     |     |     |     |     |     |
| cap_add                      |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| cap_drop                     |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| command                      |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| configs                      |  Y  |  Y  |  Y  |  Y  |     |     |     |
| - external                   |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - file                       |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - mode                       |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - name                       |  Y  |  Y  |     |     |     |     |     |
| cgroup_parent                |     |     |     |     |     |     |     |
| container_name               |     |     |     |     |     |     |     |
| credential_spec              |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| deploy                       |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - endpoint_mode: dnsrr       |     |     |     |     |     |     |     |
| - endpoint_mode: vip         |  Y  |  Y  |  Y  |  Y  |     |     |     |
| - labels                     |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - mode                       |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - placement                  |  P  |  P  |  P  |  P  |  P  |  P  |  P  |
| - replicas                   |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - resources                  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- limits                  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- reservations            |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - restart_policy             |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- condition: none         |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- condition: on-failure   |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- condition: any          |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- delay                   |     |     |     |     |     |     |     |
|   -- max_attempts            |     |     |     |     |     |     |     |
|   -- window                  |     |     |     |     |     |     |     |
| - update_config              |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- delay                   |     |     |     |     |     |     |     |
|   -- failure_action          |     |     |     |     |     |     |     |
|   -- max_failure_ratio       |     |     |     |     |     |     |     |
|   -- monitor                 |     |     |     |     |     |     |     |
|   -- order                   |     |     |     |     |     |     |     |
|   -- parallelism             |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - devices                    |     |     |     |     |     |     |     |
| - depends_on                 |     |     |     |     |     |     |     |
| - dns                        |     |     |     |     |     |     |     |
| - dns_search                 |     |     |     |     |     |     |     |
| - tmpfs                      |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - entrypoint                 |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - env_file                   |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - environment                |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- key=value               |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- key                     |     |     |     |     |     |     |     |
| - expose                     |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - external_links             |     |     |     |     |     |     |     |
| - extra_hosts                |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - healthcheck                |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- disable                 |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- interval                |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- retries                 |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- start_period            |  Y  |  Y  |  Y  |     |     |     |     |
|   -- test                    |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- test: NONE              |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- timeout                 |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - image                      |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - isolation                  |     |     |     |     |     |     |     |
| - labels                     |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - links                      |     |     |     |     |     |     |     |
| - logging                    |     |     |     |     |     |     |     |
|   -- driver                  |     |     |     |     |     |     |     |
|   -- options                 |     |     |     |     |     |     |     |
| - network_mode               |     |     |     |     |     |     |     |
| - networks                   |     |     |     |     |     |     |     |
|   -- aliases                 |     |     |     |     |     |     |     |
|   -- ipv4_address            |     |     |     |     |     |     |     |
|   -- ipv6_address            |     |     |     |     |     |     |     |
|   -- name                    |     |     |     |     |     |     |     |
| - pid: host                  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - ports                      |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- mode: host              |  ?  |  ?  |  ?  |  ?  |  ?  |  ?  |  ?  |
|   -- mode: ingress           |  ?  |  ?  |  ?  |  ?  |  ?  |  ?  |  ?  |
|   -- protocol                |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- published               |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- target                  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - secrets                    |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- gid                     |     |     |     |     |     |     |     |
|   -- mode                    |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- name                    |  Y  |  Y  |  Y  |     |     |     |     |
|   -- source                  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- target                  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- uid                     |     |     |     |     |     |     |     |
| - security_opt               |     |     |     |     |     |     |     |
| - stop_grace_period          |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - stop_signal                |     |     |     |     |     |     |     |
| - sysctls                    |     |     |     |     |     |     |     |
| - ulimits                    |     |     |     |     |     |     |     |
| - userns_mode                |     |     |     |     |     |     |     |
| - volumes                    |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- bind/propagation        |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- name                    |  Y  |  Y  |  Y  |     |     |     |     |
|   -- read_only               |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- source                  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- target                  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- type: bind              |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- type: volume            |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- volume/nocopy           |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - restart                    |     |     |     |     |     |     |     |
| - domainname                 |     |     |     |     |     |     |     |
| - hostname                   |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - ipc                        |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - mac_address                |     |     |     |     |     |     |     |
| - privileged                 |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - read_only                  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - shm_size                   |     |     |     |     |     |     |     |
| - stdin_open                 |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - tty                        |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
| - user                       |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- numerical               |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |
|   -- name                    |     |     |     |     |     |     |     |
| - working_dir                |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |  Y  |

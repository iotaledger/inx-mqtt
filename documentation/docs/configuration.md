---
description: This section describes the configuration parameters and their types for INX-MQTT.
keywords:
- IOTA Node 
- Hornet Node
- Dashboard
- Configuration
- JSON
- Customize
- Config
- reference
---


# Core Configuration

INX-MQTT uses a JSON standard format as a config file. If you are unsure about JSON syntax, you can find more information in the [official JSON specs](https://www.json.org).

You can change the path of the config file by using the `-c` or `--config` argument while executing `inx-mqtt` executable.

For example:
```bash
inx-mqtt -c config_defaults.json
```

You can always get the most up-to-date description of the config parameters by running:

```bash
inx-mqtt -h --full
```

## <a id="app"></a> 1. Application

| Name            | Description                                                                                            | Type    | Default value |
| --------------- | ------------------------------------------------------------------------------------------------------ | ------- | ------------- |
| checkForUpdates | Whether to check for updates of the application or not                                                 | boolean | true          |
| stopGracePeriod | The maximum time to wait for background processes to finish during shutdown before terminating the app | string  | "5m"          |

Example:

```json
  {
    "app": {
      "checkForUpdates": true,
      "stopGracePeriod": "5m"
    }
  }
```

## <a id="inx"></a> 2. INX

| Name    | Description                            | Type   | Default value    |
| ------- | -------------------------------------- | ------ | ---------------- |
| address | The INX address to which to connect to | string | "localhost:9029" |

Example:

```json
  {
    "inx": {
      "address": "localhost:9029"
    }
  }
```

## <a id="mqtt"></a> 3. MQTT

| Name                                 | Description                                   | Type   | Default value |
| ------------------------------------ | --------------------------------------------- | ------ | ------------- |
| bufferSize                           | The size of the client buffers in bytes       | int    | 0             |
| bufferBlockSize                      | The size per client buffer R/W block in bytes | int    | 0             |
| [subscriptions](#mqtt_subscriptions) | Configuration for subscriptions               | object |               |
| [websocket](#mqtt_websocket)         | Configuration for websocket                   | object |               |
| [tcp](#mqtt_tcp)                     | Configuration for TCP                         | object |               |

### <a id="mqtt_subscriptions"></a> Subscriptions

| Name                           | Description                                                                                                    | Type  | Default value |
| ------------------------------ | -------------------------------------------------------------------------------------------------------------- | ----- | ------------- |
| maxTopicSubscriptionsPerClient | The maximum number of topic subscriptions per client before the client gets dropped (DOS protection)           | int   | 1000          |
| topicsCleanupThresholdCount    | The number of deleted topics that trigger a garbage collection of the subscription manager                     | int   | 10000         |
| topicsCleanupThresholdRatio    | The ratio of subscribed topics to deleted topics that trigger a garbage collection of the subscription manager | float | 1.0           |

### <a id="mqtt_websocket"></a> Websocket

| Name        | Description                                                    | Type    | Default value    |
| ----------- | -------------------------------------------------------------- | ------- | ---------------- |
| enabled     | Whether to enable the websocket connection of the MQTT broker  | boolean | true             |
| bindAddress | The websocket bind address on which the MQTT broker listens on | string  | "localhost:1888" |

### <a id="mqtt_tcp"></a> TCP

| Name                   | Description                                              | Type    | Default value    |
| ---------------------- | -------------------------------------------------------- | ------- | ---------------- |
| enabled                | Whether to enable the TCP connection of the MQTT broker  | boolean | false            |
| bindAddress            | The TCP bind address on which the MQTT broker listens on | string  | "localhost:1883" |
| [auth](#mqtt_tcp_auth) | Configuration for auth                                   | object  |                  |
| [tls](#mqtt_tcp_tls)   | Configuration for TLS                                    | object  |                  |

### <a id="mqtt_tcp_auth"></a> Auth

| Name         | Description                                                         | Type    | Default value                                                      |
| ------------ | ------------------------------------------------------------------- | ------- | ------------------------------------------------------------------ |
| enabled      | Whether to enable auth for TCP connections                          | boolean | false                                                              |
| passwordSalt | The auth salt used for hashing the passwords of the users           | string  | "0000000000000000000000000000000000000000000000000000000000000000" |
| users        | The list of allowed users with their password+salt as a scrypt hash | object  | []                                                                 |

### <a id="mqtt_tcp_tls"></a> TLS

| Name            | Description                                                              | Type    | Default value     |
| --------------- | ------------------------------------------------------------------------ | ------- | ----------------- |
| enabled         | Whether to enable TLS for TCP connections                                | boolean | false             |
| privateKeyPath  | The path to the private key file (x509 PEM) for TCP connections with TLS | string  | "private_key.pem" |
| certificatePath | The path to the certificate file (x509 PEM) for TCP connections with TLS | string  | "certificate.pem" |

Example:

```json
  {
    "mqtt": {
      "bufferSize": 0,
      "bufferBlockSize": 0,
      "subscriptions": {
        "maxTopicSubscriptionsPerClient": 1000,
        "topicsCleanupThresholdCount": 10000,
        "topicsCleanupThresholdRatio": 1
      },
      "websocket": {
        "enabled": true,
        "bindAddress": "localhost:1888"
      },
      "tcp": {
        "enabled": false,
        "bindAddress": "localhost:1883",
        "auth": {
          "enabled": false,
          "passwordSalt": "0000000000000000000000000000000000000000000000000000000000000000",
          "users": null
        },
        "tls": {
          "enabled": false,
          "privateKeyPath": "private_key.pem",
          "certificatePath": "certificate.pem"
        }
      }
    }
  }
```

## <a id="profiling"></a> 4. Profiling

| Name        | Description                                       | Type    | Default value    |
| ----------- | ------------------------------------------------- | ------- | ---------------- |
| enabled     | Whether the profiling plugin is enabled           | boolean | false            |
| bindAddress | The bind address on which the profiler listens on | string  | "localhost:6060" |

Example:

```json
  {
    "profiling": {
      "enabled": false,
      "bindAddress": "localhost:6060"
    }
  }
```

## <a id="prometheus"></a> 5. Prometheus

| Name            | Description                                                     | Type    | Default value    |
| --------------- | --------------------------------------------------------------- | ------- | ---------------- |
| enabled         | Whether the prometheus plugin is enabled                        | boolean | false            |
| bindAddress     | The bind address on which the Prometheus HTTP server listens on | string  | "localhost:9312" |
| mqttMetrics     | Whether to include MQTT metrics                                 | boolean | true             |
| goMetrics       | Whether to include go metrics                                   | boolean | false            |
| processMetrics  | Whether to include process metrics                              | boolean | false            |
| promhttpMetrics | Whether to include promhttp metrics                             | boolean | false            |

Example:

```json
  {
    "prometheus": {
      "enabled": false,
      "bindAddress": "localhost:9312",
      "mqttMetrics": true,
      "goMetrics": false,
      "processMetrics": false,
      "promhttpMetrics": false
    }
  }
```


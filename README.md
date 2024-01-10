# INX-MQTT

[![Go](https://github.com/iotaledger/inx-mqtt/actions/workflows/build.yml/badge.svg)](https://github.com/iotaledger/inx-mqtt/actions/workflows/build.yml)

INX-MQTT extends the node endpoints to provide an Event API to listen to live changes happening in the Tangle.

This API is by default reachable using MQTT over WebSockets.

## Version compatibility
* `1.x` versions are compatible with Stardust and [HORNET 2.x](https://github.com/iotaledger/hornet) and provide the Event API described in [TIP-28](https://github.com/iotaledger/tips/blob/main/tips/TIP-0028/tip-0028.md).
* `2.x` versions should be used with IOTA 2.0 and [iota-core](https://github.com/iotaledger/iota-core) and provide the Event API described in [TIP-48](https://github.com/iotaledger/tips/pull/153).

## Setup
We recommend not using this repo directly but using our pre-built [Docker images](https://hub.docker.com/r/iotaledger/inx-mqtt) together with our [Docker setup](https://wiki.iota.org/hornet/how_tos/using_docker/).

You can find the corresponding documentation in the [IOTA Wiki](https://wiki.iota.org/hornet/inx-plugins/mqtt/welcome/).

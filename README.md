# [Oneway-Filesync](https://github.com/danlapid/oneway-filesync/)

Sync files between different hosts over a one way network link such as a Data Diode

[![Build Status](https://github.com/danlapid/oneway-filesync/actions/workflows/build.yml/badge.svg)](https://github.com/danlapid/oneway-filesync/actions?query=workflow%3ABuild)
[![Test Status](https://github.com/danlapid/oneway-filesync/actions/workflows/test.yml/badge.svg)](https://github.com/danlapid/oneway-filesync/actions?query=workflow%3ATest)
[![Coverage Status](https://coveralls.io/repos/github/danlapid/oneway-filesync/badge.svg?branch=main)](https://coveralls.io/github/danlapid/oneway-filesync?branch=main)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](https://opensource.org/licenses/MIT)

## Usage

### Download binaries from [Releases](https://github.com/danlapid/oneway-filesync/releases)

Create a config.toml file for both the sender and the receiver similarly to the example in the repo

On the source machine:

```
./sender
./watcher
```

On the target machine:

```
./receiver
```

To send a specific file through that is not in the watched folder:

``` ./sendfiles <file/dir path> ```

## Data flow

### Sender side:

QueueReader (From DB) -> FileReader -> FecEncoder -> BandwidthLimiter -> UdpSender 

### -> Data Diode -> 

### Receiver side:

UdpReceiver -> ShareAssember -> FecDecoder -> FileWriter -> FileCloser (Updates receiver DB)

## Config

- ReceiverIP : The IP the receiver will listen on and the sender will send to
- ReceiverPort : The port the receiver will listen on and the sender will send to
- BandwidthLimit : in Bytes/Second the sender will limit itself to this amount, suggested to be a little under link speed, if you get "buffers are filling up" error code then you might need more compute power on the receiver
- ChunkSize : Data length sent in each udp datagram should be 42 bytes smaller than link MTU (14 Ethernet, 20 IP, 8 UDP) for compute efficiency it is suggested to increase the link MTU and then increase this value as well
- EncryptedOutput : If true the files will be encrypted in a zip file with password `filesync` before being sent and saved to the receiver as the encrypted zip
- ChunkFecRequired : Reed Solomon FEC parameter, the amount of shares that must arrive for the chunk to be reconstructed
- ChunkFecTotal : Reed Solomon FEC parameter, the total amount of shares that will be sent, it is suggested that this will be a multiple of ChunkFecRequired
- OutDir : Directory to write output files to on the receiver, the original directory structure will be perserved and appended to this path
- WatchDir : Directory the watcher will detect file changes on and send every changed files from


# Gossip

## Description

This is a distributed Gossip program. Each "node" (executing instance of the program) is identified by a TCP/IP address and periodically gossips a single digit (i.e., 0 through 9). The digit can change over time, and it reports the digit of a node (including its own) when it discovers a change. You can also input the TCP/IP address of another node to gossip with, preceded by a + sign:

```
>> +128.84.213.13:5678
>>
128.84.213.13:5678 --> 9
>> 6
128.84.213.43:9876 --> 6
>> 7
128.84.213.43:9876 --> 7
>>
128.84.213.13:5678 --> 1
>> ?
128.84.213.13:5678 --> 1
128.84.213.43:9876 --> 7
>>
```

All communication will take place over TCP/IP happening in a "pull" style: when node X sets up a connection to node Y , node Y reports back with its state. The response is in ASCII CSV format consisting of a line per node with three fields separated by commas:
> 1. the node identifier in x.x.x.x:port format
> 2. the time at which the digit was updated by the node. The time is reported as seconds since 0 hours, 0 minutes, 0 seconds, January 1, 1970, Coordinated Universal Time, without including leap seconds (this is the time that Linux reports by default)
> 3. the digit itself

No spaces or tabs can be used. Each line is terminated by a single newline (line feed) character (ASCII code 10 or 0xA). After sending this CSV table, node Y closes the connection.
> For example, in the example above, the response may look like this:
>```
> 128.84.213.13:5678,1630281124,1
> 128.84.213.43:9876,1630282312,7
>```

Each node maintains a map from TCP/IP address to a pair (time, digit). Whenever a node learns of a new TCP/IP address, or an existing TCP/IP address with a later time, then it updates the map accordingly. The node also updates its own entry when the user enters a digit. Initially the map is empty

When the user types +x.x.x.x:port, the node tries to set up a connection to the given TCP/IP address. The connection may fail, and that’s ok

Every 3 seconds, the node selects a random entry in its map, and try to set up a TCP/IP connection to the selected node in order to attempt a gossip

## Usage
**change the following boolean variable to true in `main.go` if you want to use your public ip address**

`var public_ip bool = false`

---

```
/* launch commands (default server port is 6410 if unspecified) */
go run . [SERVER_PORT]
```

```
/* runtime commands */
>> [COMMAND]

Available Commands:
[+IP:PORT]: add a node to gossip with ("pull" style)
[!]: display connection records (user-level)
[?]: display local CSV table records (debug-level)
[a]: toggle on/off adversarial mode (reply to new gossip requests with invalid info)
```

## Robustness

- Thread-safe (many threads reading and writing a single table)

- Gossip response might get ignored if it is not following the specifications

- The program ignores updates with future or older timestamps

- In a response, the program only considers the first 256 lines received

- After initial successful connection, a node will be blocklisted if it cannot be reached later

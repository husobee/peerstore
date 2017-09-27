# PeerStore

## TLDR;

PeerStore is a peer to peer file storage and sharing product.

### How to Build

```bash

mkdir -p ~/golang/src/github.com/husobee/
tar -zxf ~/peerstore.tar.gz -C ~/golang/src/github.com/husobee/
cd ~/golang/src/github.com/husobee/

GOPATH=~/golang/ go get -u ./...
GOPATH=~/golang/ make linux # to just make the linux exe
GOPATH=~/golang/ make release # will create a "releases/" dir and build windows/mac/linux binaries
```

### How to Run

Starting a peerstore server:

```
./release/peerstore_server-latest-linux-amd64 -initialPeerAddr :3000 -addr :3001 -dataPath .peerstore/3001
```

I would suggest if you run more than one to use a different -dataPath for each
server running.  That will allow you to see the particular keys that are loaded
into that server.  If you are starting the first server the initial peer addr
is not that important.


Starting the peerstore client:

```
./release/peerstore_client-latest-linux-amd64 -peerAddr :3001 -localPath ~/peerstore/ -operation backup
```
This will take everything from `~/peerstore/` directory, recursively, and load
it into the server at peerAddr, which is port :3001 on localhost in this example

```
./release/peerstore_client-latest-linux-amd64 -filedest ~/test.txt.restored -peerAddr :3001 -filename ~/peerstore/test.txt -operation getfile
```

This command will restore the file from ~/peerstore/test.txt to the file called
~/test.txt.restored



## Description

A distributed, peer to peer file storage and sharing project.  This project
aims at creating a peer to peer application for secure file storage and sharing.
In the end the application will support the following features:

* File storage through a distributed hash table (DHT)
* Synchronization of files across client computers
* Authentication and Authorization of multiple users
* File Sharing across users
* Secure communications between peers in the DHT 
* Strong Encryption of all files before they are inserted into DHT 

Deliverables:

1. Develop DHT implementation, allowing for storage of files
    - Client/Server peer to peer communications
    - Insertions/Lookups in the DHT for files (Chord or Kademlia)
2. Add functionality to support multiple users and auth
    - Authentication of peers (so we know peers are legitimate)
    - Authentication of users to access files (not allowing access to
    non-owners)
    - Encryption of connections between peers
3. Synchronization of files between multiple clients
    - Propogation from file change through DHT, to other computers
    connected.
    - Handle consensus of file changes across computers
4. Storage of files in DHT are to be encrypted.  Allow sharing of files
    - Implement encryption to allow for encrypted storage
    - Implement session key scheme for sharing of encrypted files

## Technology Stack

This application is written in [go](https://golang.org).  This language was
chosen for the following reasons:
    1. Compiled, strongly typed, cross platform
    2. Speed

## Deliverable 1

The goal for deliverable #1 is create a distributed hash table (DHT) which will
evenly distribute the storage of files across nodes, along with the file
transfer primatives used to store files within the nodes of this DHT.

A distributed hash table is a mechanism by which all of the nodes in the peer to
peer network are able to figure out which node is responsible for the action, or
value in question.  To this end, the DHT needs to have a deterministic way of
taking a "key" and assigning it to a "node" in the network.  For this project
we will be implementing the [Chord][chord] protocol.

Chord was selected as the DHT algorithm of choice for the following reasons:
    1. Ease of implementation and understandability
    2. Even distribution of keys across nodes based on "consistent hashing"
    3. Eventual consistency of key node mapping for joins/departs with Sanitize

Chord is a logical ring network overlay topology.  In the most naive
implementation, a node knows about it's successor, and it's predecessor, which
allows the node to send a query around in a ring of connected machines.  There
is a concept of a routing table within Chord using a finger table which allows
for increased performance in lookups of keys, but has a drawback in that there
are more messages required on node join and depart to keep all the other node's
finger tables up to date.

As Chord is merely an algorithm for figuring out to which node a key belongs, we
will not only have to implement all of the chord specific Remote Proceedure
Calls (RPC) for the chord primatives, but also the application layer primatives
for retrieving and pushing of file data, as well as storage of the files on the
particular nodes in the network.

For the file primatives, we will be implimenting the following RPC functionality
on every node:

* GetFile(key) => bytes
* PostFile(key, bytes) => success/failure
* DeleteFile(key) => success/failure
* PatchFile(key, bytes) => success/failure

This way, after we find the responsible node for a given key, we can then
perform any of the four standard CRUD based operations on that given file.






[chord]: docs/chord_sigcomm.pdf

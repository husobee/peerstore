# PeerStore Design Document

## Introduction

Files are very important, and therefore must be stored in multiple locations.
What better way to protect files from a centralized failure than to put them
into a completely distributed system.  Moreover the performance benefits of
having documents sharded across a network, amoungst multiple nodes allows for
a better user experience.

### Purpose

PeerStore is a peer to peer file storage and sharing product.  As mentioned in
the [description of the project](../../README.md), the premise is to provide a
completely distributed secure peer to peer application for secure file storage
and sharing.  This application will in the end support the following features:

* File storage through a distributed hash table (DHT)
  * Implemented with the Chord algorithm
* Synchronization of files across client computers
* Authentication and Authorization of multiple users
* File Sharing across users
* Secure communications between peers in the DHT
* Strong Encryption of all files before they are inserted into DHT

### Scope

This design document will be broken into sections based completely on the four
primary milestones the project will be broken into, and thus will be designed
independently of the other milestones.  If a future milestone alters the design
in any way of a prior milestone, the design changes will be noted and explained
in that particular milestone's section of this document.

This document attempts to provide a journey, and not just the destination of
the project, and thus will be a living document.

#### Milestones

1. Develop DHT implementation and allow for storage of files
2. Ability to support multiple users and authentication
3. Synchronization of files between multiple clients
4. Encrypted file storage


### Definitions

- *Peer* - a node participating in the network
- *DHT* - Distributed Hash Table; a mechanism by which one can lookup which peer
in the PeerStore network is holding any given resource.
- *Chord* - [The algorithm](../chord_sigcomm.pdf) used to accomplish a DHT in
the PeerStore network
- *RPC* - Remote Proceedure Call, a function or method that can be called from a
peer in the network.

### Structure of Milestone Documentation

Each milestone top level section will start with a brief description of what
the objectives are for that particular milestone.  Then there will be a sub
section outlining the Use Cases, and Component Architecture.  A Discussion will
then be outlined as a sub-section describing in great detail the specifications
of the implementation developed.  This will then be followed by the dependencies
that were used in the implementation, and why those dependancies were needed for
the implementation.

The final milestone sub-section will be an explaination of how to run and test
the implementation, which will prove the milestone objectives were
accomplished.

## Milestone 1 - DHT implementation and file storage

A few key objectives in this milestone are as follows:

 * Client/Server peer to peer communications
 * OS I/O abstraction for reading/writing files to disk
 * Insertion of files in correct nodes for DHT
 * Lookups in DHT for nodes containing files looked for

### Use Cases

There are a number of use cases that can be split into two independent modules
within the milestone.  These two are: the distributed hash table, used to find
relevent files amoungst the peers; the file sharing implementation.

#### File Sharing Use Cases

The file sharing protocol developed is based on these high level operations:
Post; Get; Delete.  These three methods basically enumerate the operations that
peer nodes will be able to perform against peer nodes via RPC calls.  The
implementation details of the protocol are outlined further in the Discussion
section.

Below is a high level use case diagram outlining the particulars of the
operations that a peer can make against remote peer servers.

![File Sharing Use Case Diagram Client/Server](./Milestone1/FileUseCaseDiagram.png)

As seen, the peer to peer file transmission only requires simple crud methods
for the server to implement, Post, Get and Delete.  These methods are
implemented as server handler functions, routed by the method in the request.

From the client perspective the application in addition to the three methods
needs to accept a directory of files which it will register with the client, so
that each of the files within the directory are posted to the peer servers.

#### DHT Routing Use Cases

The routing of which node to store any particular file falls on the DHT.  We are
implementing the Chord Protocol.  Below are use case diagrams that outline the
operations a Chord node has to accept and perform against other nodes.

![Chord Use Case Diagram](./Milestone1/ChordUseCaseDiagram.png)

In the bare basic sense, the Chord protocol requires each node to be able to
accept queries as to whom the successor of a given key would be based on their
information.  Each node has an understanding of who their successor is and each
node asks the next if they know the successor of the key given.

As realized this algorithm just described would be very ineffecient.  To speed
up the lookups the Chord algorithm articulates the need for a finger table,
which contains a subset of all of it's successor nodes.  In this case the
LocalNode will call successor locally to see if it's successor would be
the successor of the key given, and if not finds the node in it's finger table
who is closest, and asks that node if they are the successor.

When a node initializes or leaves, that node is responsible for fixing it's
peer's finger table, and predecessors, as well as the job of transferring all
the keys for which it responsible.


### Component Architectures

The components created in this milestone are outlined below:

![Milestone 1 Component Diagram](./Milestone1/ComponentDiagram.png)

As seen by the above diagram, there are three primary packages within this
design:  File, Chord, Protocol.

#### File Package

The file package is solely responsible for everything that has to do with
reading and writing of files to and from a file system.  The block storage
component performs the reading and writing for both the client (transport)
as well as the reading and writing of files for the server.

The file package also employs the protocol handlers for the server aspect of the
package.  By creating a custom handler type in the server package which we will
cover shortly, we can effectively hand off the protocol logic to the protocol
package, and the business logic of what to do with a file to the file package.

#### Protocol Package

Within the Protocol package we have two very distinct primary components, and
two secondary components.  As a primary component, we have a Server component.
This component handles all incoming connection processing, and connection
closing, as well as the method routing based on the type of request.

The server works off of a goroutine pool for connection handling, which is
user specified.  This is slightly different than how the http package handles
incoming connection requests, for example, because that package spawns a new
goroutine per request, which could easily overload the application in a DOS
type situation.

The Transport primary component within the Protocol package is the opposite of
a server, it initiates connections to backend servers, and sends requests, and
waits for responses from the server.

It is pretty clear by now what the two secondary packages are for, Request and
Response.  Request is the datastructure by which a request is formatted, and
ultimately packaged for transmission over the connection.  The Response is the
structure for how the server will respond to any given request.

#### Chord Package

The Chord package handles everything that is required for the Chord algorithm.
Principally, this package manages the Local Node's finger table, predecessor,
and performs the search for successors of various keys.

This package, much like the File package, manages the business logic required
for the chord protocol to operate, including data structures that are imbedded
in the request and response "Data" fields.

### Discussion (Specification)

// TODO: Fill in with more details!


### Dependencies

The dependencies used for this project are as follows, followed by a brief
explaination as to why, and how the help.

- [github.com/pkg/errors](github.com/pkg/errors)
    - This is not the standard golang error package, and is used because there
    are some wonderful features, such as errors.Wrap which will wrap the error
    you get from a given function with a string you pick.  This is extremely
    helpful for figuring out nested error conditions.
- [context](https://golang.org/pkg/context/)
    - This is part of the standard library, but is basically a mechanism by
    which you can pass unstructured context data along with function calls or
    structs.  This is used primarily to house number of workers, and channel
    buffer size in the server.go
- [encoding/gob](https://golang.org/pkg/encoding/gob/)
    - This is the encoding package we are using for converting structures into
    bytes.  Every request is serialized using the gob encoding, and deserialized
    on the server using the gob encoding.  Even in the business logic we are 
    serializing and deserializing the higher level request "Data" bytes with
    this libary
- [encoding/hex](https://golang.org/pkg/encoding/hex/)
    - We are using the encoding hex to make clean strings for the file names
    that we are storing on the various nodes in the Chord DHT.  The file name
    is used to create a "key" which is used by the Chord algorithm to find
    which node that file should be stored on.  When it get's stored on the node
    we write it to a file that is the hex representation of the key
- [net](https://golang.org/pkg/net/)
    - The net package is a standard library that we are using to create the 
    TCPListener for the server to accept network connections on. We also use
    this package to "Dial" the server with the net.Dial function.

For dependency management we are using the golang tool `dep` which has stores a
manifest in the repository "GoPkg.toml" which will keep track of all the non
standard library dependencies and version pin them to a working version.


### Resources

 - [The Chord Algorithm](../chord_sigcomm.pdf)
 - [Golang Documentation](https://golang.org/doc/)


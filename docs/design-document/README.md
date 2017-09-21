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

### Discussion (Specification)

###


### Dependencies



### Resources

 - [The Chord Algorithm](../chord_sigcomm.pdf)


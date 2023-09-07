# Raft Leader Election

<h2>Introduction</h2>

<p>
  This is the first in a series of assignments in which you'll build a
  fault-tolerant key/value storage system. You'll start in this
  assignment by implementing the leader election features of Raft,
  a replicated state machine protocol. In Assignment 3 you will complete
  Raft's log consensus agreement features. You will implement Raft as a
  Go object with associated methods, available to be used as a module in
  a larger service. Once you have completed Raft, the course assignments
  will conclude with a key/value service built on top of Raft.
</p>

<h2>Raft Overview</h2>
<p>
  The Raft protocol is used to manage replica servers for services
  that must continue operation in the face of failure (e.g.
  server crashes, broken or flaky networks). The challenge is that,
  in the face of these failures, the replicas won't always hold identical data.
  The Raft protocol helps sort out what the correct data is.
</p>

<p>
  Raft's basic approach for this is to implement a replicated state
  machine. Raft organizes client requests into a sequence, called
  the log, and ensures that all the replicas agree on the the
  contents of the log. Each replica executes the client requests
  in the log in the order they appear in the log, applying those
  requests to the service's state. Since all the live replicas
  see the same log contents, they all execute the same requests
  in the same order, and thus continue to have identical service
  state. If a server fails but later recovers, Raft takes care of
  bringing its log up to date. Raft will continue to operate as
  long as at least a majority of the servers are alive and can
  talk to each other. If there is no such majority, Raft will
  make no progress, but will pick up where it left off as soon as
  a majority is alive again.
</p>


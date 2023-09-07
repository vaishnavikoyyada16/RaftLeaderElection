package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	"math/rand"
	"sync"
	"time"
)
import "labrpc"

// import "bytes"
// import "encoding/gob"

// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make().
type ApplyMsg struct {
	Index       int
	Command     interface{}
	UseSnapshot bool   // ignore for lab2; only used in lab3
	Snapshot    []byte // ignore for lab2; only used in lab3
}
type LogEntry struct {
	Term    int
	Index   int
	Command interface{}
}

const (
	FOLLOWER  = 0
	CANDIDATE = FOLLOWER + 1
	LEADER    = CANDIDATE + 1
)
const (
	ELECTION_TI_MIN = 150
	ELECTION_TI_MAX = 300
	HEARTBEAT_TI    = 30
)

// A Go object implementing a single Raft peer.
type Raft struct {
	mu        sync.Mutex
	peers     []*labrpc.ClientEnd
	persister *Persister
	me        int // index into peers[]

	// Your data here.
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.
	currentTerm   int
	votedFor      int
	log           []LogEntry
	commitIndex   int
	lastApplied   int
	nextIndex     []int
	matchIndex    []int
	state         int
	electionTimer *time.Timer
	voteCh        chan struct{}
	appendCh      chan struct{}
	voteAcquired  int
	applyCh       chan ApplyMsg // channel on which the tester or service expects Raft to send ApplyMsg messages.
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {

	var term int
	var isleader bool
	rf.mu.Lock()
	term = rf.currentTerm
	if rf.state == LEADER {
		isleader = true
	}
	rf.mu.Unlock()
	// Your code here.
	return term, isleader
}

// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
func (rf *Raft) persist() {
	// Your code here.
	// Example:
	// w := new(bytes.Buffer)
	// e := gob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// data := w.Bytes()
	// rf.persister.SaveRaftState(data)
}

// restore previously persisted state.
func (rf *Raft) readPersist(data []byte) {
	// Your code here.
	// Example:
	// r := bytes.NewBuffer(data)
	// d := gob.NewDecoder(r)
	// d.Decode(&rf.xxx)
	// d.Decode(&rf.yyy)
}

// example RequestVote RPC arguments structure.
type RequestVoteArgs struct {
	// Your data here.
	Term         int
	CandidateID  int
	LastLogIndex int
	LastLogTerm  int
}

// example RequestVote RPC reply structure.
type RequestVoteReply struct {
	// Your data here.
	Term        int
	VoteGranted bool
}

type AppendEntriesArgs struct {
	Term         int
	LeaderID     int
	PrevLogIndex int
	PrevLogTerm  int
	Entries      []LogEntry
	LeaderCommit int
}

type AppendEntriesReply struct {
	Term      int
	Success   bool
	NextTrail int
}

// example RequestVote RPC handler.
func (rf *Raft) RequestVote(args RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here.
	if rf.currentTerm > args.Term {
		reply.VoteGranted = false
		reply.Term = rf.currentTerm
		return
	}
	rf.state = FOLLOWER
	reply.Term = args.Term
	rf.votedFor = args.CandidateID
	reply.VoteGranted = true
	return
}

func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	if args.Term >= rf.currentTerm {
		if args.Term != rf.currentTerm {
			rf.currentTerm = args.Term
			rf.changetoFollower(FOLLOWER)
		}
		reply.Success = true
		return
	} else {
		reply.Term = rf.currentTerm
		reply.Success = false
	}
	if args.PrevLogIndex > (len(rf.log) - 1) {
		reply.NextTrail = 1 + (len(rf.log) - 1)
		reply.Success = false
		return
	}
	if args.PrevLogTerm != rf.log[args.PrevLogIndex].Term {
		l := args.PrevLogIndex
		for rf.log[args.PrevLogIndex].Term == rf.log[l].Term {
			l--
		}
		reply.NextTrail = 1 + l
		reply.Success = false
		return
	}
	if args.LeaderCommit > rf.commitIndex && args.LeaderCommit < (len(rf.log)-1) {
		rf.commitIndex = (len(rf.log) - 1)
	} else {
		rf.commitIndex = args.LeaderCommit
	}
}

// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// returns true if labrpc says the RPC was delivered.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
func (rf *Raft) sendRequestVote(server int, args RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}
func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	return ok
}

// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	isLeader := true
	return index, term, isLeader
}

// the tester calls Kill() when a Raft instance won't
// be needed again. you are not required to do anything
// in Kill(), but it might be convenient to (for example)
// turn off debug output from this instance.
func (rf *Raft) Kill() {
	// Your code here, if desired.
}

// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.

func (rf *Raft) changetoFollower(state int) {
	if state == int(rf.state) {
		return
	} else {
		rf.votedFor = -1
		rf.state = FOLLOWER
	}
}

func (rf *Raft) changetoCandidate(state int) {
	if state == int(rf.state) {
		return
	} else {
		rf.state = CANDIDATE
		rf.startElection()
	}
}

func (rf *Raft) changetoLeader(state int) {
	if state != int(rf.state) {
		for i := range rf.peers {
			rf.matchIndex[i] = 0
			rf.nextIndex[i] = (len(rf.log) - 1) + 1
		}
		rf.state = LEADER
	} else {
		return
	}
}

func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me
	// Your initialization code here.
	rf.state = FOLLOWER
	rf.votedFor = -1
	rf.voteCh = make(chan struct{})
	rf.appendCh = make(chan struct{})

	rf.nextIndex = make([]int, len(rf.peers))
	rf.matchIndex = make([]int, len(rf.peers))
	rf.applyCh = applyCh
	rf.log = make([]LogEntry, 1)
	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())
	go rf.startLoop()
	return rf
}
func randomelectiontime() time.Duration {
	rnd := rand.Int63n(((ELECTION_TI_MAX - ELECTION_TI_MIN) + ELECTION_TI_MIN))
	return time.Duration(rnd) * time.Millisecond

}
func (rf *Raft) startLoop() {
	rf.electionTimer = time.NewTimer(randomelectiontime())
	for {
		switch rf.state {
		case CANDIDATE:
			select {
			case <-rf.electionTimer.C:
				rf.electionTimer.Reset(randomelectiontime())
				rf.startElection()
			case <-rf.appendCh:
				rf.changetoFollower(FOLLOWER)
			default:
				if rf.voteAcquired > len(rf.peers)/2 {
					rf.changetoLeader(LEADER)
				}
			}
		case FOLLOWER:
			select {
			case <-rf.electionTimer.C:
				rf.mu.Lock()
				rf.changetoCandidate(CANDIDATE)
				rf.mu.Unlock()
			case <-rf.appendCh:
				rf.electionTimer.Reset(randomelectiontime())
			case <-rf.voteCh:
				rf.electionTimer.Reset(randomelectiontime())
			}
		case LEADER:
			sendAppendEntriesTo := func(server int) bool {
				var args AppendEntriesArgs
				var reply AppendEntriesReply
				args.Term = rf.currentTerm
				args.LeaderID = rf.me
				args.LeaderCommit = rf.commitIndex
				args.PrevLogIndex = rf.nextIndex[server] - 1
				args.PrevLogTerm = rf.log[args.PrevLogIndex].Term
				if (len(rf.log) - 1) >= rf.nextIndex[server] {
					args.Entries = rf.log[rf.nextIndex[server]:]
				}

				if rf.state == LEADER && rf.sendAppendEntries(server, &args, &reply) {
					if reply.Success == false && rf.state != LEADER {
						return false
					} else {
						rf.nextIndex[server] += len(args.Entries)
						rf.matchIndex[server] = rf.nextIndex[server] - 1
					}

					if reply.Success == false {
						if rf.state != LEADER {
							return false
						}
						if reply.Term > rf.currentTerm {
							rf.currentTerm = reply.Term
							rf.changetoFollower(FOLLOWER)
						} else {
							rf.nextIndex[server] = reply.NextTrail
							return true
						}
					} else {
						rf.nextIndex[server] += len(args.Entries)
						rf.matchIndex[server] = rf.nextIndex[server] - 1
					}
				}
				return false
			}

			for i := range rf.peers {
				if i == rf.me {
					continue
				}
				go func(server int) {
					for {
						if sendAppendEntriesTo(server) == false {
							break
						}
					}
				}(i)
			}
			time.Sleep(HEARTBEAT_TI * time.Millisecond)
		}

	}
}

func (rf *Raft) startElection() {
	rf.currentTerm = 1 + rf.currentTerm
	rf.votedFor = rf.me
	rf.voteAcquired = 1
	rf.electionTimer.Reset(randomelectiontime())
	var a RequestVoteArgs
	a.Term = rf.currentTerm
	a.CandidateID = rf.votedFor
	a.LastLogIndex = (len(rf.log) - 1)
	a.LastLogTerm = rf.log[a.LastLogIndex].Term
	for k := range rf.peers {
		if rf.votedFor == k {
			continue
		}
		go func(server int) {
			var r RequestVoteReply
			if rf.state == CANDIDATE {
				if rf.sendRequestVote(server, a, &r) {
					rf.mu.Lock()
					defer rf.mu.Unlock()
					if rf.currentTerm < r.Term && r.VoteGranted == false {
						rf.currentTerm = r.Term
						rf.changetoFollower(FOLLOWER)
					} else {
						rf.voteAcquired = rf.voteAcquired + 1
					}
				}
			}
		}(k)
	}
}

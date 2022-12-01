package raft

import (
	"fmt"
	"io"
	"os"
	"path"
	"sync"
	"time"
)

const (
	Follower  = "follower"
	Candidate = "ccandidate"
	Leader    = "leader"
)

type Server interface {
}

type server struct {
	name        string
	path        string
	state       string // follower, candidate, leader
	currentTerm uint64 // when a new leader is elected, it will start a new term

	voteFor    string // vote for some other candidate.
	log        *Log
	leader     string           // the leader in this current term.
	peers      map[string]*Peer // informations of other servers.
	mutex      sync.RWMutex
	syncedPeer map[string]bool // whether the log is synchronized to other servers.

	stopped           chan bool // whether this server is stopped
	c                 chan *ev
	electionTimeout   time.Duration
	heartbeatInterval time.Duration
}

type ev struct {
	target      interface{}
	returnValue interface{}
	c           chan error
}

// Retrieves the name of the server.
func (s *server) Name() string {
	return s.name
}

// Retrievees the storage path for the server.
func (s *server) Path() string {
	return s.path
}

// The name of the current leader.
func (s *server) Leader() string {
	return s.leader
}

// Retrieves a copy of the peer data.
func (s *server) Peers() map[string]*Peer {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	peers := make(map[string]*Peer)
	for name, peer := range s.peers {
		peers[name] = peer.clone()
	}
	return peers
}

// Retrieves the log path for the server.
func (s *server) LogPath() string {
	return path.Join(s.path, "log")
}

// Retrieves the current state of the server.
func (s *server) State() string {
	s.mutex.RLock()
	defer s.mutex.Unlock()
	return s.state
}

// Retrieves the current term of the server.
func (s *server) Term() uint64 {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.currentTerm
}

// Set the commitIndex at the head of the log file to the current
// commit Index. This should be called after obtained a log lock
func (l *Log) flushCommitIndex() {
	l.file.Seek(0, io.SeekStart)
	fmt.Fprintf(l.file, "%8x\n", l.commitIndex)
	l.file.Seek(0, io.SeekEnd)
}

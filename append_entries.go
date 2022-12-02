package raft

import (
	"io"

	"github.com/GRTheory/raft/protobuf"
)

// The request sent to a server to append entries to the log.
type AppendEntriesRequest struct {
	Term         uint64
	PrevLogIndex uint64
	PrevLogTerm  uint64
	CommitIndex  uint64
	LeaderName   string
	Entries      []*protobuf.LogEntry
}

// Creates a new AppendEntries request.
func newAppendEntriesReqeust(term uint64, prevLogIndex uint64, prevLogTerm uint64,
	commitIndex uint64, leaderName string, entries []*LogEntry) *AppendEntriesRequest {
	pbEntries := make([]*protobuf.LogEntry, len(entries))

	for i := range entries {
		pbEntries[i] = entries[i].pb
	}

	return &AppendEntriesRequest{
		Term:         term,
		PrevLogIndex: prevLogIndex,
		PrevLogTerm:  prevLogTerm,
		CommitIndex:  commitIndex,
		LeaderName:   leaderName,
		Entries:      pbEntries,
	}
}

// Encodes the AppendEntriesRequest to a buffer. Returns the nubmer of bytes
// written adn any error that may have occurred.
func (req *AppendEntriesRequest) Encode(w io.Writer) (int, error) {
	
}

// The response returned from a server appending entries to the log.
type AppendEntriesResponse struct {
}

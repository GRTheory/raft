package raft

// Transporter is the interface for allowing the host application to transport
// reqeusts to other nodes.
type Transporter interface {
	SendVoteRequest(server Server, peer *Peer, req *RequestVoteRequest) *RequestVoteResponse
	SendAppendEntriesRequest(server Server, peer *Peer, req *AppendEntriesRequest) *AppendEntriesResponse
	
}

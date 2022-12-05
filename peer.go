package raft

import (
	"sync"
	"time"
)

type Peer struct {
	server            *server
	Name              string `json:"name"`
	ConnectionString  string `json:"connectionString"`
	prevLogIndex      uint64
	stopChan          chan bool
	heartbeatInterval time.Duration
	lastActivity      time.Time
	sync.RWMutex
}

// Create a new peer.
func newPeer(server *server, name string, connectionString string, heartbeatInterval time.Duration) *Peer {
	return &Peer{
		server:            server,
		Name:              name,
		ConnectionString:  connectionString,
		heartbeatInterval: heartbeatInterval,
	}
}

// Sets the heartbeat timeout.
func (p *Peer) setHeartbeatInterval(duration time.Duration) {
	p.heartbeatInterval = duration
}

// Retrieves the previous log index.
func (p *Peer) getPrevLogIndex() uint64 {
	p.RLock()
	defer p.RUnlock()
	return p.prevLogIndex
}

// Sets the previous log index.
func (p *Peer) setPrevLogIndex(value uint64) {
	p.Lock()
	defer p.Unlock()
	p.prevLogIndex = value
}

func (p *Peer) setLastActivity(now time.Time) {
	p.Lock()
	defer p.Unlock()
	p.lastActivity = now
}

// func (p *Peer) startHeartbeat() {
// 	p.stopChan = make(chan bool)
// 	c := make(chan bool)

// 	p.setLastActivity(time.Now())

// 	p.server.routineGroup.Add(1)
// 	go func() {
// 		defer p.server.routineGroup.Done()

// 	}()
// }

func (p *Peer) clone() *Peer {
	p.Lock()
	defer p.Unlock()
	return &Peer{}
}

// Listens to the heartbeat timeout and flushes an AppendEntries RPC.
// func (p *Peer) heartbeat(c chan bool) {
// 	stopChan := p.stopChan

// 	c <- true

// 	ticker := time.Tick(p.heartbeatInterval)

// 	// debugln("peer.heartbeat: ", p.Name, p.heartbeatInterval)

// 	for {
// 		select{
// 		case flush := <- stopChan:
// 			if flush{
// 				// before we can safely remove a node
// 				// we must flush the remove command to the node first

// 			}
// 		}
// 	}
// }

func (p *Peer) flush() {
	// debugln("peer.heartbeat.flush: ", p.Name)
	prevLogIndex := p.getPrevLogIndex()
	term := p.server.currentTerm

	entries, prevLogTerm := p.server.log.getEntriesAfter(prevLogIndex, p.server.maxLogEntriesPerReqesut)

	if entries != nil {
		p.sendAppendEntriesRequest(newAppendEntriesReqeust(term, prevLogIndex, prevLogTerm, p.server.log.CommitIndex(), p.server.name, entries))
	} else {
		// TODO send snapshot to the peers
	}
}

// Sends an AppendEntries request to the peer through the transport.
func (p *Peer) sendAppendEntriesRequest(req *AppendEntriesRequest) {
	// tracef

	resp := p.server.Transporter().SendAppendEntriesRequest(p.server, p, req)
	if resp == nil {
		// TODO
		// dispatch the event
		// debugln("peer.append.timeout: ", p.server.Name(), "->", p.Name)
		return
	}
	// tranceln("peer.append.resp: ", p.server.Name(), "<-", p.Name)

	p.setLastActivity(time.Now())
	// If successful then update the previous log index.
	p.Lock()
	if resp.Success() {
		if len(req.Entries) > 0 {
			p.prevLogIndex = req.Entries[len(req.Entries)-1].GetIndex()

			// if peer append a log entry from the current term
			// we set append to true.
			if req.Entries[len(req.Entries)-1].GetTerm() == p.server.currentTerm {
				resp.append = true
			}
		}
		// traceln("peer.append.resp.success: ", p.Name, "; idx =", p.prevLogIndex)
		// If it was unsuccessful then decrement the previous log index and
		// we'll try again next time.
	} else {
		if resp.Term() > p.server.Term() {
			// this happens when there is a new leader comes up that this *leader* has not
			// known yet.
			// this server cannot know until the new leader send a ae with higher term
			// or this server finish processing this response.
			// debugln("peer.append.resp.not.update: new.leader.found")
		} else if resp.Term() == req.Term && resp.CommitIndex() >= p.prevLogIndex {
			// we may miss a response from peer
			// so maybe the peer has committed the logs we just sent
			// but we did not receive the successful reply and did not increase
			// the prevLogIndex

			// peer failed to truncate the log and sent a fail reply at this time
			// we just need to update peer's prevLog index to commitIndex

			p.prevLogIndex = resp.CommitIndex()
			// debugln("peer.append.resp.update: ", p.Name, "; idx =", p.revLogIndex)

		} else if p.prevLogIndex > 0 {
			// Decrement the previous log index down until we find a match. Don't
			// let it go below where the peer's commit index is though.
			p.prevLogIndex--
			// if it not enough, we directly decrease to the index of the resp's index.
			if p.prevLogIndex > resp.Index() {
				p.prevLogIndex = resp.Index()
			}

			// debugln("peer.append.resp.decrement: ", p.Name, "; idx =", p.prevLogIndex)
		}
	}
	p.Unlock()

	// Attach the peer to resp, thus server can know where it comes from
	resp.peer = p.Name
	// Send response to server fro processing.
	p.server.sendAsync(resp)
}

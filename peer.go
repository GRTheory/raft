package raft

import (
	"sync"
	"time"
)

type Peer struct {
	server            *server
	Name              string `json:"name"`
	ConnectionString  string `json:"connectionString"`
	preLogIndex       uint64
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
	return p.preLogIndex
}

// Sets the previous log index.
func (p *Peer) setPrevLogIndex(value uint64) {
	p.Lock()
	defer p.Unlock()
	p.preLogIndex = value
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

// func (p *Peer) flush(){
// 	// debugln("peer.heartbeat.flush: ", p.Name)
// 	prevLogIndex := p.getPrevLogIndex()
// 	term := p.server.currentTerm

// 	entries, prevLogTerm := p.server.log.getEntriesAfter(prevLogIndex, p.server.maxLogEntriesPerReqesut)

// 	if entries != nil {
		
// 	}
// }

// // Sends an AppendEntries request to the peer through the transport.
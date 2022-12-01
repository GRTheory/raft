package raft

import "sync"

type Peer struct {
	server *server
	sync.RWMutex
}

func (p *Peer) clone() *Peer {
	p.Lock()
	defer p.Unlock()
	return &Peer{}
}

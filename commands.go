package raft

// Join command interface
type JoinCommand interface {
	Command
	NodeName() string
}
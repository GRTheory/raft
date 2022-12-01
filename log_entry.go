package raft

import "github.com/GRTheory/raft/protobuf"

type LogEntry struct {
	pb *protobuf.LogEntry
}
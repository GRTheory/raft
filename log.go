package raft

import (
	"fmt"
	"io"
	"os"
	"sync"
)

// A log is a collection of log entries that are persisted to durable storage.
type Log struct {
	ApplyFunc   func(*LogEntry, Command) (interface{}, error)
	file        *os.File
	path        string
	entries     []*LogEntry
	commitIndex uint64
	mutex       sync.RWMutex
	startIndex  uint64 // the index before the first entry in the Log entries
	startTerm   uint64
	initialized bool
}

// The results of the applying a log entry.
type logResult struct {
	returnValue interface{}
	err         error
}

// Create a new log.
func newLog() *Log {
	return &Log{
		entries: make([]*LogEntry, 0),
	}
}

// The last committed index in the log.
func (l *Log) CommitIndex() uint64 {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	return l.commitIndex
}

// The current index in the log.
func (l *Log) currentIndex() uint64 {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	return l.internalCurrentIndex()
}

// The current index in the log without locking.
func (l *Log) internalCurrentIndex() uint64 {
	if len(l.entries) == 0 {
		return l.startIndex
	}
	return l.entries[len(l.entries)-1].Index()
}

// The next index in the log.
func (l *Log) nextIndex() uint64 {
	return l.currentIndex() + 1
}

// Determines if the log contains zero entries.
func (l *Log) isEmpty() bool {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	return (len(l.entries) == 0) && (l.startIndex == 0)
}

// The name of the last command in the log.
func (l *Log) lastCommandName() string {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	if len(l.entries) > 0 {
		if entry := l.entries[len(l.entries)-1]; entry != nil {
			return entry.CommandName()
		}
	}
	return ""
}

// The current term in the log.
func (l *Log) currentTerm() uint64 {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	if len(l.entries) == 0 {
		return l.startTerm
	}
	return l.entries[len(l.entries)-1].Term()
}

// Opens the log file and reads existing entries. The log can remain open and
// cotinue to append entries to the end of the log.
func (l *Log) open(path string) error {
	// Read all the entries from the log if one exists.
	var readBytes int64

	var err error
	// debugln("log.open.open ", path)
	l.file, err = os.OpenFile(path, os.O_RDWR, 0600)
	l.path = path

	if err != nil {
		// if the log file does not exist before
		// we create the log file and set commitIndex to 0
		if os.IsNotExist(err) {
			l.file, err = os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0600)
			// debugln("log.open.create ", path)
			if err == nil {
				l.initialized = true
			}
			return err
		}
		return err
	}
	// debugln("log.open.exist “, path)

	// Read the file and decode entries.
	for {
		// Instantiate log entry and decode into it.
		entry := &LogEntry{}
		entry.Position, _ = l.file.Seek(0, os.SEEK_CUR)

		n, err := entry.Decode(l.file)
		if err != nil {
			if err == io.EOF {
				// debugln("open.log.append: finish ")
			} else {
				if err = os.Truncate(path, readBytes); err != nil {
					return fmt.Errorf("raft.Log: Unable to recover: %v", err)
				}
			}
			break
		}
		if entry.Index() > l.startIndex {
			// Append entry.
			l.entries = append(l.entries, entry)
			if entry.Index() <= l.commitIndex {
				// if the entry index is smaller than committed index, add this entry
				// to state machine.
				command, err := newCommand(entry.CommandName(), entry.Command())
				if err != nil {
					continue
				}
				l.ApplyFunc(entry, command)
			}
			// debugln("open.log.append log index ", entry.Index())
		}

		readBytes += int64(n)
	}
	// debugln("open.log.recovery number of log ", len(l.entries))
	l.initialized = true
	return nil
}

// Closes the log file.
func (l *Log) close() {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.file != nil {
		l.file.Close()
		l.file = nil
	}
	l.entries = make([]*LogEntry, 0)
}

// sync to disk
func (l *Log) sync() error {
	return l.file.Sync()
}

// Creates a log entry associated with this log.
func (l *Log) createEntry(term uint64, command Command, e *ev) (*LogEntry, error) {
	return newLogEntry(l, e, l.nextIndex(), term, command)
}

// Retrieves an entry from the log. If the entry has been eliminated because
// of a snapshot then nil is returned.
func (l *Log) getEntry(index uint64) *LogEntry {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	if index <= l.startIndex || index > (l.startIndex+uint64(len(l.entries))) {
		return nil
	}
	return l.entries[index-l.startIndex-1]
}

// Checks if the log contains a given index/term combination.
func (l *Log) containsEntry(index uint64, term uint64) bool {
	entry := l.getEntry(index)
	return (entry != nil && entry.Term() == term)
}

func (l *Log) getEntriesAfter(index uint64, maxLogEntriesPerReqesut uint64) ([]*LogEntry, uint64) {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	// Return nil if index is before the start of the log.
	if index < l.startIndex {
		// traceln("log.entriesAfter.before: ", index, " ", l.startIndex)
		return nil, 0
	}

	// Return an error if the index doesn't exist.
	if index > (uint64(len(l.entries)) + l.startIndex) {
		panic(fmt.Sprintf("raft: Index is beyond end of log: %v %v", len(l.entries), index))
	}

	// If we're going from the beginning of the log then return the whole log.
	if index == l.startIndex {
		// traceln("log.entriesAfter.beginning: ", index, " ", l.startIndex)
		return l.entries, l.startTerm
	}

	// traceln("log.entriesAfter.partial: ", index, " ", l.entries[len(l.entries)-1].Index)

	entries := l.entries[index-l.startIndex:]
	length := len(entries)

	// traceln("log.entriesAfter: startIndex:", l.startIndex, " length", len(l.entries))

	if uint64(length) < maxLogEntriesPerReqesut {
		// Determine the term at the given entry and return a subslice.
		return entries, l.entries[index-1-l.startIndex].Term()
	} else {
		return entries[:maxLogEntriesPerReqesut], l.entries[index-1-l.startIndex].Term()
	}
}

// Retrieves the last index and term that has been committed to the log.
func (l *Log) commitInfo() (index uint64, term uint64) {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	// If we don't have any committed entries then just return zeros.
	if l.commitIndex == 0 {
		return 0, 0
	}

	// No new commit log after snapshot
	if l.commitIndex == l.startIndex {
		return l.startIndex, l.startTerm
	}

	// Return the last index & term from the last committed entry.
	// debugln("commitInfo.get.[", l.commitIndex, "/", l.startIndex, "]")
	entry := l.entries[l.commitIndex-1-l.startIndex]
	return entry.Index(), entry.Term()
}

// Retrieves the last index and term that has been appended to the log.
func (l *Log) lastInfo() (index uint64, term uint64) {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	// If we don't have any entries then jsut reutrn zeros.
	if len(l.entries) == 0 {
		return l.startIndex, l.startTerm
	}

	// Return the last index & term
	entry := l.entries[len(l.entries)-1]
	return entry.Index(), entry.Term()
}

// Update the commit index
func (l *Log) updateCommitIndex(index uint64) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if index > l.commitIndex {
		l.commitIndex = index
	}
	// debugln("update.commit.index ", index)
}

// Update the commit index and writes entries after that index to the stabe storage.
func (l *Log) setCommitIndex(index uint64) error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	// this is not error any more after limited the number of sending entries
	// commit up to what we alraady have.
	if index > l.startIndex+uint64(len(l.entries)) {
		// debugln("raft.Log: Commit index", index, "set back to ", len(l.entries))
		index = l.startIndex + uint64(len(l.entries))
	}

	// Do not allow previous indices to be committed again.

	// This could happens, since the guarantee is that the new leader has up-to-dated
	// log entries rather than has nost up-to-dated committed index

	// For example, Leader 1 send log 80 to follower 2 and follower 3
	// follower 2 and follower 3 all got the new entries and reply
	// leader 1 committed entry 80 and send reply to follower 2 and follower 3
	// follower 2 receive the new committed index and update committed index to 80
	// leader 1 fail to send the committed index to follower 3
	// follower 3 promote to leader (server 1 and server 2 will vote, since leader 3
	// has up-to-dated the entries)
	// when new leader 3 send heartbeat with committed index = 0 to follower 2,
	// follower 2 should reply success and let leader 3 update the committed index to 80

	if index < l.commitIndex {
		return nil
	}

	// Find all entries whose index is between the previous index and the current index.
	for i := l.commitIndex + 1; i <= index; i++ {
		entryIndex := i - 1 - l.startIndex
		entry := l.entries[entryIndex]

		// Update commit index.
		l.commitIndex = entry.Index()

		// Decode the command.
		command, err := newCommand(entry.CommandName(), entry.Command())
		if err != nil {
			return err
		}

		// Apply the changes to the state machine and store the error code.
		returnValue, err := l.ApplyFunc(entry, command)

		// debugf("setCommitIndex.set.result index； %v,entries index: %v", i, entryIndex)
		if entry.event != nil {
			entry.event.returnValue = returnValue
			entry.event.c <- err
		}

		_, isJoinCommand := command.(JoinCommand)

		// we can only commit up to the most recent join command
		// if there is a join in this batch of commands.
		// after this commit, we need to recalculate the majority.
		if isJoinCommand {
			return nil
		}
	}
	return nil
}

// Set the commitIndex at the head of the log file to the current
// commit Index. This should be called after obtained a log lock
func (l *Log) flushCommitIndex() {
	l.file.Seek(0, io.SeekStart)
	fmt.Fprintf(l.file, "%8x\n", l.commitIndex)
	l.file.Seek(0, io.SeekEnd)
}
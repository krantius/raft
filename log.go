package raft

import (
	"sync"

	"github.com/krantius/logging"
)

type Log struct {
	CurrentTerm  int
	CurrentIndex int
	CommitIndex  int
	logs         []LogEntry
	mu           sync.Mutex
}

type LogEntry struct {
	Term      int
	Index     int
	Committed bool
	Cmd       Command
}

// Append is called when a leader requests a follower to append entries
func (l *Log) Append(term, prevIndex, prevTerm, commitIndex int, entries []LogEntry) bool {
	logging.Infof("term=%d prevIndex=%d prevTerm=%d commitIndex=%d entriesLen=%d", term, prevIndex, prevTerm, commitIndex, len(entries))
	if term < l.CurrentTerm {
		return false
	}

	l.CurrentTerm = term

	// First one, append and move on
	/*if l.CurrentIndex == -1 {
		l.append(entries)
		return true
	}*/

	// First one
	if entries[0].Index == 0 && l.CurrentIndex == -1 {
		l.append(entries)
		return true
	}

	if l.CurrentIndex < prevIndex {
		logging.Errorf("Current Index %d < prevIndex %d", l.CurrentIndex, prevIndex)
		return false
	}

	if l.logs[prevIndex].Term != prevTerm {
		logging.Errorf("PrevIndex %d term %d not equal %d", prevIndex, l.logs[prevIndex].Term, prevTerm)
		return false
	}

	/*difference := l.CurrentIndex - prevIndex
	if difference != 0 {
		// Trim off any old entries
		l.logs = l.logs[:len(l.logs)-difference]
		// TODO probably need to update the fsm with the deletes...
	}*/

	// Actually add the new entries
	l.append(entries)

	// Commit whatever is in the commitIndex
	// TODO currently only committing on heartbeats
	//l.commit(commitIndex)

	logging.Tracef("Applied term=%d index=%d commit=%d", term, l.CurrentIndex, l.CommitIndex)

	return true
}

func (l *Log) appendCmd(c Command) LogEntry {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.CurrentIndex++

	le := LogEntry{
		Term:  l.CurrentTerm,
		Index: l.CurrentIndex,
		Cmd:   c,
	}

	l.logs = append(l.logs, le)

	return le
}

// append will just add to the entries and upate the term/index
func (l *Log) append(entries []LogEntry) {
	l.logs = append(l.logs, entries...)

	last := l.logs[len(l.logs)-1]
	l.CurrentIndex = last.Index
	l.CurrentTerm = last.Term

	logging.Tracef("Appended the goods, %+v", l.logs)
}

// commit will set all entries between the current index and the provided index
func (l *Log) commit(index int) []LogEntry {
	if l.CommitIndex >= index {
		return nil
	}

	committedLogs := []LogEntry{}
	for i := l.CommitIndex + 1; i <= index; i++ {
		l.logs[i].Committed = true
		committedLogs = append(committedLogs, l.logs[i])
	}

	l.CommitIndex = index
	logging.Infof("Committed index=%d %+v", index, committedLogs)

	return committedLogs
}

func (l *Log) walk(le LogEntry) LogEntry {
	if le.Index == 0 {
		// Probably won't get here, but might as well check
		logging.Warning("Tried to walk on the very first log entry")
		return le
	}

	return l.logs[le.Index-1]
}

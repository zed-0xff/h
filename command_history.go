package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const (
	MAX_COMMAND_HISTORY_ENTRIES = 1024
)

type CommandHistory struct {
	entries       []CommandHistoryEntry
	pos           int
	fileTimestamp time.Time
	fileSize      int64
	ready         bool
}

type CommandHistoryEntry struct {
	Command string
	Ts      int64
}

var commandHistory = &CommandHistory{}

func (sh *CommandHistory) Add(command string) {
	if !sh.ready {
		return
	}
	if len(command) == 0 {
		return
	}
	if len(sh.entries) > 0 && sh.entries[len(sh.entries)-1].Command == command {
		return
	}
	sh.entries = append(sh.entries, CommandHistoryEntry{Command: command, Ts: time.Now().UnixNano()})
	sh.pos = len(sh.entries) - 1
	go sh.Save()
}

func (sh *CommandHistory) Prev() string {
	if sh.ready && sh.pos > 0 {
		sh.pos--
		return sh.entries[sh.pos].Command
	}
	return ""
}

func (sh *CommandHistory) Next() string {
	if sh.ready && sh.pos < len(sh.entries)-1 {
		sh.pos++
		return sh.entries[sh.pos].Command
	}
	return ""
}

func (sh *CommandHistory) fileName() string {
	path, err := getAppDir()
	if err != nil {
		return ""
	}
	return filepath.Join(path, "command_history.json")
}

func (sh *CommandHistory) Save() {
	fname := sh.fileName()
	if fname == "" {
		return
	}

	err := os.MkdirAll(filepath.Dir(fname), 0700)
	if err != nil {
		lastErrMsg = "[?] " + err.Error()
		return
	}

	merged := sh

	fi, err := os.Stat(fname)
	if err == nil && (fi.ModTime() != sh.fileTimestamp || fi.Size() != sh.fileSize) {
		merged = &CommandHistory{}
		merged.Load()
		merged.entries = append(merged.entries, sh.entries...)
		merged.sort_uniq()
	}

	entries := merged.entries
	if len(merged.entries) > MAX_COMMAND_HISTORY_ENTRIES {
		// keep last
		entries = entries[len(merged.entries)-MAX_COMMAND_HISTORY_ENTRIES:]
	}

	f, err := os.Create(fname)
	if err != nil {
		lastErrMsg = "[?] " + err.Error()
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.Encode(entries)
}

func (sh *CommandHistory) sort_uniq() {
	sort.Slice(sh.entries, func(i, j int) bool {
		return sh.entries[i].Ts < sh.entries[j].Ts
	})

	for i := 0; i < len(sh.entries)-1; {
		if sh.entries[i].Ts == sh.entries[i+1].Ts {
			sh.entries = append(sh.entries[:i], sh.entries[i+1:]...)
		} else {
			i++
		}
	}
}

func initCommandHistory() {
	commandHistory.Load()
}

func (sh *CommandHistory) Load() {
	sh.tryLoad()
	sh.pos = len(sh.entries)
	sh.ready = true
}

func (sh *CommandHistory) tryLoad() {
	fname := sh.fileName()
	if fname == "" {
		return
	}
	fi, err := os.Stat(fname)
	if err != nil {
		return
	}

	sh.fileTimestamp = fi.ModTime()
	sh.fileSize = fi.Size()

	f, err := os.Open(fname)
	if err != nil {
		return
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	if err := dec.Decode(&sh.entries); err != nil {
		lastErrMsg = "[?] " + err.Error()
	}
}

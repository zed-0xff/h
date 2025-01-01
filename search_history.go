package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const (
	MAX_SEARCH_HISTORY_ENTRIES = 1024
)

type SearchHistory struct {
	entries       []SearchHistoryEntry
	pos           int
	fileTimestamp time.Time
	fileSize      int64
	ready         bool
}

type SearchHistoryEntry struct {
	Mode    int
	Pattern []byte
	Ts      int64
}

var searchHistory = &SearchHistory{}

func (sh *SearchHistory) Add(mode int, pattern []byte) {
	if !sh.ready {
		return
	}
	if len(pattern) == 0 {
		return
	}
	if len(sh.entries) > 0 && sh.entries[len(sh.entries)-1].Mode == mode && bytes.Equal(sh.entries[len(sh.entries)-1].Pattern, pattern) {
		return
	}
	sh.entries = append(sh.entries, SearchHistoryEntry{Mode: mode, Pattern: pattern, Ts: time.Now().UnixNano()})
	sh.pos = len(sh.entries) - 1
	go sh.Save()
}

func (sh *SearchHistory) Prev() (int, []byte) {
	if sh.ready && sh.pos > 0 {
		sh.pos--
		return sh.entries[sh.pos].Mode, sh.entries[sh.pos].Pattern
	}
	return -1, nil
}

func (sh *SearchHistory) Next() (int, []byte) {
	if sh.ready && sh.pos < len(sh.entries)-1 {
		sh.pos++
		return sh.entries[sh.pos].Mode, sh.entries[sh.pos].Pattern
	}
	return -1, nil
}

func getAppDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "h"), nil
}

func (sh *SearchHistory) fileName() string {
	path, err := getAppDir()
	if err != nil {
		return ""
	}
	return filepath.Join(path, "search_history.json")
}

func (sh *SearchHistory) Save() {
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
		merged = &SearchHistory{}
		merged.Load()
		merged.entries = append(merged.entries, sh.entries...)
		merged.sort_uniq()
	}

	entries := merged.entries
	if len(merged.entries) > MAX_SEARCH_HISTORY_ENTRIES {
		// keep last
		entries = entries[len(merged.entries)-MAX_SEARCH_HISTORY_ENTRIES:]
	}

	f, err := os.Create(fname)
	if err != nil {
		lastErrMsg = "[?] " + err.Error()
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.Encode(entries)
}

func (sh *SearchHistory) sort_uniq() {
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

func initSearch() {
	searchHistory.Load()
	if searchHistory.ready && len(searchHistory.entries) > 0 {
		lastSearch := searchHistory.entries[len(searchHistory.entries)-1]
		g_searchMode = lastSearch.Mode
		g_searchPattern = lastSearch.Pattern
	}
}

func (sh *SearchHistory) Load() {
	sh.tryLoad()
	sh.pos = len(sh.entries)
	sh.ready = true
}

func (sh *SearchHistory) tryLoad() {
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

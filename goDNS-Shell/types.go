package main

import "sync"

type comandManager struct {
	CommandMap     map[string]Command
	CMMutex        *sync.RWMutex
	WaitingCommand string
}

type response struct {
	Chunks      []chunk
	TotalChunks int64
	ReadChunks  int64
}

type chunk struct {
	Body string
	Num  int64
}

type Command struct {
	SentValue string
	UUID      string
	Response  response
}

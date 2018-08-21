package main

import (
	"fmt"
	"strconv"
)

func (cm comandManager) getCommandToSend() string {
	cm.CMMutex.RLock()
	defer cm.CMMutex.RUnlock()
	return cm.WaitingCommand
}

func (cm *comandManager) getCommand(c string) Command {
	cm.CMMutex.RLock()
	defer cm.CMMutex.RUnlock()
	if v, ok := cm.CommandMap[c]; ok {
		return v
	}
	return Command{}
}

func (cm *comandManager) setCommandToSend(s string) {
	cm.CMMutex.Lock()
	defer cm.CMMutex.Unlock()
	cm.WaitingCommand = s
}

func (cm *comandManager) UpdateCmd(uuid, maxchunks, thischunk, vals string) {
	cm.CMMutex.Lock()
	defer cm.CMMutex.Unlock()
	c := cm.CommandMap[uuid]

	cn, e := strconv.ParseInt(thischunk, 10, 64)
	if e != nil {
		fmt.Println("Bad chunk number: ", e)
		return
	}
	mc, e := strconv.ParseInt(maxchunks, 10, 64)
	if e != nil {
		fmt.Println("Bad max chunk number: ", e)
		return
	}
	c.Response.TotalChunks = mc
	c.Response.AddChunk(cn, vals)
	c.Response.ReadChunks++
	cm.CommandMap[uuid] = c

}

func (cm *comandManager) AddCommand(c Command) {
	cm.CMMutex.Lock()
	defer cm.CMMutex.Unlock()
	cm.CommandMap[c.UUID] = c
}

package main

import (
	"encoding/base32"
	"fmt"
	"sort"
	"strings"
)

func (r *response) AddChunk(cnum int64, val string) {
	r.Chunks = append(r.Chunks, chunk{Body: val, Num: cnum})
}

func (r response) IsDone() bool {
	if r.TotalChunks == 0 || r.TotalChunks > r.ReadChunks {
		return false
	}
	return true
}

func (r response) ReadResposne() string {
	//sort chunks
	rval := ""
	sort.Slice(r.Chunks, func(i, j int) bool {
		if r.Chunks[i].Num < r.Chunks[j].Num {
			return true
		}
		return false
	})
	for _, x := range r.Chunks {
		rval += x.Body
	}
	rval = strings.Replace(rval, "-", "==", -1)
	v, e := base32.HexEncoding.DecodeString(rval)
	if e != nil {
		fmt.Println("ReadResponse Error:", e)
	}
	return string(v)
}

package main

import (
	"flag"
	"fmt"
	"github.com/hashicorp/raft"
	"io"
	"net/http"
	"os"
)

func main() {
	httpAddr := flag.String("http-addr", ":8000", "htt addr")
	raftAddr := flag.String("raft-addr", "localhost:7000", "raft addr")
	raftDir := flag.String("raft-dir", "raft", "raft dir")
	raftId := flag.String("raft-id", "", "raft id")
	join := flag.String("join", "", "join cluster")
	flag.Parse()
	if *raftId == "" {
		hostname, err := os.Hostname()
		if err != nil {
			panic(err)
		}
		raftId = &hostname
	}
	ap := &App{HttpAddr: *httpAddr, RaftID: *raftId}
	_ = os.MkdirAll(*raftDir, os.ModePerm)
	lw := &LogWriter{}
	rf, err := newRaft(*raftDir, *raftId, *raftAddr, ap, lw)
	if err != nil {
		panic(err)
	}
	ap.Rf = rf
	ob := make(chan raft.Observation)
	observer := raft.NewObserver(ob, false, func(o *raft.Observation) bool {
		return true
	})
	rf.RegisterObserver(observer)
	if *join == "" {
		err = bootstrap(rf, *raftId, *raftAddr)
		if err != nil {
			panic(err)
		}
	} else {
		get, err := http.Get(*join + "?id=" + *raftId + "&addr=" + *raftAddr)
		if err != nil {
			panic(err)
		}
		if get.StatusCode != http.StatusOK {
			all, err := io.ReadAll(get.Body)
			if err != nil {
				panic(err)
			}
			panic(string(all))
		}
	}
	go func() {
		for v := range ob {
			switch v.Data.(type) {
			case raft.LeaderObservation:
				fmt.Println(v.Data)
			}
		}
	}()
	err = ap.Run()
	if err != nil {
		panic(err)
	}
}

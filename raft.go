package main

import (
	"github.com/hashicorp/raft"
	"github.com/hashicorp/raft-boltdb"
	"io"
	"net"
	"path/filepath"
	"time"
)

type LogWriter struct {
}

func (l *LogWriter) Write(p []byte) (n int, err error) {
	return 0, nil
}

func newRaft(baseDir, raftId, addr string, fsm raft.FSM, lw io.Writer) (*raft.Raft, error) {
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(raftId)
	config.LogOutput = lw
	stableStore, err := raftboltdb.NewBoltStore(filepath.Join(baseDir, "stable.dat"))
	if err != nil {
		return nil, err
	}
	logStore, err := raftboltdb.NewBoltStore(filepath.Join(baseDir, "log.dat"))
	if err != nil {
		return nil, err
	}
	snapshotStore, err := raft.NewFileSnapshotStore(baseDir, 3, lw)
	if err != nil {
		return nil, err
	}
	advA, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, err
	}
	transport, err := raft.NewTCPTransport(addr, advA, 2, 5*time.Second, lw)
	if err != nil {
		return nil, err
	}
	return raft.NewRaft(config, fsm, stableStore, logStore, snapshotStore, transport)
}

func bootstrap(rf *raft.Raft, raftId, raftAddr string) error {
	config := rf.GetConfiguration().Configuration()
	if len(config.Servers) != 0 {
		return nil
	}
	return rf.BootstrapCluster(raft.Configuration{Servers: []raft.Server{
		{
			ID:      raft.ServerID(raftId),
			Address: raft.ServerAddress(raftAddr),
		},
	}}).Error()
}

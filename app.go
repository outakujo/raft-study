package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/hashicorp/raft"
	"io"
	"net/http"
	"time"
)

type App struct {
	HttpAddr string
	Rf       *raft.Raft
	cmd      []byte
	RaftID   string
}

func (a *App) Persist(sink raft.SnapshotSink) error {
	defer sink.Close()
	_, err := sink.Write(a.cmd)
	return err
}

func (a *App) Release() {
}

func (a *App) Apply(l *raft.Log) interface{} {
	a.cmd = l.Data
	fmt.Println("index", a.Rf.LastIndex())
	fmt.Println("apply", string(l.Data), l.Index)
	return nil
}

func (a *App) Snapshot() (raft.FSMSnapshot, error) {
	return a, nil
}

func (a *App) Restore(snapshot io.ReadCloser) error {
	all, err := io.ReadAll(snapshot)
	if err != nil {
		return err
	}
	a.cmd = all
	return nil
}

func (a *App) Run() error {
	en := gin.New()
	en.GET("/", func(c *gin.Context) {
		_, leaderID := a.Rf.LeaderWithID()
		err := a.Rf.VerifyLeader().Error()
		if err != nil {
			c.JSON(http.StatusOK, leaderID)
			return
		}
		id := c.Query("id")
		addr := c.Query("addr")
		if id != "" && addr != "" {
			err := a.Rf.AddVoter(raft.ServerID(id), raft.ServerAddress(addr),
				a.Rf.GetConfiguration().Index(), 2*time.Second).Error()
			if err != nil {
				c.JSON(http.StatusInternalServerError, err.Error())
				return
			}
			c.JSON(http.StatusOK, "ok")
		} else {
			c.JSON(http.StatusOK, "ok")
		}
	})
	en.Use(func(c *gin.Context) {
		path := c.Request.URL.Path
		err := a.Rf.Apply([]byte(path), 2*time.Second).Error()
		if err != nil {
			c.JSON(http.StatusOK, err.Error())
			return
		}
		c.JSON(http.StatusOK, "ok")
	})
	return en.Run(a.HttpAddr)
}

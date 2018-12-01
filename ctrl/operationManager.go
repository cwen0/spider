package main

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/unrolled/render"
	"github.com/urfave/negroni"
	"github.com/zhouqiang-cl/hack/types"
)

// APIPrefix is api prefix
const APIPrefix = "/operation"

type State struct {
	operation string
	partition types.Partition
}

type Log struct {
	operation string
	parameter string
	time      string
}

// Manager is the operation manager.
type Manager struct {
	sync.RWMutex
	addr string
	s    *http.Server
}

// NewManager creates the node with given address
func NewManager(addr string) *Manager {
	n := &Manager{
		addr: addr,
	}

	return n
}

// Run runs operation manager
func (c *Manager) Run() error {
	c.s = &http.Server{
		Addr:    c.addr,
		Handler: c.createHandler(),
	}
	return c.s.ListenAndServe()
}

// Close closes the Node.
func (c *Manager) Close() {
	if c.s != nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		c.s.Shutdown(ctx)
		cancel()
	}
}

func (c *Manager) createHandler() http.Handler {
	engine := negroni.New()
	recover := negroni.NewRecovery()
	engine.Use(recover)

	router := mux.NewRouter()
	subRouter := c.createRouter()
	router.PathPrefix(APIPrefix).Handler(
		negroni.New(negroni.Wrap(subRouter)),
	)

	engine.UseHandler(router)
	return engine
}

func (c *Manager) createRouter() *mux.Router {
	rd := render.New(render.Options{
		IndentJSON: true,
	})

	router := mux.NewRouter().PathPrefix(APIPrefix).Subrouter()

	failpointHandler := newFailpointHandler(c, rd)
	partitionHandler := newPartitionHandler(c, rd)
	topologyHandler := newTopologynHandler(c, rd)
	evictLeaderHandler := newEvictLeaderHandler(c, rd)
	logHandler := newLogHandler(c, rd)
	stateHandler := newStateHandler(c, rd)

	// failpoint route
	router.HandleFunc("/failpoint", failpointHandler.CreateFailpoint).Methods("POST")
	router.HandleFunc("/failpoint", failpointHandler.GetFailpoint).Methods("GET")

	// network partition route
	router.HandleFunc("/partition", partitionHandler.CreateNetworkPartition).Methods("POST")
	router.HandleFunc("/partition", partitionHandler.GetNetworkPartiton).Methods("GET")

	// topology route
	router.HandleFunc("/topology", topologyHandler.GetTopology).Methods("GET")

	// evict leader route
	router.HandleFunc("/evictleader", evictLeaderHandler.EvictLeader).Methods("POST")

	// log route
	router.HandleFunc("/log", logHandler.GetLogs).Methods("GET")

	// state route
	router.HandleFunc("/state", stateHandler.GetState).Methods("GET")

	return router
}
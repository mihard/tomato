package resource

import (
	"log"
	"sync"

	"github.com/alileza/tomato/config"
	"github.com/alileza/tomato/resource/db/sql"
	"github.com/alileza/tomato/resource/http/client"
	"github.com/alileza/tomato/resource/http/server"
	"github.com/alileza/tomato/resource/queue"

	"github.com/pkg/errors"
)

var (
	ErrInvalidType   = errors.New("invalid resource type")
	ErrNotFound      = errors.New("resource not found")
	ErrInvalidParams = errors.New("invalid resource params")
)

type Resource interface {
	// Ready returns nil, if resource is ready to be use.
	Ready() error

	// Close returns nil, if resource successfully terminated.
	Close() error
}

type Manager interface {
	Get(name string) (Resource, error)
	Close()
}

type manager struct {
	resources []*config.Resource
	cache     sync.Map
	log       log.Logger
}

func NewManager(cfgs []*config.Resource) *manager {
	return &manager{resources: cfgs}
}

func (mgr *manager) Get(name string) (Resource, error) {
	for _, resourceCfg := range mgr.resources {
		if resourceCfg.Name == name {
			cache, ok := mgr.cache.Load(resourceCfg)
			if ok {
				return cache.(Resource), nil
			}

			var r Resource
			switch resourceCfg.Type {
			case client.Name:
				r = client.New(resourceCfg.Params)
			case sql.Name:
				r = sql.New(resourceCfg.Params)
			case server.Name:
				r = server.New(resourceCfg.Params)
			case queue.Name:
				r = queue.New(resourceCfg.Params)
			default:
				return nil, ErrInvalidType
			}
			mgr.cache.Store(resourceCfg, r)

			return r, nil
		}
	}
	return nil, errors.WithMessage(ErrNotFound, "resource:"+name)
}

func (mgr *manager) Close() {
	mgr.cache.Range(func(resourceCfg interface{}, resource interface{}) bool {
		cfg := resource.(*config.Resource)
		r := resource.(Resource)
		if err := r.Close(); err != nil {
			mgr.log.Printf("[ERR] %s: failed to terminate resource : %v\n", cfg.Key(), err)
			return false
		}
		return true
	})
}

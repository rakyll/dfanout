package fanout

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/dfanout/dfanout/fanout/clientcache"
	pb "github.com/dfanout/dfanout/proto"
	"github.com/mailgun/groupcache"
)

type Cache struct {
	pool  *groupcache.HTTPPool
	group *groupcache.Group
}

func NewFanoutCache(me string, peers []string, ccache *clientcache.Cache, adminService pb.AdminService, ttl time.Duration) *Cache {
	pool := groupcache.NewHTTPPool("http://" + me)
	if len(peers) > 0 {
		pool.Set(peers...)
	}
	return &Cache{
		// TODO: Check whether the size is sufficient.
		pool: pool,
		group: groupcache.NewGroup("dfanout-fanout-cache", 128<<20, groupcache.GetterFunc(
			func(ctx groupcache.Context, key string, dest groupcache.Sink) error {
				log.Printf("Looking up to cache for %q", key)
				resp, err := adminService.GetFanout(context.Background(), &pb.GetFanoutRequest{
					FanName: key,
				})
				if err != nil {
					return err
				}
				if len(resp.Endpoints) == 0 {
					return errors.New("no endpoints found")
				}
				for _, e := range resp.Endpoints {
					if _, err := ccache.RegisterHTTPClient(key, e); err != nil {
						return nil
					}
				}
				return dest.SetProto(resp, time.Now().Add(ttl))
			},
		)),
	}
}

func (c *Cache) Fanout(ctx context.Context, fanout string) ([]*pb.Endpoint, error) {
	var resp pb.GetFanoutResponse
	if err := c.group.Get(ctx, fanout, groupcache.ProtoSink(&resp)); err != nil {
		return nil, err
	}
	return resp.Endpoints, nil
}

func (c *Cache) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.pool.ServeHTTP(w, r)
}

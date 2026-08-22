package metrics

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type routeKey struct {
	method string
	path   string
	status int
}

type Registry struct {
	startedUnix int64
	requests    atomic.Uint64
	inFlight    atomic.Int64
	durations   atomic.Uint64
	mu          sync.RWMutex
	routes      map[routeKey]uint64
}

func NewRegistry(now time.Time) *Registry {
	return &Registry{startedUnix: now.UTC().Unix(), routes: make(map[routeKey]uint64)}
}

func (r *Registry) Begin() func(method, path string, status int, elapsed time.Duration) {
	r.inFlight.Add(1)
	started := time.Now()
	return func(method, path string, status int, elapsed time.Duration) {
		r.inFlight.Add(-1)
		r.requests.Add(1)
		if elapsed <= 0 {
			elapsed = time.Since(started)
		}
		r.durations.Add(uint64(elapsed.Microseconds()))
		r.mu.Lock()
		r.routes[routeKey{method: method, path: path, status: status}]++
		r.mu.Unlock()
	}
}

func (r *Registry) WritePrometheus(writer io.Writer) error {
	r.mu.RLock()
	keys := make([]routeKey, 0, len(r.routes))
	values := make(map[routeKey]uint64, len(r.routes))
	for key, value := range r.routes {
		keys = append(keys, key)
		values[key] = value
	}
	r.mu.RUnlock()
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].path != keys[j].path {
			return keys[i].path < keys[j].path
		}
		if keys[i].method != keys[j].method {
			return keys[i].method < keys[j].method
		}
		return keys[i].status < keys[j].status
	})
	if _, err := fmt.Fprintf(writer, "# TYPE dust_http_requests_total counter\ndust_http_requests_total %d\n", r.requests.Load()); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "# TYPE dust_http_in_flight gauge\ndust_http_in_flight %d\n", r.inFlight.Load()); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "# TYPE dust_http_duration_microseconds_total counter\ndust_http_duration_microseconds_total %d\n", r.durations.Load()); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "# TYPE dust_process_started_unix gauge\ndust_process_started_unix %d\n", r.startedUnix); err != nil {
		return err
	}
	for _, key := range keys {
		_, err := fmt.Fprintf(writer, "dust_http_route_requests_total{method=%s,path=%s,status=%q} %d\n", strconv.Quote(key.method), strconv.Quote(key.path), strconv.Itoa(key.status), values[key])
		if err != nil {
			return err
		}
	}
	return nil
}

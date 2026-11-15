package topology

import (
	"sync"
	"time"
)

// LinkEnd is one side of a link: a port on a switch.
type LinkEnd struct {
	DPID uint64
	Port uint32
}

// Link is a unidirectional link discovered by LLDP: an LLDP frame
// sent out of Src was received on Dst.
type Link struct {
	Src      LinkEnd
	Dst      LinkEnd
	LastSeen time.Time
}

// LinkRegistry stores the links discovered between switches.
type LinkRegistry struct {
	mu    sync.RWMutex
	links map[LinkEnd]Link // keyed by Src: a port has one neighbour
}

func NewLinkRegistry() *LinkRegistry {
	return &LinkRegistry{
		links: make(map[LinkEnd]Link),
	}
}

// Update records that src -> dst was just seen.
// It reports whether the lik is new (or its detination changed)
func (r *LinkRegistry) Update(src, dst LinkEnd) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	old, exists := r.links[src]
	r.links[src] = Link{Src: src, Dst: dst, LastSeen: time.Now()}

	return !exists || old.Dst != dst
}

// Links returns a copy of all known links.
func (r *LinkRegistry) Links() []Link {
	r.mu.RLock()
	defer r.mu.RUnlock()

	links := make([]Link, 0, len(r.links))
	for _, l := range r.links {
		links = append(links, l)
	}
	return links
}

// Expire removes every link not seen for longer than maxAge and
// return the links it removed
func (r *LinkRegistry) Expire(maxAge time.Duration) []Link {
	r.mu.Lock()
	defer r.mu.Unlock()

	var expired []Link
	for src, l := range r.links {
		if time.Since(l.LastSeen) > maxAge {
			expired = append(expired, l)
			delete(r.links, src)
		}
	}
	return expired
}

// RemovePort removes every link that starts or ends at the given port
// and return the links it removed
func (r *LinkRegistry) RemovePort(end LinkEnd) []Link {
	r.mu.Lock()
	defer r.mu.Unlock()

	var removed []Link
	for src, l := range r.links {
		if l.Src == end || l.Dst == end {
			removed = append(removed, l)
			delete(r.links, src)
		}
	}
	return removed
}

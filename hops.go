package dht

import (
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

type LookupMetrics struct {
	m            *sync.Mutex
	startTime    time.Time
	totalHops    int
	tree         map[peer.ID]*hop
	ogPeers      map[peer.ID]*hop
	closestPeers []peer.ID
}

func NewLookupMetrics() *LookupMetrics {
	return &LookupMetrics{
		m:            new(sync.Mutex),
		startTime:    time.Now(),
		totalHops:    0,
		tree:         make(map[peer.ID]*hop),
		ogPeers:      make(map[peer.ID]*hop),
		closestPeers: make([]peer.ID, 0),
	}
}

func (l *LookupMetrics) addNewPeers(causePeer peer.ID, p []peer.ID) {
	l.m.Lock()
	defer l.m.Unlock()

	// get parent hop
	parentHop, ok := l.searchOgPeer(causePeer)
	if !ok {
		// if cause peer not in the tree, create new entrance an level 0
		parentHop = newHop(causePeer)
		parentHop.original = true

		// add the parent hop to the level 0 tree
		l.tree[causePeer] = parentHop
		l.ogPeers[causePeer] = parentHop
	}

	// iter throught the new peers to add to the tree
	for _, pi := range p {
		// check whether there is already an original peer with the same ID in the tree
		var h *hop
		_, ok = l.searchOgPeer(pi)

		// to avoid having an endless loop over links between hops, create non-original childs to fill the tree
		if ok {
			// if the there is a peer with the same PeerId, we create a replica with origin=false
			h = newHop(pi)
			h.original = false
		} else {
			// if the there isn't a peer with the same PeerId, we create an original one
			h = newHop(pi)
			h.original = true
			l.ogPeers[pi] = h
		}
		// link always the child hop to the parent hop
		parentHop.addSubHop(h)
	}
	l.totalHops++
}

func (l *LookupMetrics) setClosestPeers(cPeers []peer.ID) {
	for _, p := range cPeers {
		l.closestPeers = append(l.closestPeers, p)
	}
}

// GetOgPeers is a non-thread safe operation, please use only for retrieving results
func (l *LookupMetrics) GetOgPeers() map[peer.ID]*hop {
	return l.ogPeers
}

// GetPeerTree is a non-thread safe operation, please use only for retrieving results
func (l *LookupMetrics) GetPeerTree() map[peer.ID]*hop {
	return l.tree
}

func (l *LookupMetrics) GetClosestPeers() []peer.ID {
	return l.closestPeers
}

func (l *LookupMetrics) GetTotalHops() int {
	return l.totalHops
}

func (l *LookupMetrics) GetMinHopsForPeerSet(peerSet []peer.ID) int {
	return l.getHopsForPeerSet(peerSet)
}

// means go through the tree len(peerSet) times to get the total of hops to discover the peers
func (l *LookupMetrics) getHopsForPeerSet(peerSet []peer.ID) int {
	l.m.Lock()
	defer l.m.Unlock()

	// although the hop gives you the minum distance to the peer,
	// whe want the biggest one of those shortest distances
	var biggestSetHop int

	// iter through the peer set to see the sortest depth at when we found it
	for _, p := range peerSet {
		var shortestHop int
		for _, hop := range l.tree {
			// if the target peer is already in the initial hop list, keep searching for the rest of peers (shortest distance)
			if p == hop.causePeer {
				continue
			}
			dist := hop.getShortestDistToPeer(p)
			// keep track of the shortest hop distance to the peer (only when the dist > 0)
			if dist > 0 {
				if shortestHop == 0 {
					shortestHop = dist
				}
				if dist < shortestHop {
					shortestHop = dist
				}
			}
		}
		// Once the shortest distance has been computed, compare it with the biggestHop one ()
		if shortestHop > biggestSetHop {
			biggestSetHop = shortestHop // TODO: we still have to figure it out whether we want to add the seed peers as hops
		}
	}
	return biggestSetHop
}

func (l *LookupMetrics) GetStartTime() time.Time {
	return l.startTime
}

func (l *LookupMetrics) GetTreeDepth() int {
	return l.getTreeDepth()
}

func (l *LookupMetrics) GetContectedPeers() int {
	return l.totalHops
}

func (l *LookupMetrics) getTreeDepth() int {
	l.m.Lock()
	defer l.m.Unlock()

	var maxHops int

	// go through the entire tree checking which is the largest branch
	for _, v := range l.tree {
		auxHops := v.getNumberOfHops() // no previous hops since we are at parent
		if auxHops > maxHops {
			maxHops = auxHops
		}
	}
	return maxHops // the first hop is always our self-host peer id, so don't count it
}

func (l *LookupMetrics) searchOgPeer(peerID peer.ID) (*hop, bool) {
	h, ok := l.ogPeers[peerID]
	return h, ok
}

type hop struct {
	m               sync.Mutex
	firstReportTime time.Time        `json:"first-report-time"` // Time when the peer was first reported or tracked in the tree
	causePeer       peer.ID          `json:"cause-peer"`        // peer that we contacted in the specific hop
	original        bool             `json:"original"`          // identify if this peer is a original of existing one (came later in the lookup)
	hops            map[peer.ID]*hop `json:"hops"`              // subsecuent hops from this same one
}

func newHop(causePeer peer.ID) *hop {
	return &hop{
		firstReportTime: time.Now(),
		causePeer:       causePeer,
		original:        false,
		hops:            make(map[peer.ID]*hop),
	}
}

func (h *hop) len() int {
	return len(h.hops)
}

func (h *hop) addSubHop(subHop *hop) {
	h.m.Lock()
	defer h.m.Unlock()

	// add it to the map of the hop parent
	h.hops[subHop.causePeer] = subHop
}

func (h *hop) getNumberOfHops() int {
	h.m.Lock()
	defer h.m.Unlock()

	var parentBase, maxHops int

	// iter through the hops asking for their lenght
	if h.len() > 0 {
		parentBase += 1
	}
	for _, v := range h.hops {
		hopsNumber := v.getNumberOfHops()
		if hopsNumber > maxHops {
			maxHops = hopsNumber
		}
	}
	return parentBase + maxHops
}

func (h *hop) getShortestDistToPeer(target peer.ID) int {
	h.m.Lock()
	defer h.m.Unlock()

	depth := 1
	var shortestDist int // init at 0 to show that we didn't found it

	// check if the peer is in the current list of hops (return depth straight away)
	for _, nextHop := range h.hops {
		if target == nextHop.causePeer {
			return depth
		}
	}

	// if the peer wasn't inside the direct hop peers, call following ones
	for _, nextHop := range h.hops {
		hopCount := nextHop.getShortestDistToPeer(target)
		// check if the hopCount is smaller that the original
		// track if original is still 0 and if the new one is also 0
		if hopCount > 0 {
			if shortestDist == 0 {
				shortestDist = hopCount
				continue
			}
			if hopCount < shortestDist {
				shortestDist = hopCount
			}
		}
	}
	// f we found the peer, add the depth to the measurement
	if shortestDist > 0 {
		shortestDist += depth // add the current depth to the shortest distance
	}
	// return the shortest distance
	return shortestDist
}

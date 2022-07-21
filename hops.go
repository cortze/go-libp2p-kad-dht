package dht

import (
	"sync"

	"github.com/libp2p/go-libp2p-core/peer"
	log "github.com/sirupsen/logrus"
)

// TODO: 	missing unitary test verification
// 			+ add the code to the query.run() function
// 			+ is it interesting to keep track of the peer IDs of each lookup?

// TOTALLY NEW to calculate the HOPS on the lookup
type queryHops struct {
	m         sync.Mutex
	hopRounds map[peer.ID]*hop
}

func newQueryHops() *queryHops {
	log.Debug("New query hops")
	return &queryHops{
		hopRounds: make(map[peer.ID]*hop),
	}
}

func (qh *queryHops) addNewPeer(causePeer peer.ID, p peer.ID) {
	qh.m.Lock()
	defer qh.m.Unlock()

	log.WithFields(log.Fields{
		"cause": causePeer.String(),
		"peer":  p.String(),
	}).Debug("Adding new peer")

	// get parent hop
	parentHop, ok := qh.searchPeer(causePeer)
	if !ok {
		// if casue peer not in the tree, create new entrance an level 0
		parentHop = newHop(causePeer)

		// add the parent hop to the level 0 tree
		qh.hopRounds[causePeer] = parentHop
	}

	// check whether there is already a peer with the same ID in the tree
	h, ok := qh.searchPeer(p)
	if !ok {
		// if the peer is not there yet at the tree, create a new instance and link it to parent hop
		h = newHop(p)
	}

	// only add the hop to the parent if there wasn't any child peer already in the tree
	if h.len() == 0 {
		// link always the child hop to the parent hop
		parentHop.addSubHop(h)
	}

}

// means go through the tree len(peerSet) times to get the total of hops to discover the peers
func (qh *queryHops) getHopsForPeerSet(peerSet []peer.ID) int {
	qh.m.Lock()
	defer qh.m.Unlock()

	// although the hop gives you the minum distance to the peer,
	// whe want the biggest one of those shortest distances
	var biggestSetHop int

	// iter through the peer set to see the sortest depth at when we found it
	for _, p := range peerSet {
		var shortestHop int
		for _, hop := range qh.hopRounds {
			// if the target peer is already in the initial hop list, keep searching for the rest of peers (shortest distance)
			if p == hop.causePeer {
				shortestHop = 1
				continue
			}
			dist := hop.getShortestDistToPeer(p)
			// keep track of the shortest hop distance to the peer (only when the dist > 0)
			if dist > 0 {
				// add to dist the hop of the seed peers
				dist++
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

	if biggestSetHop == 0 {
		log.Warn("peers in closest on, not found in hops tree")
	}

	log.WithFields(log.Fields{
		"peerSetLen": len(peerSet),
		"hops":       biggestSetHop,
	}).Debug("Adding new peer")

	return biggestSetHop

}

func (qh *queryHops) getHops() int {
	qh.m.Lock()
	defer qh.m.Unlock()

	//peerCache := make(map[peer.ID]bool)
	var maxHops int

	// go through the entire tree checking which is the largest branch
	for _, v := range qh.hopRounds {
		//	peerCache[v.causePeer] = true            // add to the cache the seed peers
		auxHops := v.getNumberOfHops() // no previous hops since we are at parent
		if auxHops > maxHops {
			maxHops = auxHops
		}
	}
	return maxHops + 1 // add the first hop of searching in the routing table
}

func (qh *queryHops) searchPeer(peerID peer.ID) (*hop, bool) {
	log.WithFields(log.Fields{
		"peer": peerID.String(),
	}).Debug("searching peer")

	// iter through the number of initial hops
	for p, h := range qh.hopRounds {
		if p == peerID {
			return h, true
		}
		auxH, ok := h.searchPeer(peerID)
		if ok {
			return auxH, true
		}
	}
	// if previosu search didn't succeed, return failure searching
	return nil, false
}

type hop struct {
	m         sync.Mutex
	causePeer peer.ID
	hops      map[peer.ID]*hop
}

func newHop(causePeer peer.ID) *hop {
	log.WithFields(log.Fields{
		"cause": causePeer.String(),
	}).Debug("new hop")

	return &hop{
		causePeer: causePeer,
		hops:      make(map[peer.ID]*hop),
	}
}

func (h *hop) len() int {
	return len(h.hops)
}

func (h *hop) searchPeer(peerID peer.ID) (*hop, bool) {
	h.m.Lock()
	defer h.m.Unlock()

	log.WithFields(log.Fields{
		"peer": peerID.String(),
	}).Debug("searching peer in hop")

	// iter through each of the hops in the list
	for p, hp := range h.hops {
		if p == peerID {
			return hp, true
		}
		auxH, ok := hp.searchPeer(peerID)
		if ok {
			return auxH, true
		}
	}
	// if previosu search didn't succeed, return failure searching
	return nil, false
}

func (h *hop) addSubHop(subHop *hop) {
	h.m.Lock()
	defer h.m.Unlock()

	// add it to the map of the hop parent
	h.hops[subHop.causePeer] = subHop
}

func (h *hop) getCausePeer() peer.ID {
	h.m.Lock()
	defer h.m.Unlock()

	return h.causePeer
}

func (h *hop) getHops() map[peer.ID]*hop {
	h.m.Lock()
	defer h.m.Unlock()
	return h.hops
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

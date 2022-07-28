package dht

import (
	"sync"

	"github.com/libp2p/go-libp2p-core/peer"
	log "github.com/sirupsen/logrus"
)

// TODO: -- simplyfy the tree by removing queryHops (the begining of the tree is always the key that we are seatching for)
type Hops struct {
	Total     int
	ToClosest int
}

type queryHops struct {
	m       sync.Mutex
	tree    map[peer.ID]*hop
	ogPeers map[peer.ID]*hop
}

func newQueryHops() *queryHops {
	log.Trace("New query hops")
	return &queryHops{
		tree:    make(map[peer.ID]*hop),
		ogPeers: make(map[peer.ID]*hop),
	}
}

func (qh *queryHops) addNewPeers(causePeer peer.ID, p []peer.ID) {
	qh.m.Lock()
	defer qh.m.Unlock()

	log.WithFields(log.Fields{
		"cause": causePeer.String(),
		"peer":  len(p),
	}).Trace("Adding new peers")

	// get parent hop
	parentHop, ok := qh.searchOgPeer(causePeer)
	if !ok {
		// if casue peer not in the tree, create new entrance an level 0
		parentHop = newHop(causePeer)
		parentHop.original = true

		// add the parent hop to the level 0 tree
		qh.tree[causePeer] = parentHop
		qh.ogPeers[causePeer] = parentHop
	}

	// iter throught the new peers to add to the tree
	for _, pi := range p {

		log.WithFields(log.Fields{
			"cause": causePeer.String(),
			"peer":  pi.String(),
		}).Trace("Adding new peer")

		// check whether there is already an original peer with the same ID in the tree
		var h *hop
		_, ok = qh.searchOgPeer(pi)

		// to avoid having an endless loop over links between hops, create non-original childs to fill the tree
		if ok {
			// if the there is a peer with the same PeerId, we create a replica with origin=false
			h = newHop(pi)
			h.original = false
		} else {
			// if the there isn't a peer with the same PeerId, we create an original one
			h = newHop(pi)
			h.original = true
			qh.ogPeers[pi] = h

		}
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
		for _, hop := range qh.tree {
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

	log.WithFields(log.Fields{
		"peerSetLen": len(peerSet),
		"hops":       biggestSetHop,
	}).Trace("Adding new peer")

	return biggestSetHop

}

func (qh *queryHops) getHops() int {
	qh.m.Lock()
	defer qh.m.Unlock()

	//peerCache := make(map[peer.ID]bool)
	var maxHops int

	// go through the entire tree checking which is the largest branch
	for _, v := range qh.tree {
		//	peerCache[v.causePeer] = true            // add to the cache the seed peers
		auxHops := v.getNumberOfHops() // no previous hops since we are at parent
		if auxHops > maxHops {
			maxHops = auxHops
		}
	}
	return maxHops // the first hop is always our self-host peer id, so don't count it
}

func (qh *queryHops) searchOgPeer(peerID peer.ID) (*hop, bool) {
	log.WithFields(log.Fields{
		"peer": peerID.String(),
	}).Trace("searching peer")

	// iter through the ogPeers tree (optimized version)
	h, ok := qh.ogPeers[peerID]
	return h, ok

	// --- Depecated: was adding to much overhead when adding pers ---
	//
	// // iter through the number of initial hops
	// for p, h := range qh.tree {
	// 	if p == peerID && h.original {
	// 		return h, true
	// 	}
	// 	auxH, ok := h.searchPeer(peerID)
	// 	if ok {
	// 		if !auxH.original {
	// 			log.Panic("pointer to non-original hop has been received at QueryHops")
	// 		}
	// 		return auxH, true
	// 	}
	// }
	// // if previosu search didn't succeed, return failure searching
	// return nil, false
}

type hop struct {
	m         sync.Mutex
	causePeer peer.ID
	original  bool // identify if this peer is a original of existing one (came later in the lookup)
	hops      map[peer.ID]*hop
}

func newHop(causePeer peer.ID) *hop {
	log.WithFields(log.Fields{
		"cause": causePeer.String(),
	}).Trace("new hop")

	return &hop{
		causePeer: causePeer,
		original:  false,
		hops:      make(map[peer.ID]*hop),
	}
}

func (h *hop) len() int {
	return len(h.hops)
}

// --- Depecated: was adding to much overhead when adding pers ---
// moved to a OgPeers cache in queryHops
func (h *hop) searchPeer(peerID peer.ID) (*hop, bool) {
	h.m.Lock()
	defer h.m.Unlock()

	log.WithFields(log.Fields{
		"peer": peerID.String(),
	}).Trace("searching peer in hop")

	// iter through each of the hops in the list
	for p, hp := range h.hops {
		// return only the hop of the peer that is not a original of an existing one
		if p == peerID && hp.original {
			return hp, true
		}
		auxH, ok := hp.searchPeer(peerID)
		if ok {
			if !auxH.original {
				log.Panic("pointer to non-original hop has been received at Hop")
			}
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

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

	// link always the child hop to the parent hop
	parentHop.addSubHop(h)
}

func (qh *queryHops) getHops() int {
	qh.m.Lock()
	defer qh.m.Unlock()

	var maxHops int

	// go through the entire tree checking which is the largest branch
	for _, v := range qh.hopRounds {
		auxHops := v.GetNumberOfHops() // no previous hops since we are at parent
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

func (h *hop) CausePeer() peer.ID {
	h.m.Lock()
	defer h.m.Unlock()

	return h.causePeer
}

func (h *hop) Hops() map[peer.ID]*hop {
	h.m.Lock()
	defer h.m.Unlock()
	return h.hops
}

func (h *hop) GetNumberOfHops() int {
	var parentBase, maxHops int

	// iter through the hops asking for their lenght
	if h.len() > 0 {
		parentBase += 1
	}
	for _, v := range h.hops {
		hopsNumber := v.GetNumberOfHops()
		if hopsNumber > maxHops {
			maxHops = hopsNumber
		}
	}
	return parentBase + maxHops
}

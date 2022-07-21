package dht

import (
	"testing"

	"github.com/libp2p/go-libp2p-core/peer"
	//log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestHop(t *testing.T) {
	//log.SetLevel(log.DebugLevel)

	//  schema:
	// peer 0 	-- peer 1
	// 			-- peer 2	-- peer 3
	//

	peerByteSet := []string{
		"0",
		"1",
		"2",
		"3"}

	var peerIDSet []peer.ID
	// compose peer.IDs from strings (casted to bytes)
	for _, s := range peerByteSet {
		p := peer.ID(s)
		peerIDSet = append(peerIDSet, p)
	}

	// generate hop from peer 0
	hop := newHop(peerIDSet[0])

	// generate hop for peer 1 and 2
	hop1 := newHop(peerIDSet[1])
	hop2 := newHop(peerIDSet[2])
	hop3 := newHop(peerIDSet[3])

	// add subhops of peers 1 and 2 from the main peer 0
	hop.addSubHop(hop1)
	hop.addSubHop(hop2)

	hop2.addSubHop(hop3)

	require.Equal(t, hop.len(), 2)
	require.Equal(t, hop1.len(), 0)
	require.Equal(t, hop2.len(), 1)

	// check the len of the hops
	l := hop.getNumberOfHops()
	require.Equal(t, 2, l)

	// test the shortest hop to a given peer
	shortestHop := hop.getShortestDistToPeer(peerIDSet[3])
	require.Equal(t, 2, shortestHop)
}

func TestHopsQuery(t *testing.T) {

	//  schema:
	// peer 0 	-- peer 2	-- peer 8	-- peer 5
	// 			-- peer 3	-- peer 6
	//
	// peer 1 	-- peer 4	-- peer 6	-- peer 7
	// 			-- peer 5	-- peer 7

	peerByteSet := []string{
		"0",
		"1",
		"2",
		"3",
		"4",
		"5",
		"6",
		"7",
		"8"}

	var peerIDSet []peer.ID
	// compose peer.IDs from strings (casted to bytes)
	for _, s := range peerByteSet {
		p := peer.ID(s)
		peerIDSet = append(peerIDSet, p)
	}

	// generate the parent queryHop
	qHop := newQueryHops()

	// add peer 3 and 4 as child hops from 1
	qHop.addNewPeer(peerIDSet[0], peerIDSet[2])
	qHop.addNewPeer(peerIDSet[0], peerIDSet[3])

	require.Equal(t, 2, qHop.hopRounds[peerIDSet[0]].len())

	// add peer 5 and 6 as child hops from 2
	qHop.addNewPeer(peerIDSet[1], peerIDSet[4])
	qHop.addNewPeer(peerIDSet[1], peerIDSet[5])

	require.Equal(t, 2, qHop.hopRounds[peerIDSet[1]].len())

	// add peer 7 as child hop from 4 and 5
	qHop.addNewPeer(peerIDSet[3], peerIDSet[6])
	qHop.addNewPeer(peerIDSet[4], peerIDSet[6])

	// add peer 9 as child of 3
	qHop.addNewPeer(peerIDSet[2], peerIDSet[8])

	// add peer 5 as child of 9
	qHop.addNewPeer(peerIDSet[8], peerIDSet[4])

	h2, ok := qHop.searchPeer(peerIDSet[2])
	require.Equal(t, true, ok)
	h3, ok := qHop.searchPeer(peerIDSet[3])
	require.Equal(t, true, ok)
	h4, ok := qHop.searchPeer(peerIDSet[4])
	require.Equal(t, true, ok)

	require.Equal(t, 1, h2.len())
	require.Equal(t, 1, h3.len())
	require.Equal(t, 1, h4.len())

	// add peer 8 as child hop from 6 and 7
	qHop.addNewPeer(peerIDSet[5], peerIDSet[7])
	qHop.addNewPeer(peerIDSet[6], peerIDSet[7])

	h5, ok := qHop.searchPeer(peerIDSet[5])
	require.Equal(t, true, ok)
	h6, ok := qHop.searchPeer(peerIDSet[6])
	require.Equal(t, true, ok)

	require.Equal(t, 1, h5.len())
	require.Equal(t, 1, h6.len())

	hops := qHop.getHops()
	require.Equal(t, 4, hops)

	// test the shortest hop to a given peer
	var peerArr1 []peer.ID = []peer.ID{peerIDSet[2], peerIDSet[7]}

	shortestHop := qHop.getHopsForPeerSet(peerArr1)
	require.Equal(t, 3, shortestHop)

	// test the shortest hop to a given peer
	var peerArr2 []peer.ID = []peer.ID{peerIDSet[4], peerIDSet[5]}

	shortestHop = qHop.getHopsForPeerSet(peerArr2)
	require.Equal(t, 2, shortestHop)

	// test the shortest hop to a given peer
	var peerArr3 []peer.ID = []peer.ID{peerIDSet[6], peerIDSet[7]}

	shortestHop = qHop.getHopsForPeerSet(peerArr3)
	require.Equal(t, 3, shortestHop)

	// test the shortest hop to a given peer
	var peerArr4 []peer.ID = []peer.ID{peerIDSet[0], peerIDSet[1]}

	shortestHop = qHop.getHopsForPeerSet(peerArr4)
	require.Equal(t, 1, shortestHop)

}

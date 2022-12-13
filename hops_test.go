package dht

import (
	"testing"

	"github.com/libp2p/go-libp2p/core/peer"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestHop(t *testing.T) {
	log.SetLevel(log.InfoLevel)

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

	shortestHop = hop.getShortestDistToPeer(peerIDSet[2])
	require.Equal(t, 1, shortestHop)

	shortestHop = hop.getShortestDistToPeer(peerIDSet[1])
	require.Equal(t, 1, shortestHop)

	shortestHop = hop.getShortestDistToPeer(peerIDSet[0])
	require.Equal(t, 0, shortestHop)
}

func TestHopsQuery(t *testing.T) {

	//  schema:
	// peer 0 	-- peer 2	-- peer 6	-- peer 7
	// 					   `-- peer 7	-- peer 6
	// 		   `-- peer 3	-- peer 8	-- peer 7
	// 								   `-- peer 6
	// 					   `-- peer 6
	//
	// peer 1 	-- peer 4	-- peer 7
	//
	// 			-- peer 5	-- peer 9 	-- peer 7
	//								   `-- peer 6
	// 		    		   `-- peer 10 	-- peer 11	--peer6

	peerByteSet := []string{
		"0000",
		"1111",
		"2222",
		"3333",
		"4444",
		"5555",
		"6666",
		"7777",
		"8888",
		"aaaa",
		"bbbb",
		"cccc"}

	var peerIDSet []peer.ID
	// compose peer.IDs from strings (casted to bytes)
	for _, s := range peerByteSet {
		p := peer.ID(s)
		peerIDSet = append(peerIDSet, p)
	}

	// generate the parent queryHop
	qHop := newQueryHops()

	// ---- level 1 and 2 of the tree ----
	// add peer 2 and 3 as child hops from 0
	qHop.addNewPeers(peerIDSet[0], []peer.ID{peerIDSet[2], peerIDSet[3]})

	require.Equal(t, 2, qHop.tree[peerIDSet[0]].len())

	// add peer 4 and 5 as child hops from 1
	qHop.addNewPeers(peerIDSet[1], []peer.ID{peerIDSet[4], peerIDSet[5]})

	hops := qHop.getHops()
	require.Equal(t, 1, hops)

	require.Equal(t, 2, qHop.tree[peerIDSet[1]].len())

	// ---- level 3 of the tree ----
	// add peer 6 as child hop from 2 and 3
	qHop.addNewPeers(peerIDSet[2], []peer.ID{peerIDSet[6]})
	qHop.addNewPeers(peerIDSet[3], []peer.ID{peerIDSet[6]})

	// add peer 7 as child hop from 2 and 4
	qHop.addNewPeers(peerIDSet[2], []peer.ID{peerIDSet[7]})
	qHop.addNewPeers(peerIDSet[4], []peer.ID{peerIDSet[7]})

	// add 8 from 3
	qHop.addNewPeers(peerIDSet[3], []peer.ID{peerIDSet[8]})

	hops = qHop.getHops()
	require.Equal(t, 2, hops)

	// ---- level 4 of the tree ----
	// add 6 and 7 from 8
	qHop.addNewPeers(peerIDSet[8], []peer.ID{peerIDSet[6], peerIDSet[7]})

	// add peer 7 and 6 to depend from eachother
	qHop.addNewPeers(peerIDSet[6], []peer.ID{peerIDSet[7]})
	qHop.addNewPeers(peerIDSet[7], []peer.ID{peerIDSet[6]})

	// add 9 and 10 from 5
	qHop.addNewPeers(peerIDSet[5], []peer.ID{peerIDSet[9], peerIDSet[10]})

	// add 6 and 7 from 9
	qHop.addNewPeers(peerIDSet[9], []peer.ID{peerIDSet[6], peerIDSet[7]})

	// add 11 from 10
	qHop.addNewPeers(peerIDSet[10], []peer.ID{peerIDSet[11]})

	hops = qHop.getHops()
	require.Equal(t, 3, hops)

	// TEMPORARY
	require.Equal(t, 2, qHop.tree[peerIDSet[0]].len())

	// ---- level 5 of the tree ----
	// add 6 from 11
	qHop.addNewPeers(peerIDSet[11], []peer.ID{peerIDSet[6]})

	hops = qHop.getHops()
	require.Equal(t, 4, hops)

	// TEMPORARY
	require.Equal(t, 2, qHop.tree[peerIDSet[0]].len())

	// ---- TESTING THE LOGIC / DEPTH ---

	// level 1
	h0, ok := qHop.searchOgPeer(peerIDSet[0])
	require.Equal(t, true, ok)
	h1, ok := qHop.searchOgPeer(peerIDSet[1])
	require.Equal(t, true, ok)

	require.Equal(t, 2, h0.len())
	require.Equal(t, 2, h1.len())

	// level 2
	h2, ok := qHop.searchOgPeer(peerIDSet[2])
	require.Equal(t, true, ok)
	h3, ok := qHop.searchOgPeer(peerIDSet[3])
	require.Equal(t, true, ok)
	h4, ok := qHop.searchOgPeer(peerIDSet[4])
	require.Equal(t, true, ok)
	h5, ok := qHop.searchOgPeer(peerIDSet[5])
	require.Equal(t, true, ok)

	require.Equal(t, 2, h2.len())
	require.Equal(t, 2, h3.len())
	require.Equal(t, 1, h4.len())
	require.Equal(t, 2, h5.len())

	// level 3
	h6, ok := qHop.searchOgPeer(peerIDSet[6])
	require.Equal(t, true, ok)
	h7, ok := qHop.searchOgPeer(peerIDSet[7])
	require.Equal(t, true, ok)
	h8, ok := qHop.searchOgPeer(peerIDSet[8])
	require.Equal(t, true, ok)
	h9, ok := qHop.searchOgPeer(peerIDSet[9])
	require.Equal(t, true, ok)
	h10, ok := qHop.searchOgPeer(peerIDSet[10])
	require.Equal(t, true, ok)

	require.Equal(t, 1, h6.len())
	require.Equal(t, 1, h7.len())
	require.Equal(t, 2, h8.len())
	require.Equal(t, 2, h9.len())
	require.Equal(t, 1, h10.len())

	// level 4
	h11, ok := qHop.searchOgPeer(peerIDSet[11])
	require.Equal(t, true, ok)
	require.Equal(t, 1, h11.len())

	// -- Test the Tree Depth

	hops = qHop.getHops()
	require.Equal(t, 4, hops)

	// -- Test the shortest hop to a given peer
	var peerArr1 []peer.ID = []peer.ID{peerIDSet[2], peerIDSet[7]}

	shortestHop := qHop.getHopsForPeerSet(peerArr1)
	require.Equal(t, 2, shortestHop)

	var peerArr2 []peer.ID = []peer.ID{peerIDSet[4], peerIDSet[5]}

	shortestHop = qHop.getHopsForPeerSet(peerArr2)
	require.Equal(t, 1, shortestHop)

	var peerArr3 []peer.ID = []peer.ID{peerIDSet[6], peerIDSet[7]}

	shortestHop = qHop.getHopsForPeerSet(peerArr3)
	require.Equal(t, 2, shortestHop)

	var peerArr4 []peer.ID = []peer.ID{peerIDSet[0], peerIDSet[1]}

	shortestHop = qHop.getHopsForPeerSet(peerArr4)
	require.Equal(t, 0, shortestHop)

	var peerArr5 []peer.ID = []peer.ID{peerIDSet[11], peerIDSet[6]}

	shortestHop = qHop.getHopsForPeerSet(peerArr5)
	require.Equal(t, 3, shortestHop)

}

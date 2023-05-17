package dht

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/stretchr/testify/require"

	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
)

func TestInvalidRemotePeers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mn, err := mocknet.FullMeshLinked(5)
	if err != nil {
		t.Fatal(err)
	}
	defer mn.Close()
	hosts := mn.Hosts()

	os := []Option{testPrefix, DisableAutoRefresh(), Mode(ModeServer)}
	d, err := New(ctx, hosts[0], os...)
	if err != nil {
		t.Fatal(err)
	}
	for _, proto := range d.serverProtocols {
		// Hang on every request.
		hosts[1].SetStreamHandler(proto, func(s network.Stream) {
			defer s.Reset() // nolint
			<-ctx.Done()
		})
	}

	err = mn.ConnectAllButSelf()
	if err != nil {
		t.Fatal("failed to connect peers", err)
	}

<<<<<<< HEAD
<<<<<<< HEAD
	// Wait at a bit for a peer in our routing table.
	for i := 0; i < 100 && d.routingTable.Size() == 0; i++ {
		time.Sleep(10 * time.Millisecond)
	}
	if d.routingTable.Size() == 0 {
		t.Fatal("failed to fill routing table")
	}

	ctx1, cancel1 := context.WithTimeout(ctx, 1*time.Second)
	defer cancel1()

	done := make(chan error, 1)
	go func() {
		_, _, err := d.GetClosestPeers(ctx1, testCaseCids[0].KeyString())
		done <- err
	}()

=======
>>>>>>> 8c9fdff (fix: don't add unresponsive DHT servers to the Routing Table (#820))
=======
>>>>>>> d373974 (fix: don't add unresponsive DHT servers to the Routing Table (#820))
	time.Sleep(100 * time.Millisecond)

	// hosts[1] isn't added to the routing table because it isn't responding to
	// the DHT request
	require.Equal(t, 0, d.routingTable.Size())
}

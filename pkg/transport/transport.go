package transport

import (
	"context"
	"fmt"
	"gx/ipfs/QmR8BauakNcBa3RbE4nbQu76PDiJgoQgz8AJdhJuiU4TAw/go-cid"
	"gx/ipfs/QmRKLtwMw131aK7ugC3G7ybpumMz78YrJe5dzneyindvG1/go-multiaddr"
	"gx/ipfs/QmUymf8fJtideyv3z727BcZUifGBjMZMpCJqu3Gxk5aRUk/go-libp2p-peerstore"
	"gx/ipfs/QmVrjR2KMe57y4YyfHdYa3yKD278gN8W7CTiqSuYmxjA7F/go-libp2p-host"
	"gx/ipfs/QmXnpYYg2onGLXVxM4Q5PEFcx29k8zeJQkPeLAk9h9naxg/go-libp2p"
	"gx/ipfs/QmXnpYYg2onGLXVxM4Q5PEFcx29k8zeJQkPeLAk9h9naxg/go-libp2p/p2p/host/routed"
	"gx/ipfs/QmadRyQYRn64xHb5HKy2jRFp2Der643Cgo7NEjFgs4MX2k/go-libp2p-kad-dht"
	"gx/ipfs/QmadRyQYRn64xHb5HKy2jRFp2Der643Cgo7NEjFgs4MX2k/go-libp2p-kad-dht/opts"
	"gx/ipfs/QmcqU6QUDSXprb1518vYDGczrTJTyGwLG9eUa5iNX4xUtS/go-libp2p-peer"
	"gx/ipfs/QmdQmRSSAGmZvBcbETygeTbsqLLn4k69ELvTxVbEiZxGmA/go-libp2p-pubsub"
	"gx/ipfs/QmerPMzPk1mJVowm8KgmoknWa4yCYvvugMPsgWmDNUvDLW/go-multihash"
)

var BootstrapAddresses = []string{
	"/dnsaddr/bootstrap.libp2p.io/ipfs/QmSoLer265NRgSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd",
	"/ip4/104.131.131.82/tcp/4001/ipfs/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",            // mars.i.ipfs.io
	"/ip4/104.236.179.241/tcp/4001/ipfs/QmSoLPppuBtQSGwKDZT2M73ULpjvfd3aZ6ha4oFGL1KrGM",           // pluto.i.ipfs.io
	"/ip4/128.199.219.111/tcp/4001/ipfs/QmSoLSafTMBsPKadTEgaXctDQVcqN88CNLHXMkTNwMKPnu",           // saturn.i.ipfs.io
	"/ip4/104.236.76.40/tcp/4001/ipfs/QmSoLV4Bbm51jM9C4gDYZQ9Cy3U6aXMJDAbzgu2fzaDs64",             // venus.i.ipfs.io
	"/ip4/178.62.158.247/tcp/4001/ipfs/QmSoLer265NRgSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd",            // earth.i.ipfs.io
	"/ip6/2604:a880:1:20::203:d001/tcp/4001/ipfs/QmSoLPppuBtQSGwKDZT2M73ULpjvfd3aZ6ha4oFGL1KrGM",  // pluto.i.ipfs.io
	"/ip6/2400:6180:0:d0::151:6001/tcp/4001/ipfs/QmSoLSafTMBsPKadTEgaXctDQVcqN88CNLHXMkTNwMKPnu",  // saturn.i.ipfs.io
	"/ip6/2604:a880:800:10::4a:5001/tcp/4001/ipfs/QmSoLV4Bbm51jM9C4gDYZQ9Cy3U6aXMJDAbzgu2fzaDs64", // venus.i.ipfs.io
	"/ip6/2a03:b0c0:0:1010::23:1001/tcp/4001/ipfs/QmSoLer265NRgSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd", // earth.i.ipfs.io
}

type Transport struct {
	node      *host.Host
	dhtclient *dht.IpfsDHT
	Rnode     *routedhost.RoutedHost
	Pubsub    *pubsub.PubSub

	CurrentTopic *pubsub.Subscription
}

func Connect(ctx context.Context) (*Transport, error) {
	node, err := libp2p.New(ctx)
	if err != nil {
		return nil, err
	}

	dhtclient, err := dht.New(ctx, node, dhtopts.Client(false))
	if err != nil {
		return nil, err
	}

	rnode := routedhost.Wrap(node, dhtclient)

	err = bootstrapDHT(ctx, rnode)
	if err != nil {
		return nil, err
	}

	pubSub, err := pubsub.NewFloodSub(ctx, rnode, pubsub.WithStrictSignatureVerification(false))
	if err != nil {
		return nil, err
	}

	return &Transport{
		node:      &node,
		dhtclient: dhtclient,
		Rnode:     rnode,
		Pubsub:    pubSub,
	}, nil
}

func (transport *Transport) Join(ctx context.Context, topic string) error {
	c, err := cid.V1Builder{Codec: cid.Raw, MhType: multihash.ID}.Sum([]byte(topic))
	if err != nil {
		return err
	}

	go func() {
		transport.dhtclient.Provide(ctx, c, true)
	}()
	go func() {
		pc := transport.dhtclient.FindProvidersAsync(ctx, c, 80)
		for peerInfo := range pc {
			go func() {
				fmt.Printf("dialing %s, %d\n", peerInfo.ID.Pretty(), len(peerInfo.Addrs))
				for _, a := range peerInfo.Addrs {
					fmt.Println(a)
				}
				err := transport.Rnode.Connect(ctx, peerInfo)
				fmt.Printf("dial status: %s\n", err)
			}()
		}
		fmt.Println("discovery done")

	}()

	sub, err := transport.Pubsub.Subscribe(topic)
	if err != nil {
		return err
	}
	transport.CurrentTopic = sub
	go func() {
		for {
			msg, err := sub.Next(ctx)
			if peer.ID(msg.From).Pretty() == transport.Rnode.ID().Pretty() {
				continue
			}
			if err != nil {
				panic(err)
			}
			println(string(msg.Data) + "-" + string(peer.ID(msg.From).Pretty()))
		}
	}()
	return nil
}

func bootstrapDHT(ctx context.Context, node *routedhost.RoutedHost) error {
	/*ctx, cancel := context.WithCancel(ctx)
	defer cancel()*/

	done := make(chan error, len(BootstrapAddresses))
	for _, addrs := range BootstrapAddresses {
		pi, err := peerstore.InfoFromP2pAddr(multiaddr.StringCast(addrs))
		if err != nil {
			panic(err)
		}
		go func() {
			done <- node.Connect(ctx, *pi)
		}()
	}
	for range BootstrapAddresses {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-done:
			if err == nil {
				return nil
			}
			fmt.Printf("bootstrap error: %s", err)
		}
	}
	return fmt.Errorf("failed to bootstrap DHT")
}

func (transport *Transport) Close() {
	transport.Rnode.Close()
	(*transport.node).Close()
	transport.dhtclient.Close()
}

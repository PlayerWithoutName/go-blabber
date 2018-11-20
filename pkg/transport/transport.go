package transport

import (
	"context"
	"fmt"

	"gx/ipfs/QmQAGG1zxfePqj2t7bLxyN8AFccZ889DDR9Gn8kVLDrGZo/go-libp2p-peerstore"
	"gx/ipfs/QmQsw6Nq2A345PqChdtbWVoYbSno7uqRDHwYmYpbPHmZNc/go-libp2p-kad-dht"
	"gx/ipfs/QmQsw6Nq2A345PqChdtbWVoYbSno7uqRDHwYmYpbPHmZNc/go-libp2p-kad-dht/opts"
	"gx/ipfs/QmR8BauakNcBa3RbE4nbQu76PDiJgoQgz8AJdhJuiU4TAw/go-cid"
	"gx/ipfs/QmRKLtwMw131aK7ugC3G7ybpumMz78YrJe5dzneyindvG1/go-multiaddr"
	"gx/ipfs/QmVvV8JQmmqPCwXAaesWJPheUiEFQJ9HWRhWhuFuxVQxpR/go-libp2p"
	"gx/ipfs/QmWY1pHdRP1rA2ifUuCu1ZwFJ8ZzpSEcgXsu9haH21AYKd/go-libp2p-quic-transport"
	"gx/ipfs/QmZBH87CAPFHcc7cYmBqeSQ98zQ3SX9KUxiYgzPmLWNVKz/go-libp2p-routing"
	"gx/ipfs/QmahxMNoNuSsgQefo9rkpcfRFmQrMN6Q99aztKXf63K7YJ/go-libp2p-host"
	"gx/ipfs/Qmc3BYVGtLs8y3p4uVpARWyo3Xk2oCBFF1AhYUVMPWgwUK/go-libp2p-pubsub"
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
	"/ip4/10.1.2.197/tcp/4001/ipfs/QmZatNPNW8DnpMRgSuUJjmzc6nETJi21c2quikh4jbmPKk",                // m6k ipfs node
	"/ip4/51.75.35.194/udp/4001/quic/ipfs/QmVGX47BzePPqEzpkTwfUJogPZxHcifpSXsGdgyHjtk5t7",         // m6k potato
}

type Transport struct {
	dhtclient *dht.IpfsDHT
	Node      host.Host
	Pubsub    *pubsub.PubSub

	CurrentTopic *pubsub.Subscription
}

func Connect(ctx context.Context) (*Transport, error) {
	var dhtclient *dht.IpfsDHT
	node, err := libp2p.New(ctx, libp2p.DefaultTransports, libp2p.Transport(libp2pquic.NewTransport), libp2p.Routing(func(bh host.Host) (routing.PeerRouting, error) {
		d, err := dht.New(ctx, bh, dhtopts.Client(true))
		if err != nil {
			return nil, err
		}
		if err := d.Bootstrap(ctx); err != nil {
			return nil, err
		}

		dhtclient = d

		return d, err
	}))
	if err != nil {
		return nil, err
	}

	pubSub, err := pubsub.NewFloodSub(ctx, node, pubsub.WithStrictSignatureVerification(true))
	if err != nil {
		return nil, err
	}

	err = bootstrapDHT(ctx, node)
	if err != nil {
		return nil, err
	}

	return &Transport{
		Node:      node,
		dhtclient: dhtclient,
		Pubsub:    pubSub,
	}, nil
}

func (transport *Transport) Join(ctx context.Context, topic string) error {
	c, err := cid.V1Builder{Codec: cid.Raw, MhType: multihash.ID}.Sum([]byte(topic))
	if err != nil {
		return err
	}

	fmt.Printf("Prov: %s\n", c.String())
	fmt.Printf("Self: /ipfs/%s\n", transport.Node.ID().Pretty())

	go func() {
		transport.dhtclient.Provide(ctx, c, true)
	}()
	go func() {
		pc := transport.dhtclient.FindProvidersAsync(ctx, c, 80)
		for peerInfo := range pc {
			go func(peerInfo peerstore.PeerInfo) {
				fmt.Printf("dialing %s, %d\n", peerInfo.ID.Pretty(), len(peerInfo.Addrs))
				for _, a := range peerInfo.Addrs {
					fmt.Println(a)
				}
				err := transport.Node.Connect(ctx, peerInfo)
				fmt.Printf("dial status: %s\n", err)
			}(peerInfo)
		}
		fmt.Println("discovery done")

	}()

	sub, err := transport.Pubsub.Subscribe(topic)
	if err != nil {
		return err
	}
	transport.CurrentTopic = sub
	return nil
}

func bootstrapDHT(ctx context.Context, node host.Host) error {
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
	transport.Node.Close()
	transport.dhtclient.Close()
}

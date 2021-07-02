module github.com/textileio/powergate/v2

go 1.16

require (
	github.com/apoorvam/goterminal v0.0.0-20180523175556-614d345c47e5
	github.com/caarlos0/spin v1.1.0
	github.com/charmbracelet/bubbles v0.7.6
	github.com/charmbracelet/bubbletea v0.13.1
	github.com/cheggaaa/pb/v3 v3.0.7
	github.com/containerd/continuity v0.0.0-20200228182428-0f16d7a0959c // indirect
	github.com/davidlazar/go-crypto v0.0.0-20200604182044-b73af7476f6c // indirect
	github.com/desertbit/timer v0.0.0-20180107155436-c41aec40b27f // indirect
	github.com/dustin/go-humanize v1.0.0
	github.com/filecoin-project/go-address v0.0.5
	github.com/filecoin-project/go-dagaggregator-unixfs v0.1.0
	github.com/filecoin-project/go-data-transfer v1.4.3
	github.com/filecoin-project/go-fil-commcid v0.0.0-20201016201715-d41df56b4f6a
	github.com/filecoin-project/go-fil-commp-hashhash v0.0.0-20210329070555-d36b89a2504c
	github.com/filecoin-project/go-fil-markets v1.2.5
	github.com/filecoin-project/go-jsonrpc v0.1.4-0.20210217175800-45ea43ac2bec
	github.com/filecoin-project/go-state-types v0.1.1-0.20210506134452-99b279731c48
	github.com/filecoin-project/lotus v1.10.0
	github.com/gin-contrib/location v0.0.2
	github.com/gin-contrib/static v0.0.0-20191128031702-f81c604d8ac2
	github.com/gin-gonic/gin v1.6.3
	github.com/golang/protobuf v1.4.3
	github.com/google/go-cmp v0.5.5
	github.com/google/uuid v1.2.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.2
	github.com/improbable-eng/grpc-web v0.14.0
	github.com/ipfs/go-blockservice v0.1.4
	github.com/ipfs/go-car v0.0.4
	github.com/ipfs/go-cid v0.0.7
	github.com/ipfs/go-datastore v0.4.5
	github.com/ipfs/go-ds-badger v0.2.6
	github.com/ipfs/go-ds-badger2 v0.1.1-0.20200708190120-187fc06f714e
	github.com/ipfs/go-graphsync v0.7.0 // indirect
	github.com/ipfs/go-ipfs v0.8.0
	github.com/ipfs/go-ipfs-blockstore v1.0.4
	github.com/ipfs/go-ipfs-exchange-offline v0.0.1
	github.com/ipfs/go-ipfs-files v0.0.8
	github.com/ipfs/go-ipfs-http-client v0.1.0
	github.com/ipfs/go-ipld-cbor v0.0.5
	github.com/ipfs/go-ipld-format v0.2.0
	github.com/ipfs/go-log/v2 v2.1.3
	github.com/ipfs/go-merkledag v0.3.2
	github.com/ipfs/go-unixfs v0.2.6
	github.com/ipfs/interface-go-ipfs-core v0.4.0
	github.com/ipld/go-car v0.2.1-0.20210322190947-cffd36d39d90 // indirect
	github.com/jessevdk/go-assets v0.0.0-20160921144138-4f4301a06e15
	github.com/libp2p/go-libp2p v0.13.0
	github.com/libp2p/go-libp2p-core v0.8.5
	github.com/libp2p/go-libp2p-kad-dht v0.11.1
	github.com/logrusorgru/aurora v0.0.0-20200102142835-e9ef32dff381
	github.com/mattn/go-runewidth v0.0.12
	github.com/mitchellh/go-homedir v1.1.0
	github.com/muesli/termenv v0.7.4
	github.com/multiformats/go-multiaddr v0.3.1
	github.com/multiformats/go-multiaddr-dns v0.2.0
	github.com/multiformats/go-multihash v0.0.14
	github.com/olekukonko/tablewriter v0.0.5
	github.com/ory/dockertest/v3 v3.6.3
	github.com/oschwald/geoip2-golang v1.4.0
	github.com/otiai10/copy v1.4.2
	github.com/phayes/freeport v0.0.0-20180830031419-95f893ade6f2
	github.com/rs/cors v1.7.0
	github.com/spf13/cobra v1.1.3
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.7.0
	github.com/textileio/dsutils v1.0.1
	github.com/textileio/go-ds-measure v0.1.1-0.20210323185620-1df9394d5b7a
	github.com/textileio/go-ds-mongo v0.1.4
	github.com/textileio/go-metrics-opentelemetry v0.0.0-20210323190205-79a1865cff3a
	go.opencensus.io v0.22.6 // indirect
	go.opentelemetry.io/contrib/instrumentation/runtime v0.18.0
	go.opentelemetry.io/otel v0.19.0
	go.opentelemetry.io/otel/exporters/metric/prometheus v0.18.0
	go.opentelemetry.io/otel/metric v0.19.0
	google.golang.org/grpc v1.39.0
	google.golang.org/protobuf v1.25.0
	nhooyr.io/websocket v1.8.6 // indirect
)

replace github.com/ipfs/go-unixfs => github.com/ipfs/go-unixfs v0.2.2

replace github.com/dgraph-io/badger/v2 => github.com/dgraph-io/badger/v2 v2.2007.2

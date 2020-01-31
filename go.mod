module github.com/textileio/filecoin

go 1.13

require (
	contrib.go.opencensus.io/exporter/prometheus v0.1.0
	github.com/AlecAivazis/survey/v2 v2.0.5
	github.com/caarlos0/spin v1.1.0
	github.com/filecoin-project/go-address v0.0.0-20200107215422-da8eea2842b5
	github.com/filecoin-project/go-sectorbuilder v0.0.2-0.20200123143044-d9cc96c53c55
	github.com/filecoin-project/lotus v0.2.7-0.20200129090654-df747c8216b2
	github.com/golang/protobuf v1.3.3
	github.com/google/go-cmp v0.4.0
	github.com/gorilla/websocket v1.4.1
	github.com/gosuri/uilive v0.0.4
	github.com/ip2location/ip2location-go v8.2.0+incompatible
	github.com/ipfs/go-cid v0.0.4
	github.com/ipfs/go-datastore v0.3.1
	github.com/ipfs/go-ds-badger v0.2.0
	github.com/ipfs/go-ipld-cbor v0.0.3
	github.com/ipfs/go-log v1.0.1
	github.com/ipfs/go-log/v2 v2.0.2
	github.com/libp2p/go-libp2p v0.5.1
	github.com/libp2p/go-libp2p-core v0.3.0
	github.com/libp2p/go-libp2p-crypto v0.1.0
	github.com/libp2p/go-libp2p-kad-dht v0.5.0
	github.com/libp2p/go-libp2p-peer v0.2.0
	github.com/logrusorgru/aurora v0.0.0-20200102142835-e9ef32dff381
	github.com/manifoldco/promptui v0.7.0
	github.com/mitchellh/go-homedir v1.1.0
	github.com/multiformats/go-multiaddr v0.2.0
	github.com/multiformats/go-multihash v0.0.10
	github.com/olekukonko/tablewriter v0.0.4
	github.com/pelletier/go-toml v1.6.0 // indirect
	github.com/spf13/afero v1.2.2 // indirect
	github.com/spf13/cast v1.3.1 // indirect
	github.com/spf13/cobra v0.0.5
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/spf13/viper v1.6.2
	github.com/stretchr/testify v1.4.0
	go.opencensus.io v0.22.2
	golang.org/x/xerrors v0.0.0-20191204190536-9bdfabe68543
	google.golang.org/genproto v0.0.0-20191206224255-0243a4be9c8f // indirect
	google.golang.org/grpc v1.27.0
	gopkg.in/ini.v1 v1.51.1 // indirect
	gopkg.in/yaml.v2 v2.2.7 // indirect
)

replace github.com/coreos/go-systemd => github.com/coreos/go-systemd/v22 v22.0.0

replace github.com/filecoin-project/filecoin-ffi => ./extern/filecoin-ffi

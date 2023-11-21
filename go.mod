module github.com/superisaac/rpcmux

go 1.21

toolchain go1.21.3

// replace (
// 	github.com/superisaac/jlib => ./vendors/jlib/
// 	github.com/superisaac/jlib/http => ./vendors/jlib/http/
// )

require (
	github.com/go-redis/redis/v8 v8.11.5
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.9.0
	github.com/stretchr/testify v1.7.0
	github.com/superisaac/jsoff v0.5.4
	gopkg.in/yaml.v2 v2.4.0
)

require (
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/golang-jwt/jwt v3.2.2+incompatible // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/mdlayher/socket v0.4.1 // indirect
	github.com/mdlayher/vsock v1.2.1 // indirect
	github.com/mitchellh/mapstructure v1.4.3 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/net v0.9.0 // indirect
	golang.org/x/sync v0.1.0 // indirect
	golang.org/x/sys v0.7.0 // indirect
	golang.org/x/text v0.9.0 // indirect
	gopkg.in/yaml.v3 v3.0.0 // indirect
)

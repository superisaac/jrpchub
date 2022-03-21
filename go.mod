module github.com/superisaac/rpcz

go 1.17

replace (
	github.com/superisaac/jsonz => ./vendors/jsonz/
	github.com/superisaac/jsonz/http => ./vendors/jsonz/http/
)

require (
	github.com/sirupsen/logrus v1.8.1
	github.com/superisaac/jsonz v0.1.12
)

require (
	github.com/dgrijalva/jwt-go v3.2.0+incompatible // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/mitchellh/mapstructure v1.4.3 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	golang.org/x/net v0.0.0-20211015210444-4f30a5c0130f // indirect
	golang.org/x/sys v0.0.0-20211019181941-9d821ace8654 // indirect
	golang.org/x/text v0.3.7 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

module github.com/saveio/pylons

go 1.14

replace (
	github.com/saveio/carrier => ../carrier
	github.com/saveio/themis => ../themis
	github.com/saveio/themis-go-sdk => ../themis-go-sdk
)

require (
	github.com/gogo/protobuf v1.3.2
	github.com/golang/protobuf v1.5.2
	github.com/mattn/go-sqlite3 v2.0.3+incompatible
	github.com/ontio/ontology-eventbus v0.9.1
	github.com/saveio/carrier v0.0.0-00010101000000-000000000000
	github.com/saveio/themis v0.0.0-00010101000000-000000000000
	github.com/saveio/themis-go-sdk v0.0.0-00010101000000-000000000000
)

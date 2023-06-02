module github.com/saveio/pylons

go 1.16

require (
	github.com/gogo/protobuf v1.3.2
	github.com/golang/protobuf v1.5.2
	github.com/mattn/go-sqlite3 v2.0.3+incompatible
	github.com/onsi/ginkgo v1.16.4 // indirect
	github.com/onsi/gomega v1.13.0 // indirect
	github.com/ontio/ontology-eventbus v0.9.1
	github.com/orcaman/concurrent-map v0.0.0-20210501183033-44dafcb38ecc // indirect
	github.com/saveio/carrier v0.0.0-20230322093539-24eaadd546b5
	github.com/saveio/themis v1.0.175
	github.com/saveio/themis-go-sdk v0.0.0-20230314033227-3033a22d3bcd
	github.com/tjfoc/gmsm v1.2.3 // indirect
)

replace (
	github.com/saveio/carrier => ../carrier
	github.com/saveio/themis => ../themis
	github.com/saveio/themis-go-sdk => ../themis-go-sdk
)

module code.cloudfoundry.org/metrics-discovery

go 1.12

require (
	cloud.google.com/go v0.44.3 // indirect
	code.cloudfoundry.org/go-envstruct v1.5.0
	code.cloudfoundry.org/go-loggregator v0.0.0-20190809213911-969cb33bee6a // pinned
	github.com/go-kit/kit v0.9.0 // indirect
	github.com/go-stack/stack v1.8.0 // indirect
	github.com/google/go-cmp v0.3.1 // indirect
	github.com/hashicorp/golang-lru v0.5.3 // indirect
	github.com/jpillora/backoff v0.0.0-20180909062703-3050d21c67d7 // indirect
	github.com/mwitkow/go-conntrack v0.0.0-20190716064945-2f068394615f // indirect
	github.com/nats-io/jwt v0.2.12 // indirect
	github.com/nats-io/nats-server/v2 v2.0.2 // indirect
	github.com/nats-io/nats.go v1.8.1
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.0
	github.com/prometheus/client_golang v1.1.0 // indirect
	github.com/prometheus/client_model v0.0.0-20190812154241-14fe0d1b01d4 // indirect
	github.com/prometheus/common v0.6.0
	github.com/prometheus/prometheus v2.11.0+incompatible // pinned
	golang.org/x/net v0.0.0-20190724013045-ca1201d0de80 // indirect
	golang.org/x/sys v0.0.0-20190813064441-fde4db37ae7a // indirect
	google.golang.org/grpc v1.22.2 // indirect
	gopkg.in/yaml.v2 v2.2.2
)

replace github.com/prometheus/common => github.com/prometheus/common v0.5.0

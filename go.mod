module github.com/webitel/webitel-wfm

go 1.23.4

replace github.com/armon/go-metrics v0.5.3 => github.com/hashicorp/go-metrics v0.5.3

require (
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.0-20241127180247-a33202765966.1
	buf.build/gen/go/webitel/engine/grpc/go v1.5.1-20241204053309-7eac59c4b6c7.1
	buf.build/gen/go/webitel/engine/protocolbuffers/go v1.36.0-20241204053309-7eac59c4b6c7.1
	buf.build/gen/go/webitel/logger/grpc/go v1.5.1-20240911114117-1d910a772b4f.1
	buf.build/gen/go/webitel/logger/protocolbuffers/go v1.36.0-20240911114117-1d910a772b4f.1
	github.com/VictoriaMetrics/fastcache v1.12.2
	github.com/bufbuild/protovalidate-go v0.8.0
	github.com/georgysavva/scany/v2 v2.1.3
	github.com/google/go-cmp v0.6.0
	github.com/google/uuid v1.6.0
	github.com/google/wire v0.6.0
	github.com/huandu/go-sqlbuilder v1.33.1
	github.com/jackc/pgerrcode v0.0.0-20240316143900-6e2875d9b438
	github.com/jackc/pgx/v5 v5.7.1
	github.com/pashagolub/pgxmock/v4 v4.3.0
	github.com/pressly/goose/v3 v3.24.0
	github.com/rabbitmq/amqp091-go v1.10.0
	github.com/stretchr/testify v1.10.0
	github.com/urfave/cli/v2 v2.27.5
	github.com/webitel/engine v0.0.0-20240620111912-86e1807cf401
	github.com/webitel/webitel-go-kit v0.0.20
	github.com/webitel/webitel-go-kit/logging/wlog v0.0.0-20241119150325-b21de048f596
	go.opentelemetry.io/otel v1.33.0
	go.opentelemetry.io/otel/trace v1.33.0
	golang.org/x/sync v0.10.0
	google.golang.org/genproto/googleapis/api v0.0.0-20241219192143-6b3ec007d9bb
	google.golang.org/grpc v1.69.2
	google.golang.org/protobuf v1.36.0
)

require (
	buf.build/gen/go/grpc-ecosystem/grpc-gateway/protocolbuffers/go v1.36.0-20240617172850-a48fcebcf8f1.1 // indirect
	buf.build/gen/go/webitel/webitel-go/grpc/go v1.5.1-20241211101732-846cb7ad222f.1 // indirect
	buf.build/gen/go/webitel/webitel-go/protocolbuffers/go v1.36.0-20241211101732-846cb7ad222f.1 // indirect
	cel.dev/expr v0.19.1 // indirect
	github.com/antlr4-go/antlr/v4 v4.13.1 // indirect
	github.com/armon/go-metrics v0.5.3 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.6 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/fatih/color v1.18.0 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/gofrs/flock v0.12.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/cel-go v0.22.1 // indirect
	github.com/grafana/otel-profiling-go v0.5.1 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.25.1 // indirect
	github.com/hashicorp/consul/api v1.30.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-hclog v1.6.3 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/golang-lru v1.0.2 // indirect
	github.com/hashicorp/serf v0.10.1 // indirect
	github.com/huandu/xstrings v1.5.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mfridman/interpolate v0.0.2 // indirect
	github.com/miekg/dns v1.1.43 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/sethvargo/go-retry v0.3.0 // indirect
	github.com/stoewer/go-strcase v1.3.0 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/webitel/wlog v0.0.0-20240909100805-822697e17a45 // indirect
	github.com/xrash/smetrics v0.0.0-20240521201337-686a1a2994c1 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/contrib/bridges/otelzap v0.5.0 // indirect
	go.opentelemetry.io/contrib/propagators/jaeger v1.33.0 // indirect
	go.opentelemetry.io/contrib/samplers/jaegerremote v0.27.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.33.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.33.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.33.0 // indirect
	go.opentelemetry.io/otel/log v0.6.0 // indirect
	go.opentelemetry.io/otel/metric v1.33.0 // indirect
	go.opentelemetry.io/otel/sdk v1.33.0 // indirect
	go.opentelemetry.io/proto/otlp v1.4.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/crypto v0.31.0 // indirect
	golang.org/x/exp v0.0.0-20241217172543-b2144cdd0a67 // indirect
	golang.org/x/net v0.33.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20241219192143-6b3ec007d9bb // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

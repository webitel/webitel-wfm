module github.com/webitel/webitel-wfm

go 1.22.5

require (
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.34.2-20240508200655-46a4cf4ba109.2
	buf.build/gen/go/webitel/engine/grpc/go v1.4.0-20240704112916-e6cc7cc3a9e3.2
	buf.build/gen/go/webitel/engine/protocolbuffers/go v1.34.2-20240704112916-e6cc7cc3a9e3.2
	buf.build/gen/go/webitel/logger/grpc/go v1.3.0-20240404135439-f6c7830c29dd.2
	buf.build/gen/go/webitel/logger/protocolbuffers/go v1.33.0-20240404135439-f6c7830c29dd.1
	github.com/VictoriaMetrics/fastcache v1.12.2
	github.com/bufbuild/protovalidate-go v0.6.3
	github.com/georgysavva/scany/v2 v2.1.3
	github.com/google/go-cmp v0.6.0
	github.com/google/uuid v1.6.0
	github.com/google/wire v0.6.0
	github.com/huandu/go-sqlbuilder v1.28.1
	github.com/jackc/pgerrcode v0.0.0-20240316143900-6e2875d9b438
	github.com/jackc/pgx/v5 v5.6.0
	github.com/pashagolub/pgxmock/v4 v4.2.0
	github.com/pressly/goose/v3 v3.21.1
	github.com/rabbitmq/amqp091-go v1.10.0
	github.com/stretchr/testify v1.9.0
	github.com/urfave/cli/v2 v2.27.4
	github.com/webitel/engine v0.0.0-20240620111912-86e1807cf401
	github.com/webitel/webitel-go-kit v0.0.17
	github.com/webitel/webitel-go-kit/logging/wlog v0.0.0-20240829172959-ea097c65d185
	go.opentelemetry.io/otel v1.29.0
	go.opentelemetry.io/otel/trace v1.29.0
	golang.org/x/sync v0.8.0
	google.golang.org/genproto/googleapis/api v0.0.0-20240822170219-fc7c04adadcd
	google.golang.org/grpc v1.65.0
	google.golang.org/protobuf v1.34.2
)

require (
	buf.build/gen/go/grpc-ecosystem/grpc-gateway/protocolbuffers/go v1.34.2-20231027202514-3f42134f4c56.2 // indirect
	buf.build/gen/go/webitel/webitel-go/grpc/go v1.3.0-20240503115012-a17a535809fe.3 // indirect
	buf.build/gen/go/webitel/webitel-go/protocolbuffers/go v1.34.0-20240503115012-a17a535809fe.1 // indirect
	github.com/antlr4-go/antlr/v4 v4.13.0 // indirect
	github.com/armon/go-metrics v0.4.1 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.4 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/fatih/color v1.16.0 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/cel-go v0.20.1 // indirect
	github.com/grafana/otel-profiling-go v0.5.1 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.21.0 // indirect
	github.com/hashicorp/consul/api v1.27.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-hclog v1.6.3 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/golang-lru v1.0.2 // indirect
	github.com/hashicorp/serf v0.10.1 // indirect
	github.com/huandu/xstrings v1.4.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20231201235250-de7065d80cb9 // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mfridman/interpolate v0.0.2 // indirect
	github.com/miekg/dns v1.1.43 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/sethvargo/go-retry v0.2.4 // indirect
	github.com/stoewer/go-strcase v1.3.0 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/webitel/wlog v0.0.0-20240805104530-8e14129556dd // indirect
	github.com/xrash/smetrics v0.0.0-20240521201337-686a1a2994c1 // indirect
	go.opentelemetry.io/contrib/propagators/jaeger v1.29.0 // indirect
	go.opentelemetry.io/contrib/samplers/jaegerremote v0.23.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.29.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.28.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.28.0 // indirect
	go.opentelemetry.io/otel/log v0.4.0 // indirect
	go.opentelemetry.io/otel/metric v1.29.0 // indirect
	go.opentelemetry.io/otel/sdk v1.29.0 // indirect
	go.opentelemetry.io/proto/otlp v1.3.1 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/crypto v0.26.0 // indirect
	golang.org/x/exp v0.0.0-20240325151524-a685a6edb6d8 // indirect
	golang.org/x/net v0.28.0 // indirect
	golang.org/x/sys v0.24.0 // indirect
	golang.org/x/text v0.17.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240814211410-ddb44dafa142 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

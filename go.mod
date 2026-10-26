module github.com/webitel/webitel-wfm

go 1.26.0

replace github.com/armon/go-metrics v0.5.3 => github.com/hashicorp/go-metrics v0.5.3

require (
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.12-20260825204119-511051f7f437.2
	buf.build/gen/go/webitel/engine/grpc/go v1.6.2-20260514114320-059aef6effba.1
	buf.build/gen/go/webitel/engine/protocolbuffers/go v1.36.12-20260514114320-059aef6effba.2
	buf.build/gen/go/webitel/logger/grpc/go v1.6.2-20260416145218-c8c0564a5ed9.1
	buf.build/gen/go/webitel/logger/protocolbuffers/go v1.36.12-20260416145218-c8c0564a5ed9.2
	buf.build/gen/go/webitel/webitel-go/grpc/go v1.6.2-20260528093848-00aedb783a79.1
	buf.build/gen/go/webitel/webitel-go/protocolbuffers/go v1.36.12-20260528093848-00aedb783a79.1
	buf.build/go/protovalidate v1.4.0
	github.com/fsnotify/fsnotify v1.9.0
	github.com/georgysavva/scany/v2 v2.1.4
	github.com/google/uuid v1.6.0
	github.com/huandu/go-sqlbuilder v1.43.0
	github.com/jackc/pgerrcode v0.0.0-20250907135507-afb5586c32a6
	github.com/jackc/pgx/v5 v5.11.0
	github.com/pressly/goose/v3 v3.28.0
	github.com/spf13/pflag v1.0.10
	github.com/stretchr/testify v1.12.1
	github.com/urfave/cli/v2 v2.27.7
	github.com/webitel/webitel-go-kit/appconfig v0.0.0-20260901092450-f7cbb06aceb5
	github.com/webitel/webitel-go-kit/cmd/protoc-gen-go-webitel v0.0.0-20240829153325-0ae7f6059b52
	github.com/webitel/webitel-go-kit/infra/discovery v0.0.0-20260901092450-f7cbb06aceb5
	github.com/webitel/webitel-go-kit/infra/health v0.0.0-20260901092450-f7cbb06aceb5
	github.com/webitel/webitel-go-kit/infra/otel v0.2.0
	github.com/webitel/webitel-go-kit/infra/pgw v0.0.0-20260901092450-f7cbb06aceb5
	github.com/webitel/webitel-go-kit/infra/pubsub/rabbitmq v0.0.0-20260901092450-f7cbb06aceb5
	github.com/webitel/webitel-go-kit/infra/transport v0.0.0-20260901092450-f7cbb06aceb5
	github.com/webitel/webitel-go-kit/pkg/cache v0.0.0-20260901092450-f7cbb06aceb5
	github.com/webitel/webitel-go-kit/pkg/errors v0.1.0
	github.com/webitel/webitel-go-kit/pkg/interceptors v0.1.1
	go.opentelemetry.io/contrib/bridges/otelslog v0.17.0
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.71.0
	go.opentelemetry.io/otel v1.46.0
	go.opentelemetry.io/otel/sdk v1.46.0
	go.opentelemetry.io/otel/trace v1.46.0
	go.uber.org/fx v1.24.0
	golang.org/x/sync v0.23.0
	google.golang.org/genproto/googleapis/api v0.0.0-20260911204522-f61a6ca850bd
	google.golang.org/grpc v1.83.2
	google.golang.org/protobuf v1.36.12
)

require (
	buf.build/gen/go/grpc-ecosystem/grpc-gateway/protocolbuffers/go v1.36.12-20240502201324-7530ea77434f.1 // indirect
	cel.dev/cel-go v0.32.0 // indirect
	cel.dev/expr v0.25.3 // indirect
	github.com/antlr4-go/antlr/v4 v4.13.1 // indirect
	github.com/armon/go-metrics v0.5.3 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.7 // indirect
	github.com/dgraph-io/ristretto/v2 v2.2.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/fatih/color v1.18.0 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/gofrs/flock v0.12.1 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.28.0 // indirect
	github.com/hashicorp/consul/api v1.34.3 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-hclog v1.6.3 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/golang-lru v1.0.2 // indirect
	github.com/hashicorp/serf v0.10.1 // indirect
	github.com/huandu/go-clone v1.7.3 // indirect
	github.com/huandu/xstrings v1.5.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/mfridman/interpolate v0.0.2 // indirect
	github.com/miekg/dns v1.1.43 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/rabbitmq/amqp091-go v1.14.0 // indirect
	github.com/redis/go-redis/v9 v9.18.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/sagikazarmark/locafero v0.11.0 // indirect
	github.com/sethvargo/go-retry v0.4.0 // indirect
	github.com/sourcegraph/conc v0.3.1-0.20240121214520-5f936abd7ae8 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/spf13/viper v1.21.0 // indirect
	github.com/stretchr/objx v0.5.3 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/webitel/webitel-go-kit/pkg/safemap v0.1.1-0.20260617101709-72b6b829c7ef // indirect
	github.com/xrash/smetrics v0.0.0-20240521201337-686a1a2994c1 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/runtime v0.68.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc v0.19.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp v0.19.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.43.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v1.43.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.43.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.43.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.43.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v1.43.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.43.0 // indirect
	go.opentelemetry.io/otel/log v0.19.0 // indirect
	go.opentelemetry.io/otel/metric v1.46.0 // indirect
	go.opentelemetry.io/otel/sdk/log v0.19.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.46.0 // indirect
	go.opentelemetry.io/proto/otlp v1.10.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/dig v1.19.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/exp v0.0.0-20260824195058-e88cd73687aa // indirect
	golang.org/x/net v0.58.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260904194346-d0f1323225a4 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
)

module gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api

go 1.21

require (
	github.com/Masterminds/semver/v3 v3.2.1
	github.com/ProtonMail/go-crypto v1.0.0
	github.com/apparentlymart/go-versions v1.0.2
	github.com/avast/retry-go v3.0.0+incompatible
	github.com/avast/retry-go/v4 v4.5.1
	github.com/aws/aws-sdk-go-v2 v1.24.1
	github.com/aws/aws-sdk-go-v2/config v1.26.6
	github.com/aws/aws-sdk-go-v2/credentials v1.16.16
	github.com/aws/aws-sdk-go-v2/feature/s3/manager v1.15.14
	github.com/aws/aws-sdk-go-v2/service/ecs v1.38.0
	github.com/aws/aws-sdk-go-v2/service/eks v1.37.1
	github.com/aws/aws-sdk-go-v2/service/kms v1.27.9
	github.com/aws/aws-sdk-go-v2/service/s3 v1.48.0
	github.com/aws/aws-sdk-go-v2/service/sts v1.26.7
	github.com/aws/smithy-go v1.19.0
	github.com/bmatcuk/doublestar/v4 v4.6.1
	github.com/docker/docker v24.0.7+incompatible
	github.com/doug-martin/goqu/v9 v9.19.0
	github.com/dustin/go-humanize v1.0.1
	github.com/fatih/color v1.7.0
	github.com/go-chi/chi/v5 v5.0.11
	github.com/go-chi/cors v1.2.1
	github.com/go-git/go-git/v5 v5.11.0
	github.com/go-ozzo/ozzo-validation/v4 v4.3.0
	github.com/golang-migrate/migrate/v4 v4.17.0
	github.com/gomodule/redigo v1.8.9
	github.com/google/uuid v1.6.0
	github.com/graph-gophers/dataloader v5.0.0+incompatible
	github.com/graph-gophers/graphql-go v1.5.0
	github.com/graph-gophers/graphql-transport-ws v0.0.2
	github.com/hashicorp/go-getter v1.7.3
	github.com/hashicorp/go-retryablehttp v0.7.5
	github.com/hashicorp/go-slug v0.13.3
	github.com/hashicorp/go-tfe v1.43.0
	github.com/hashicorp/go-version v1.6.0
	github.com/hashicorp/hc-install v0.6.2
	github.com/hashicorp/hcl/v2 v2.19.1
	github.com/hashicorp/jsonapi v1.2.0
	github.com/hashicorp/terraform-config-inspect v0.0.0-20231204233900-a34142ec2a72
	github.com/hashicorp/terraform-exec v0.20.0
	github.com/hashicorp/terraform-registry-address v0.2.3
	github.com/hashicorp/terraform-svchost v0.1.1
	github.com/in-toto/in-toto-golang v0.9.0
	github.com/jackc/pgconn v1.14.1
	github.com/jackc/pgerrcode v0.0.0-20220416144525-469b46aa5efa
	github.com/jackc/pgproto3/v2 v2.3.2
	github.com/jackc/pgx/v4 v4.18.1
	github.com/lestrrat-go/jwx v1.2.28
	github.com/mitchellh/go-ps v1.0.0
	github.com/opencontainers/image-spec v1.1.0-rc6
	github.com/prometheus/client_golang v1.18.0
	github.com/prometheus/procfs v0.12.0
	github.com/qiangxue/go-env v1.0.1
	github.com/secure-systems-lab/go-securesystemslib v0.8.0
	github.com/sigstore/sigstore v1.8.1
	github.com/stretchr/testify v1.8.4
	github.com/swaggo/http-swagger v1.3.4
	github.com/zclconf/go-cty v1.14.2
	gitlab.com/infor-cloud/martian-cloud/tharsis/go-limiter v0.0.0-20230411193226-3247984d5abc
	gitlab.com/infor-cloud/martian-cloud/tharsis/go-redisstore v0.0.0-20230411194747-eb2ce440abb8
	gitlab.com/infor-cloud/martian-cloud/tharsis/graphql-query-complexity v0.2.0
	gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go v0.31.0
	go.opentelemetry.io/contrib/propagators/aws v1.22.0
	go.opentelemetry.io/otel v1.22.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.22.0
	go.opentelemetry.io/otel/sdk v1.22.0
	go.opentelemetry.io/otel/trace v1.22.0
	go.uber.org/zap v1.26.0
	google.golang.org/grpc v1.61.0
	gopkg.in/yaml.v2 v2.4.0
	gotest.tools/v3 v3.5.1
	k8s.io/api v0.29.1
	k8s.io/apimachinery v0.29.1
	k8s.io/client-go v0.29.1
)

require (
	cloud.google.com/go v0.112.0 // indirect
	cloud.google.com/go/compute v1.23.3 // indirect
	cloud.google.com/go/compute/metadata v0.2.3 // indirect
	cloud.google.com/go/iam v1.1.5 // indirect
	cloud.google.com/go/storage v1.36.0 // indirect
	github.com/KyleBanks/depth v1.2.1 // indirect
	github.com/Microsoft/go-winio v0.6.1 // indirect
	github.com/agext/levenshtein v1.2.3 // indirect
	github.com/apparentlymart/go-textseg/v15 v15.0.0 // indirect
	github.com/asaskevich/govalidator v0.0.0-20230301143203-a9d515a09cc2 // indirect
	github.com/aws/aws-sdk-go v1.50.2 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.5.4 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.14.11 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.2.10 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.5.10 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.7.3 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.2.10 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.10.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.2.10 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.10.10 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.16.10 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.18.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.21.7 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bgentry/go-netrc v0.0.0-20140422174119-9fd32a8b3d3d // indirect
	github.com/cenkalti/backoff/v4 v4.2.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/cloudflare/circl v1.3.7 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.2.0 // indirect
	github.com/docker/distribution v2.8.2+incompatible // indirect
	github.com/docker/go-connections v0.5.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/emicklei/go-restful/v3 v3.11.2 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/jsonpointer v0.20.2 // indirect
	github.com/go-openapi/jsonreference v0.20.4 // indirect
	github.com/go-openapi/spec v0.20.14 // indirect
	github.com/go-openapi/swag v0.22.8 // indirect
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/gnostic-models v0.6.9-0.20230804172637-c7be7c783f49 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/google/go-containerregistry v0.18.0 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/s2a-go v0.1.7 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.2 // indirect
	github.com/googleapis/gax-go/v2 v2.12.0 // indirect
	github.com/gorilla/websocket v1.5.1 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.19.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-safetemp v1.0.0 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/hashicorp/terraform-json v0.21.0 // indirect
	github.com/hasura/go-graphql-client v0.9.1 // indirect
	github.com/jackc/chunkreader/v2 v2.0.1 // indirect
	github.com/jackc/pgio v1.0.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20231201235250-de7065d80cb9 // indirect
	github.com/jackc/pgtype v1.14.1 // indirect
	github.com/jackc/puddle v1.3.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.17.4 // indirect
	github.com/lestrrat-go/backoff/v2 v2.0.8 // indirect
	github.com/lestrrat-go/blackmagic v1.0.2 // indirect
	github.com/lestrrat-go/httpcc v1.0.1 // indirect
	github.com/lestrrat-go/iter v1.0.2 // indirect
	github.com/lestrrat-go/option v1.0.1 // indirect
	github.com/letsencrypt/boulder v0.0.0-20240123225737-adb9673c3748 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-colorable v0.1.6 // indirect
	github.com/mattn/go-isatty v0.0.16 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/go-testing-interface v1.14.1 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/pjbgf/sha1cd v0.3.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.5.0 // indirect
	github.com/prometheus/common v0.46.0 // indirect
	github.com/shibumi/go-pathspec v1.3.0 // indirect
	github.com/stretchr/objx v0.5.1 // indirect
	github.com/swaggo/files v1.0.1 // indirect
	github.com/swaggo/swag v1.16.2 // indirect
	github.com/titanous/rocacheck v0.0.0-20171023193734-afe73141d399 // indirect
	github.com/ulikunitz/xz v0.5.11 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.46.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.46.1 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.22.0 // indirect
	go.opentelemetry.io/otel/metric v1.22.0 // indirect
	go.opentelemetry.io/proto/otlp v1.1.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/crypto v0.18.0 // indirect
	golang.org/x/mod v0.14.0 // indirect
	golang.org/x/net v0.20.0 // indirect
	golang.org/x/oauth2 v0.16.0 // indirect
	golang.org/x/sync v0.6.0 // indirect
	golang.org/x/sys v0.16.0 // indirect
	golang.org/x/term v0.16.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	golang.org/x/time v0.5.0 // indirect
	golang.org/x/tools v0.17.0 // indirect
	golang.org/x/xerrors v0.0.0-20231012003039-104605ab7028 // indirect
	google.golang.org/api v0.157.0 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/genproto v0.0.0-20240123012728-ef4313101c80 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240123012728-ef4313101c80 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240123012728-ef4313101c80 // indirect
	google.golang.org/protobuf v1.32.0 // indirect
	gopkg.in/go-jose/go-jose.v2 v2.6.2 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/klog/v2 v2.120.1 // indirect
	k8s.io/kube-openapi v0.0.0-20240117194847-208609032b15 // indirect
	k8s.io/utils v0.0.0-20240102154912-e7106e64919e // indirect
	nhooyr.io/websocket v1.8.10 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.4.1 // indirect
	sigs.k8s.io/yaml v1.4.0 // indirect
)

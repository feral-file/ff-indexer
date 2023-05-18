module github.com/bitmark-inc/nft-indexer

go 1.18

require (
	github.com/99designs/gqlgen v0.17.24
	github.com/aws/aws-sdk-go v1.43.9
	github.com/bitmark-inc/account-vault-ethereum v0.2.3
	github.com/bitmark-inc/autonomy-account v0.0.5-1
	github.com/bitmark-inc/autonomy-notification v0.0.3
	github.com/bitmark-inc/bitmark-sdk-go v0.3.0
	github.com/bitmark-inc/config-loader v0.0.5
	github.com/bitmark-inc/tzkt-go v0.0.5
	github.com/chromedp/chromedp v0.8.6
	github.com/cloudflare/cloudflare-go v0.62.0
	github.com/ethereum/go-ethereum v1.10.23
	github.com/fatih/structs v1.1.0
	github.com/gabriel-vasile/mimetype v1.4.1
	github.com/getsentry/sentry-go v0.13.0
	github.com/gin-contrib/cors v1.3.1
	github.com/gin-gonic/gin v1.8.1
	github.com/hasura/go-graphql-client v0.8.0
	github.com/lib/pq v1.10.2
	github.com/sirupsen/logrus v1.9.0
	github.com/spf13/viper v1.12.0
	github.com/stretchr/testify v1.8.2
	github.com/uber-go/tally v3.4.2+incompatible
	github.com/vektah/gqlparser/v2 v2.5.1
	github.com/wealdtech/go-ens/v3 v3.5.2
	go.mongodb.org/mongo-driver v1.10.1
	go.uber.org/cadence v0.18.3
	go.uber.org/yarpc v1.58.0
	go.uber.org/zap v1.17.0
	golang.org/x/crypto v0.0.0-20220824171710-5757bc0c5503
	golang.org/x/net v0.7.0
	golang.org/x/sync v0.0.0-20220819030929-7fc1605a5dde
	google.golang.org/grpc v1.46.2
	google.golang.org/protobuf v1.28.1
	gorm.io/driver/postgres v1.3.1
	gorm.io/gorm v1.23.3
)

require (
	github.com/agnivade/levenshtein v1.1.1 // indirect
	github.com/aws/aws-sdk-go-v2 v1.17.5 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.18.14 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.13.14 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.12.23 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.1.29 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.4.23 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.3.30 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.9.23 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssm v1.35.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.12.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.14.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.18.4 // indirect
	github.com/aws/smithy-go v1.13.5 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.2 // indirect
	github.com/decred/dcrd/dcrec/secp256k1 v1.0.3 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v2 v2.0.0 // indirect
	github.com/hashicorp/golang-lru v0.5.5-0.20210104140557-80c98217689d // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/urfave/cli/v2 v2.24.4 // indirect
	github.com/xrash/smetrics v0.0.0-20201216005158-039620a65673 // indirect
)

replace github.com/apache/thrift => github.com/apache/thrift v0.0.0-20190309152529-a9b748bb0e02

require (
	blockwatch.cc/tzgo v1.14.2
	github.com/BurntSushi/toml v1.2.1 // indirect
	github.com/anmitsu/go-shlex v0.0.0-20161002113705-648efa622239 // indirect
	github.com/apache/thrift v0.0.0-20161221203622-b2a4d4ae21c7 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bitmark-inc/bitmarkd v0.13.2 // indirect
	github.com/bitmark-inc/go-utils v0.0.0-20220509034903-59f3369237d3 // indirect
	github.com/bitmark-inc/logger v0.3.5 // indirect
	github.com/bitmark-inc/traceutils v0.1.0
	github.com/btcsuite/btcd v0.22.0-beta // indirect
	github.com/btcsuite/btcd/btcec/v2 v2.2.0 // indirect
	github.com/btcsuite/btcutil v1.0.3-0.20201208143702-a53e38424cce // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/chromedp/cdproto v0.0.0-20220924210414-0e3390be1777 // indirect
	github.com/chromedp/sysutil v1.0.0 // indirect
	github.com/cihub/seelog v0.0.0-20170130134532-f561c5e57575 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/deckarep/golang-set v1.8.0 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.0.1 // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/facebookgo/clock v0.0.0-20150410010913-600d898af40a // indirect
	github.com/fatih/structtag v1.2.0 // indirect
	github.com/fsnotify/fsnotify v1.5.4 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/go-playground/locales v0.14.0 // indirect
	github.com/go-playground/universal-translator v0.18.0 // indirect
	github.com/go-playground/validator/v10 v10.11.0 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/gobwas/httphead v0.1.0 // indirect
	github.com/gobwas/pool v0.2.1 // indirect
	github.com/gobwas/ws v1.1.0 // indirect
	github.com/goccy/go-json v0.9.11 // indirect
	github.com/golang/mock v1.5.0 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.2 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/ipfs/go-cid v0.1.0 // indirect
	github.com/jackc/chunkreader/v2 v2.0.1 // indirect
	github.com/jackc/pgconn v1.10.1 // indirect
	github.com/jackc/pgio v1.0.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgproto3/v2 v2.2.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20200714003250-2b9c44734f2b // indirect
	github.com/jackc/pgtype v1.9.1 // indirect
	github.com/jackc/pgx/v4 v4.14.1 // indirect
	github.com/jessevdk/go-flags v1.4.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.4 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kisielk/errcheck v1.6.1 // indirect
	github.com/klauspost/compress v1.15.9 // indirect
	github.com/klauspost/cpuid/v2 v2.0.11 // indirect
	github.com/leodido/go-urn v1.2.1 // indirect
	github.com/magiconair/properties v1.8.6 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-isatty v0.0.16 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/miguelmota/go-ethereum-hdwallet v0.1.1 // indirect
	github.com/minio/blake2b-simd v0.0.0-20160723061019-3f5f724cb5b1 // indirect
	github.com/minio/sha256-simd v1.0.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/montanaflynn/stats v0.6.6 // indirect
	github.com/mr-tron/base58 v1.2.0 // indirect
	github.com/multiformats/go-base32 v0.0.4 // indirect
	github.com/multiformats/go-base36 v0.1.0 // indirect
	github.com/multiformats/go-multibase v0.0.3 // indirect
	github.com/multiformats/go-multihash v0.1.0 // indirect
	github.com/multiformats/go-varint v0.0.6 // indirect
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/pborman/uuid v1.2.0 // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/pelletier/go-toml/v2 v2.0.3 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_golang v1.12.1 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.32.1 // indirect
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/rjeczalik/notify v0.9.2 // indirect
	github.com/robfig/cron v1.2.0 // indirect
	github.com/shirou/gopsutil v3.21.11+incompatible // indirect
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	github.com/spf13/afero v1.9.2 // indirect
	github.com/spf13/cast v1.5.0 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/stretchr/objx v0.5.0 // indirect
	github.com/subosito/gotenv v1.4.1 // indirect
	github.com/tklauser/go-sysconf v0.3.9 // indirect
	github.com/tklauser/numcpus v0.4.0 // indirect
	github.com/twmb/murmur3 v1.1.6 // indirect
	github.com/tyler-smith/go-bip39 v1.1.0 // indirect
	github.com/uber-go/mapdecode v1.0.0 // indirect
	github.com/uber/tchannel-go v1.22.0 // indirect
	github.com/ugorji/go/codec v1.2.7 // indirect
	github.com/wealdtech/go-multicodec v1.4.0 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.1 // indirect
	github.com/xdg-go/stringprep v1.0.3 // indirect
	github.com/youmark/pkcs8 v0.0.0-20201027041543-1326539a0a0a // indirect
	github.com/yusufpapurcu/wmi v1.2.2 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	go.uber.org/net/metrics v1.3.0 // indirect
	go.uber.org/thriftrw v1.29.2 // indirect
	golang.org/x/exp/typeparams v0.0.0-20220613132600-b0d781184e0d // indirect
	golang.org/x/lint v0.0.0-20210508222113-6edffad5e616 // indirect
	golang.org/x/mod v0.6.0-dev.0.20220419223038-86c51ed26bb4 // indirect
	golang.org/x/sys v0.5.0 // indirect
	golang.org/x/text v0.7.0 // indirect
	golang.org/x/time v0.3.0 // indirect
	golang.org/x/tools v0.1.12 // indirect
	google.golang.org/genproto v0.0.0-20220519153652-3a47de7e79bd // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/natefinch/npipe.v2 v2.0.0-20160621034901-c1b8fa8bdcce // indirect
	gopkg.in/square/go-jose.v2 v2.6.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1
	honnef.co/go/tools v0.3.2 // indirect
	lukechampine.com/blake3 v1.1.7 // indirect
	nhooyr.io/websocket v1.8.7 // indirect
)

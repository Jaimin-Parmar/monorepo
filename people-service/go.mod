module people-service

go 1.19

replace (
	people-service/proto/account => /home/admin1/go/sidekiq/monorepo/proto/account/v1
	people-service/proto/authentication => /home/admin1/go/sidekiq/monorepo/proto/authentication/v1
)

require (
	github.com/SherClockHolmes/webpush-go v1.3.0
	github.com/aws/aws-sdk-go v1.50.2
	github.com/go-redis/redis/v7 v7.4.1
	github.com/go-sql-driver/mysql v1.7.1
	github.com/golang/protobuf v1.5.3
	github.com/google/uuid v1.4.0
	github.com/gorilla/handlers v1.5.2
	github.com/gorilla/mux v1.8.1
	github.com/jmoiron/sqlx v1.3.5
	github.com/lithammer/fuzzysearch v1.1.8
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.9.3
	github.com/skip2/go-qrcode v0.0.0-20200617195104-da1b6568686e
	github.com/spf13/cobra v1.8.0
	github.com/spf13/viper v1.18.2
	go.mongodb.org/mongo-driver v1.13.1
	golang.org/x/crypto v0.16.0
	golang.org/x/sys v0.15.0
	golang.org/x/text v0.14.0
	google.golang.org/grpc v1.61.0
	people-service/proto/account v0.0.0-00010101000000-000000000000
	people-service/proto/authentication v0.0.0-00010101000000-000000000000
)

require (
	github.com/felixge/httpsnoop v1.0.3 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/golang-jwt/jwt v3.2.2+incompatible // indirect
	github.com/golang/snappy v0.0.1 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/klauspost/compress v1.17.0 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/montanaflynn/stats v0.0.0-20171201202039-1bf9dbcd8cbe // indirect
	github.com/pelletier/go-toml/v2 v2.1.0 // indirect
	github.com/sagikazarmark/locafero v0.4.0 // indirect
	github.com/sagikazarmark/slog-shim v0.1.0 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.11.0 // indirect
	github.com/spf13/cast v1.6.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/youmark/pkcs8 v0.0.0-20181117223130-1be2e3e5546d // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.9.0 // indirect
	golang.org/x/exp v0.0.0-20230905200255-921286631fa9 // indirect
	golang.org/x/net v0.19.0 // indirect
	golang.org/x/sync v0.5.0 // indirect
	golang.org/x/term v0.15.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20231120223509-83a465c0220f // indirect
	google.golang.org/protobuf v1.32.0 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

module github.com/chremoas/chremoas-ng

go 1.16

require (
	github.com/Masterminds/squirrel v1.5.0
	github.com/antihax/goesi v0.0.0-20210313233113-a4c71dbef361
	github.com/astaxie/beego v1.12.3
	github.com/bwmarrin/discordgo v0.23.2
	github.com/bwmarrin/disgord v0.0.0-20200407171809-1fe97f20c0de
	github.com/chremoas/auth-srv v1.3.2
	github.com/dimfeld/httptreemux v5.0.1+incompatible
	github.com/elazarl/go-bindata-assetfs v1.0.0
	github.com/fsnotify/fsnotify v1.4.9 // indirect
	github.com/gogo/protobuf v1.3.1 // indirect
	github.com/golang/protobuf v1.4.3
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79
	github.com/jinzhu/gorm v1.9.16
	github.com/jmoiron/sqlx v1.3.1
	github.com/lib/pq v1.10.0
	github.com/micro/go-micro v1.18.0
	github.com/nsqio/go-nsq v1.0.8
	github.com/pelletier/go-toml v1.8.1 // indirect
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.7.1
	go.uber.org/zap v1.16.0
)

replace google.golang.org/grpc => google.golang.org/grpc v1.29.1

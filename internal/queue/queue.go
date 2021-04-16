package queue

import (
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/bwmarrin/discordgo"
	"github.com/chremoas/chremoas-ng/internal/common"
	"github.com/chremoas/chremoas-ng/internal/discord/members"
	"github.com/chremoas/chremoas-ng/internal/discord/roles"
	"github.com/nsqio/go-nsq"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type Queue struct {
	queue     *nsq.Config
	queueAddr string
	session   *discordgo.Session
	logger    *zap.SugaredLogger
	db        *sq.StatementBuilderType
}

func New(session *discordgo.Session, logger *zap.SugaredLogger, db *sq.StatementBuilderType) Queue {
	queue := nsq.NewConfig()
	queueAddr := fmt.Sprintf("%s:%d", viper.GetString("queue.host"), viper.GetInt("queue.port"))

	return Queue{queue: queue, queueAddr: queueAddr, session: session, logger: logger, db: db}
}

func (queue Queue) ProducerQueue() (*nsq.Producer, error) {
	// Setup NSQ producer for the commands to use
	producer, err := nsq.NewProducer(queue.queueAddr, queue.queue)
	if err != nil {
		return nil, err
	}
	if err = producer.Ping(); err != nil {
		return nil, err
	}

	return producer, nil
}

func (queue Queue) RoleConsumer() (*nsq.Consumer, error) {
	// Setup the Role Consumer handler
	roleConsumer, err := nsq.NewConsumer(common.GetTopic("role"), "discordGateway", queue.queue)
	if err != nil {
		return nil, err
	}

	// Add Role NSQ handler
	roleConsumer.AddHandler(roles.New(queue.logger, queue.session, queue.db))

	err = roleConsumer.ConnectToNSQLookupd("10.42.1.30:4161")
	if err != nil {
		return nil, err
	}

	return roleConsumer, nil
}

func (queue Queue) MemberConsumer() (*nsq.Consumer, error) {
	// Setup the Member Consumer handler
	memberConsumer, err := nsq.NewConsumer(common.GetTopic("member"), "discordGateway", queue.queue)
	if err != nil {
		return nil, err
	}

	// Add Role NSQ handler
	memberConsumer.AddHandler(members.New(queue.logger, queue.session, queue.db))

	err = memberConsumer.ConnectToNSQLookupd("10.42.1.30:4161")
	if err != nil {
		return nil, err
	}

	return memberConsumer, err
}

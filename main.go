// Declare this file to be part of the main package so it can be compiled into
// an executable.
package main

// Import all Go packages required for this file.
import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	sq "github.com/Masterminds/squirrel"
	"github.com/bwmarrin/discordgo"
	"github.com/bwmarrin/disgord/x/mux"
	"github.com/chremoas/chremoas-ng/internal/discord/roles"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/nsqio/go-nsq"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/chremoas/chremoas-ng/internal/commands"
	"github.com/chremoas/chremoas-ng/internal/config"
	"github.com/chremoas/chremoas-ng/internal/log"
)

// Version is a constant that stores the Disgord version information.
const Version = "v0.0.0"

func main() {
	var (
		err    error
		logger = log.New()
	)

	// Get the config file name if we're not using consul
	flag.String("configFile", "chremoas.yaml", "configuration file name")
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	err = viper.BindPFlags(pflag.CommandLine)
	if err != nil {
		panic(err)
	}

	_, err = config.New(viper.GetString("configFile"))
	if err != nil {
		panic(err)
	}

	Session, err := discordgo.New("Bot " + viper.GetString("bot.token"))
	if err != nil {
		logger.Fatalf("Error starting session: %s", err)
	}

	Session.Identify.Intents = discordgo.IntentsGuildMessages

	Router := mux.New()
	Router.Prefix = "!"

	// Register the mux OnMessageCreate handler that listens for and processes
	// all messages received.
	Session.AddHandler(Router.OnMessageCreate)

	// Register the build-in help command.
	_, err = Router.Route("help", "Display this message.", Router.Help)
	if err != nil {
		panic("Can't load help router something is very, very wrong")
	}

	// Print out a fancy logo!
	fmt.Printf(`
    _________ .__                                       
    \_   ___ \|  |_________  _____   _________    ______
    /    \  \/|  |  \_  __ \/     \ /  _ \__  \  /  ___/
    \     \___|   Y  \  | \/  Y Y  (  <_> ) __ \_\___ \ 
     \______  /___|  /__|  |__|_|  /\____(____  /____  >
            \/     \/            \/ %-9s \/     \/`+"\n\n", Version)

	// Setup DB connection
	db, err := NewDB(logger)
	if err != nil {
		logger.Fatalf("error opening connection to PostgreSQL: %s\n", err)
	}

	// Setup NSQ
	queue := nsq.NewConfig()
	queueAddr := fmt.Sprintf("%s:%d", viper.GetString("queue.host"), viper.GetInt("queue.port"))

	// Setup NSQ producer for the commands to use
	producer, err := nsq.NewProducer(queueAddr, queue)
	if err != nil {
		logger.Fatalf("error setting up queue producer: %s\n", err)
	}
	defer producer.Stop()
	if err = producer.Ping(); err != nil {
		logger.Fatalf("error connecting to the queue: %s\n", err)
	}

	// Setup the Consumer handlers
	topic := fmt.Sprintf("%s-discord.role", viper.GetString("namespace"))
	roleConsumer, err := nsq.NewConsumer(topic, "discordGateway", queue)
	if err != nil {
		logger.Fatalf("error setting up queue consumer: %s\n", err)
	}
	defer roleConsumer.Stop()

	// Add NSQ handlers
	roleConsumer.AddHandler(roles.New(logger, Session, db))

	err = roleConsumer.ConnectToNSQLookupd("10.42.1.30:4161")
	if err != nil {
		logger.Fatal(err)
	}

	c := commands.New(logger, db, producer)

	commandList := []struct {
		command string
		desc    string
		handler mux.HandlerFunc
	}{
		{"ping", "Sends a Pong", c.Ping},
		{"pong", "Sends a Ping", c.Pong},
		{"role", "Manages Roles", c.Role},
	}

	for _, route := range commandList {
		_, err = Router.Route(route.command, route.desc, route.handler)
		if err != nil {
			logger.Warnf("Failed to load route: %s", route.command)
		}
	}

	// Open a websocket connection to Discord
	err = Session.Open()
	if err != nil {
		logger.Fatalf("error opening connection to Discord: %s\n", err)
	}

	// Wait for a CTRL-C
	logger.Info(`Now running. Press CTRL-C to exit.`)
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Clean up
	Session.Close()

	// Exit Normally.
}

func NewDB(logger *zap.SugaredLogger) (*sq.StatementBuilderType, error) {
	var (
		err       error
		namespace = viper.GetString("namespace")
	)

	//ignoredRoles = viper.GetStringSlice("bot.ignoredRoles")

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s",
		viper.GetString("database.host"),
		viper.GetInt("database.port"),
		viper.GetString("database.username"),
		viper.GetString("database.password"),
		viper.GetString("database.roledb"),
	)

	ldb, err := sqlx.Connect(viper.GetString("database.driver"), dsn)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	err = ldb.Ping()
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	dbCache := sq.NewStmtCache(ldb)
	db := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).RunWith(dbCache)

	// Ensure required permissions exist in the database
	var (
		requiredPermissions = map[string]string{
			"role_admins": "Role Admins",
			"sig_admins":  "SIG Admins",
		}
		id int
	)

	for k, v := range requiredPermissions {
		err = db.Select("id").
			From("permissions").
			Where(sq.Eq{"name": k}).
			Where(sq.Eq{"namespace": namespace}).
			QueryRow().Scan(&id)

		switch err {
		case nil:
			logger.Infof("%s (%d) found", k, id)
		case sql.ErrNoRows:
			logger.Infof("%s NOT found, creating", k)
			err = db.Insert("permissions").
				Columns("namespace", "name", "description").
				Values(namespace, k, v).
				Suffix("RETURNING \"id\"").
				QueryRow().Scan(&id)
			if err != nil {
				logger.Error(err)
				return nil, err
			}
		default:
			logger.Error(err)
			return nil, err
		}
	}

	return &db, nil
}

// Declare this file to be part of the main package so it can be compiled into
// an executable.
package main

// Import all Go packages required for this file.
import (
	"context"
	"database/sql"
	_ "expvar"
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/bwmarrin/discordgo"
	"github.com/bwmarrin/disgord/x/mux"
	"github.com/chremoas/chremoas-ng/internal/auth/web"
	esi_poller "github.com/chremoas/chremoas-ng/internal/esi-poller"
	queue2 "github.com/chremoas/chremoas-ng/internal/queue"
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

	// Print out a fancy logo!
	fmt.Printf(`
    _________ .__                                       
    \_   ___ \|  |_________  _____   _________    ______
    /    \  \/|  |  \_  __ \/     \ /  _ \__  \  /  ___/
    \     \___|   Y  \  | \/  Y Y  (  <_> ) __ \_\___ \ 
     \______  /___|  /__|  |__|_|  /\____(____  /____  >
            \/     \/            \/ %-9s \/     \/`+"\n\n", Version)

	// =========================================================================
	// Setup the configuration
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

	// =========================================================================
	// Start Debug Service
	//
	// /debug/pprof - Added to the default mux by importing the net/http/pprof package.
	// /debug/vars - Added to the default mux by importing the expvar package.
	//
	// Not concerned with shutting this down when the application is shutdown.

	logger.Info("main: Initializing debugging support")

	go func() {
		logger.Infof("main: Debug Listening %s", "0.0.0.0:4000")
		if err = http.ListenAndServe("0.0.0.0:4000", http.DefaultServeMux); err != nil {
			logger.Errorf("main: Debug Listener closed: %v", err)
		}
	}()

	// =========================================================================
	// Setup DB connection

	db, err := NewDB(logger)
	if err != nil {
		logger.Fatalf("error opening connection to PostgreSQL: %s\n", err)
	}

	// =========================================================================
	// Start the discord session

	session, err := discordgo.New("Bot " + viper.GetString("bot.token"))
	if err != nil {
		logger.Fatalf("Error starting session: %s", err)
	}

	session.Identify.Intents = discordgo.IntentsGuildMessages

	Router := mux.New()
	Router.Prefix = "!"

	// Register the mux OnMessageCreate handler that listens for and processes
	// all messages received.
	session.AddHandler(Router.OnMessageCreate)

	// Register the build-in help command.
	_, err = Router.Route("help", "Display this message.", Router.Help)
	if err != nil {
		panic("Can't load help router something is very, very wrong")
	}

	// =========================================================================
	// Setup NSQ

	workers := queue2.New(session, logger, nsq.LogLevelError, db)

	// Setup NSQ producer for the commands to use
	producer, err := workers.ProducerQueue()
	if err != nil {
		logger.Fatalf("error setting up producer queue: %s\n", err)
	}
	defer producer.Stop()

	// Setup the Role Consumer handler
	roleConsumer, err := workers.RoleConsumer()
	if err != nil {
		logger.Fatalf("error setting up role queue consumer: %s\n", err)
	}
	defer roleConsumer.Stop()

	// Setup the Member Consumer handler
	memberConsumer, err := workers.MemberConsumer()
	if err != nil {
		logger.Fatalf("error setting up member queue consumer: %s\n", err)
	}
	defer memberConsumer.Stop()

	// =========================================================================
	// Setup commands
	c := commands.New(logger, db, producer, session)

	commandList := []struct {
		command string
		desc    string
		handler mux.HandlerFunc
	}{
		{"ping", "Sends a Pong", c.Ping},
		{"pong", "Sends a Ping", c.Pong},
		{"role", "Manages Roles", c.Role},
		{"sig", "Manages Sigs", c.Sig},
		{"filter", "Manages Filters", c.Filter},
		{"perms", "Manages Permissions", c.Perms},
		{"auth", "Manages Permissions", c.Auth},
	}

	for _, route := range commandList {
		_, err = Router.Route(route.command, route.desc, route.handler)
		if err != nil {
			logger.Warnf("Failed to load route: %s", route.command)
		}
	}

	// Open a websocket connection to Discord
	err = session.Open()
	if err != nil {
		logger.Fatalf("error opening connection to Discord: %s\n", err)
	}

	// =========================================================================
	// Start auth-web Service

	logger.Info("main: Initializing auth-web support")

	// Make a channel to listen for an interrupt or terminate signal from the OS.
	// Use a buffered channel because the signal package requires it.
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)

	authWeb, err := web.New(logger, db, producer)
	if err != nil {
		logger.Fatalf("Error starting authWeb: %s", err)
	}

	api := http.Server{
		Addr:         "0.0.0.0:3100",
		Handler:      authWeb.Auth(),
		ReadTimeout:  time.Second * 5,
		WriteTimeout: time.Second * 5,
	}

	// Make a channel to listen for errors coming from the listener. Use a
	// buffered channel so the goroutine can exit if we don't collect this error.
	serverErrors := make(chan error, 1)

	// Start the service listening for requests.
	go func() {
		logger.Infof("main: auth-web listening on %s", api.Addr)
		serverErrors <- api.ListenAndServe()
	}()

	// =========================================================================
	// Start the ESI Poller thread.
	userAgent := "chremoas-esi-srv Ramdar Chinken on TweetFleet Slack https://github.com/chremoas/chremoas-ng"
	esiPoller := esi_poller.New(userAgent, logger, db, producer)
	esiPoller.Start()
	defer esiPoller.Stop()

	// =========================================================================
	// Main loop

	logger.Info(`Now running. Press CTRL-C to exit.`)
	// Blocking main and waiting for shutdown.
	select {
	case err = <-serverErrors:
		logger.Fatal(err)

	case sig := <-shutdown:
		logger.Infof("main: %v: Start shutdown", sig)

		// Give outstanding requests a deadline for completion.
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		// Asking listener to shutdown and shed load.
		if err = api.Shutdown(ctx); err != nil {
			err = api.Close()
			if err != nil {
				logger.Fatalf("Error stopping API: %s", err)
			}
			logger.Fatalf("could not stop server gracefully: %s", err)
		}
	}

	// Clean up
	err = session.Close()
	if err != nil {
		logger.Fatalf("Error closing discord session: %s", err)
	}

	// Exit Normally.
}

func NewDB(logger *zap.SugaredLogger) (*sq.StatementBuilderType, error) {
	var (
		err error
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
			"role_admins":   "Role Admins",
			"sig_admins":    "SIG Admins",
			"server_admins": "Server Admins",
		}
		id int
	)

	for k, v := range requiredPermissions {
		err = db.Select("id").
			From("permissions").
			Where(sq.Eq{"name": k}).
			QueryRow().Scan(&id)

		switch err {
		case nil:
			logger.Infof("%s (%d) found", k, id)
		case sql.ErrNoRows:
			logger.Infof("%s NOT found, creating", k)
			err = db.Insert("permissions").
				Columns("name", "description").
				Values(k, v).
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

// Declare this file to be part of the main package so it can be compiled into
// an executable.
package main

// Import all Go packages required for this file.
import (
	"context"
	_ "expvar"
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/bwmarrin/disgord/x/mux"
	"github.com/chremoas/chremoas-ng/internal/common"
	discordMembers "github.com/chremoas/chremoas-ng/internal/discord/members"
	discordRoles "github.com/chremoas/chremoas-ng/internal/discord/roles"
	"github.com/gregjones/httpcache"
	_ "github.com/lib/pq"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/chremoas/chremoas-ng/internal/auth/web"
	"github.com/chremoas/chremoas-ng/internal/commands"
	"github.com/chremoas/chremoas-ng/internal/config"
	"github.com/chremoas/chremoas-ng/internal/database"
	esiPoller "github.com/chremoas/chremoas-ng/internal/esi-poller"
	"github.com/chremoas/chremoas-ng/internal/log"
	"github.com/chremoas/chremoas-ng/internal/queue"
)

// Version is a constant that stores the Disgord version information.
const Version = "v0.0.0"

func main() {
	environment := os.Getenv("ENVIRONMENT")
	dbg := os.Getenv("DEBUG")
	debug, err := strconv.ParseBool(dbg)
	if err != nil {
		fmt.Printf("Error parsing DEBUG '%s' is not a boolean value", dbg)
	}

	logger := log.New(environment, debug)
	logger.Info("Starting up", zap.String("env", environment), zap.Bool("debug", debug))
	dependencies := common.Dependencies{Logger: logger, Context: context.Background()}

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

	_, err = config.New(viper.GetString("configFile"), logger)
	if err != nil {
		panic(err)
	}

	// put guildID somewhere useful
	dependencies.GuildID = viper.GetString("bot.discordServerId")

	// =========================================================================
	// Start Debug Service
	//
	// /debug/pprof - Added to the default mux by importing the net/http/pprof package.
	// /debug/vars - Added to the default mux by importing the expvar package.
	//
	// Not concerned with shutting this down when the application is shutdown.

	dependencies.Logger.Info("main: Initializing debugging support")

	debugAddr := fmt.Sprintf("%s:%d", viper.GetString("net.host"), viper.GetInt("net.debugPort"))
	go func() {
		dependencies.Logger.Info("main: Debug Listener", zap.String("debugAddr", debugAddr))
		if err = http.ListenAndServe(debugAddr, http.DefaultServeMux); err != nil {
			dependencies.Logger.Error("main: Debug Listener closed", zap.Error(err))
		}
	}()

	// =========================================================================
	// Setup DB connection

	dependencies.DB, err = database.New(dependencies.Logger)
	if err != nil {
		dependencies.Logger.Fatal("error opening connection to PostgreSQL", zap.Error(err))
	}

	// =========================================================================
	// Start the discord session

	dependencies.Session, err = discordgo.New("Bot " + viper.GetString("bot.token"))
	if err != nil {
		dependencies.Logger.Fatal("Error starting session", zap.Error(err))
	}

	defer func() {
		err := dependencies.Session.Close()
		if err != nil {
			dependencies.Logger.Error("Error closing discord connection", zap.Error(err))
		}
	}()

	// Let's use a caching http client
	dependencies.Session.Client = httpcache.NewMemoryCacheTransport().Client()

	dependencies.Session.Identify.Intents = discordgo.IntentsAll

	Router := mux.New()
	Router.Prefix = "!"

	// Register the mux OnMessageCreate handler that listens for and processes
	// all messages received.
	dependencies.Session.AddHandler(Router.OnMessageCreate)

	// Register the build-in help command.
	_, err = Router.Route("help", "Display this message.", Router.Help)
	if err != nil {
		panic("Can't load help router something is very, very wrong")
	}

	// =========================================================================
	// Setup RabbitMQ
	// =========================================================================
	queueURI := fmt.Sprintf("amqp://%s:%s@%s:%d/%s",
		viper.GetString("queue.username"),
		viper.GetString("queue.password"),
		viper.GetString("queue.host"),
		viper.GetInt("queue.port"),
		viper.GetString("namespace"),
	)

	// Consumers
	members := discordMembers.New(dependencies)
	membersConsumer, err := queue.NewConsumer(queueURI, "members", "direct", "members",
		"members", "members", members.HandleMessage, dependencies.Logger)
	if err != nil {
		dependencies.Logger.Error("Error setting up members consumer", zap.Error(err))
	}
	defer func() {
		err := membersConsumer.Shutdown()
		if err != nil {
			dependencies.Logger.Error("error shutting down members consumer", zap.Error(err))
		}
	}()

	roles := discordRoles.New(dependencies)
	rolesConsumer, err := queue.NewConsumer(queueURI, "roles", "direct", "roles",
		"roles", "roles", roles.HandleMessage, dependencies.Logger)
	if err != nil {
		dependencies.Logger.Error("Error setting up members consumer", zap.Error(err))
	}
	defer func() {
		err := rolesConsumer.Shutdown()
		if err != nil {
			dependencies.Logger.Error("error shutting down roles consumer", zap.Error(err))
		}
	}()

	// Producers
	dependencies.MembersProducer, err = queue.NewPublisher(queueURI, "members", "direct",
		"members", dependencies.Logger)
	if err != nil {
		dependencies.Logger.Error("Error setting up members producer", zap.Error(err))
	}
	defer dependencies.MembersProducer.Shutdown()

	dependencies.RolesProducer, err = queue.NewPublisher(queueURI, "roles", "direct",
		"roles", dependencies.Logger)
	if err != nil {
		dependencies.Logger.Error("Error setting up roles producer", zap.Error(err))
	}
	defer dependencies.RolesProducer.Shutdown()

	// =========================================================================
	// Setup commands
	c := commands.New(dependencies)

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
		{"version", "Returns Chremoas version", c.Version},
	}

	for _, route := range commandList {
		_, err = Router.Route(route.command, route.desc, route.handler)
		if err != nil {
			dependencies.Logger.Warn("Failed to load route", zap.String("route", route.command))
		}
	}

	// Open a websocket connection to Discord
	err = dependencies.Session.Open()
	if err != nil {
		dependencies.Logger.Fatal("error opening connection to Discord", zap.Error(err))
	}

	// =========================================================================
	// Start auth-web Service

	dependencies.Logger.Info("main: Initializing auth-web support")

	// Make a channel to listen for an interrupt or terminate signal from the OS.
	// Use a buffered channel because the signal package requires it.
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)

	authWeb, err := web.New(dependencies)
	if err != nil {
		dependencies.Logger.Fatal("Error starting authWeb", zap.Error(err))
	}

	webUI := http.Server{
		Addr: fmt.Sprintf("%s:%d",
			viper.GetString("net.host"),
			viper.GetInt("net.webPort")),
		Handler:      authWeb.Auth(),
		ReadTimeout:  time.Second * 5,
		WriteTimeout: time.Second * 5,
	}

	// Make a channel to listen for errors coming from the listener. Use a
	// buffered channel so the goroutine can exit if we don't collect this error.
	serverErrors := make(chan error, 1)

	// Start the service listening for requests.
	go func() {
		dependencies.Logger.Info("main: auth-web listening", zap.String("webUI.Addr", webUI.Addr))
		serverErrors <- webUI.ListenAndServe()
	}()

	// =========================================================================
	// Start the ESI Poller thread.
	userAgent := "chremoas-ng Ramdar Chinken on TweetFleet Slack https://github.com/chremoas/chremoas-ng"
	esi := esiPoller.New(userAgent, dependencies)
	esi.Start()
	defer esi.Stop()

	// =========================================================================
	// Main loop

	dependencies.Logger.Info(`Now running. Press CTRL-C to exit.`)
	// Blocking main and waiting for shutdown.
	select {
	case err = <-serverErrors:
		dependencies.Logger.Fatal("server error", zap.Error(err))

	case sig := <-shutdown:
		dependencies.Logger.Info("main: Start shutdown", zap.Any("signal", sig))

		// Give outstanding requests a deadline for completion.
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		// Asking listener to shutdown and shed load.
		if err = webUI.Shutdown(ctx); err != nil {
			err = webUI.Close()
			if err != nil {
				dependencies.Logger.Fatal("Error stopping API", zap.Error(err))
			}
			dependencies.Logger.Fatal("could not stop server gracefully", zap.Error(err))
		}
	}

	// Exit Normally.
}

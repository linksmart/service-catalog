// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package main

import (
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/gorilla/context"
	"github.com/justinas/alice"
	_ "github.com/linksmart/go-sec/auth/keycloak/validator"
	"github.com/linksmart/go-sec/auth/validator"
	"github.com/linksmart/service-catalog/v3/catalog"
	"github.com/oleksandr/bonjour"
	"github.com/rs/cors"
	uuid "github.com/satori/go.uuid"
	"github.com/urfave/negroni"
)

var (
	confPath    = flag.String("conf", "conf/service-catalog.json", "Configuration file path")
	profile     = flag.Int("profile", 0, "Activate runtime profiling HTTP server on the given port")
	version     = flag.Bool("version", false, "Print the API version")
	Version     string // set with build flag
	BuildNumber string // set with build flag
)

const LINKSMART = `
╦   ╦ ╔╗╔ ╦╔═  ╔═╗ ╔╦╗ ╔═╗ ╦═╗ ╔╦╗
║   ║ ║║║ ╠╩╗  ╚═╗ ║║║ ╠═╣ ╠╦╝  ║
╩═╝ ╩ ╝╚╝ ╩ ╩  ╚═╝ ╩ ╩ ╩ ╩ ╩╚═  ╩
`

func main() {
	flag.Parse()
	if *version {
		fmt.Println(Version)
		return
	}
	fmt.Print(LINKSMART)
	logger.Printf("Starting Service Catalog")

	if Version != "" {
		logger.Printf("Version: %s", Version)
	}
	if BuildNumber != "" {
		logger.Printf("Build Number: %s", BuildNumber)
	}

	if *profile != 0 {
		logger.Println("Activating runtime profiling server on port", *profile)
		go func() { logger.Println(http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", *profile), nil)) }()
	}

	// Load configuration
	config, err := loadConfig(*confPath)
	if err != nil {
		logger.Fatalf("Error reading config file %v: %s", *confPath, err)
	}
	if config.ID == "" {
		config.ID = uuid.NewV4().String()
	}

	// Setup storage
	var storage catalog.Storage

	switch config.Storage.Type {
	case catalog.CatalogBackendMemory:
		storage = catalog.NewMemoryStorage()
	case catalog.CatalogBackendLevelDB:
		storage, err = catalog.NewLevelDBStorage(config.Storage.DSN, nil)
		if err != nil {
			logger.Fatalf("Failed to start LevelDB storage: %s", err)
		}
	default:
		logger.Fatalf("Could not create catalog API storage. Unsupported type: %v", config.Storage.Type)
	}

	var listeners []catalog.Listener
	controller, err := catalog.NewController(storage, listeners...)
	if err != nil {
		storage.Close()
		logger.Fatalf("Failed to start the controller: %s", err)
	}

	// Create http api
	httpAPI := catalog.NewHTTPAPI(controller, config.ID, config.Description, Version)
	go serveHTTP(httpAPI, config)

	// Create mqtt api
	go catalog.StartMQTTManager(controller, config.MQTT, config.ID)

	// Announce service using DNS-SD
	var bonjourS *bonjour.Server
	if config.DNSSDEnabled {
		bonjourS, err = bonjour.Register(config.Description, catalog.DNSSDServiceType, "", config.HTTP.BindPort, []string{"uri=/"}, nil)
		if err != nil {
			logger.Printf("Failed to register DNS-SD service: %s", err)
		} else {
			logger.Println("Registered service via DNS-SD using type", catalog.DNSSDServiceType)
		}
	}

	// Ctrl+C / Kill handling
	handler := make(chan os.Signal, 1)
	signal.Notify(handler, os.Interrupt, os.Kill)
	<-handler
	logger.Println("Shutting down...")

	// Stop bonjour registration
	if bonjourS != nil {
		bonjourS.Shutdown()
		time.Sleep(1e9)
	}

	// Shutdown storage
	err = controller.Stop()
	if err != nil {
		logger.Println(err.Error())
	}

	logger.Println("Stopped")
}

func serveHTTP(httpAPI *catalog.HttpAPI, config *Config) {

	commonHandlers := alice.New(
		context.ClearHandler,
		loggingHandler,
		//commonHeaders,
		cors.AllowAll().Handler,
	)

	// Append auth handler if enabled
	if config.Auth.Enabled {
		// Setup ticket validator
		v, err := validator.Setup(
			config.Auth.Provider,
			config.Auth.ProviderURL,
			config.Auth.ServiceID,
			config.Auth.BasicEnabled,
			config.Auth.Authz)
		if err != nil {
			logger.Fatalln(err)
		}

		commonHandlers = commonHandlers.Append(v.Handler)
	}

	// Configure http router
	r := newRouter()
	// generic handlers
	r.get("/health", commonHandlers.ThenFunc(healthHandler))
	r.options("/{path:.*}", commonHandlers.ThenFunc(optionsHandler))

	// service handlers
	r.get("/", commonHandlers.ThenFunc(httpAPI.List))
	r.post("/", commonHandlers.ThenFunc(httpAPI.Post))
	// Accept an id with zero or one slash: [^/]+/?[^/]*
	// -> [^/]+ one or more of anything but slashes /? optional slash [^/]* zero or more of anything but slashes
	r.get("/{id:[^/]+/?[^/]*}", commonHandlers.ThenFunc(httpAPI.Get))
	r.put("/{id:[^/]+/?[^/]*}", commonHandlers.ThenFunc(httpAPI.Put))
	r.delete("/{id:[^/]+/?[^/]*}", commonHandlers.ThenFunc(httpAPI.Delete))
	r.get("/{path}/{op}/{value:.*}", commonHandlers.ThenFunc(httpAPI.Filter))

	// Configure the middleware
	n := negroni.New(
		negroni.NewRecovery(),
	)
	// Mount router
	n.UseHandler(r)

	// Start listener
	n.Run(fmt.Sprintf("%s:%s", config.HTTP.BindAddr, strconv.Itoa(config.HTTP.BindPort)))
}

package main

import (
	"flag"
	"log"
	"github.com/influxdata/influxdb/client/v2"
	"github.com/warmans/anflux/influx"
	"github.com/warmans/resty"
	"github.com/gorilla/mux"
	"github.com/warmans/anflux/server"
	"github.com/warmans/ctxhandler"
	"net/http"
)

var serverbind = flag.String("server.bind", ":8888", "Bind http server to this address")
var influxHost = flag.String("influx.host", "http://localhost:8086", "InfluxDB Host")
var influxUsername = flag.String("influx.username", "", "InfluxDB username")
var influxPassword = flag.String("influx.password", "", "InfluxDB password")
var influxDB = flag.String("influx.db", "notes", "InfluxDB database")

func main() {

	flag.Parse()

	log.Printf("Creating influx client for host/db: %s -> %s\n", *influxHost, *influxDB)
	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     *influxHost,
		Username: *influxUsername,
		Password: *influxPassword,
	})
	if err != nil {
		log.Fatalf("Failed to connect to influxdb: %s", err.Error())
	}
	defer c.Close()

	store := influx.NewInflux(c, *influxDB)
	if err := store.Initialize(); err != nil {
		log.Fatalf("Failed to initialize influx: %s", err.Error())
	}

	routes := mux.NewRouter()
	routes.Handle("/note/{system}/{subsystem}", ctxhandler.Ctx(resty.Restful(&server.NoteHandler{Influx: store})))
	routes.Handle("/stream", &server.EventStreamHandler{Influx: store})
	routes.Handle("/watch", &server.WatchHandler{})

	log.Printf("Listening on %s...", *serverbind)
	log.Fatal(http.ListenAndServe(*serverbind, routes))
}


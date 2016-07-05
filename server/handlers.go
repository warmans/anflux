package server

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/influxdata/influxdb/client/v2"
	"github.com/warmans/anflux/influx"
	"github.com/warmans/resty"
	"golang.org/x/net/context"
	"html/template"
	"log"
)

type NoteHandler struct {
	resty.DefaultRESTHandler
	Influx *influx.Influx
}

func (h *NoteHandler) HandlePost(rw http.ResponseWriter, r *http.Request, ctx context.Context) {
	defer r.Body.Close()

	if err := r.ParseForm(); err != nil {
		Fail(rw, "Failed to parse form", http.StatusBadRequest)
		return
	}

	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		Fail(rw, fmt.Sprintf("Failed to read body: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	fields := map[string]interface{}{"title": r.Form.Get("title"), "text": string(bodyBytes)}

	//just use the system/subsystem vars as tags
	pt, err := client.NewPoint("notes", mux.Vars(r), fields, time.Now())
	if err != nil {
		Fail(rw, fmt.Sprintf("Failed to create point: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	if err := h.Influx.AddPoint(pt); err != nil {
		Fail(rw, fmt.Sprintf("Failed to write point: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	fmt.Fprint(rw, "OK")
}

//---------------------------------------------------------------------

var upgrader = websocket.Upgrader{} // use default options

type EventStreamHandler struct {
	Influx *influx.Influx
}

func (h *EventStreamHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	c, err := upgrader.Upgrade(rw, r, nil)
	if err != nil {
		Fail(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	defer c.Close()

	//check for client disconnect
	closeChecker := NewCloseChecker(c)

	influxObserver := influx.NewChanObserver()
	defer h.Influx.Accept(influxObserver)()

	for {
		select {
		case <-closeChecker.C:
			return
		case note := <-influxObserver.C:
			if err := c.WriteJSON(note); err != nil {
				Fail(rw, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}
}

//---------------------------------------------------------------------

var tmpl = template.Must(template.New("watch").Parse(`
<!DOCTYPE HTML>
<html>
   	<head></head>
   	<body>
		<div id="log" style="font-family: monospace"></div>
		<script type="text/javascript">
		function connect() {
			var ws = new WebSocket("ws://"+window.location.host+"/stream");
			var logEl = document.getElementById("log");

			ws.onopen = function() {
				console.log("socket open...");
			};
			ws.onmessage = function (evt) {
				console.log("Message is received...", evt.data);
				logEl.innerHTML += "<div>"+evt.data+"</div>";
			};
			ws.onclose = function() {
				console.log("Connection is closed, attempting re-connect...");
				setTimeout(connect, 1000);
			};
		 }
		 connect();
		</script>
	</body>
</html>
`))

type WatchHandler struct {}

func (h *WatchHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	if err := tmpl.Execute(rw, nil); err != nil {
		Fail(rw, err.Error(), http.StatusInternalServerError)
	}
}

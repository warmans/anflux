package influx

import (
	"fmt"
	"sync"

	"github.com/influxdata/influxdb/client/v2"
)

const (
	EV_QUERY = "QUERY"
	EV_POINT = "POINT"
)

func NewChanObserver() *ChanObserver {
	return &ChanObserver{C: make(chan *Notification, 1000)}
}

type Observer interface {
	Notify(event string, data interface{})
}

type Notification struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
}

type ChanObserver struct {
	C chan *Notification
}

func (o *ChanObserver) Notify(event string, data interface{}) {
	switch t := data.(type) {
	case *client.Point:
		data = t.String()
	}
	o.C <- &Notification{Event: event, Data: data}
}

func NewInflux(client client.Client, database string) *Influx {
	return &Influx{conn: client, database: database, observers: make(map[Observer]int)}
}

type Influx struct {
	conn      client.Client
	database  string
	observers map[Observer]int
	sync.RWMutex
}

func (i *Influx) Initialize() error {
	_, err := i.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", i.database))
	return err
}

func (i *Influx) Exec(cmd string) (res []client.Result, err error) {
	q := client.Query{
		Command:  cmd,
		Database: i.database,
	}
	if response, err := i.conn.Query(q); err == nil {
		if response.Error() != nil {
			return res, response.Error()
		}
		res = response.Results
	} else {
		return res, err
	}

	i.NotifyAll(EV_QUERY, cmd)

	return res, nil
}

func (i *Influx) AddPoint(point *client.Point) error {
	// Create a new point batch
	bp, err := client.NewBatchPoints(client.BatchPointsConfig{Precision: "s", Database: i.database})
	if err != nil {
		return fmt.Errorf("Failed to create point batch: %s", err.Error())
	}
	bp.AddPoint(point)

	// Write the batch
	if err := i.conn.Write(bp); err != nil {
		return fmt.Errorf("Failed to write point: %s", err.Error())
	}

	i.NotifyAll(EV_POINT, point)

	return nil
}

func (i *Influx) NotifyAll(event string, data interface{}) {
	for o := range i.observers {
		o.Notify(event, data)
	}
}

func (i *Influx) Accept(o Observer) func() {
	i.Lock()
	i.observers[o] = 0
	i.Unlock()
	return func() {
		i.Lock()
		delete(i.observers, o)
		i.Unlock()
	}
}

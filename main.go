package main

import (
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	"github.com/HackerLoop/rotonde-client.go"
	"github.com/HackerLoop/rotonde/shared"
	log "github.com/Sirupsen/logrus"
	"github.com/tarm/serial"
	"github.com/vitaminwater/handlers.go"
)

type Client struct {
	client.Client
}

func (c *Client) sendError(port string, err error) {
	c.SendEvent("SERIAL_STATUS", rotonde.Object{
		"port":    port,
		"status":  "ERROR",
		"message": fmt.Sprintf("%s", err),
	})
}

func (c *Client) sendSuccess(port string) {
	c.SendEvent("SERIAL_STATUS", rotonde.Object{
		"port":   port,
		"status": "SUCCESS",
	})
}

func (c *Client) startPort(name string, baud int) (endChan chan bool) {
	endChan = make(chan bool)
	var stop, isStopped = func() (func(), func() bool) {
		stopChan := make(chan bool)
		isStoppedChan := make(chan bool)
		go func() {
			stop := false
			for {
				select {
				case stop = <-stopChan:
				case isStoppedChan <- stop:
				}
			}
		}()
		return func() {
				stopChan <- true
				endChan <- true
			}, func() bool {
				return <-isStoppedChan
			}
	}()

	var readMutex = new(sync.Mutex)

	conf := &serial.Config{Name: name, Baud: baud, ReadTimeout: time.Second * 1}
	s, err := serial.OpenPort(conf)
	if err != nil {
		c.sendError(name, err)
		log.Warning(err)
		go stop()
		return
	}

	c.OnNamedAction("SERIAL_CLOSE", func(m interface{}) bool {
		log.Info("SERIAL_CLOSE")
		action := m.(rotonde.Action)
		if action.Data["port"] != name {
			return !isStopped()
		}

		stop()
		readMutex.Lock()
		s.Close()
		readMutex.Unlock()
		c.sendSuccess(name)
		return !isStopped()
	})

	c.OnNamedAction("SERIAL_WRITE", func(a interface{}) bool {
		log.Info("SERIAL_WRITE")
		if isStopped() {
			return false
		}
		action := a.(rotonde.Action)
		if action.Data["port"] != name {
			return !isStopped()
		}
		log.Info(action)

		buf, err := base64.StdEncoding.DecodeString(action.Data["data"].(string))
		if err != nil {
			c.sendError(name, err)
			log.Warning(err)
			return !isStopped()
		}
		log.Info(string(buf))

		_, err = s.Write(buf)
		if err != nil {
			c.sendError(name, err)
			log.Warning(err)
			return !isStopped()
		}
		c.sendSuccess(name)
		return !isStopped()
	})

	go func() {
		buf := make([]byte, 100)
		for {
			readMutex.Lock()
			n, err := s.Read(buf)
			readMutex.Unlock()
			if err != nil && fmt.Sprintf("%s", err) != "EOF" {
				//c.sendError(name, err)
				log.Warning(err)
				return
			}
			if isStopped() {
				return
			}
			if n == 0 {
				continue
			}
			log.Info("Received ", base64.StdEncoding.EncodeToString(buf[:n]))
			c.SendEvent("SERIAL_READ", rotonde.Object{
				"port": name,
				"data": base64.StdEncoding.EncodeToString(buf[:n]),
			})
		}
	}()

	c.sendSuccess(name)
	return
}

func main() {
	log.Info("Connecting to rotonde...")
	c := &Client{*client.NewClient("ws://rotonde:4224/")}
	log.Info("connected.")

	open_action := &rotonde.Definition{"SERIAL_OPEN", "action", false, rotonde.FieldDefinitions{}}
	open_action.PushField("port", "string", "")
	open_action.PushField("baud", "number", "baud")
	c.AddLocalDefinition(open_action)

	close_action := &rotonde.Definition{"SERIAL_CLOSE", "action", false, rotonde.FieldDefinitions{}}
	close_action.PushField("port", "string", "")
	c.AddLocalDefinition(close_action)

	write_action := &rotonde.Definition{"SERIAL_WRITE", "action", false, rotonde.FieldDefinitions{}}
	write_action.PushField("port", "string", "")
	write_action.PushField("data", "string", "")
	c.AddLocalDefinition(write_action)

	status_event := &rotonde.Definition{"SERIAL_STATUS", "event", false, rotonde.FieldDefinitions{}}
	status_event.PushField("port", "string", "")
	status_event.PushField("status", "string", "")
	c.AddLocalDefinition(status_event)

	read_event := &rotonde.Definition{"SERIAL_READ", "event", false, rotonde.FieldDefinitions{}}
	read_event.PushField("port", "string", "")
	read_event.PushField("data", "string", "")
	c.AddLocalDefinition(read_event)

	c.OnNamedAction("SERIAL_OPEN", func() handlers.HandlerFunc {
		var isOpen, openned, closed = func() (func(string) bool, func(string), func(string)) {
			var mutex = new(sync.Mutex)
			var openPorts = map[string]bool{}
			return func(port string) bool {
					mutex.Lock()
					defer mutex.Unlock()
					isOpen, ok := openPorts[port]
					return ok && isOpen
				}, func(port string) {
					mutex.Lock()
					defer mutex.Unlock()
					openPorts[port] = true
				}, func(port string) {
					mutex.Lock()
					defer mutex.Unlock()
					openPorts[port] = false
				}
		}()
		return func(m interface{}) bool {
			a := m.(rotonde.Action)
			log.Info("SERIAL_OPEN", a)

			port := a.Data["port"].(string)
			baud := a.Data["baud"].(float64)

			if !isOpen(port) {
				endChan := c.startPort(port, int(baud))
				openned(port)
				go func() {
					<-endChan
					log.Infof("Port closed: %s", port)
					closed(port)
				}()
			}
			return true
		}
	}())

	select {}
}

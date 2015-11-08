package main

import (
	"github.com/HackerLoop/rotonde-client.go"
	"github.com/HackerLoop/rotonde/shared"
	log "github.com/Sirupsen/logrus"
)

func main() {
	client := client.NewClient("ws://rotonde:4224/")

	open_action := &rotonde.Definition{"SERIAL_OPEN", "action", rotonde.FieldDefinitions{}}
	open_action.PushField("port", "string", "")
	open_action.PushField("baud", "number", "baud")
	client.AddLocalDefinition(open_action)

	close_action := &rotonde.Definition{"SERIAL_CLOSE", "action", rotonde.FieldDefinitions{}}
	close_action.PushField("port", "string", "")
	client.AddLocalDefinition(close_action)

	write_action := &rotonde.Definition{"SERIAL_WRITE", "action", rotonde.FieldDefinitions{}}
	write_action.PushField("port", "string", "")
	write_action.PushField("data", "string", "")
	client.AddLocalDefinition(write_action)

	read_event := &rotonde.Definition{"SERIAL_READ", "event", rotonde.FieldDefinitions{}}
	read_event.PushField("port", "string", "")
	read_event.PushField("data", "string", "")
	client.AddLocalDefinition(read_event)

	client.OnNamedAction("SERIAL_OPEN", func(m interface{}) bool {
		log.Info("SERIAL_OPEN")
		return true
	})

	client.OnNamedAction("SERIAL_CLOSE", func(m interface{}) bool {
		log.Info("SERIAL_CLOSE")
		return true
	})

	client.OnNamedAction("SERIAL_WRITE", func(m interface{}) bool {
		log.Info("SERIAL_WRITE")
		return true
	})

	select {}
}

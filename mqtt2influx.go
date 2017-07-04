package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/eclipse/paho.mqtt.golang"
	"github.com/influxdata/influxdb/client/v2"
	"github.com/jinzhu/configor"
	"strconv"
)

var Config = struct {
	MQTT struct {
		Host     string `required:"true" default:"tcp://127.0.0.1:1883"`
		User     string
		Password string
		Topic    string `default:"#"`
	}

	INFLUX struct {
		Host     string `required:"true" default:"http://localhost:8086"`
		User     string
		Password string
		Database string `required:"true"`
		Interval time.Duration
	}

	SYNC []struct {
		Pattern     string
		Measurement string
	}
}{}

var pointCollection []*client.Point

const (
	eventFieldName = "occurred"
)

func main() {

	config := flag.String("file", "config.toml", "configuration file")
	flag.StringVar(&Config.MQTT.Host, "mh", "tcp://127.0.0.1:1883", "source broker connection string")
	flag.StringVar(&Config.MQTT.User, "mu", "", "source broker username")
	flag.StringVar(&Config.MQTT.Password, "mp", "", "source broker password")
	flag.StringVar(&Config.MQTT.Topic, "t", "#", "source topic")
	//debugMode := flag.Bool("debug", false, "turn on debug output")

	flag.Parse()
	configor.Load(&Config, *config)

	sourceBrokerString := Config.MQTT.Host
	sourceUserString := Config.MQTT.User
	sourcePassString := Config.MQTT.Password
	sourceTopic := Config.MQTT.Topic

	sourceOpts := mqtt.NewClientOptions().AddBroker(sourceBrokerString).SetClientID("mqtt2influx")
	sourceOpts.SetAutoReconnect(true)

	if sourceUserString != "" {
		sourceOpts.SetUsername(sourceUserString)
	}

	if sourcePassString != "" {
		sourceOpts.SetPassword(sourcePassString)
	}

	// Create a new HTTPClient
	influxClient, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     Config.INFLUX.Host,
		Username: Config.INFLUX.User,
		Password: Config.INFLUX.Password,
	})
	if err != nil {
		log.Fatal(err)
	}

	sourceOpts.OnConnect = func(sourceClient mqtt.Client) {
		if token := sourceClient.Subscribe(sourceTopic, 2, syncCallback); token.Wait() && token.Error() != nil {
			fmt.Println(token.Error())
		}
	}

	sourceClient := mqtt.NewClient(sourceOpts)

	if token := sourceClient.Connect(); token.Wait() && token.Error() != nil {
		fmt.Printf("source host: %v\n", token.Error())
		os.Exit(1)
	}

	defer sourceClient.Disconnect(10)

	fmt.Println("source mqtt connected...")

	t := time.NewTicker(time.Second * Config.INFLUX.Interval)
	for {
		if len(pointCollection) > 0 {
			pointCollection = writePoints(pointCollection, influxClient)
		}
		<-t.C
	}
}

func writePoints(pointCollection []*client.Point, influxClient client.Client) []*client.Point {
	fmt.Printf("writing %d Points to influx\n", len(pointCollection))
	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  Config.INFLUX.Database,
		Precision: "s",
	})
	if err != nil {
		log.Print(err)
	}
	bp.AddPoints(pointCollection)
	if err := influxClient.Write(bp); err != nil {
		log.Print(err)
	} else {
		pointCollection = nil
	}
	return pointCollection
}

func syncCallback(mqttClient mqtt.Client, message mqtt.Message) {
	fmt.Printf("message: %s, %s\n", message.Topic(), message.Payload())

	for _, sync := range Config.SYNC {

		tags, fields := gatherInfluxData(sync.Pattern, message.Topic(), message.Payload())

		if fields == nil {
			continue
		}

		point, err := client.NewPoint(sync.Measurement, tags, fields, time.Now())
		if err != nil {
			log.Print(err)
		}

		// "store" new point until next batch interval kicks in
		pointCollection = append(pointCollection, point)

		// only process first pattern match
		break
	}
}

func isNumeric(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

func isBoolean(s string) bool {
	_, err := strconv.ParseBool(s)
	return err == nil
}

func gatherInfluxData(pattern string, topic string, payload []byte) (map[string]string, map[string]interface{}) {
	var myExp = regexp.MustCompile(pattern)
	match := myExp.FindStringSubmatch(topic)

	// skip this pattern if no matches found
	if len(match) == 0 {
		return nil, nil
	}

	tags := make(map[string]string)
	fieldName := ""

	// collect tags and field
	for i, name := range myExp.SubexpNames() {
		if i != 0 {
			if name == "" {
				if fieldName != "" {
					fieldName += "."
				}
				fieldName += match[i]
			} else {
				tags[name] = match[i]
			}
		}
	}

	if fieldName == "" {
		// no unnamed group exists => use last subtopic
		topics := strings.Split(topic, "/")
		fieldName = topics[len(topics)-1]
	}

	fields := make(map[string]interface{})

	payloadIsNumericOrBoolean := isNumeric(string(payload)) || isBoolean(string(payload))

	if payloadIsNumericOrBoolean {
		fields[fieldName] = payload
	} else {
		// if payload is neither numeric nor boolean we want to record the payload as tag
		// with a "occurred" field of "true" e.g.:
		// /foo/event STATUS_CHANGE	 => event=POWERUP occurred=true
		tags[fieldName] = string(payload)
		fields[eventFieldName] = true
	}

	return tags, fields
}

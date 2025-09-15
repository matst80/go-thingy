package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/matst80/go-thingy/thingy"
)

var (
	deviceName = flag.String("name", "Office", "name of remote peripheral")
	mqttBroker = flag.String("mqtt", "tcp://10.10.3.12:1883", "address of the mqtt broker")
)

func main() {
	flag.Parse()

	opts := mqtt.NewClientOptions().AddBroker(*mqttBroker)
	opts.SetClientID("go-thingy")
	mqttClient := mqtt.NewClient(opts)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	defer mqttClient.Disconnect(250)

	t, err := thingy.New(*deviceName)
	if err != nil {
		log.Fatalf("can't connect to Thingy: %s", err)
	}
	defer t.Disconnect()

	if err := t.TemperatureEnable(); err != nil {
		log.Fatalf("failed to enable temperature notification: %s", err)
	} else {
		publishDiscovery(mqttClient, t.MAC(), "temperature", "temperature", *deviceName, "°C")
	}

	if err := t.PressureEnable(); err != nil {
		log.Fatalf("failed to enable pressure notification: %s", err)
	} else {
		publishDiscovery(mqttClient, t.MAC(), "pressure", "pressure", *deviceName, "hPa")
	}
	if err := t.HumidityEnable(); err != nil {
		log.Fatalf("failed to enable humidity notification: %s", err)
	} else {
		publishDiscovery(mqttClient, t.MAC(), "humidity", "humidity", *deviceName, "%")
	}
	if err := t.GasEnable(); err != nil {
		log.Fatalf("failed to enable gas notification: %s", err)
	} else {
		publishDiscovery(mqttClient, t.MAC(), "eco2", "carbon_dioxide", *deviceName, "ppm")
		publishDiscovery(mqttClient, t.MAC(), "number", "volatile_organic_compounds", *deviceName, "ppb")
	}

	go func() {
		for {
			select {
			case temp, ok := <-t.TempNotif:
				if !ok {
					return
				}
				//fmt.Printf("Temperature: %.2f C\n", temp)

				publishState(mqttClient, t.MAC(), "temperature", *deviceName, temp)
			case press, ok := <-t.PressNotif:
				if !ok {
					return
				}
				//fmt.Printf("Pressure: %.2f hPa\n", press)

				publishState(mqttClient, t.MAC(), "pressure", *deviceName, press)
			case humid, ok := <-t.HumidNotif:
				if !ok {
					return
				}
				//fmt.Printf("Humidity: %d %%\n", humid)

				publishState(mqttClient, t.MAC(), "humidity", *deviceName, humid)
			case gas, ok := <-t.GasNotif:
				if !ok {
					return
				}
				fmt.Printf("Gas: eCO2: %d ppm, TVOC: %d ppb\n", gas.ECO2, gas.TVOC)

				publishState(mqttClient, t.MAC(), "eco2", *deviceName, gas.ECO2)

				publishState(mqttClient, t.MAC(), "tvoc", *deviceName, gas.TVOC)
			}
		}
	}()

	fmt.Println("Press any key to exit...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}

func publishDiscovery(mqttClient mqtt.Client, mac, sensorType, deviceClass, deviceName, unit string) {

	topic := fmt.Sprintf("homeassistant/sensor/%s/%s/config", deviceName, sensorType)
	payload := map[string]interface{}{
		"name":         fmt.Sprintf("%s %s", deviceName, sensorType),
		"stat_t":       fmt.Sprintf("homeassistant/sensor/%s/state", deviceName),
		"val_tpl":      fmt.Sprintf("{{ value_json.%s }}", sensorType),
		"unit_of_meas": unit,
		"dev_cla":      deviceClass,
		"uniq_id":      fmt.Sprintf("%s_%s", mac, sensorType),
		"device": map[string]interface{}{
			"identifiers":  []string{mac},
			"name":         deviceName,
			"manufacturer": "Nordic Semiconductor",
			"model":        "Thingy:52",
		},
	}
	payloadBytes, _ := json.Marshal(payload)
	mqttClient.Publish(topic, 0, false, payloadBytes)
}

func publishState(mqttClient mqtt.Client, mac, sensorType string, deviceName string, value interface{}) {

	topic := fmt.Sprintf("homeassistant/sensor/%s/state", deviceName)
	payload := map[string]interface{}{
		sensorType: value,
	}
	payloadBytes, _ := json.Marshal(payload)
	mqttClient.Publish(topic, 0, false, payloadBytes)
}

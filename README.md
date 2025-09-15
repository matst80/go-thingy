# Go Thingy

This project provides a Go application and a library to interact with Nordic Thingy:52 devices over Bluetooth LE. It reads sensor data from the Thingy and publishes it to an MQTT broker for integration with Home Assistant using the MQTT Discovery protocol.

## Application Usage

The `go-thingy` application connects to a specified Thingy:52 device, reads its sensor data, and forwards it to an MQTT broker.

### Prerequisites

- Go 1.22 or later
- A Nordic Thingy:52 device
- An MQTT broker (e.g., Mosquitto)

### Installation

1.  Clone the repository:
    ```sh
    git clone https://github.com/matst80/go-thingy.git
    cd go-thingy
    ```

2.  Install dependencies:
    ```sh
    go mod tidy
    ```

### Building

Build the application using the following command:

```sh
go build .
```

### Running

Run the application with `sudo` as it requires elevated privileges for BLE access. You can specify the Thingy's name and the MQTT broker address using command-line flags.

```sh
sudo ./go-thingy --name "YourThingyName" --mqtt "tcp://your-broker-address:1883"
```

#### Flags

-   `--name`: The name of the BLE peripheral to connect to (e.g., "Office"). Defaults to "Office".
-   `--mqtt`: The address of the MQTT broker. Defaults to "tcp://10.10.3.12:1883".

## Library Usage

The `thingy` package provides a simple API to connect to a Thingy:52 device and receive sensor notifications.

### Example

Here's a basic example of how to use the library:

```go
package main

import (
	"fmt"
	"log"

	"github.com/matst80/go-thingy/thingy"
)

func main() {
	// Connect to the Thingy device named "Office"
	t, err := thingy.New("Office")
	if err != nil {
		log.Fatalf("Failed to connect to Thingy: %s", err)
	}
	defer t.Disconnect()

	// Enable temperature notifications
	if err := t.TemperatureEnable(); err != nil {
		log.Fatalf("Failed to enable temperature notifications: %s", err)
	}

	// Read from the temperature notification channel
	for temp := range t.TempNotif {
		fmt.Printf("Temperature: %.2f°C\n", temp)
	}
}
```

## Home Assistant Integration

This application uses the Home Assistant MQTT Discovery protocol. Once the application is running and connected to your MQTT broker, the Thingy:52 sensors will automatically appear as entities in Home Assistant.

Ensure that the MQTT integration is set up in Home Assistant and that the broker address provided to the `go-thingy` application is correct.

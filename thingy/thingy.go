package thingy

import (
	"context"
	"encoding/binary"
	"fmt"
	"strings"
	"time"

	"github.com/go-ble/ble"
	"github.com/go-ble/ble/examples/lib/dev"
)

const (
	TCS_UUID = "ef6801009b3549339b1052ffa9740042"

	TES_UUID       = "ef6802009b3549339b1052ffa9740042"
	TES_TEMP_UUID  = "ef6802019b3549339b1052ffa9740042"
	TES_PRESS_UUID = "ef6802029b3549339b1052ffa9740042"
	TES_HUMID_UUID = "ef6802039b3549339b1052ffa9740042"
	TES_GAS_UUID   = "ef6802049b3549339b1052ffa9740042"
	TES_COLOR_UUID = "ef6802059b3549339b1052ffa9740042"
	TES_CONF_UUID  = "ef6802069b3549339b1052ffa9740042"

	UIS_UUID     = "ef6803009b3549339b1052ffa9740042"
	UIS_LED_UUID = "ef6803019b3549339b1052ffa9740042"
	UIS_BTN_UUID = "ef6803029b3549339b1052ffa9740042"
	UIS_PIN_UUID = "ef6803039b3549339b1052ffa9740042"

	TMS_UUID              = "ef6804009b3549339b1052ffa9740042"
	TMS_CONF_UUID         = "ef6804019b3549339b1052ffa9740042"
	TMS_TAP_UUID          = "ef6804029b3549339b1052ffa9740042"
	TMS_ORIENTATION_UUID  = "ef6804039b3549339b1052ffa9740042"
	TMS_QUATERNION_UUID   = "ef6804049b3549339b1052ffa9740042"
	TMS_STEP_COUNTER_UUID = "ef6804059b3549339b1052ffa9740042"
	TMS_RAW_DATA_UUID     = "ef6804069b3549339b1052ffa9740042"
	TMS_EULER_UUID        = "ef6804079b3549339b1052ffa9740042"
	TMS_ROTATION_UUID     = "ef6804089b3549339b1052ffa9740042"
	TMS_HEADING_UUID      = "ef6804099b3549339b1052ffa9740042"
	TMS_GRAVITY_UUID      = "ef68040a9b3549339b1052ffa9740042"

	TSS_UUID              = "ef6805009b3549339b1052ffa9740042"
	TSS_CONF_UUID         = "ef6805019b3549339b1052ffa9740042"
	TSS_SPEAKER_DATA_UUID = "ef6805029b3549339b1052ffa9740042"
	TSS_SPEAKER_STAT_UUID = "ef6805039b3549339b1052ffa9740042"
	TSS_MIC_UUID          = "ef6805049b3549339b1052ffa9740042"
)

type Thingy struct {
	cln     ble.Client
	profile *ble.Profile
	mac     string

	TempNotif  chan float32
	PressNotif chan float32
	HumidNotif chan uint8
	GasNotif   chan struct{ ECO2, TVOC uint16 }
	ColorNotif chan struct{ Red, Green, Blue, Clear uint16 }
	BtnNotif   chan bool
}

func New(deviceName string) (*Thingy, error) {
	d, err := dev.NewDevice("default")
	if err != nil {
		return nil, err
	}
	ble.SetDefaultDevice(d)

	filter := func(a ble.Advertisement) bool {
		return strings.Contains(strings.ToUpper(a.LocalName()), strings.ToUpper(deviceName))
	}

	ctx := ble.WithSigHandler(context.WithTimeout(context.Background(), 60*time.Second))
	cln, err := ble.Connect(ctx, filter)
	if err != nil {
		return nil, err
	}

	p, err := cln.DiscoverProfile(true)
	if err != nil {
		return nil, err
	}

	mac := cln.Addr().String()
	mac = strings.ReplaceAll(mac, ":", "")

	t := &Thingy{
		cln:     cln,
		profile: p,
		mac:     mac,

		TempNotif:  make(chan float32, 10),
		PressNotif: make(chan float32, 10),
		HumidNotif: make(chan uint8, 10),
		GasNotif:   make(chan struct{ ECO2, TVOC uint16 }, 10),
		ColorNotif: make(chan struct{ Red, Green, Blue, Clear uint16 }, 10),
		BtnNotif:   make(chan bool, 10),
	}

	go func() {
		<-cln.Disconnected()
		close(t.TempNotif)
		close(t.PressNotif)
		close(t.HumidNotif)
		close(t.GasNotif)
		close(t.ColorNotif)
		close(t.BtnNotif)
	}()

	return t, nil
}

func (t *Thingy) MAC() string {
	return t.mac
}

func (t *Thingy) Disconnect() {
	t.cln.CancelConnection()
}

func (t *Thingy) onTempNotif(data []byte) {
	integer := int8(data[0])
	decimal := uint8(data[1])
	temperature := float32(integer) + float32(decimal)/100.0
	t.TempNotif <- temperature
}

func (t *Thingy) onPressNotif(data []byte) {
	integer := binary.LittleEndian.Uint32(data[0:4])
	decimal := uint8(data[4])
	pressure := float32(integer) + float32(decimal)/100.0
	t.PressNotif <- pressure
}

func (t *Thingy) onHumidNotif(data []byte) {
	humid := uint8(data[0])
	t.HumidNotif <- humid
}

func (t *Thingy) onGasNotif(data []byte) {
	eco2 := binary.LittleEndian.Uint16(data[0:2])
	tvoc := binary.LittleEndian.Uint16(data[2:4])
	t.GasNotif <- struct{ ECO2, TVOC uint16 }{eco2, tvoc}
}

func (t *Thingy) onColorNotif(data []byte) {
	red := binary.LittleEndian.Uint16(data[0:2])
	green := binary.LittleEndian.Uint16(data[2:4])
	blue := binary.LittleEndian.Uint16(data[4:6])
	clear := binary.LittleEndian.Uint16(data[6:8])
	t.ColorNotif <- struct{ Red, Green, Blue, Clear uint16 }{red, green, blue, clear}
}

func (t *Thingy) onBtnNotif(data []byte) {
	t.BtnNotif <- data[0] == 1
}

func (t *Thingy) getCharacteristic(serviceUUID, charUUID string) (*ble.Characteristic, error) {
	sUUID := ble.MustParse(serviceUUID)
	cUUID := ble.MustParse(charUUID)

	for _, s := range t.profile.Services {
		if s.UUID.Equal(sUUID) {
			for _, c := range s.Characteristics {
				if c.UUID.Equal(cUUID) {
					return c, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("characteristic %s not found in service %s", charUUID, serviceUUID)
}

func (t *Thingy) TemperatureEnable() error {
	c, err := t.getCharacteristic(TES_UUID, TES_TEMP_UUID)
	if err != nil {
		return err
	}
	return t.cln.Subscribe(c, false, t.onTempNotif)
}

func (t *Thingy) TemperatureDisable() error {
	c, err := t.getCharacteristic(TES_UUID, TES_TEMP_UUID)
	if err != nil {
		return err
	}
	return t.cln.Unsubscribe(c, false)
}

func (t *Thingy) PressureEnable() error {
	c, err := t.getCharacteristic(TES_UUID, TES_PRESS_UUID)
	if err != nil {
		return err
	}
	return t.cln.Subscribe(c, false, t.onPressNotif)
}

func (t *Thingy) PressureDisable() error {
	c, err := t.getCharacteristic(TES_UUID, TES_PRESS_UUID)
	if err != nil {
		return err
	}
	return t.cln.Unsubscribe(c, false)
}

func (t *Thingy) HumidityEnable() error {
	c, err := t.getCharacteristic(TES_UUID, TES_HUMID_UUID)
	if err != nil {
		return err
	}
	return t.cln.Subscribe(c, false, t.onHumidNotif)
}

func (t *Thingy) HumidityDisable() error {
	c, err := t.getCharacteristic(TES_UUID, TES_HUMID_UUID)
	if err != nil {
		return err
	}
	return t.cln.Unsubscribe(c, false)
}

func (t *Thingy) GasEnable() error {
	c, err := t.getCharacteristic(TES_UUID, TES_GAS_UUID)
	if err != nil {
		return err
	}
	return t.cln.Subscribe(c, false, t.onGasNotif)
}

func (t *Thingy) GasDisable() error {
	c, err := t.getCharacteristic(TES_UUID, TES_GAS_UUID)
	if err != nil {
		return err
	}
	return t.cln.Unsubscribe(c, false)
}

func (t *Thingy) ColorEnable() error {
	c, err := t.getCharacteristic(TES_UUID, TES_COLOR_UUID)
	if err != nil {
		return err
	}
	return t.cln.Subscribe(c, false, t.onColorNotif)
}

func (t *Thingy) ColorDisable() error {
	c, err := t.getCharacteristic(TES_UUID, TES_COLOR_UUID)
	if err != nil {
		return err
	}
	return t.cln.Unsubscribe(c, false)
}

func (t *Thingy) ButtonEnable() error {
	c, err := t.getCharacteristic(UIS_UUID, UIS_BTN_UUID)
	if err != nil {
		return err
	}
	return t.cln.Subscribe(c, false, t.onBtnNotif)
}

func (t *Thingy) ButtonDisable() error {
	c, err := t.getCharacteristic(UIS_UUID, UIS_BTN_UUID)
	if err != nil {
		return err
	}
	return t.cln.Unsubscribe(c, false)
}

package iec_client

import (
	"fmt"
	"github.com/thinkgos/go-iecp5/asdu"
	"github.com/thinkgos/go-iecp5/cs104"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrorNoConnection = fmt.Errorf("no connection to server")
)

type ConnectionStateHandler func(bool)
type DataHandler func(typ DataType, iot int, data interface{})

type Logger interface {
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

type IEC104Client struct {
	client     *cs104.Client
	serverIP   string
	serverPort int
	commonAddr int
	Logger     Logger

	closer                 chan struct{}
	mu                     sync.Mutex
	connectionStateHandler ConnectionStateHandler
	dataHandler            DataHandler

	Connected      atomic.Bool
	Telemetry      map[int]TelemetryPoint
	Teleindication map[int]TeleindPoint
	Telecontrol    map[int]TelecontrolPoint
	Teleregulation map[int]TeleregulationPoint
}

func NewIEC104Client(host string, port, commonAddr int) *IEC104Client {
	client := &IEC104Client{
		serverIP:       host,
		serverPort:     port,
		commonAddr:     commonAddr,
		closer:         make(chan struct{}),
		Telemetry:      make(map[int]TelemetryPoint),
		Teleindication: make(map[int]TeleindPoint),
		Telecontrol:    make(map[int]TelecontrolPoint),
		Teleregulation: make(map[int]TeleregulationPoint),
	}

	go client.run(time.Second * 15)
	return client
}

func (c *IEC104Client) UpdateConfig(host string, port, commonAddr int) {
	c.mu.Lock()
	c.mu.Unlock()

	c.serverIP = host
	c.serverPort = port
	c.commonAddr = commonAddr
}

func (c *IEC104Client) RegisterConnectionStateHandler(handler ConnectionStateHandler) {
	c.connectionStateHandler = handler
}

func (c *IEC104Client) RegisterDataHandler(handler DataHandler) {
	c.dataHandler = handler
}

func (c *IEC104Client) Connect() error {
	if c.Connected.Load() {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	option := cs104.NewOption()
	option.SetAutoReconnect(true)
	option.SetReconnectInterval(5 * time.Second)

	err := option.AddRemoteServer(fmt.Sprintf("%s:%d", c.serverIP, c.serverPort))
	if err != nil {
		return err
	}

	c.client = cs104.NewClient(c, option)
	c.client.LogMode(false)

	c.client.SetOnConnectHandler(func(client *cs104.Client) {
		c.Connected.Store(true)
		if c.connectionStateHandler != nil {
			c.connectionStateHandler(true)
		}
		c.Logger.Infof("Connected to server: %s:%d", c.serverIP, c.serverPort)
		client.SendStartDt()
	})

	c.client.SetConnectionLostHandler(func(client *cs104.Client) {
		c.Connected.Store(false)
		if c.connectionStateHandler != nil {
			c.connectionStateHandler(false)
		}
		c.Logger.Infof("Disconnected from server: %s:%d", c.serverIP, c.serverPort)
	})

	err = c.client.Start()
	if err != nil {
		return fmt.Errorf("connect error: %v", err)
	}

	return nil
}

func (c *IEC104Client) Disconnect() error {
	if !c.Connected.Load() || c.client == nil {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.client.Close()
	c.Connected.Store(false)
	return nil
}

func (c *IEC104Client) Close() {
	close(c.closer)
	c.Disconnect()
}

// SendTelecontrol sends a telecontrol command (digital control) to the server
// TODO only support single command & with select
func (c *IEC104Client) SendTelecontrol(offset int, value bool) error {
	if !c.Connected.Load() || c.client == nil {
		return ErrorNoConnection
	}

	ioa := offset + 24577
	c.mu.Lock()
	defer c.mu.Unlock()

	err := asdu.SingleCmd(c.client, asdu.C_SC_NA_1, asdu.CauseOfTransmission{
		Cause: asdu.Activation,
	}, 1, asdu.SingleCommandInfo{
		Ioa:   asdu.InfoObjAddr(ioa),
		Value: value,
		Qoc: asdu.QualifierOfCommand{
			InSelect: true,
		},
	})
	if err != nil {
		return fmt.Errorf("send Select Command error: %v", err)
	}

	time.Sleep(time.Millisecond * 200)
	err = asdu.SingleCmd(c.client, asdu.C_SC_NA_1, asdu.CauseOfTransmission{
		Cause: asdu.Activation,
	}, 1, asdu.SingleCommandInfo{
		Ioa:   asdu.InfoObjAddr(ioa),
		Value: value,
	})
	if err != nil {
		return fmt.Errorf("send Telecontrol Command error: %v", err)
	}

	return nil
}

// SendTelemetry sends a telemetry command (analog control) to the server
// TODO only support float
func (c *IEC104Client) SendTelemetry(offset int, value float64) error {
	if !c.Connected.Load() || c.client == nil {
		return ErrorNoConnection
	}

	ioa := offset + 25089

	c.mu.Lock()
	defer c.mu.Unlock()

	err := asdu.SetpointCmdFloat(c.client, asdu.C_SE_NC_1, asdu.CauseOfTransmission{Cause: asdu.Activation}, 1, asdu.SetpointCommandFloatInfo{
		Ioa:   asdu.InfoObjAddr(ioa),
		Value: float32(value),
	})
	if err != nil {
		return fmt.Errorf("send Telemetry Command error: %v", err)
	}
	return nil
}

func (c *IEC104Client) InterrogationHandler(asdu.Connect, *asdu.ASDU) error {
	return nil
}

func (c *IEC104Client) CounterInterrogationHandler(asdu.Connect, *asdu.ASDU) error {
	return nil
}

func (c *IEC104Client) ReadHandler(asdu.Connect, *asdu.ASDU) error {
	return nil
}

func (c *IEC104Client) TestCommandHandler(asdu.Connect, *asdu.ASDU) error {
	return nil
}

func (c *IEC104Client) ClockSyncHandler(asdu.Connect, *asdu.ASDU) error {
	return nil
}

func (c *IEC104Client) ResetProcessHandler(asdu.Connect, *asdu.ASDU) error {
	return nil
}

func (c *IEC104Client) DelayAcquisitionHandler(asdu.Connect, *asdu.ASDU) error {
	return nil
}

func (c *IEC104Client) ASDUHandler(client asdu.Connect, a *asdu.ASDU) error {
	if a.CommonAddr != asdu.CommonAddr(c.commonAddr) {
		return nil
	}
	switch a.Identifier.Type {
	case asdu.M_ME_NC_1:
		data := a.GetMeasuredValueFloat()
		for _, d := range data {
			c.Telemetry[int(d.Ioa)] = TelemetryPoint{
				DataPoint: DataPoint{
					Address:   int(d.Ioa),
					Timestamp: d.Time,
				},
				Value: float64(d.Value),
			}

			if c.dataHandler != nil {
				c.dataHandler(Telemetry, int(d.Ioa), float64(d.Value))
			}
		}
	case asdu.M_ME_NA_1, asdu.M_ME_ND_1:
		data := a.GetMeasuredValueNormal()
		for _, d := range data {
			c.Telemetry[int(d.Ioa)] = TelemetryPoint{
				DataPoint: DataPoint{
					Address:   int(d.Ioa),
					Timestamp: d.Time,
				},
				Value: float64(d.Value),
			}

			if c.dataHandler != nil {
				c.dataHandler(Telemetry, int(d.Ioa), float64(d.Value))
			}
		}

	case asdu.M_SP_NA_1:
		data := a.GetSinglePoint()
		for _, d := range data {
			c.Teleindication[int(d.Ioa)] = TeleindPoint{
				DataPoint: DataPoint{
					Address:   int(d.Ioa),
					Timestamp: d.Time,
				},
				Value: d.Value,
			}

			if c.dataHandler != nil {
				c.dataHandler(Teleindication, int(d.Ioa), d.Value)
			}
		}
	case asdu.M_ME_NB_1:
		data := a.GetMeasuredValueScaled()
		for _, d := range data {
			c.Telemetry[int(d.Ioa)] = TelemetryPoint{
				DataPoint: DataPoint{
					Address:   int(d.Ioa),
					Timestamp: d.Time,
				},
				Value: float64(d.Value),
			}

			if c.dataHandler != nil {
				c.dataHandler(Telemetry, int(d.Ioa), float64(d.Value))
			}
		}

	default:
		c.Logger.Debugf("Invalid ASDU type: %s", a.Identifier.Type)
	}
	return nil
}

func (c *IEC104Client) run(callInterval time.Duration) {
	time.Sleep(time.Second * 5)
	c.allCall()
	timer := time.NewTimer(callInterval)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			c.allCall()
			timer.Reset(callInterval)
		case <-c.closer:
			return
		}
	}
}

func (c *IEC104Client) allCall() {
	if !c.Connected.Load() {
		return
	}
	coa := asdu.CauseOfTransmission{
		Cause: asdu.Activation,
	}
	ca := asdu.CommonAddr(c.commonAddr)
	err := asdu.InterrogationCmd(c.client, coa, ca, asdu.QOIStation)
	if err != nil {
		c.Logger.Infof("104 interrogation error = %v", err)
	}
}

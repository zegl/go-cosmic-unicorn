//go:generate pioasm -o go cosmic_unicorn.pio cosmic_unicorn_pio.go

package cosmic

import (
	"fmt"
	"image/color"
	"machine"
	"time"

	pio "github.com/tinygo-org/pio/rp2-pio"
)

const WIDTH = 32
const HEIGHT = 32

// pin assignments
const COLUMN_CLOCK = machine.GP13
const COLUMN_DATA = machine.GP14
const COLUMN_LATCH = machine.GP15
const COLUMN_BLANK = machine.GP16

const ROW_BIT_0 = machine.GP17
const ROW_BIT_1 = machine.GP18
const ROW_BIT_2 = machine.GP19
const ROW_BIT_3 = machine.GP20

const LIGHT_SENSOR = machine.GP28

const MUTE = machine.GP22

const I2S_DATA = machine.GP9
const I2S_BCLK = machine.GP10
const I2S_LRCLK = machine.GP11

const I2C_SDA = machine.GP4
const I2C_SCL = machine.GP5

const SWITCH_A = machine.GP0
const SWITCH_B = machine.GP1
const SWITCH_C = machine.GP3
const SWITCH_D = machine.GP6

const SWITCH_SLEEP = machine.GP27

const SWITCH_VOLUME_UP = machine.GP7
const SWITCH_VOLUME_DOWN = machine.GP8
const SWITCH_BRIGHTNESS_UP = machine.GP21
const SWITCH_BRIGHTNESS_DOWN = machine.GP26

const ROW_COUNT = 16
const BCD_FRAME_COUNT = 2 // TODO: set to 14 to support 14-bit colors again
const FRAME_COL_SIZE = 32 * 2 * 3

type CosmicUnicorn struct {
	bitstream  [BCD_FRAME_COUNT][16 * FRAME_COL_SIZE]bool
	brightness uint8
}

func (c *CosmicUnicorn) clear() {
	for frame := 0; frame < BCD_FRAME_COUNT; frame++ {
		for idx := 0; idx < 16*FRAME_COL_SIZE; idx++ {
			c.bitstream[frame][idx] = false
		}
	}
}

func (c *CosmicUnicorn) SetPixel(x, y int, r, g, b uint8) {
	if x < 0 || y < 0 || x > 31 || y > 31 {
		return
	}

	x = (WIDTH - 1) - x
	y = (HEIGHT - 1) - y

	// map coordinates into display space
	if y < 16 {
		// move to top half of display (which is actually the right half of the framebuffer)
		x += 32
	} else {
		// remap y coordinate
		y -= 16
	}

	on := func(val uint8, frame int) bool {
		if val < 255/3 {
			return false
		} else if val < 255/3*2 {
			return frame == 1
		} else {
			return true
		}
	}

	for frame := 0; frame < BCD_FRAME_COUNT; frame++ {
		c.bitstream[frame][y*FRAME_COL_SIZE+(x*3+0)] = on(b, frame)
		c.bitstream[frame][y*FRAME_COL_SIZE+(x*3+1)] = on(g, frame)
		c.bitstream[frame][y*FRAME_COL_SIZE+(x*3+2)] = on(r, frame)
	}
}

func (c *CosmicUnicorn) SetColor(x, y int, col color.Color) {
	r, g, b, _ := col.RGBA()
	c.SetPixel(x, y, uint8(r), uint8(g), uint8(b))
}

func (c *CosmicUnicorn) Init() {
	c.brightness = 255

	COLUMN_CLOCK.Configure(machine.PinConfig{Mode: machine.PinOutput})
	COLUMN_CLOCK.Set(false)
	COLUMN_DATA.Configure(machine.PinConfig{Mode: machine.PinOutput})
	COLUMN_DATA.Set(false)
	COLUMN_LATCH.Configure(machine.PinConfig{Mode: machine.PinOutput})
	COLUMN_LATCH.Set(false)
	COLUMN_BLANK.Configure(machine.PinConfig{Mode: machine.PinOutput})
	COLUMN_BLANK.Set(true)

	// initialise the row select, and set them to a non-visible row to avoid flashes during setup
	ROW_BIT_0.Configure(machine.PinConfig{Mode: machine.PinOutput})
	ROW_BIT_0.Set(true)
	ROW_BIT_1.Configure(machine.PinConfig{Mode: machine.PinOutput})
	ROW_BIT_1.Set(true)
	ROW_BIT_2.Configure(machine.PinConfig{Mode: machine.PinOutput})
	ROW_BIT_2.Set(true)
	ROW_BIT_3.Configure(machine.PinConfig{Mode: machine.PinOutput})
	ROW_BIT_3.Set(true)

	// wtf
	// configure full output current in register 2

	var reg1 uint16 = 0b1111111111001110

	// clock the register value to the first 11 driver chips
	for j := uint16(0); j < 11; j++ {
		for i := uint16(0); i < 16; i++ {
			if reg1&(uint16(1)<<(15-i)) != 0 {
				COLUMN_DATA.Set(true)
			} else {
				COLUMN_DATA.Set(false)
			}
			time.Sleep(time.Microsecond * 10)
			COLUMN_CLOCK.Set(true)
			time.Sleep(time.Microsecond * 10)
			COLUMN_CLOCK.Set(false)
		}
	}

	// clock the last chip and latch the value
	for i := uint16(0); i < 16; i++ {
		if reg1&(uint16(1)<<(15-i)) != 0 {
			COLUMN_DATA.Set(true)
		} else {
			COLUMN_DATA.Set(false)
		}

		time.Sleep(time.Microsecond * 10)
		COLUMN_CLOCK.Set(true)
		time.Sleep(time.Microsecond * 10)
		COLUMN_CLOCK.Set(false)

		if i == 4 {
			COLUMN_LATCH.Set(true)
		}
	}
	COLUMN_LATCH.Set(false)

	// reapply the blank as the above seems to cause a slight glow.
	// Note, this will produce a brief flash if a visible row is selected (which it shouldn't be)
	COLUMN_BLANK.Set(false)
	time.Sleep(time.Microsecond * 10)
	COLUMN_BLANK.Set(true)
	// wtf

	time.Sleep(100 * time.Millisecond)

	// COLUMN_BLANK.Set(false)
	MUTE.Set(true)

	// Sleep to catch prints.
	print("boot1")
	time.Sleep(time.Second * 5)
	print("boot2")

	Pio := pio.PIO0

	offset, err := Pio.AddProgram(cosmic_unicornInstructions, cosmic_unicornOrigin)
	if err != nil {
		panic(err.Error())
	}
	println("Loaded program at", offset)

	sm := Pio.StateMachine(0)
	cfg := cosmic_unicornProgramDefaultConfig(offset)

	var pins_to_set uint32 = uint32(1)<<uint32(COLUMN_BLANK) | uint32(0b1111)<<uint32(ROW_BIT_0)
	sm.SetPinsMasked(pins_to_set, pins_to_set)
	sm.SetPindirsConsecutive(COLUMN_CLOCK, 8, true)

	// osr shifts right, autopull on, autopull threshold 32
	cfg.SetOutShift(true, true, 32)

	// configure out, set, and sideset pins
	cfg.SetOutPins(ROW_BIT_0, 4)
	cfg.SetSetPins(COLUMN_DATA, 3)
	cfg.SetSidesetPins(COLUMN_CLOCK)

	// join fifos as only tx needed (gives 8 deep fifo instead of 4)
	cfg.SetFIFOJoin(pio.FifoJoinTx)

	// piolib.NewParallel8Tx()

	// DMA???

	sm.Init(offset, cfg)
	sm.SetEnabled(true)

	p := [72]uint8{
		64 - 1, // 8
		1,      // 8
		0b1,    // 8
		0b10,   // 8
		0b100,
		// yolo
		// 16
	}
	frame := 1
	var bcd_ticks uint32 = (1 << frame)
	p[68] = 0   // (bcd_ticks & 0xff) >> 0
	p[69] = 255 // (bcd_ticks & 0xff00) >> 8
	p[70] = 255 // (bcd_ticks & 0xff0000) >> 16
	p[71] = 0   // (bcd_ticks & 0xff000000) >> 24
	_ = bcd_ticks

	// 32

	for {
		time.Sleep(time.Microsecond * 100)
		for i := 0; i < 18; i++ {

			var v uint32
			v = v | (uint32(p[i*4+0]) << 0)
			v = v | (uint32(p[i*4+1]) << 8)
			v = v | (uint32(p[i*4+2]) << 16)
			v = v | (uint32(p[i*4+3]) << 24)

			println(i, v)
			fmt.Printf("%032b\n", v)

			for {
				if sm.IsTxFIFOFull() {
					println("full")
					time.Sleep(time.Second)
					continue
				} else {
					break
				}
			}

			sm.TxPut(v)
		}
	}
}

//go:nobounds
func (c *CosmicUnicorn) Draw() {
	for frame := uint8(0); frame < BCD_FRAME_COUNT; frame++ {
		for row := 0; row < 16; row++ {

			ROW_BIT_0.Set(row&0b1 == 0b1)
			ROW_BIT_1.Set(row&0b10 == 0b10)
			ROW_BIT_2.Set(row&0b100 == 0b100)
			ROW_BIT_3.Set(row&0b1000 == 0b1000)

			for idx := 0; idx < FRAME_COL_SIZE; idx++ {
				COLUMN_DATA.Set(false)
				b := c.bitstream[frame][row*FRAME_COL_SIZE+idx]
				if b {
					COLUMN_DATA.Set(true)
				}

				COLUMN_CLOCK.Set(true)
				time.Sleep(1)
				COLUMN_CLOCK.Set(false)
			}

			time.Sleep(1)

			COLUMN_LATCH.Set(true) // latch high, blank high
			COLUMN_BLANK.Set(true)

			time.Sleep(1)

			COLUMN_BLANK.Set(false) // blank low (enable output)
			COLUMN_LATCH.Set(false)
			COLUMN_DATA.Set(false)

			time.Sleep(time.Microsecond * 50)

			COLUMN_BLANK.Set(true) // blank high (disable output)
			COLUMN_LATCH.Set(false)
			COLUMN_DATA.Set(false)

		}
	}
}

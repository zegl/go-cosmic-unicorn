package cosmic

import (
	"fmt"
	"image/color"
	"machine"
	"time"
)

const WIDTH = 32
const HEIGHT = 32

// Assign all GPIO pins
const COLUMN_CLOCK = machine.GP13
const COLUMN_DATA = machine.GP14
const COLUMN_LATCH = machine.GP15
const COLUMN_BLANK = machine.GP16

const ROW_BIT_0 = machine.GP17
const ROW_BIT_1 = machine.GP18
const ROW_BIT_2 = machine.GP19
const ROW_BIT_3 = machine.GP20

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
const FRAME_COUNT = 2 // TODO: Support even more colors!
const FRAME_COL_SIZE = 32 * 2 * 3

type CosmicUnicorn struct {
	frames     [FRAME_COUNT][16 * FRAME_COL_SIZE]bool
	brightness uint8
}

func (c *CosmicUnicorn) clear() {
	for frame := 0; frame < FRAME_COUNT; frame++ {
		for idx := 0; idx < 16*FRAME_COL_SIZE; idx++ {
			c.frames[frame][idx] = false
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

	for frame := 0; frame < FRAME_COUNT; frame++ {
		c.frames[frame][y*FRAME_COL_SIZE+(x*3+0)] = on(b, frame)
		c.frames[frame][y*FRAME_COL_SIZE+(x*3+1)] = on(g, frame)
		c.frames[frame][y*FRAME_COL_SIZE+(x*3+2)] = on(r, frame)
	}
}

func (c *CosmicUnicorn) SetColor(x, y int, col color.Color) {
	r, g, b, _ := col.RGBA()
	c.SetPixel(x, y, uint8(r), uint8(g), uint8(b))
}

func (c *CosmicUnicorn) rowSleepDuration() time.Duration {
	return time.Microsecond * 1000 / FRAME_COUNT / 255 * time.Duration(c.brightness)
}

func (c *CosmicUnicorn) ChangeBrightness(delta int) {
	b := int(c.brightness) + delta
	if b > 255 {
		b = 255
	}
	if b < 0 {
		b = 0
	}
	c.brightness = uint8(b)

	fmt.Printf("Updated brightness: brightness=%d rowSleepDuration=%s\n", c.brightness, c.rowSleepDuration())
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

	time.Sleep(100 * time.Millisecond)

	COLUMN_BLANK.Set(false)

}

var i uint64

func tick() {
	i++
}

//go:nobounds
func (c *CosmicUnicorn) Draw() {
	rowSleepDuration := c.rowSleepDuration()

	for frame := uint8(0); frame < FRAME_COUNT; frame++ {
		for row := 0; row < ROW_COUNT; row++ {

			ROW_BIT_0.Set(row&0b1 == 0b1)
			ROW_BIT_1.Set(row&0b10 == 0b10)
			ROW_BIT_2.Set(row&0b100 == 0b100)
			ROW_BIT_3.Set(row&0b1000 == 0b1000)

			for idx := 0; idx < FRAME_COL_SIZE; idx++ {
				COLUMN_DATA.Set(false)
				b := c.frames[frame][row*FRAME_COL_SIZE+idx]
				if b {
					COLUMN_DATA.Set(true)
				}

				COLUMN_CLOCK.Set(true)
				tick()
				COLUMN_CLOCK.Set(false)
			}

			tick()

			COLUMN_LATCH.Set(true) // latch high, blank high
			COLUMN_BLANK.Set(true)

			tick()

			COLUMN_BLANK.Set(false) // blank low (enable output)
			COLUMN_LATCH.Set(false)
			COLUMN_DATA.Set(false)

			// Brightness is correlated with how long the LEDs are turned on before turning them off.
			// Based on testing. The maximum "on" time before flickering seems to be
			// around 1000µs when rendering with one frame.
			time.Sleep(rowSleepDuration)

			COLUMN_BLANK.Set(true) // blank high (disable output)
			COLUMN_LATCH.Set(false)
			COLUMN_DATA.Set(false)
		}
	}
}

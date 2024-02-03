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
const FRAME_COUNT = 8 // TODO: Support even more colors!!?!
const FRAME_COL_SIZE = 32 * 2 * 3

var GAMMA_8BIT = [256]uint8{
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 2, 2, 2,
	2, 2, 2, 3, 3, 3, 3, 3, 4, 4, 4, 4, 5, 5, 5, 5,
	6, 6, 6, 7, 7, 7, 8, 8, 8, 9, 9, 9, 10, 10, 11, 11,
	11, 12, 12, 13, 13, 13, 14, 14, 15, 15, 16, 16, 17, 17, 18, 18,
	19, 19, 20, 21, 21, 22, 22, 23, 23, 24, 25, 25, 26, 27, 27, 28,
	29, 29, 30, 31, 31, 32, 33, 34, 34, 35, 36, 37, 37, 38, 39, 40,
	40, 41, 42, 43, 44, 45, 46, 46, 47, 48, 49, 50, 51, 52, 53, 54,
	55, 56, 57, 58, 59, 60, 61, 62, 63, 64, 65, 66, 67, 68, 69, 70,
	71, 72, 73, 74, 76, 77, 78, 79, 80, 81, 83, 84, 85, 86, 88, 89,
	90, 91, 93, 94, 95, 96, 98, 99, 100, 102, 103, 104, 106, 107, 109, 110,
	111, 113, 114, 116, 117, 119, 120, 121, 123, 124, 126, 128, 129, 131, 132, 134,
	135, 137, 138, 140, 142, 143, 145, 146, 148, 150, 151, 153, 155, 157, 158, 160,
	162, 163, 165, 167, 169, 170, 172, 174, 176, 178, 179, 181, 183, 185, 187, 189,
	191, 193, 194, 196, 198, 200, 202, 204, 206, 208, 210, 212, 214, 216, 218, 220,
	222, 224, 227, 229, 231, 233, 235, 237, 239, 241, 244, 246, 248, 250, 252, 255}

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

	r32 := (uint32(r) * uint32(c.brightness)) >> 8
	g32 := (uint32(g) * uint32(c.brightness)) >> 8
	b32 := (uint32(b) * uint32(c.brightness)) >> 8

	gammaR := GAMMA_8BIT[r32]
	gammaG := GAMMA_8BIT[g32]
	gammaB := GAMMA_8BIT[b32]

	for frame := 0; frame < FRAME_COUNT; frame++ {
		c.frames[frame][y*FRAME_COL_SIZE+(x*3+0)] = gammaB&0b1 == 1
		c.frames[frame][y*FRAME_COL_SIZE+(x*3+1)] = gammaG&0b1 == 1
		c.frames[frame][y*FRAME_COL_SIZE+(x*3+2)] = gammaR&0b1 == 1

		gammaR = gammaR >> 1
		gammaG = gammaG >> 1
		gammaB = gammaB >> 1

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
			bcd_ticks := (1 << (frame + 3)) + 1
			for k := 0; k < bcd_ticks; k++ {
				tick()
			}

			COLUMN_BLANK.Set(true) // blank high (disable output)
			COLUMN_LATCH.Set(false)
			COLUMN_DATA.Set(false)
		}
	}
}

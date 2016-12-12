/// Author: Bernhard Tittelbach, btittelbach@github  (c) 2014

package bbhw

import (
	"errors"
	"fmt"
	"golang.org/x/sys/unix"
	"os"
)

// Uses the /sys/class/gpio/**/* file-interface provided by the linux kernel.
// Slightly slower than mmapped implementations but will work on any linux system with GPIOs.
type SysfsGPIO struct {
	Number uint
	fd     *os.File
}

// Constants for GPIO edge callbacks through sysfs.
const (
	RISING = iota
	FALLING
	BOTH
	NONE
)

// SysFS managed GPIO ------------------------------------

// Instantinate a new GPIO to control through sysfs. Takes GPIO numer (same as in sysfs) and direction bbhw.IN or bbhw.OUT
//
// See http://kilobaser.com/blog/2014-07-15-beaglebone-black-gpios#1gpiopin regarding the numbering of GPIO pins.
func NewSysfsGPIO(number uint, direction int) (gpio *SysfsGPIO, err error) {
	gpio = new(SysfsGPIO)
	gpio.Number = number

	if err := gpio.enable_export(); err != nil {
		return nil, err
	}
	err = gpio.SetDirection(direction)
	if err != nil {
		return nil, err
	}
	//check if file really exists and open for OUT
	gpio.fd, err = os.OpenFile(fmt.Sprintf("/sys/class/gpio/gpio%d/value", gpio.Number), os.O_RDWR|os.O_SYNC, 0666)
	if err != nil {
		return nil, err
	}
	return gpio, nil
}

// Wrapper around NewSysfsGPIO. Does not return an error but panics instead. Useful to avoid multiple return values.
// This is the function with the same signature as all the other New*GPIO*s
func NewSysfsGPIOOrPanic(number uint, direction int) (gpio *SysfsGPIO) {
	gpio, err := NewSysfsGPIO(number, direction)
	if err != nil {
		panic(err)
	}
	return gpio
}

func (gpio *SysfsGPIO) ReOpen() (err error) {
	if gpio == nil || gpio.fd == nil {
		return fmt.Errorf("gpio is nil")
	}
	prevfd := gpio.fd
	gpio.fd, err = os.OpenFile(gpio.fd.Name(), os.O_RDWR|os.O_SYNC, 0666)
	if err != nil {
		return
	}
	prevfd.Close()
	return nil
}

func (gpio *SysfsGPIO) enable_export() error {
	if gpio == nil {
		panic("gpio == nil")
	}
	_, err := os.Stat(fmt.Sprintf("/sys/class/gpio/gpio%d", gpio.Number))
	if err == nil {
		// already exported
		return nil
	} else if err != nil && !os.IsNotExist(err) {
		// some other error
		return err
	}
	fd, err := os.OpenFile("/sys/class/gpio/export", os.O_WRONLY|os.O_SYNC, 0666)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(fd, "%d\n", gpio.Number)
	return err
}

func (gpio *SysfsGPIO) CheckDirection() (direction int, err error) {
	var df *os.File
	var n int
	err = nil
	direction = -1
	if gpio == nil {
		panic("gpio == nil")
	}
	filename := fmt.Sprintf("/sys/class/gpio/gpio%d/direction", gpio.Number)
	df, err = os.OpenFile(filename, os.O_RDONLY|os.O_SYNC, 0666)
	if err != nil {
		return
	}
	defer df.Close()
	buf := make([]byte, 16)
	df.Seek(0, 0)
	n, err = df.Read(buf) //go knows how long our buf is, right ??
	if err != nil {
		return
	}
	if n == 0 {
		err = errors.New("wtf ?")
		return
	}
	if string(buf)[0:2] == "in" {
		direction = IN
	} else if string(buf)[0:3] == "out" {
		direction = OUT
	} else {
		err = fmt.Errorf("direction '%s' is neither in nor out !!!", buf)
	}
	return
}

func (gpio *SysfsGPIO) GetEdge() (edge string, err error) {
	var df *os.File
	var n int
	err = nil
	edge = ""
	if gpio == nil {
		panic("gpio == nil")
	}
	filename := fmt.Sprintf("/sys/class/gpio/gpio%d/edge", gpio.Number)
	df, err = os.OpenFile(filename, os.O_RDONLY|os.O_SYNC, 0666)
	if err != nil {
		return
	}
	defer df.Close()
	buf := make([]byte, 16)
	df.Seek(0, 0)
	n, err = df.Read(buf) //go knows how long our buf is, right ??
	if err != nil {
		return
	}
	if n == 0 {
		err = errors.New("Edge file is empty")
	}
	if string(buf)[0:4] == "none" {
		edge = "none"
	} else if string(buf)[0:4] == "both" {
		edge = "both"
	} else if string(buf)[0:6] == "rising" {
		edge = "rising"
	} else if string(buf)[0:7] == "falling" {
		edge = "falling"
	} else {
		err = errors.New("Invalid edge state")
	}
	return
}

func (gpio *SysfsGPIO) SetDirection(direction int) error {
	if gpio == nil {
		panic("gpio == nil")
	}
	df, err := os.OpenFile(fmt.Sprintf("/sys/class/gpio/gpio%d/direction", gpio.Number),
		os.O_WRONLY|os.O_SYNC, 0666)
	if err != nil {
		return err
	}
	defer df.Close()
	if direction == OUT {
		fmt.Fprintln(df, "out")
	} else {
		fmt.Fprintln(df, "in")
	}
	return nil
}

//this inverts the meaning of 0 and 1 in /sys/class/gpio/gpio*/value
func (gpio *SysfsGPIO) SetActiveLow(activelow bool) error {
	if gpio == nil {
		panic("gpio == nil")
	}
	df, err := os.OpenFile(fmt.Sprintf("/sys/class/gpio/gpio%d/active_low", gpio.Number),
		os.O_WRONLY|os.O_SYNC, 0666)
	if err != nil {
		return err
	}
	defer df.Close()
	if activelow {
		fmt.Fprintln(df, "1")
	} else {
		fmt.Fprintln(df, "0")
	}
	return nil
}

func (gpio *SysfsGPIO) SetEdge(edge int) error {
	if gpio == nil {
		panic("gpio == nil")
	}
	df, err := os.OpenFile(fmt.Sprintf("/sys/class/gpio/gpio%d/edge", gpio.Number),
		os.O_WRONLY|os.O_SYNC, 0666)
	if err != nil {
		return err
	}
	defer df.Close()
	if edge == RISING {
		fmt.Fprintln(df, "rising")
	} else if edge == FALLING {
		fmt.Fprintln(df, "falling")
	} else if edge == BOTH {
		fmt.Fprintln(df, "both")
	} else if edge == NONE {
		fmt.Fprintln(df, "none")
	} else {
		return errors.New("Edge value invalid")
	}
	return nil
}

// Monitor pin using Unix Poll with a specified timeout (negative value for infinite timeout)
func (gpio *SysfsGPIO) SetEdgeCallback(callback *chan bool, timeout int) error {
	if gpio == nil {
		panic("gpio == nil")
	}
	edge, err := gpio.GetEdge()
	if err != nil {
		return err
	}
	if edge == "none" {
		err = errors.New("Edge value is set to NONE")
		return err
	}
	go func() {
		defer close(*callback)

		for {
			//First do a dummy read before we poll
			gpio.GetState()
			fds := []unix.PollFd{{Fd: int32(gpio.fd.Fd()), Events: unix.POLLPRI}}
			_, err := unix.Poll(fds, timeout)
			if err != nil {
				break
			}
			state, err := gpio.GetState()
			if err != nil {
				break
			}
			*callback <- state
		}
	}()
	return nil
}

func (gpio *SysfsGPIO) GetState() (state bool, err error) {
	if gpio == nil {
		panic("gpio == nil")
	}
	var n int
	if gpio.fd == nil {
		panic("gpio.fd == nil")
	}
	_, err = gpio.fd.Seek(0, 0)

	// if err = gpio.ReOpen(); err != nil {
	// 	return
	// }
	buf := make([]byte, 16)
	n, err = gpio.fd.Read(buf) //go knows how long our buffer is, right ??
	if err != nil {
		return
	}
	if n != 2 {
		err = errors.New("wtf ?")
		return
	}
	if buf[0] == '1' {
		state = true
	} else {
		state = false
	}
	return
}

func (gpio *SysfsGPIO) SetState(state bool) error {
	if gpio == nil || gpio.fd == nil {
		panic("gpio == nil")
	}
	v := "0"
	if state {
		v = "1"
	}
	gpio.fd.Truncate(0)
	_, err := fmt.Fprintln(gpio.fd, v)
	return err
}

func (gpio *SysfsGPIO) SetStateNow(state bool) error { return gpio.SetState(state) }

//closes filedescriptor
//does NOT unexport gpio, since gpio_mmap_collection and gpio_mmap depend on the gpio remaining exported and the gpiobank activated
func (gpio *SysfsGPIO) Close() {
	gpio.fd.Close()
	gpio = nil
}

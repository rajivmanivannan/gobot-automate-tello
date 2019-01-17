/*
To automated the DJI Tello Drone using Gobot Framework
*/
package main

import (
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"time"

	"gobot.io/x/gobot"
	"gobot.io/x/gobot/platforms/dji/tello"
	"gocv.io/x/gocv"
)

const (
	frameX    = 960
	frameY    = 720
	frameSize = frameX * frameY * 3
)

func main() {
	// Driver: Tello Driver
	drone := tello.NewDriver("8890")

	// OpenCV window to watch the live video stream from Tello.
	window := gocv.NewWindow("Tello")

	//FFMPEG to convert live video from drone and show in the supported video format in window.
	ffmpeg := exec.Command("ffmpeg", "-hwaccel", "auto", "-hwaccel_device", "opencl", "-i", "pipe:0",
		"-pix_fmt", "bgr24", "-s", strconv.Itoa(frameX)+"x"+strconv.Itoa(frameY), "-f", "rawvideo", "pipe:1")
	ffmpegIn, _ := ffmpeg.StdinPipe()
	ffmpegOut, _ := ffmpeg.StdoutPipe()

	work := func() {
		//Starting FFMPEG.
		if err := ffmpeg.Start(); err != nil {
			fmt.Println(err)
			return
		}
		// Event: Listening the drone connect event to start the video streaming from drone.
		drone.On(tello.ConnectedEvent, func(data interface{}) {
			fmt.Println("Connected to Tello.")
			drone.StartVideo()
			drone.SetVideoEncoderRate(tello.VideoBitRateAuto)
			drone.SetExposure(0)

			//For continued streaming of video.
			gobot.Every(100*time.Millisecond, func() {
				drone.StartVideo()
			})
		})

		//Event: Piping the video data from drone into the FFMPEG library installed in mac.
		drone.On(tello.VideoFrameEvent, func(data interface{}) {
			pkt := data.([]byte)
			if _, err := ffmpegIn.Write(pkt); err != nil {
				fmt.Println(err)
			}
		})

		//TakeOff the Drone.
		gobot.After(5*time.Second, func() {
			drone.TakeOff()
			fmt.Println("Tello Taking Off...")
		})

		//Land the Drone.
		gobot.After(15*time.Second, func() {
			drone.Land()
			fmt.Println("Tello Landing...")
		})
	}
	//Robot: Tello (Drone)
	robot := gobot.NewRobot("tello",
		[]gobot.Connection{},
		[]gobot.Device{drone},
		work,
	)

	// calling Start(false) lets the Start routine return immediately without an additional blocking goroutine
	robot.Start(false)

	// now handle video frames from ffmpeg stream in main thread, to be macOs friendly
	for {
		buf := make([]byte, frameSize)
		if _, err := io.ReadFull(ffmpegOut, buf); err != nil {
			fmt.Println(err)
			continue
		}
		img, _ := gocv.NewMatFromBytes(frameY, frameX, gocv.MatTypeCV8UC3, buf)
		if img.Empty() {
			continue
		}

		window.IMShow(img)
		if window.WaitKey(1) >= 0 {
			break
		}
	}
}

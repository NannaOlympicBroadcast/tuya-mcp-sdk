package main

import (
	"mcp-sdk/examples/mcp"

	_ "github.com/pion/mediadevices/pkg/driver/camera"
)

func main() {
	println("Recording photo...")
	photoPath, err := mcp.TakePhoto("test")
	if err != nil {
		println("failed to take photo: ", err)
		return
	}
	println("Photo taken successfully: ", photoPath)
	println("Opening photo...")
	err = mcp.OpenPhoto(photoPath)
	if err != nil {
		println("failed to open photo: ", err)
		return
	}
	println("Photo opened successfully")
}

const (
	photoPath = "photo"
	// width     = 1920
	// height    = 1080
)

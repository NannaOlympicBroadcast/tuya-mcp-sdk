package mcp

import (
	"context"
	"errors"
	"fmt"
	"image/jpeg"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/pion/mediadevices"
	_ "github.com/pion/mediadevices/pkg/driver/camera"
	"github.com/pion/mediadevices/pkg/prop"
)

const (
	_photoPath = "static/photo"
)

type Photo struct {
}

func (t *Photo) Register(mcpServer *server.MCPServer) {
	mcpServer.AddTool(
		mcp.NewTool("take_photo",
			mcp.WithDescription("Take a photo"),
			mcp.WithString("name",
				mcp.Description("The name of the photo; e.g. 'photo1'"),
			),
			mcp.WithBoolean("is_view",
				mcp.Description("Whether to view the photo after taking it; e.g. 'true'"),
			),
		),
		handleTakePhotoTool,
	)

	mcpServer.AddTool(
		mcp.NewTool("view_photo",
			mcp.WithDescription("View a photo"),
			mcp.WithString("name",
				mcp.Description("The name of the photo; e.g. 'photo1'"),
				mcp.Required(),
			),
		),
		handleViewPhotoTool,
	)
}

func handleTakePhotoTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := request.GetString("name", time.Now().Format("20060102150405"))

	isView := request.GetBool("is_view", false)

	photoPath, err := TakePhoto(name)
	if err != nil {
		return nil, fmt.Errorf("failed to take photo: %v", err)
	}

	respText := fmt.Sprintf("Photo taken successfully: %s", name)

	if isView {
		err := OpenPhoto(photoPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open photo: %v", err)
		}
		respText = fmt.Sprintf("Photo taken successfully and viewed: %s", name)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{Type: "text", Text: respText},
		},
	}, nil
}

func handleViewPhotoTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := request.GetString("name", "")
	if name == "" {
		return nil, fmt.Errorf("missing name parameter")
	}

	dir := _photoPath
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read photo directory: %v", err)
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if strings.HasPrefix(file.Name(), name) {
			photoPath := fmt.Sprintf("%s/%s", dir, file.Name())
			err := OpenPhoto(photoPath)
			if err != nil {
				return nil, fmt.Errorf("failed to open photo: %v", err)
			}
			break
		}
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{Type: "text", Text: "Photo viewed successfully"},
		},
	}, nil
}

func TakePhoto(name string) (path string, err error) {
	defer func() {
		if r := recover(); r != nil {
			println("panic: ", r)
			err = errors.New("主人我手抖了，没有拍到，你再摆个Pose吧")
		}
	}()
	stream, err := mediadevices.GetUserMedia(mediadevices.MediaStreamConstraints{
		Video: func(constraint *mediadevices.MediaTrackConstraints) {
			// Query for ideal resolutions
			constraint.Width = prop.Int(1024)
			constraint.Height = prop.Int(768)
		},
	})
	if err != nil {
		println("failed to get user media: ", err.Error())
		return "", fmt.Errorf("failed to get user media: %v", err)
	}

	// Since track can represent audio as well, we need to cast it to
	// *mediadevices.VideoTrack to get video specific functionalities
	track := stream.GetVideoTracks()[0]
	videoTrack := track.(*mediadevices.VideoTrack)
	defer videoTrack.Close()

	// Create a new video reader to get the decoded frames. Release is used
	// to return the buffer to hold frame back to the source so that the buffer
	// can be reused for the next frames.
	videoReader := videoTrack.NewReader(false)
	frame, release, err := videoReader.Read()
	defer release()
	// Since frame is the standard image.Image, it's compatible with Go standard
	// library. For example, capturing the first frame and store it as a jpeg image.
	if _, err := os.Stat(_photoPath); err == nil {
		os.Remove(_photoPath)
	}
	os.MkdirAll(_photoPath, 0755)
	photoPath := fmt.Sprintf("%s/%s_%s.jpg", _photoPath, name, time.Now().Format("20060102150405"))
	output, err := os.Create(photoPath)
	if err != nil {
		println("failed to create photo: ", err)
		return "", fmt.Errorf("failed to create photo: %v", err)
	}
	defer output.Close()
	jpeg.Encode(output, frame, nil)

	return photoPath, nil
}

func OpenPhoto(path string) error {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", path}
	default: // Linux
		cmd = "xdg-open"
	}
	if len(args) == 0 {
		args = append(args, path)
	}
	return exec.Command(cmd, args...).Start()
}

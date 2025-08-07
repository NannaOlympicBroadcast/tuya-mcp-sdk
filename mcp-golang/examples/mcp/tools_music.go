package mcp

import (
	"context"
	"errors"
	"fmt"
	"log"
	"mcp-sdk/pkg/utils"
	"os"
	"strings"
	"sync"

	"github.com/hajimehoshi/oto/v2"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/hajimehoshi/go-mp3"
)

type cmd string

const (
	cmdPlay cmd = "play"
	cmdStop cmd = "stop"
)

const (
	_musicPath = "static/music"
)

var music *Music

func init() {
	music = newMusic()
}

type Music struct {
	path        string
	c           *oto.Context
	mu          sync.RWMutex
	currentSong string
	isPlaying   bool
	player      oto.Player
	cmd         chan cmd
}

func (m *Music) Register(mcpServer *server.MCPServer) {
	// Music tool
	mcpServer.AddTool(
		mcp.NewTool("play_music",
			mcp.WithDescription("Play music by music name"),
			mcp.WithString("name",
				mcp.Description("Music name; e.g. 'classic'"),
			),
		),
		handleMusicTool,
	)

	// Stop music tool
	mcpServer.AddTool(
		mcp.NewTool("stop_music",
			mcp.WithDescription("Stop playing music"),
		),
		handleStopMusicTool,
	)
}

// handleMusicTool handles music tool
func handleMusicTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	musicName := request.GetString("name", "classic")

	utils.Go(func() {
		music := GetMusic()
		err := music.Play(musicName)
		if err != nil {
			log.Println("error playing music", err)
		}
	})

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Playing music: %s", musicName),
			},
		},
	}, nil
}

// handleStopMusicTool handles stop music tool
func handleStopMusicTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	music := GetMusic()
	music.Stop()

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: "Music stopped",
			},
		},
	}, nil
}

func newMusic() *Music {
	op := &oto.NewContextOptions{}
	op.SampleRate = 44100
	op.ChannelCount = 2
	op.Format = oto.FormatSignedInt16LE

	c, ready, err := oto.NewContextWithOptions(op)
	if err != nil {
		panic(err)
	}

	<-ready

	music := &Music{
		path: _musicPath,
		c:    c,
		cmd:  make(chan cmd, 1),
	}

	utils.Go(music.loop)
	return music
}

func (m *Music) loop() {
	for cmd := range m.cmd {
		switch cmd {
		case cmdStop:
			m.stop()
		case cmdPlay:
			if !m.IsPlaying() && !m.player.IsPlaying() {
				m.play()
			}
		}
	}
}

func GetMusic() *Music {
	return music
}

func (m *Music) stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	// 关闭播放器
	if m.player != nil {
		err := m.player.Close()
		if err != nil {
			fmt.Println("error closing player", err)
		}
		m.player = nil
	}

	m.isPlaying = false
	fmt.Println("music stopped")
}

func (m *Music) play() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.player == nil {
		fmt.Println("music player is nil")
		return
	}

	m.player.Play()
	m.isPlaying = true
	fmt.Println("music playing")
}

func (m *Music) Stop() {
	m.cmd <- cmdStop
}

func (m *Music) Play(musicName string) error {
	defer func() {
		if r := recover(); r != nil {
			log.Println("panic in Play", r)
		}
	}()

	if m.IsPlaying() {
		m.stop()
	}

	// 查找音乐文件
	musicPath, err := m.lookupMusic(musicName)
	if err != nil {
		return err
	}

	// 打开文件
	f, err := os.Open(musicPath)
	if err != nil {
		return err
	}

	// 创建解码器
	d, err := mp3.NewDecoder(f)
	if err != nil {
		f.Close()
		return err
	}

	// 创建播放器
	player := m.c.NewPlayer(d)

	m.player = player

	fmt.Printf("Playing: %s, Length: %d[bytes]\n", musicName, d.Length())
	m.cmd <- cmdPlay
	return nil
}

// 获取当前播放状态
func (m *Music) IsPlaying() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isPlaying
}

// 获取当前歌曲
func (m *Music) GetCurrentSong() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentSong
}

func (m *Music) lookupMusic(musicName string) (string, error) {
	dir := m.path
	files, err := os.ReadDir(dir)
	if err != nil {
		dir = "examples/" + dir
		files, err = os.ReadDir(dir)
		if err != nil {
			return "", errors.New("failed to read directory")
		}
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if strings.Contains(strings.ToLower(file.Name()), strings.ToLower(musicName)) {
			return fmt.Sprintf("%s/%s", dir, file.Name()), nil
		}
	}
	return "", errors.New("music not found")
}

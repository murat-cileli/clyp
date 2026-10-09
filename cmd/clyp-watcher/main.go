package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

const appID = "bio.murat.clyp"

var (
	activeClipboardHandler func(string, []byte)
	memoryReleaseMu        sync.Mutex
	memoryReleaseTimer     *time.Timer
)

func main() {
	lock, alreadyRunning, err := acquireInstanceLock()
	if err != nil {
		log.Fatal(err)
	}
	if alreadyRunning {
		return
	}
	defer lock.Close()

	history, err := openHistory()
	if err != nil {
		log.Fatal(err)
	}
	defer history.Close()
	debug.FreeOSMemory()

	activeClipboardHandler = func(mimeType string, data []byte) {
		if err := storeClipboardData(history, mimeType, data); err != nil {
			log.Printf("Failed to save clipboard content: %v", err)
		}
	}

	if os.Getenv("WAYLAND_DISPLAY") != "" {
		log.Println("Trying native Wayland clipboard backend.")
		err = runWayland()
		if err == nil {
			return
		}
		if !errors.Is(err, errBackendUnavailable) {
			log.Printf("Wayland clipboard backend stopped: %v", err)
		}
		if os.Getenv("DISPLAY") == "" {
			log.Fatal("Wayland compositor does not provide ext-data-control-v1 and XWayland is unavailable")
		}
		log.Println("Falling back to the X11/XWayland clipboard backend.")
	}

	if os.Getenv("DISPLAY") == "" {
		log.Fatal("No supported display found")
	}

	log.Println("Using X11 clipboard backend.")
	if err := runX11(); err != nil {
		log.Fatal(err)
	}
}

func storeClipboardData(history *historyStore, mimeType string, data []byte) error {
	if len(data) == 0 {
		return nil
	}

	var content string
	var itemType byte
	isImage := strings.HasPrefix(strings.ToLower(mimeType), "image/")
	if isImage {
		content = base64.StdEncoding.EncodeToString(data)
		itemType = 2
	} else {
		content = strings.TrimSpace(strings.ToValidUTF8(string(data), ""))
		itemType = 1
	}

	if content == "" {
		return nil
	}
	if err := history.Save(content, itemType); err != nil {
		return fmt.Errorf("store %s: %w", mimeType, err)
	}
	if isImage || len(data) >= 1<<20 {
		scheduleMemoryRelease()
	}
	return nil
}

func scheduleMemoryRelease() {
	memoryReleaseMu.Lock()
	defer memoryReleaseMu.Unlock()
	if memoryReleaseTimer != nil {
		memoryReleaseTimer.Stop()
	}
	memoryReleaseTimer = time.AfterFunc(250*time.Millisecond, debug.FreeOSMemory)
}

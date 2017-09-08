package main

import (
	"context"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"

	_ "net/http/pprof"

	"github.com/getlantern/systray"
)

func main() {
	go func() {
		log.Println(http.ListenAndServe(":6060", nil))
	}()

	systray.Run(onReady, onExit)
}

// UI

func onExit() {
	log.Println("onExit")
}

func onReady() {
	systray.SetTitle("Pb")
	systray.SetTooltip("Pasteboard Tooltip")

	// main logic

	var item string
	var err error
	var size, idx int

	size = 20
	idx = 0
	store := make([]string, size)
	menu := make([]*systray.MenuItem, size)
	texts := make(chan string)
	errs := make(chan error)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go watchClipboard(ctx, texts, errs)

	for {
		select {
		case <-ctx.Done():
			if err = ctx.Err(); err != nil {
				log.Fatal(err)
				return
			}
			log.Println("ctx done")
			return
		case err = <-errs:
			if err != nil {
				return
			}
		case item = <-texts:
			if uniqueText(item, store) {
				store[idx] = item
				addItemToMenu(idx, store, menu)
				handleIndex(&idx, size)
			}
		}
	}
}

func addItemToMenu(i int, store []string, menu []*systray.MenuItem) {
	if menu[i] == nil {
		menu[i] = createTrayBtn(store[i])
		go listenMenuChecked(i, menu[i], store[i])
	} else {
		menu[i].SetTitle(getTitle(store[i]))
		menu[i].SetTooltip(store[i])
		go listenMenuChecked(i, menu[i], store[i])
	}
}

func listenMenuChecked(i int, menuItem *systray.MenuItem, text string) {
	select {
	case <-menuItem.ClickedCh:
		writePasteBoard(text)
		go listenMenuChecked(i, menuItem, text)
		return
	}
}

func getTitle(item string) (title string) {
	if len(item) > 20 {
		title = item[:20] + "..."
	} else {
		title = item
	}
	title = strings.TrimSpace(title)
	title = strings.Replace(title, "\n", " ", -1)
	return
}

func createTrayBtn(item string) *systray.MenuItem {
	return systray.AddMenuItem(getTitle(item), item)
}

// Backend

// handleIndex handles the current index of store.
func handleIndex(idx *int, size int) {
	*idx++
	if *idx >= size {
		*idx = 0
	}
}

// uniqueText checks if the value of s is in the store.
func uniqueText(s string, store []string) bool {
	for i := range store {
		if store[i] == s {
			return false
		}
	}
	return true
}

// watchClipboard runs forever watching the pastboard.
func watchClipboard(ctx context.Context, out chan<- string, errs chan<- error) {
	for {
		text, err := readPasteBoard()
		if err != nil {
			errs <- err
		}

		if len(text) > 0 {
			out <- text
		}

		time.Sleep(2 * time.Second)
	}
}

// readClipboard reads the current value of the pastboard.
func readPasteBoard() (string, error) {
	cmd := exec.Command("pbpaste")
	b, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// writePasteBoard fills out the pasteboard with a string.
func writePasteBoard(s string) {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(s)
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

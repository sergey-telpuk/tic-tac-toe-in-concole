package main

import (
	"fmt"
	"github.com/jroimartin/gocui"
	"log"
	"math/rand"
	"strconv"
	"time"
)

const (
	Red     = "\033[1;31m%2s\033[0m"
	Green   = "\033[1;32m%s\033[0m"
	Yellow  = "\033[1;33m%s\033[0m"
	Purple  = "\033[1;34m%s\033[0m"
	Magenta = "\033[1;35m%s\033[0m"
	Teal    = "\033[1;36m%s\033[0m"
	White   = "\033[1;37m%s\033[0m"

	YOU = "YOU"
	BOT = "BOT"
)

var views = map[string][]int{
	"1": {0, 0, 8, 3},
	"2": {9, 0, 17, 3},
	"3": {18, 0, 26, 3},

	"4": {0, 4, 8, 7},
	"5": {9, 4, 17, 7},
	"6": {18, 4, 26, 7},

	"7": {0, 8, 8, 11},
	"8": {9, 8, 17, 11},
	"9": {18, 8, 26, 11},
}

var winCombinations = [][]int{
	{1, 2, 3},
	{4, 5, 6},
	{7, 8, 9},

	{1, 4, 7},
	{2, 5, 8},
	{3, 6, 9},

	{1, 5, 9},
	{3, 5, 7},
}

var currentGameX []int
var currentGameO []int
var freeSteps = []int{1, 2, 3, 4, 5, 6, 7, 8, 9}
var winner = make(chan string)
var reset = make(chan bool)
var closed = make(chan bool)
var botStep = make(chan bool)
var userStep = make(chan bool)
var counterSteps = 0

func main() {
	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		log.Panicln(err)
	}
	defer g.Close()
	g.Mouse = true

	g.SetManagerFunc(layout)

	if err := keyBindings(g); err != nil {
		log.Panicln(err)
	}

	go winnerView(g)
	go whoFirstWindow(g)
	go resetView(g)
	go closeWindow(g)
	go listenBotStep(g)

	go func() {
		if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
			log.Panicln(err)
		}
	}()

	<-closed
}

func listenBotStep(g *gocui.Gui) {
	randV := func() *gocui.View {
		rand.Seed(time.Now().Unix()) // initialize global pseudo random generator
		value := freeSteps[rand.Intn(len(freeSteps))]
		itoa := strconv.Itoa(value)
		v, _ := g.View(itoa)
		return v
	}

	for {
		select {
		case <-botStep:
			time.Sleep(1 * time.Second)
			if len(freeSteps) == 0 {
				return
			}

			g.Update(func(gui *gocui.Gui) error {
				_ = handler(g, randV(), BOT)
				go func() {
					userStep <- true
				}()
				return nil
			})
		}
	}
}

func closeWindow(g *gocui.Gui) {
	v, _ := g.SetView("close", 30, 0, 33, 2)
	fmt.Fprintln(v, "X")

	_ = g.SetKeybinding("close", gocui.MouseLeft, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		closed <- true
		return nil
	})
}
func whoFirstWindow(g *gocui.Gui) {
	v, _ := g.SetView("whoFirstWindow", 40, 3, 56, 6)
	v.Title = "Who first?"

	botV, _ := g.SetView("botStep", 40, 4, 47, 6)
	fmt.Fprintln(botV, BOT)
	youV, _ := g.SetView("userStep", 49, 4, 56, 6)
	fmt.Fprintln(youV, YOU)

	_ = g.SetKeybinding("botStep", gocui.MouseLeft, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		g.DeleteKeybindings(botV.Name())
		g.DeleteKeybindings(youV.Name())
		v.Title = "1"
		go func() {
			botStep <- true
		}()
		return nil
	})

	_ = g.SetKeybinding("userStep", gocui.MouseLeft, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		g.Update(func(gui *gocui.Gui) error {
			g.DeleteKeybindings(botV.Name())
			g.DeleteKeybindings(youV.Name())
			v.Title = "1"
			go func() {
				userStep <- true
			}()
			return nil
		})

		return nil
	})
}

func winnerView(g *gocui.Gui) {
	for {
		who := <-winner
		g.Update(func(gui *gocui.Gui) error {
			g.DeleteView("winner")
			g.DeleteKeybindings("winner")
			v, _ := g.SetView("winner", 60, 3, 83, 5)
			fmt.Fprintln(v, "\033[1;32mWINNER:\033[0m \033[1;31m"+who+"\033[0m \033[1;32mTry again?\033[0m")

			_ = g.SetKeybinding("winner", gocui.MouseLeft, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
				resetGame(g)
				return nil
			})
			return nil
		})
	}
}

func resetView(g *gocui.Gui) {
	for {
		<-reset
		maxX, maxY := g.Size()

		g.Update(func(gui *gocui.Gui) error {
			v, _ := g.SetView("reset", maxX/2-7, maxY/2, maxX/2+20, maxY/2+2)

			fmt.Fprintln(v, "Click me for resetting!")

			_ = g.SetKeybinding("reset", gocui.MouseLeft, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
				resetGame(g)
				return nil
			})
			return nil
		})
	}
}
func resetGame(g *gocui.Gui) {
	for key, _ := range views {
		g.DeleteView(key)
		g.DeleteKeybindings(key)
	}
	g.DeleteView("reset")
	g.DeleteKeybindings("reset")
	g.DeleteView("winner")
	g.DeleteKeybindings("winner")

	g.DeleteView("whoFirstWindow")
	g.DeleteView("botStep")
	g.DeleteKeybindings("botStep")
	g.DeleteView("userStep")
	g.DeleteKeybindings("userStep")
	freeSteps = []int{1, 2, 3, 4, 5, 6, 7, 8, 9}
	counterSteps = 0
	currentGameX = nil
	currentGameO = nil
	go listenBotStep(g)
	go whoFirstWindow(g)
	if err := keyBindings(g); err != nil {
		log.Panicln(err)
	}
}

func layout(g *gocui.Gui) error {
	for key, values := range views {
		if v, err := g.SetView(key, values[0], values[1], values[2], values[3]); err != nil {
			if err != gocui.ErrUnknownView {
				return err
			}
			fmt.Fprintf(v, "")
		}
	}

	return nil
}

func keyBindings(g *gocui.Gui) error {

	for key, _ := range views {
		err := g.SetKeybinding(string(key), gocui.MouseLeft, gocui.ModNone, func(gui *gocui.Gui, view *gocui.View) error {

			select {
			case <-userStep:
				handler(gui, view, YOU)
				botStep <- true
			default:
				return nil
			}

			return nil
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func handler(g *gocui.Gui, v *gocui.View, who string) error {
	v.Clear()
	g.DeleteKeybindings(v.Name())
	v.Title = who

	s := v.Name()

	step, _ := strconv.Atoi(s)
	nextStep := string(stepper(step))

	color := func() string {
		if nextStep != "X" {
			return Yellow
		} else {
			return Red
		}
	}()

	_, _ = fmt.Fprintf(v, color, nextStep)
	if counterSteps >= 9 {
		reset <- true
		return nil
	}

	if w, ok := tryToFindWinner(step, nextStep); ok {
		winner <- w
		return nil
	}

	return nil
}

var step = 'X'

func stepper(currentStep int) rune {
	if step == 'X' {
		step = 'O'
	} else {
		step = 'X'
	}

	counterSteps++

	removeFromFreeSteps(currentStep)

	return step
}

func tryToFindWinner(cell int, step string) (string, bool) {
	check := func(candidate string, steps []int) (string, bool) {
		for _, winComb := range winCombinations {
			count := 0
			for _, item := range winComb {
				if !contains(steps, item) {
					break
				}
				count++
				if count == 3 {
					return candidate, true
				}
			}
		}
		return "", false
	}

	if step == "X" {
		currentGameX = append(currentGameX, cell)
		return check("X", currentGameX)
	}

	currentGameO = append(currentGameO, cell)
	return check("O", currentGameO)

}

func contains(a []int, x int) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}

func removeFromFreeSteps(s int) {
	for i, v := range freeSteps {
		if v == s {
			freeSteps = append(freeSteps[:i], freeSteps[i+1:]...)
			break
		}
	}
}

package main

import (
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/creack/pty"
	"github.com/hinshun/vt10x"
	"golang.org/x/exp/slices"
)

type DiscordTerminal struct {
	Running bool

	Bot *Bot

	OwnerUserID string

	Msg           *discordgo.Message
	Pty           *os.File
	Term          vt10x.Terminal
	CurrentScreen string
	LastScreen    string
}

func ParseCtrlSequences(ins string) string {
	match := regexp.MustCompile(`\^([A-Z])`).FindAllStringSubmatch(ins, -1)
	for _, m := range match {
		ac := []byte(m[1])[0]
		ins = strings.ReplaceAll(ins, m[0], string(ac-64))
	}
	return ins
}

func (term *DiscordTerminal) PTYUpdater() {
	for term.Running {
		data := make([]byte, 8192)
		_, err := term.Pty.Read(data)
		if err != nil {
			term.Bot.Session.ChannelMessageDelete(term.Msg.ChannelID, term.Msg.ID)
			term.Bot.Session.ChannelMessageSend(term.Msg.ChannelID, "Terminal dead")
			term.Running = false
			return
		}
		term.Term.Write(data)
		//fmt.Printf("%v\r", term.String())
		term.CurrentScreen = term.Term.String()
	}
}

func (term *DiscordTerminal) ScreenUpdater() {
	var err error
	for term.Running {
		if term.CurrentScreen != term.LastScreen {
			term.Msg, err = term.Bot.Session.ChannelMessageEdit(term.Msg.ChannelID, term.Msg.ID, "```\n"+term.CurrentScreen+"```")
			if err != nil {
				log.Printf("Cannot update terminal: %s", err)
			}
			term.LastScreen = term.CurrentScreen
		}
		time.Sleep(time.Second)
	}
}

func NewDiscordTerminal(bot *Bot, cid string, ownerid string) {
	var err error
	this := &DiscordTerminal{Bot: bot, OwnerUserID: ownerid}

	this.Term = vt10x.New(vt10x.WithSize(W, H))
	c := exec.Command(os.Getenv("SHELL"))
	c.Env = os.Environ()
	c.Env = append(c.Env, "TERM=vt100")
	this.Pty, err = pty.StartWithSize(c, &pty.Winsize{Rows: H, Cols: W, X: 0, Y: 0})
	if err != nil {
		this.Bot.Session.ChannelMessageSend(cid, "Cannot start terminal: "+err.Error())
		return
	}

	this.Msg, _ = this.Bot.Session.ChannelMessageSend(cid, "Waiting for pty...")
	bot.Terminals = append(bot.Terminals, this)
	this.Running = true

	go this.PTYUpdater()
	this.ScreenUpdater()

	idx := slices.IndexFunc(bot.Terminals, func(t *DiscordTerminal) bool { return t == this })
	if idx == -1 {
		panic("cannot remove terminal belonging to " + this.OwnerUserID)
	}
	bot.Terminals[idx] = bot.Terminals[len(bot.Terminals)-1]
	bot.Terminals = bot.Terminals[:len(bot.Terminals)-1]
}

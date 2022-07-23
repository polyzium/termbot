package main

import (
	"bytes"
	"fmt"
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

var (
	BOT_KEYS = [...]Key{
		{
			Name:   "LF [\\n]",
			InSeq:  "\\n",
			OutSeq: "\n",
		},
		{
			Name:   "CR (Enter) [\\r]",
			InSeq:  "\\r",
			OutSeq: "\r",
		},
		{
			Name:   "Backspace [\\b]",
			InSeq:  "\\b",
			OutSeq: "\b",
		},
		{
			Name:   "Tab [\\t]",
			InSeq:  "\\t",
			OutSeq: "\t",
		},
		{
			Name:   "Escape [ESC]",
			InSeq:  "[ESC]",
			OutSeq: "\x1b",
		},
		{
			Name:   "F1",
			InSeq:  "[F1]",
			OutSeq: "\x1bOP",
		},
		{
			Name:   "F2",
			InSeq:  "[F2]",
			OutSeq: "\x1bOQ",
		},
		{
			Name:   "F3",
			InSeq:  "[F3]",
			OutSeq: "\x1bOR",
		},
		{
			Name:   "F4",
			InSeq:  "[F4]",
			OutSeq: "\x1bOS",
		},
		{
			Name:   "F5",
			InSeq:  "[F5]",
			OutSeq: "\x1b[15~",
		},
		{
			Name:   "F6",
			InSeq:  "[F6]",
			OutSeq: "\x1b[17~",
		},
		{
			Name:   "F7",
			InSeq:  "[F7]",
			OutSeq: "\x1b[18~",
		},
		{
			Name:   "F8",
			InSeq:  "[F8]",
			OutSeq: "\x1b[19~",
		},
		{
			Name:   "F9",
			InSeq:  "[F9]",
			OutSeq: "\x1b[20~",
		},
		{
			Name:   "F10",
			InSeq:  "[F10]",
			OutSeq: "\x1b[21~",
		},
		{
			Name:   "F11",
			InSeq:  "[F11]",
			OutSeq: "\x1b[23~",
		},
		{
			Name:   "F12",
			InSeq:  "[F12]",
			OutSeq: "\x1b[24~",
		},
		{
			Name:   "UP",
			InSeq:  "[UP]",
			OutSeq: "\x1b[A",
		},
		{
			Name:   "DOWN",
			InSeq:  "[DOWN]",
			OutSeq: "\x1b[B",
		},
		{
			Name:   "RIGHT",
			InSeq:  "[RIGHT]",
			OutSeq: "\x1b[C",
		},
		{
			Name:   "LEFT",
			InSeq:  "[LEFT]",
			OutSeq: "\x1b[D",
		},
		{
			Name:   "INS",
			InSeq:  "[INS]",
			OutSeq: "\x1b[2~",
		},
		{
			Name:   "DEL",
			InSeq:  "[DEL]",
			OutSeq: "\x1b[3~",
		},
		{
			Name:   "PGUP",
			InSeq:  "[PGUP]",
			OutSeq: "\x1b[5~",
		},
		{
			Name:   "PGDN",
			InSeq:  "[PGDN]",
			OutSeq: "\x1b[6~",
		},
	}
)

type DiscordTerminal struct {
	Running bool

	Bot *Bot

	Owner *discordgo.User

	Msg           *discordgo.Message
	Pty           *os.File
	Term          vt10x.Terminal
	CurrentScreen string
	LastScreen    string
}

type Key struct {
	Name   string
	InSeq  string
	OutSeq string
}

// discordgo is fucking retarded in this regard, why the fuck do you need a string pointer???
func StringPointer(str string) *string {
	return &str
}

func ParseSequences(ins string) string {
	// ^D, ^C, ^Z, etc...
	match := regexp.MustCompile(`\^([A-Z])`).FindAllStringSubmatch(ins, -1)
	for _, m := range match {
		ac := []byte(m[1])[0]
		ins = strings.ReplaceAll(ins, m[0], string(ac-64))
	}
	// Keys
	for _, k := range BOT_KEYS {
		ins = strings.ReplaceAll(ins, k.InSeq, k.OutSeq)
	}
	return ins
}

func (bot *Bot) Exec(i *discordgo.Interaction, cmd string, args string) {
	var c *exec.Cmd
	if args != "" {
		aargs := strings.Split(args, " ")
		c = exec.Command(cmd, aargs...)
	} else {
		c = exec.Command(cmd)
	}
	c.Env = os.Environ()

	// cerr := c.Run()
	out, _ := c.CombinedOutput()
	/* if err != nil {
		bot.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Error: " + err.Error(),
				// Flags:           0,

			},
		})
		return
	} */
	// if cerr != nil {
	// 	bot.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
	// 		Type: discordgo.InteractionResponseChannelMessageWithSource,
	// 		Data: &discordgo.InteractionResponseData{
	// 			Content: "Execution error: " + err.Error(),
	// 			// Flags:           0,

	// 		},
	// 	})
	// }

	bot.Session.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "```ansi\n" + string(out) + "```",
			// Flags:           0,

		},
	})
	/* 	_, err := bot.Session.InteractionResponseEdit(i, &discordgo.WebhookEdit{
	   		Content: "```ansi\n" + string(out) + "```",
	   	})
	   	if err != nil {
	   		bot.Session.ChannelMessageSend(i.ChannelID, "Cannot edit an interaction response: "+err.Error())
	   	} */
}

func (term *DiscordTerminal) PTYUpdater() {
	for term.Running {
		data := make([]byte, 8192)
		_, err := term.Pty.Read(data)
		if err != nil {
			term.Bot.Session.ChannelMessageDelete(term.Msg.ChannelID, term.Msg.ID)
			term.Running = false
			return
		}
		term.Term.Write(data)
		//fmt.Printf("%v\r", term.String())
		term.CurrentScreen = term.Term.String()
		if bytes.Contains(data, []byte("\a")) { // Check for \a aka \x07 (BEL)
			term.Bot.Session.ChannelMessageSend(term.Msg.ChannelID, fmt.Sprintf("<@%s> BEL", term.Owner.ID))
		}
	}
}

func (term *DiscordTerminal) ScreenUpdater() {
	var err error
	for term.Running {
		if term.CurrentScreen != term.LastScreen {
			// term.Msg, err = term.Bot.Session.ChannelMessageEdit(term.Msg.ChannelID, term.Msg.ID, "```\n"+term.CurrentScreen+"```")
			msgcontent := "```\n" + term.CurrentScreen + "```"
			term.Msg, err = term.Bot.Session.ChannelMessageEditComplex(&discordgo.MessageEdit{
				Content:    &msgcontent,
				Components: term.Msg.Components,
				ID:         term.Msg.ID,
				Channel:    term.Msg.ChannelID,
			})
			if err != nil {
				log.Printf("Cannot update terminal: %s", err)
			}
			term.LastScreen = term.CurrentScreen
		}
		time.Sleep(2 * time.Second)
	}
}

func (term *DiscordTerminal) Close() {
	err := term.Pty.Close()
	if err != nil {
		term.Bot.Session.ChannelMessageSend(term.Msg.ChannelID, "Error closing pty: "+err.Error())
	}
	term.Running = false
}

func NewDiscordTerminal(bot *Bot, cid string, owner *discordgo.User) {
	var err error
	this := &DiscordTerminal{Bot: bot, Owner: owner}

	this.Term = vt10x.New(vt10x.WithSize(W, H))
	c := exec.Command(os.Getenv("SHELL"))
	c.Env = os.Environ()
	c.Env = append(c.Env, "TERM=vt100")
	this.Pty, err = pty.StartWithSize(c, &pty.Winsize{Rows: H, Cols: W, X: 0, Y: 0})
	if err != nil {
		this.Bot.Session.ChannelMessageSend(cid, "Cannot start terminal: "+err.Error())
		return
	}

	this.Msg, err = this.Bot.Session.ChannelMessageSendComplex(cid, &discordgo.MessageSend{
		Content: "Waiting for pty...",
		Components: []discordgo.MessageComponent{
			&discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					&discordgo.Button{
						Label:    "Move down",
						Style:    discordgo.PrimaryButton,
						CustomID: "here",
					},
					/* &discordgo.Button{
						Label:    "Show keys",
						Style:    discordgo.SecondaryButton,
						CustomID: "keys",
					}, */
					&discordgo.Button{
						Label:    "Close",
						Style:    discordgo.DangerButton,
						CustomID: "close",
					}},
			},
		},
	})
	if err != nil {
		log.Println("Unable to create message for terminal: " + err.Error())
	}
	bot.Terminals = append(bot.Terminals, this)
	this.Running = true

	go this.PTYUpdater()
	this.ScreenUpdater()

	idx := slices.IndexFunc(bot.Terminals, func(t *DiscordTerminal) bool { return t == this })
	if idx == -1 {
		panic("cannot remove terminal belonging to " + this.Owner.ID)
	}
	bot.Terminals[idx] = bot.Terminals[len(bot.Terminals)-1]
	bot.Terminals = bot.Terminals[:len(bot.Terminals)-1]
	bot.Session.ChannelMessageEditComplex(&discordgo.MessageEdit{Content: StringPointer("Terminal closed"), Components: []discordgo.MessageComponent{}, Channel: this.Msg.ChannelID, ID: this.Msg.ID})
}

package main

import (
	"fmt"
	"log"
	"math"
	"math/rand"
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

	buttons = []discordgo.MessageComponent{
		&discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				&discordgo.Button{
					Label:    "Set as active",
					Style:    discordgo.PrimaryButton,
					CustomID: "active",
				},
				&discordgo.Button{
					Label:    "Move down",
					Style:    discordgo.SecondaryButton,
					CustomID: "here",
				},
				&discordgo.Button{
					Label:    "Close",
					Style:    discordgo.DangerButton,
					CustomID: "close",
				}},
		},
	}
)

type DiscordTerminal struct {
	ID                 int
	Running            bool
	ScheduledForUpdate bool

	Bot *Bot

	Owner       *discordgo.User
	SharedUsers []string
	CloseSignal chan bool

	Msg           *discordgo.Message
	Pty           *os.File
	Term          vt10x.Terminal
	CurrentScreen string
	LastScreen    string
}

func (term *DiscordTerminal) AllowedToControl(user *discordgo.User) bool {
	if user.ID == term.Owner.ID {
		return true
	}
	if slices.Index(term.SharedUsers, user.ID) != -1 {
		return true
	}
	return false
}

type Key struct {
	Name   string
	InSeq  string
	OutSeq string
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

func StringANSI(t vt10x.Terminal) string {
	t.Lock()
	defer t.Unlock()

	var view []rune
	cols, rows := t.Size()
	var prevcolorfg vt10x.Color = 0
	var prevcolorbg vt10x.Color = 0
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			attr := t.Cell(x, y)
			if attr.FG != prevcolorfg {
				if attr.FG == vt10x.DefaultFG {
					view = append(view, []rune("\x1b[39m")...)
				} else {
					view = append(view, []rune(fmt.Sprintf("\x1b[%dm", 30+attr.FG%8))...)
				}
			}
			if attr.BG != prevcolorbg {
				if attr.BG == vt10x.DefaultBG {
					view = append(view, []rune("\x1b[49m")...)
				} else {
					view = append(view, []rune(fmt.Sprintf("\x1b[%dm", 40+attr.BG%8))...)
				}
			}
			view = append(view, attr.Char)
			prevcolorfg = attr.FG
			prevcolorbg = attr.BG
		}
		view = append(view, '\n')
	}

	view = []rune(regexp.MustCompile("(?m) +$").ReplaceAllString(string(view), ""))
	view = []rune(strings.ReplaceAll(string(view), "    ", "\t"))

	return string(view)
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
		},
	})
	/* 	_, err := bot.Session.InteractionResponseEdit(i, &discordgo.WebhookEdit{
	   		Content: "```ansi\n" + string(out) + "```",
	   	})
	   	if err != nil {
	   		bot.Session.ChannelMessageSend(i.ChannelID, "Cannot edit an interaction response: "+err.Error())
	   	} */
}

func (bot *Bot) Macro(i *discordgo.Interaction, name string) {
	if bot.Config.UserPrefs[i.Member.User.ID].ActiveSession == nil {
		bot.Session.InteractionRespond(i, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "You don't have an active session!",
				Flags:   uint64(discordgo.MessageFlagsEphemeral),
			},
		})
		return
	}
	for _, m := range bot.Config.Macros {
		if m.Name == name {
			if !m.Whitelist || (m.Whitelist && slices.Contains(m.AllowedIDs, i.Member.User.ID)) {
				bot.Config.UserPrefs[i.Member.User.ID].ActiveSession.Pty.WriteString(m.In)
				bot.Session.InteractionRespond(i, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Executed macro " + m.Name,
						Flags:   uint64(discordgo.MessageFlagsEphemeral),
					},
				})
			} else {
				bot.Session.InteractionRespond(i, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "You cannot use this macro!",
						Flags:   uint64(discordgo.MessageFlagsEphemeral),
					},
				})
			}
			return
		}
	}
	bot.RespondError(i, "No such macro")
}

func (term *DiscordTerminal) PTYUpdater() {
	for term.Running {
		data := make([]byte, 8192)
		_, err := term.Pty.Read(data)
		if err != nil {
			// term.Bot.Session.ChannelMessageDelete(term.Msg.ChannelID, term.Msg.ID)
			term.Running = false
			return
		}
		term.Term.Write(data)
		if term.Bot.Config.UserPrefs[term.Owner.ID].Color {
			term.CurrentScreen = StringANSI(term.Term)
		} else {
			term.CurrentScreen = term.Term.String()
		}
		/* if bytes.Contains(data, []byte("\a")) { // Check for \a aka \x07 (BEL)
			term.Bot.Session.ChannelMessageSend(term.Msg.ChannelID, fmt.Sprintf("<@%s> BEL", term.Owner.ID))
		} */
	}
}

func (term *DiscordTerminal) ScreenUpdater() {
	for term.Running {
		if term.CurrentScreen != term.LastScreen || term.ScheduledForUpdate {
			// term.Msg, err = term.Bot.Session.ChannelMessageEdit(term.Msg.ChannelID, term.Msg.ID, "```\n"+term.CurrentScreen+"```")
			msgcontent := "```ansi\n" + term.CurrentScreen + "```"
			err2k := "Oops! Looks like you've reached Discord's 2000 character limit.\nDon't worry, your terminal is still running.\n\nTry disabling colors, and it'll be back."
			var newmsg *discordgo.Message
			newmsg, err := term.Bot.Session.ChannelMessageEditComplex(&discordgo.MessageEdit{
				Content:    &msgcontent,
				Components: buttons,
				Embed:      term.Embed(),
				ID:         term.Msg.ID,
				Channel:    term.Msg.ChannelID,
			})
			if err != nil {
				newmsg, err = term.Bot.Session.ChannelMessageEditComplex(&discordgo.MessageEdit{
					Content:    &err2k,
					Components: buttons,
					Embed:      term.Embed(),
					ID:         term.Msg.ID,
					Channel:    term.Msg.ChannelID,
				})
				if err != nil {
					log.Printf("Cannot send an error message: %s", err)
				}
			}
			if newmsg != nil {
				term.Msg = newmsg
			}
			term.LastScreen = term.CurrentScreen
			term.ScheduledForUpdate = false
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
	<-term.CloseSignal
}

func (term *DiscordTerminal) FormatControlledBy() string {
	out := ""
	for uid, p := range term.Bot.Config.UserPrefs {
		if p.ActiveSession == term {
			if out == "" {
				out += "<@" + uid + ">"
			} else {
				out += ", <@" + uid + ">"
			}
		}
	}
	if out == "" {
		return "None"
	}
	return out
}

func (term *DiscordTerminal) Embed() *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Type:        discordgo.EmbedTypeRich,
		Title:       "Session ID " + fmt.Sprint(term.ID),
		Description: term.Term.Title(),
		Color:       0x00FFFF,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Owner",
				Value:  term.Owner.Mention(),
				Inline: false,
			},
			{
				Name:   "Controlled by",
				Value:  term.FormatControlledBy(),
				Inline: false,
			},
		},
	}
}

func NewDiscordTerminal(bot *Bot, cid string, owner *discordgo.User) {
	var err error
	this := &DiscordTerminal{Bot: bot, Owner: owner, CloseSignal: make(chan bool), SharedUsers: bot.Config.UserPrefs[owner.ID].DefaultSharedUsers, ID: int(math.Floor(100000 + rand.Float64()*900000))}

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
		Content:    "Waiting for pty...",
		Components: buttons,
		Embed:      this.Embed(),
	})
	if err != nil {
		log.Println("Unable to create message for terminal: " + err.Error())
		this.Pty.Close()
		return
	}
	bot.Terminals = append(bot.Terminals, this)
	this.Running = true
	bot.Config.UserPrefs[owner.ID].ActiveSession = this

	go this.PTYUpdater()
	this.ScreenUpdater()

	idx := slices.IndexFunc(bot.Terminals, func(t *DiscordTerminal) bool { return t == this })
	if idx == -1 {
		panic("cannot remove terminal belonging to " + this.Owner.ID)
	}
	bot.Terminals[idx] = bot.Terminals[len(bot.Terminals)-1]
	bot.Terminals = bot.Terminals[:len(bot.Terminals)-1]
	// closedstr := "Terminal closed"
	// bot.Session.ChannelMessageEditComplex(&discordgo.MessageEdit{Content: &closedstr, Components: []discordgo.MessageComponent{}, Channel: this.Msg.ChannelID, ID: this.Msg.ID})
	bot.Session.ChannelMessageDelete(this.Msg.ChannelID, this.Msg.ID)
	this.CloseSignal <- true
}

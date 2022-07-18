package main

import (
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

const (
	BOT_PREFIX = "$"
)

type Bot struct {
	Session   *discordgo.Session
	Terminals []*DiscordTerminal
}

func (bot *Bot) Error(cid string, ierr error) {
	_, err := bot.Session.ChannelMessageSend(cid, ierr.Error())
	if err != nil {
		log.Printf("Cannot send error message: %s", err.Error())
	}
}

func (bot *Bot) MessageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore self
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Paste your userID so the bot can only be used by you
	if m.Author.ID != "552930095141224479" {
		bot.Session.ChannelMessageSend(m.ChannelID, "Insufficient permissions")
		return
	}

	prefix := regexp.MustCompile(`^\` + BOT_PREFIX + `(.+)`)
	msg := prefix.FindAllStringSubmatch(m.Content, -1) // No prefix shall be there
	if msg != nil {
		if rm, _ := regexp.MatchString(`^\.\w+`, msg[0][1]); rm { // Command
			cmd := strings.ReplaceAll(msg[0][1], ".", "") // Remove the dot, all commands start with the prefix+dot
			bot.CommandHandler(m, cmd)
			return
		} else { // Terminal input
			instr, _ := strconv.Unquote(`"` + msg[0][1] + `"`) // Unescape sequences
			instr = ParseCtrlSequences(instr)                  // Parse ^A, ^Z, ^C, etc. See the function.
			for _, t := range bot.Terminals {
				if t.OwnerUserID == m.Author.ID {
					t.Pty.WriteString(instr)                                           // Write message input to the terminal's pty
					err := bot.Session.ChannelMessageDelete(m.ChannelID, m.Message.ID) // Remove user's command msg
					if err != nil {
						log.Printf("Cannot remove user command message: %s", err)
					}
					return
				}
			}
		}
	}
}

func (bot *Bot) Shutdown() {
	err := bot.Session.Close()
	if err != nil {
		log.Printf("Cannot shutdown bot: %s", err)
	}
}

func NewTerminalBot(token string) *Bot {
	this := Bot{}
	sess, err := discordgo.New("Bot " + token)
	if err != nil {
		panic(err)
	}
	this.Session = sess

	this.Session.AddHandler(this.MessageHandler)
	this.Session.Identify.Intents = discordgo.IntentMessageContent | discordgo.IntentGuildMessages
	err = this.Session.Open()
	if err != nil {
		panic(err)
	}

	return &this
}

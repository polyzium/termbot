package main

import (
	"log"
	"regexp"

	"github.com/bwmarrin/discordgo"
)

const (
	BOT_PREFIX  = "$"
	BOT_OWNERID = "552930095141224479" // Paste your userID here so the bot can only be used by you
)

type Prefs struct {
	ActiveSession *DiscordTerminal

	DefaultSharedUsers []string
	Color              bool
}
type Bot struct {
	Session   *discordgo.Session
	Terminals []*DiscordTerminal
	UserPrefs map[string]*Prefs
}

func (bot *Bot) RespondError(i *discordgo.Interaction, err string) {
	bot.Session.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: err,
			Flags:   uint64(discordgo.MessageFlagsEphemeral),
		},
	})
}

func (bot *Bot) CreatePrefIfNotExistsFor(user *discordgo.User) {
	if _, ok := bot.UserPrefs[user.ID]; !ok {
		bot.UserPrefs[user.ID] = &Prefs{
			ActiveSession:      nil,
			DefaultSharedUsers: []string{},
			Color:              false,
		}
	}
}

func (bot *Bot) MessageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore self
	if m.Author.ID == s.State.User.ID {
		return
	}

	bot.CreatePrefIfNotExistsFor(m.Author)

	prefix := regexp.MustCompile(`^\` + BOT_PREFIX + `(.+)`)
	msg := prefix.FindAllStringSubmatch(m.Content, -1) // No prefix shall be there
	if msg != nil {
		/* if !(m.Author.ID == BOT_OWNERID) {
			bot.Session.ChannelMessageSend(m.ChannelID, "Insufficient permissions")
			return
		} */

		/* if rm, _ := regexp.MatchString(`^\.\w+`, msg[0][1]); rm { // Command
			cmd := strings.ReplaceAll(msg[0][1], ".", "") // Remove the dot, all commands start with the prefix+dot
			bot.CommandHandler(m, cmd)
			return
		} else { // Terminal input */
		instr := ParseSequences(msg[0][1])
		t := bot.UserPrefs[m.Author.ID].ActiveSession
		if t == nil {
			bot.Session.ChannelMessageSend(m.ChannelID, "No active session")
			return
		}
		if m.ChannelID == t.Msg.ChannelID {
			t.Pty.WriteString(instr) // Write message input to the terminal's pty
		} else {
			chn, _ := bot.Session.Channel(t.Msg.ChannelID)
			bot.Session.ChannelMessageSend(m.ChannelID, "Your active session is in another channel ("+chn.Mention()+")")
		}
		err := bot.Session.ChannelMessageDelete(m.ChannelID, m.Message.ID) // Remove user's command msg
		if err != nil {
			log.Printf("Cannot remove user command message: %s", err)
		}
		return
		//}
	}
}

func (bot *Bot) InteractionHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	bot.CreatePrefIfNotExistsFor(i.Member.User)

	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		bot.CommandHandler(i)
	case discordgo.InteractionMessageComponent:
		bot.ComponentHandler(i)
	}
}

func (bot *Bot) Shutdown() {
	for _, t := range bot.Terminals {
		t.Close()
	}
	err := bot.Session.Close()
	if err != nil {
		log.Printf("Cannot shutdown bot: %s", err)
	}
}

func NewTerminalBot(token string) *Bot {
	this := Bot{UserPrefs: make(map[string]*Prefs)}
	sess, err := discordgo.New("Bot " + token)
	if err != nil {
		panic(err)
	}
	this.Session = sess

	this.Session.AddHandler(this.MessageHandler)
	this.Session.AddHandler(this.InteractionHandler)
	this.Session.Identify.Intents = discordgo.IntentMessageContent | discordgo.IntentGuildMessages
	err = this.Session.Open()
	if err != nil {
		panic(err)
	}
	this.RegisterCommands()

	return &this
}

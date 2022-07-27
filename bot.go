package main

import (
	"log"
	"os"
	"regexp"

	"github.com/bwmarrin/discordgo"
	"gopkg.in/yaml.v3"
)

const (
	BOT_PREFIX  = "$"
	BOT_OWNERID = "552930095141224479" // Paste your userID here so the bot can only be used by you
)

type Prefs struct {
	ActiveSession *DiscordTerminal `yaml:"-"`

	DefaultSharedUsers []string `yaml:"defaultsharedusers"`
	Color              bool     `yaml:"color"`
}

type Macro struct {
	Name       string   `yaml:"name"`
	In         string   `yaml:"in"`
	Whitelist  bool     `yaml:"whitelist"`
	AllowedIDs []string `yaml:"allowedids"`
}

type Config struct {
	Macros    []Macro `yaml:"macros"`
	UserPrefs map[string]*Prefs
}
type Bot struct {
	Config Config

	Session   *discordgo.Session
	Terminals []*DiscordTerminal
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
	if _, ok := bot.Config.UserPrefs[user.ID]; !ok {
		bot.Config.UserPrefs[user.ID] = &Prefs{
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
		t := bot.Config.UserPrefs[m.Author.ID].ActiveSession
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
		log.Panicf("Cannot shutdown bot: %s", err)
	}

	yamlconf, err := yaml.Marshal(bot.Config)
	if err != nil {
		log.Panicf("Cannot generate YAML: %s", err)
	}
	f, err := os.OpenFile("config.yaml", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Panicf("Cannot open config file: %s", err)
	}
	_, err = f.Write(yamlconf)
	if err != nil {
		log.Panicf("Cannot save config file: %s", err)
	}
}

func NewTerminalBot(token string) *Bot {
	config := Config{
		UserPrefs: make(map[string]*Prefs),
	}
	conf, err := os.ReadFile("config.yaml")
	if err != nil {
		panic(err)
	}
	if err := yaml.Unmarshal(conf, &config); err != nil {
		panic(err)
	}

	this := Bot{Config: config}
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

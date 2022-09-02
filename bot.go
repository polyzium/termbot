package main

import (
	"log"
	"os"
	"regexp"

	"github.com/bwmarrin/discordgo"
	"gopkg.in/yaml.v3"
)

type Prefs struct {
	ActiveSession *DiscordTerminal `yaml:"-"`

	DefaultSharedUsers []string `yaml:"defaultsharedusers"`
	Color              bool     `yaml:"color"`
	Interactive        bool     `yaml:"interactive"`
	AutoSubmit         bool     `yaml:"autosubmit"`
}

type Macro struct {
	Name       string   `yaml:"name"`
	In         string   `yaml:"in"`
	Whitelist  bool     `yaml:"whitelist"`
	AllowedIDs []string `yaml:"allowedids"`
}

type Config struct {
	Token     string            `yaml:"token"`
	Prefix    string            `yaml:"prefix"`
	OwnerID   string            `yaml:"ownerid"`
	Macros    []Macro           `yaml:"macros"`
	UserPrefs map[string]*Prefs `yaml:"userprefs"`
}
type Bot struct {
	Config Config

	Session   *discordgo.Session
	Terminals []*DiscordTerminal
}

func (bot *Bot) RespondString(i *discordgo.Interaction, msg string) {
	bot.Session.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
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
			Interactive:        false,
			AutoSubmit:         false,
		}
	}
}

func (bot *Bot) MessageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore self
	if m.Author.ID == s.State.User.ID {
		return
	}

	bot.CreatePrefIfNotExistsFor(m.Author)

	msg := [][]string{{"", ""}}
	if !bot.Config.UserPrefs[m.Author.ID].Interactive {
		prefix := regexp.MustCompile(`^\` + bot.Config.Prefix + `(.+)`)
		msg = prefix.FindAllStringSubmatch(m.Content, -1) // No prefix shall be there
	} else {
		msg[0][1] = m.Content
	}
	if msg != nil {
		instr := ParseSequences(msg[0][1])
		t := bot.Config.UserPrefs[m.Author.ID].ActiveSession
		if t == nil {
			if bot.Config.UserPrefs[m.Author.ID].Interactive {
				// Ignore other channels
				return
			}
			bot.Session.ChannelMessageSend(m.ChannelID, "No active session")
			return
		}
		if m.ChannelID == t.Msg.ChannelID {
			if bot.Config.UserPrefs[m.Author.ID].AutoSubmit {
				instr += "\r"
			}
			t.Pty.WriteString(instr) // Write message input to the terminal's pty
		} else {
			if bot.Config.UserPrefs[m.Author.ID].Interactive {
				// Ignore other channels
				return
			}
			chn, _ := bot.Session.Channel(t.Msg.ChannelID)
			bot.Session.ChannelMessageSend(m.ChannelID, "Your active session is in another channel ("+chn.Mention()+")")
		}
		err := bot.Session.ChannelMessageDelete(m.ChannelID, m.Message.ID) // Remove user's command msg
		if err != nil {
			log.Printf("Cannot remove user command message: %s", err)
		}
		return
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

func NewTerminalBot() *Bot {
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
	sess, err := discordgo.New("Bot " + config.Token)
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

	for uid := range this.Config.UserPrefs {
		u, _ := this.Session.User(uid)
		log.Println("Loaded preferences for " + u.String())
	}

	return &this
}

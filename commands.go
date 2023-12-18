package main

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"golang.org/x/exp/slices"
)

func (bot *Bot) RegisterCommands() {
	commands := make(map[string]*discordgo.ApplicationCommand)

	commands["open"] = &discordgo.ApplicationCommand{
		Name:        "open",
		Type:        discordgo.ChatApplicationCommand,
		Description: "Launches a new terminal",
	}
	commands["macro"] = &discordgo.ApplicationCommand{
		Name:        "macro",
		Type:        discordgo.ChatApplicationCommand,
		Description: "Executes a predefined macro",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "name",
				Description: "Name of the macro to be executed",
				Required:    true,
			},
		},
	}
	commands["exec"] = &discordgo.ApplicationCommand{
		Name:        "exec",
		Type:        discordgo.ChatApplicationCommand,
		Description: "Executes a command",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "cmd",
				Description: "Command in $PATH. DO NOT PUT ARGUMENTS HERE!",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "args",
				Description: "Arguments for the command",
				Required:    false,
			},
		},
	}
	commands["share"] = &discordgo.ApplicationCommand{
		Type:        discordgo.ChatApplicationCommand,
		Name:        "share",
		Description: "Share your active session with others, so they can control it",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "user",
				Description: "User to share the session with",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionBoolean,
				Name:        "default",
				Description: "Set as default for all future sessions?",
				Required:    false,
			},
		},
	}
	commands["interactive"] = &discordgo.ApplicationCommand{
		Type:        discordgo.ChatApplicationCommand,
		Name:        "interactive",
		Description: "Toggle interactive mode. When enabled, you can use your terminal without a prefix.",
	}
	commands["autosubmit"] = &discordgo.ApplicationCommand{
		Type:        discordgo.ChatApplicationCommand,
		Name:        "autosubmit",
		Description: "Toggle auto submit mode. When enabled, the bot will treat your messages as a command.",
	}

	// Remove redundant/old commands
	botCommands, err := bot.Session.ApplicationCommands(bot.Session.State.User.ID, "")
	if err != nil {
		panic(err)
	}
	for _, bc := range botCommands {
		if _, ok := commands[bc.Name]; !ok {
			err := bot.Session.ApplicationCommandDelete(bot.Session.State.User.ID, "", bc.ID)
			log.Printf("Removing old command %s", bc.Name)
			if err != nil {
				log.Panicf("Cannot remove old application command: %v", err)
			}
		}
	}

	for _, c := range commands {
		_, err := bot.Session.ApplicationCommandCreate(bot.Session.State.User.ID, "", c)
		if err != nil {
			log.Panicf("Cannot register command %s: %s", c.Name, err)
		}
	}
}

func (bot *Bot) CommandHandler(i *discordgo.InteractionCreate) {
	data := i.ApplicationCommandData()

	log.Printf("User %s executed %s", i.Member.User.String(), data.Name)

	switch data.Name {
	case "open":
		canOpen := bot.Config.UserPrefs[i.Member.User.ID].CanOpen
		if !(canOpen) {
			bot.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Insufficient permissions",
					Flags:   uint64(discordgo.MessageFlagsEphemeral),
				},
			})
			return
		}

		bot.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "OK",
				Flags:   uint64(discordgo.MessageFlagsEphemeral),
			},
		})
		NewDiscordTerminal(bot, i.ChannelID, i.Member.User)
	case "macro":
		var name string
		for _, o := range data.Options {
			switch o.Name {
			case "name":
				name = o.StringValue()
			}
		}

		bot.Macro(i.Interaction, name)
	case "exec":
		var cmd string
		var args string
		for _, o := range data.Options {
			switch o.Name {
			case "cmd":
				cmd = o.StringValue()
			case "args":
				args = o.StringValue()
			}
		}
		/* bot.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Please wait",
				// Flags:   uint64(discordgo.MessageFlagsEphemeral),
			},
		}) */
		bot.Exec(i.Interaction, cmd, args)
	case "share":
		var user *discordgo.User
		var defaultv bool = false
		for _, o := range data.Options {
			switch o.Name {
			case "user":
				user = o.UserValue(bot.Session)
			case "default":
				defaultv = o.BoolValue()
			}
		}

		// TODO: Rewrite this fucking mess
		if bot.Config.UserPrefs[i.Member.User.ID].ActiveSession == nil && !defaultv {
			bot.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "You don't have an active session!",
					Flags:   uint64(discordgo.MessageFlagsEphemeral),
				},
			})
			return
		} else if bot.Config.UserPrefs[i.Member.User.ID].ActiveSession != nil && i.Member.User.ID == bot.Config.UserPrefs[i.Member.User.ID].ActiveSession.Owner.ID {
			bot.Config.UserPrefs[i.Member.User.ID].ActiveSession.SharedUsers = append(bot.Config.UserPrefs[i.Member.User.ID].ActiveSession.SharedUsers, user.ID)
		} else if defaultv {
			if slices.Contains(bot.Config.UserPrefs[i.Member.User.ID].DefaultSharedUsers, user.ID) {
				bot.RespondString(i.Interaction, "Already shared!")
				return
			} else {
				bot.Config.UserPrefs[i.Member.User.ID].DefaultSharedUsers = append(bot.Config.UserPrefs[i.Member.User.ID].DefaultSharedUsers, user.ID)

				bot.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "You have allowed " + user.Mention() + " to control your future sessions",
						// Flags:   uint64(discordgo.MessageFlagsEphemeral),
					},
				})
				return
			}
		} else {
			bot.RespondString(i.Interaction, "You are not allowed to share someone else's session")
			return
		}
		if defaultv {
			bot.Config.UserPrefs[i.Member.User.ID].DefaultSharedUsers = append(bot.Config.UserPrefs[i.Member.User.ID].DefaultSharedUsers, user.ID)
		}
		bot.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				// Sucks that there's no ternary operator in Go
				Content: "You have allowed " + user.Mention() + " to control your " + func(a bool) string {
					if a {
						return "current and future sessions"
					} else {
						return "current session"
					}
				}(defaultv),
				// Flags: uint64(discordgo.MessageFlagsEphemeral),
			},
		})
	case "interactive":
		bot.Config.UserPrefs[i.Member.User.ID].Interactive = !bot.Config.UserPrefs[i.Member.User.ID].Interactive
		bot.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				// Sucks that there's no ternary operator in Go
				Content: "Interactive mode has been " + func(a bool) string {
					if a {
						return "en"
					} else {
						return "dis"
					}
				}(bot.Config.UserPrefs[i.Member.User.ID].Interactive) + "abled",
				Flags: uint64(discordgo.MessageFlagsEphemeral),
			},
		})
	case "autosubmit":
		bot.Config.UserPrefs[i.Member.User.ID].AutoSubmit = !bot.Config.UserPrefs[i.Member.User.ID].AutoSubmit
		bot.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				// Sucks that there's no ternary operator in Go
				Content: "Autosubmit mode has been " + func(a bool) string {
					if a {
						return "en"
					} else {
						return "dis"
					}
				}(bot.Config.UserPrefs[i.Member.User.ID].AutoSubmit) + "abled",
				Flags: uint64(discordgo.MessageFlagsEphemeral),
			},
		})
	}
}

func (bot *Bot) ComponentHandler(i *discordgo.InteractionCreate) {
	data := i.MessageComponentData()

	switch data.CustomID {
	case "here":
		for _, t := range bot.Terminals {
			if t.Msg.ID == i.Message.ID {
				if t.AllowedToControl(i.Member.User) {
					bot.Session.ChannelMessageDelete(t.Msg.ChannelID, t.Msg.ID)
					t.Msg, _ = bot.Session.ChannelMessageSendComplex(t.Msg.ChannelID, &discordgo.MessageSend{
						Content:    t.Msg.Content,
						Components: t.Msg.Components,
						Embed:      t.Embed(),
					})
				} else {
					bot.RespondString(i.Interaction, "You are not allowed to take control of this session")
				}
				return
			}
		}
	case "close":
		for _, t := range bot.Terminals {
			if t.Msg.ID == i.Message.ID {
				if t.AllowedToControl(i.Member.User) {
					t.Close()
					if bot.Config.UserPrefs[i.Member.User.ID].ActiveSession == t {
						bot.Config.UserPrefs[i.Member.User.ID].ActiveSession = nil
					}
				} else {
					bot.RespondString(i.Interaction, "You are not allowed to take control of this session")
				}
				return
			}
		}
	case "active":
		for _, t := range bot.Terminals {
			if t.Msg.ID == i.Message.ID {
				if t.AllowedToControl(i.Member.User) {
					bot.Config.UserPrefs[i.Member.User.ID].ActiveSession = t
					bot.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Set session ID " + fmt.Sprint(t.ID) + " as active",
							Flags:   uint64(discordgo.MessageFlagsEphemeral),
						},
					})
				} else {
					bot.RespondString(i.Interaction, "You are not allowed to take control of this session")
				}
				return
			}
		}
	case "color":
		for _, t := range bot.Terminals {
			if t.Msg.ID == i.Message.ID {
				if t.AllowedToControl(i.Member.User) {
					t.Color = !t.Color
					t.SafeTerm.Mutex.Lock()
					if t.Color {
						t.CurrentScreen = StringANSI(t.SafeTerm.Term)
					} else {
						t.CurrentScreen = t.SafeTerm.Term.String()
					}
					t.SafeTerm.Mutex.Unlock()

					t.Update()

					bot.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseDeferredMessageUpdate,
						Data: &discordgo.InteractionResponseData{},
					})
				} else {
					bot.RespondString(i.Interaction, "You are not allowed to take control of this session")
				}
				return
			}
		}
		/* case "keys":
		fmt.Printf("data: %+v\n", data)
		components := []discordgo.MessageComponent{}
		var row discordgo.ActionsRow
		count := 0
		for _, k := range BOT_KEYS {
			row.Components = append(row.Components, discordgo.Button{
				Label:    k.Name,
				Style:    discordgo.SecondaryButton,
				CustomID: "INPUT" + k.InSeq,
			})
			count++
			if count == 5 {
				components = append(components, row)
				count = 0
			}
		}
		fmt.Printf("components: %+v\n", components)
		err := bot.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content:    "Note: you can also use these keys in the prefixed input by putting them in square brackets (with a few exceptions).\nSee TODO LINK for more details.",
				Components: components,
				Flags:      uint64(discordgo.MessageFlagsEphemeral),
			},
		})
		if err != nil {
			log.Println("Cannot respond to interaction: " + err.Error())
		} */
		/* case "input":
		fmt.Printf("data: %+v\n", data)
		return */
	}
}

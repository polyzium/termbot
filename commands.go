package main

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
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
		Description: "Takes input from a predefined macro",
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
	commands["color"] = &discordgo.ApplicationCommand{
		Type:        discordgo.ChatApplicationCommand,
		Name:        "color",
		Description: "EXPERIMENTAL! Enable or disable color. Don't forget to set your $TERM!",
	}

	for _, c := range commands {
		bot.Session.ApplicationCommandCreate(bot.Session.State.User.ID, "", c)
	}
}

func (bot *Bot) CommandHandler(i *discordgo.InteractionCreate) {
	data := i.ApplicationCommandData()

	switch data.Name {
	case "open":
		if i.Member.User.ID != BOT_OWNERID {
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
		bot.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Not implemented",
				Flags:   uint64(discordgo.MessageFlagsEphemeral),
			},
		})
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
		if bot.UserPrefs[i.Member.User.ID].ActiveSession == nil && !defaultv {
			bot.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "You don't have an active session!",
					Flags:   uint64(discordgo.MessageFlagsEphemeral),
				},
			})
			return
		} else if bot.UserPrefs[i.Member.User.ID].ActiveSession != nil && i.Member.User.ID == bot.UserPrefs[i.Member.User.ID].ActiveSession.Owner.ID {
			bot.UserPrefs[i.Member.User.ID].ActiveSession.SharedUsers = append(bot.UserPrefs[i.Member.User.ID].ActiveSession.SharedUsers, user.ID)
		} else {
			bot.RespondError(i.Interaction, "You are not allowed to share someone else's session")
		}
		if defaultv {
			bot.UserPrefs[i.Member.User.ID].DefaultSharedUsers = append(bot.UserPrefs[i.Member.User.ID].DefaultSharedUsers, user.ID)
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
				Flags: uint64(discordgo.MessageFlagsEphemeral),
			},
		})
	case "color":
		bot.UserPrefs[i.Member.User.ID].Color = !bot.UserPrefs[i.Member.User.ID].Color
		bot.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				// Sucks that there's no ternary operator in Go
				Content: "Color display has been " + func(a bool) string {
					if a {
						return "en"
					} else {
						return "dis"
					}
				}(bot.UserPrefs[i.Member.User.ID].Color) + "abled",
				Flags: uint64(discordgo.MessageFlagsEphemeral),
			},
		})
		/* default:
		// bot.Session.ChannelMessageSend(m.ChannelID, "Unknown command")
		bot.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Unknown interaction " + data.Name,
				Flags:   uint64(discordgo.MessageFlagsEphemeral),
			},
		}) */
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
					bot.RespondError(i.Interaction, "You are not allowed to take control of this session")
				}
				return
			}
		}
	case "close":
		for _, t := range bot.Terminals {
			if t.Msg.ID == i.Message.ID {
				if t.AllowedToControl(i.Member.User) {
					t.Close()
					if bot.UserPrefs[i.Member.User.ID].ActiveSession == t {
						bot.UserPrefs[i.Member.User.ID].ActiveSession = nil
					}
				} else {
					bot.RespondError(i.Interaction, "You are not allowed to take control of this session")
				}
				return
			}
		}
	case "active":
		for _, t := range bot.Terminals {
			if t.Msg.ID == i.Message.ID {
				if t.AllowedToControl(i.Member.User) {
					bot.UserPrefs[i.Member.User.ID].ActiveSession = t
					bot.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Set session ID " + fmt.Sprint(t.ID) + " as active",
							Flags:   uint64(discordgo.MessageFlagsEphemeral),
						},
					})
				} else {
					bot.RespondError(i.Interaction, "You are not allowed to take control of this session")
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

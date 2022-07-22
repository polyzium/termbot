package main

import (
	"github.com/bwmarrin/discordgo"
)

func (bot *Bot) RegisterCommands() {
	open := &discordgo.ApplicationCommand{
		Name:        "open",
		Type:        discordgo.ChatApplicationCommand,
		Description: "Launches a new terminal",
	}
	macro := &discordgo.ApplicationCommand{
		Name:        "macro",
		Type:        discordgo.ChatApplicationCommand,
		Description: "Executes a predefined macro",
	}

	bot.Session.ApplicationCommandCreate(bot.Session.State.User.ID, "", open)
	bot.Session.ApplicationCommandCreate(bot.Session.State.User.ID, "", macro)
}

func (bot *Bot) CommandHandler(i *discordgo.InteractionCreate) {
	data := i.ApplicationCommandData()

	switch data.Name {
	case "open":
		for _, t := range bot.Terminals {
			if t.Owner.ID == i.Member.User.ID {
				bot.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Only one terminal per user allowed",
						Flags:   uint64(discordgo.MessageFlagsEphemeral),
					},
				})
				return
			}
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
			if t.Owner.ID == i.Interaction.Member.User.ID {
				bot.Session.ChannelMessageDelete(t.Msg.ChannelID, t.Msg.ID)
				t.Msg, _ = bot.Session.ChannelMessageSendComplex(t.Msg.ChannelID, &discordgo.MessageSend{
					Content:    t.Msg.Content,
					Components: t.Msg.Components,
				})
				return
			}
		}
	case "close":
		for _, t := range bot.Terminals {
			if t.Owner.ID == i.Interaction.Member.User.ID {
				t.Close()
				/* bot.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseUpdateMessage,
				}) */
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

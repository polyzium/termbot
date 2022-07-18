package main

import (
	"github.com/bwmarrin/discordgo"
)

func (bot *Bot) CommandHandler(m *discordgo.MessageCreate, cmd string) {
	switch cmd {
	case "open":
		for _, t := range bot.Terminals {
			if t.OwnerUserID == m.Author.ID {
				bot.Session.ChannelMessageSend(m.ChannelID, "Only one terminal per user allowed")
				return
			}
		}
		NewDiscordTerminal(bot, m.ChannelID, m.Author.ID)
	default:
		bot.Session.ChannelMessageSend(m.ChannelID, "Unknown command")
	}
}

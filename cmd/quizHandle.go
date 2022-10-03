package main

import "github.com/bwmarrin/discordgo"

var defaultResponse = &discordgo.InteractionResponse{
	Type: discordgo.InteractionResponseChannelMessageWithSource,
	Data: &discordgo.InteractionResponseData{
		Content: "Please place a ticker as the first argument",
		Flags:   discordgo.MessageFlagsEphemeral,
	},
}

// Handles any slash command this bot registers
// (only /prompt-quiz for now)
func SlashPromptHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// This is how to get a user ID to @
	// target := i.ApplicationCommandData().Options[0].UserValue(s).ID

	response := defaultResponse

	// Get and check first arguement
	stockTicker := i.ApplicationCommandData().Options[0]
	if stockTicker == nil || len(stockTicker.StringValue()) == 0 {
		s.InteractionRespond(i.Interaction, defaultResponse)
		return
	}

	// Create and send appropriate response for /prompt-quiz
	response = promptStockDirection(stockTicker)
	s.InteractionRespond(i.Interaction, response)
}

// promptStockDirection creates response for /prompt-quiz
func promptStockDirection(command *discordgo.ApplicationCommandInteractionDataOption) (resp *discordgo.InteractionResponse) {
	stockTicker := command.StringValue()

	resp = &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "Stock Quiz",
					Description: "Guess the stock direction of " + stockTicker,
					Fields: []*discordgo.MessageEmbedField{
						{Name: "UP"},
						{Name: "DOWN"},
					},
				},
			},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "UP",
							Style:    discordgo.SuccessButton,
							CustomID: "up",
						},
						discordgo.Button{
							Label:    "DOWN",
							Style:    discordgo.DangerButton,
							CustomID: "down",
						},
					},
				},
			},
		},
	}
	return
}

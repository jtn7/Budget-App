package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	_ "github.com/mattn/go-sqlite3"

	"github.com/bwmarrin/discordgo"
)

// Initialize DB and prepared statements
func initBot() (botSession *discordgo.Session) {
	botToken := os.Getenv("TOKEN")
	// Check if bot token was set
	if botToken == "" {
		fmt.Fprintln(os.Stderr, "Your bot token is empty.\nSet \"TOKEN\" and try again.")
		os.Exit(1)
	}

	// Create a new Discord session using the provided bot token.
	// Session is just a struct; no connecting happens here.
	botSession, err := discordgo.New("Bot " + botToken)
	if err != nil {
		fmt.Println("ERROR: could not create Discord session,", err)
		os.Exit(1)
	}
	return
}

// Opens the sqlite database and assigns it to the db var
func initDatabase() {
	externalDB, err := sql.Open("sqlite3", "./stock-quiz.db")
	if err != nil {
		fmt.Println("ERROR: could not open database:", err)
		os.Exit(1)
	}

	db = database{
		DB: externalDB,
	}
}

var db database

func main() {
	botSession := initBot()
	initDatabase()

	// Register the messageCreate func as a callback for MessageCreate events.
	botSession.AddHandler(InteractionHandler)

	// Open a websocket connection to Discord and begin listening.
	err := botSession.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}
	defer botSession.Close()

	fmt.Println("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc // Wait here on the main thread until CTRL-C or other term signal is received.
}

type interactionCreate func(s *discordgo.Session, i *discordgo.InteractionCreate)

// Map of slash commands to their handler function
var commandMap = map[string]interactionCreate{
	"/prompt-quiz": SlashPromptHandler,
}

// InteractionHandler responds to Interactions
func InteractionHandler(session *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Data.Type() {
	// Handles slash commands
	case discordgo.InteractionApplicationCommand:
		if handler, ok := commandMap[i.ApplicationCommandData().Name]; ok {
			handler(session, i)
		}
	// Handles MessageComponent events (Button clicks)
	case discordgo.InteractionMessageComponent:
		MessageComponentHandler(session, i)
	}
}

var willUpdate = &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate}

// MessageComponentHandler responds to MessageComponent interactions.
//
// MessageComponent interactions are things like buttons clicks
func MessageComponentHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.MessageComponentData().CustomID {
	// Confirming event creation
	case "up":
		s.InteractionRespond(i.Interaction, willUpdate)
		// Get people who pressed the "up" button
		upResponses, err := db.GetUpResponses(i.Message.ID)
		if err != nil {
			log.Printf("ERROR: unable to get \"up\" responses for %s: %v", i.Message.ID, err)
			return
		}

		// Make map of people in the "up" response list
		inUpResponses := make(map[string]bool)
		for _, person := range upResponses {
			inUpResponses[person] = true
		}
		// Check if the person who responded "up" is already in the "up" list
		if inUpResponses[i.Member.User.Username] {
			return
		}

		// Add user who clicked "up" to the slice
		upResponses = append(upResponses, i.Member.User.Username)

		// Get users who clicked "down" for the message edit
		err, dbResponse = db.GetDownResponses(i.Message.ID)
		if err != nil {
			log.Printf("ERROR: Could not query getFlaking with %s: %v", i.Message.ID, err)
			return
		}
		var downResponses []string
		if len(dbResponse) > 0 {
			downResponses = strings.Split(dbResponse, ",;")
		}

		// if there are any down responses then we need to put that back in the message
		if len(downResponses) > 0 {
			_, err = s.ChannelMessageEditComplex(&discordgo.MessageEdit{
				ID:      i.Message.ID,
				Channel: i.ChannelID,
				Embeds: []*discordgo.MessageEmbed{
					i.Message.Embeds[0],
					{
						Title: "Responses",
						Fields: []*discordgo.MessageEmbedField{
							{Name: "UP", Value: strings.Join(upResponses, ", ")},
							{Name: "DOWN", Value: strings.Join(downResponses, ", ")},
						},
					},
				},
			})
			if err != nil {
				log.Printf("ERROR: Could not edit event message %s: %v", i.Message.ID, err)
				return
			}
		} else { // if there are only UP responses then we don't need to put "down" responses in the message
			_, err = s.ChannelMessageEditComplex(&discordgo.MessageEdit{
				ID:      i.Message.ID,
				Channel: i.ChannelID,
				Embeds: []*discordgo.MessageEmbed{
					i.Message.Embeds[0],
					{
						Title: "Responses",
						Fields: []*discordgo.MessageEmbedField{
							{Name: "UP", Value: strings.Join(upResponses, ", ")},
						},
					},
				},
			})
			if err != nil {
				log.Printf("ERROR: Could not edit event message %s: %v", i.Message.ID, err)
				return
			}
		}

		// Insert user into the "up" responses
		_, err = db.UpdateUpResponses(strings.Join(upResponses, ",;"), i.Message.ID)
		if err != nil {
			log.Printf("ERROR: unable to update list of \"up\" with %s: %v", i.Message.ID, err)
		}

	case "down":
		s.InteractionRespond(i.Interaction, willUpdate)
		// Get people who pressed the "up" button
		err, dbResponse := db.GetDownResponses(i.Message.ID)
		if err != nil {
			log.Printf("ERROR: unable to get \"down\" responses for %s: %v", i.Message.ID, err)
			return
		}

		// Set going slice and protect against empty going list
		var upResponses []string
		if len(dbResponse) > 0 {
			upResponses = strings.Split(dbResponse, ",;")
		}

		// Make map of people in the "up" response list
		inUpResponses := make(map[string]bool)
		for _, person := range upResponses {
			inUpResponses[person] = true
		}
		// Check if the person who responded "up" is already in the "up" list
		if inUpResponses[i.Member.User.Username] {
			return
		}

	case "going":
		s.InteractionRespond(i.Interaction, willUpdate)

		// Get people going to the event
		var result string
		err := jimlib.GetGoing.QueryRow(i.Message.ID).Scan(&result)
		if err != nil {
			log.Printf("ERROR: Could not query getGoing with %s: %v", i.Message.ID, err)
			return
		}

		// Set going slice and protect against empty going list
		var going []string
		if len(result) > 0 {
			going = strings.Split(result, ",;")
		}

		// Check if person is already going
		attending := make(map[string]bool)
		for _, person := range going {
			attending[person] = true
		}
		if attending[i.Member.User.Username] {
			return
		}

		// Get users in the flaking list
		err = jimlib.GetFlaking.QueryRow(i.Message.ID).Scan(&result)
		if err != nil {
			log.Printf("ERROR: Could not query getFlaking with %s: %v", i.Message.ID, err)
			return
		}
		var flaking []string
		if len(result) > 0 {
			flaking = strings.Split(result, ",;")
		}

		// Check if the person is in the flaking list
		for ind, person := range flaking {
			// If user is in the flaking list remove them from it
			if i.Member.User.Username == person {
				flaking[ind] = flaking[len(flaking)-1]
				flaking = flaking[:len(flaking)-1]

				_, err = jimlib.UpdateEventFlaking.Exec(strings.Join(flaking, ",;"), i.Message.ID)
				if err != nil {
					log.Printf("ERROR: Could not remove user from flaking list: updateEventFlaking with %s: %v", i.Message.ID, err)
				}
				break
			}
		}

		going = append(going, i.Member.User.Username)

		if len(flaking) > 0 {
			_, err = s.ChannelMessageEditComplex(&discordgo.MessageEdit{
				ID:      i.Message.ID,
				Channel: i.ChannelID,
				Embeds: []*discordgo.MessageEmbed{
					i.Message.Embeds[0],
					{
						Title: "Attendees",
						Fields: []*discordgo.MessageEmbedField{
							{Name: "Going", Value: strings.Join(going, ", ")},
							{Name: "Flaking", Value: strings.Join(flaking, ", ")},
						},
					},
				},
			})
			if err != nil {
				log.Printf("ERROR: Could not edit event message %s: %v", i.Message.ID, err)
				return
			}
		} else {
			_, err = s.ChannelMessageEditComplex(&discordgo.MessageEdit{
				ID:      i.Message.ID,
				Channel: i.ChannelID,
				Embeds: []*discordgo.MessageEmbed{
					i.Message.Embeds[0],
					{
						Title: "Attendees",
						Fields: []*discordgo.MessageEmbedField{
							{Name: "Going", Value: strings.Join(going, ", ")},
						},
					},
				},
			})
			if err != nil {
				log.Printf("ERROR: Could not edit event message %s: %v", i.Message.ID, err)
				return
			}
		}

		// Insert user into going for event
		_, err = jimlib.UpdateEventGoing.Exec(strings.Join(going, ",;"), i.Message.ID)
		if err != nil {
			log.Printf("ERROR: Could not updateEventGoing with %s: %v", i.Message.ID, err)
		}
	case "flaking":
		s.InteractionRespond(i.Interaction, willUpdate)

		// Get flaking list for event
		var result string
		err := jimlib.GetFlaking.QueryRow(i.Message.ID).Scan(&result)
		if err != nil {
			log.Printf("ERROR: Could not query getFlaking with %s: %v", i.Message.ID, err)
			return
		}
		// Put flaking list into slice
		var flaking []string
		if len(result) > 0 {
			flaking = strings.Split(result, ",;")
		}

		// Check if person is already flaking
		flakes := make(map[string]bool)
		for _, person := range flaking {
			flakes[person] = true
		}
		if flakes[i.Member.User.Username] {
			return
		}

		// Get going list for event
		err = jimlib.GetGoing.QueryRow(i.Message.ID).Scan(&result)
		if err != nil {
			log.Printf("ERROR: Could not query getGoing with %s: %v", i.Message.ID, err)
			return
		}
		// Put going list into slice
		var going []string
		if len(result) > 0 {
			going = strings.Split(result, ",;")
		}

		// Check if the person is in the going list
		for ind, person := range going {
			// If user is in the going list remove them from it
			if i.Member.User.Username == person {
				going[ind] = going[len(going)-1]
				going = going[:len(going)-1]

				_, err = jimlib.UpdateEventGoing.Exec(strings.Join(going, ",;"), i.Message.ID)
				if err != nil {
					log.Printf("ERROR: Could not remove user from going list: updateEventGoing with %s: %v", i.Message.ID, err)
				}
				break
			}
		}

		flaking = append(flaking, i.Member.User.Username)

		if len(going) > 0 {
			mb := &jimlib.MessageBuilder{}
			mb.SetID(i.Message.ID).SetChannel(i.ChannelID)
			mb.BuildMessageEdit().SetEmbeds()

			_, err = s.ChannelMessageEditComplex(mb.BuildMessageEdit())
			if err != nil {
				log.Printf("ERROR: Could not edit event message %s: %v", i.Message.ID, err)
				return
			}

			_, err = s.ChannelMessageEditComplex(&discordgo.MessageEdit{
				ID:      i.Message.ID,
				Channel: i.ChannelID,
				Embeds: []*discordgo.MessageEmbed{
					i.Message.Embeds[0],
					{
						Title: "Attendees",
						Fields: []*discordgo.MessageEmbedField{
							{Name: "Going", Value: strings.Join(going, ", ")},
							{Name: "Flaking", Value: strings.Join(flaking, ", ")},
						},
					},
				},
			})
			if err != nil {
				log.Printf("ERROR: Could not edit event message %s: %v", i.Message.ID, err)
				return
			}
		} else {
			_, err = s.ChannelMessageEditComplex(&discordgo.MessageEdit{
				ID:      i.Message.ID,
				Channel: i.ChannelID,
				Embeds: []*discordgo.MessageEmbed{
					i.Message.Embeds[0],
					{
						Title: "Attendees",
						Fields: []*discordgo.MessageEmbedField{
							{Name: "Flaking", Value: strings.Join(flaking, ", ")},
						},
					},
				},
			})
			if err != nil {
				log.Printf("ERROR: Could not edit event message %s: %v", i.Message.ID, err)
				return
			}
		}

		// Insert user into flaking for event
		_, err = jimlib.UpdateEventFlaking.Exec(strings.Join(flaking, ",;"), i.Message.ID)
		if err != nil {
			log.Printf("ERROR: Could not updateEventFlaking with %s: %v", i.Message.ID, err)
		}
	}
}

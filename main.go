package main

import (
	"flag"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"os"
	"os/signal"
	"regexp"
	"syscall"
)

var channelID = ""

const (
	CmdRaidAbsent  = "absent"
	DescRaidAbsent = "Notify officers of your planned raid absence"
)

type config struct {
	token   string
	appID   string
	guildID string
}

func main() {
	var cfg config

	// add cmd line flag parse
	flag.StringVar(&cfg.token, "token", "", "discord bot token")
	flag.StringVar(&cfg.appID, "app-id", "", "discord bot app id")
	flag.StringVar(&cfg.guildID, "guild-id", "", "discord server guild id")
	flag.StringVar(&channelID, "officer-channel-id", "", "destination channel where raid absences are posted")
	flag.Parse()

	dg, err := discordgo.New("Bot " + cfg.token)
	if err != nil {
		fmt.Println("Error creating Discord session,", err)
		return
	}

	dg.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		fmt.Println("Bot is up!")
	})

	err = dg.Open()
	if err != nil {
		fmt.Println("Error opening connection,", err)
		return
	}

	registerCommands(dg, cfg)
	dg.AddHandler(interactionHandler)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	fmt.Println("Shutting down bot...")
	dg.Close()
}

func registerCommands(s *discordgo.Session, cfg config) {
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        CmdRaidAbsent,
			Description: DescRaidAbsent,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "date",
					Description: "The date in mm-dd format",
					Type:        discordgo.ApplicationCommandOptionString,
					Required:    true,
				},
				{
					Name:        "message",
					Description: "A message/reason for absence (optional)",
					Type:        discordgo.ApplicationCommandOptionString,
					Required:    false,
				},
			},
		},
	}

	for _, cmd := range commands {
		_, err := s.ApplicationCommandCreate(cfg.appID, cfg.guildID, cmd)
		if err != nil {
			fmt.Printf("Cannot create slash command %q: %v\n", cmd.Name, err)
		}
	}
}

func interactionHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type == discordgo.InteractionApplicationCommand {
		switch i.ApplicationCommandData().Name {
		case CmdRaidAbsent:
			handleRaidAbsentCommand(s, i)
		}
	}
}

func handleRaidAbsentCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	var date, message, mention string

	mention = i.Member.Mention()

	for _, option := range options {
		switch option.Name {
		case "date":
			date = option.StringValue()
			// todo validate date not in past
		case "message":
			message = option.StringValue()
			if message == "" {
				message = "none given"
			}
		}
	}

	match, _ := regexp.MatchString(`^(0[1-9]|1[0-2])-(0[1-9]|[12][0-9]|3[01])$`, date)
	if !match {
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Invalid date format. Please use mm-dd format.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		if err != nil {
			fmt.Printf("Error responding to send command: %v\n", err)
		}
		return
	}

	if len(message) > 200 {
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Message is too long. Maximum length is 200 characters.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		if err != nil {
			fmt.Printf("Error responding to send command: %v\n", err)
		}
		return
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Thanks for notifying the officers!",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		fmt.Printf("Error responding to send command: %v\n", err)
	}

	msg := fmt.Sprintf("%s has called out for raid on %s. reason: %s", mention, date, message)
	_, err = s.ChannelMessageSend(channelID, msg)
	if err != nil {
		fmt.Printf("Error sending message to channel %s: %v\n", channelID, err)
	}

}

package app

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

type Bot struct {
	token    string
	helpText string
	dg       *discordgo.Session
	db       *Connector
	log      *zap.SugaredLogger
}

func NewBot(token, helpText string, log *zap.SugaredLogger) (*Bot, error) {
	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, fmt.Errorf("error creating Discord session: %w", err)
	}

	bot := &Bot{
		token:    token,
		helpText: helpText,
		dg:       dg,
		log:      log.Named(fmt.Sprintf("%d", os.Getpid())),
	}

	// Register handlers
	dg.AddHandler(bot.ready)
	dg.AddHandler(bot.messageCreate)
	dg.AddHandler(bot.guildCreate)
	dg.AddHandler(bot.presenceUpdate)
	dg.AddHandler(bot.channelCreated)
	dg.AddHandler(bot.channelDeleted)
	dg.AddHandler(bot.voiceChannelUpdate)

	// We need information about guilds (which includes their channels),
	// messages, presence and voice states.
	dg.Identify.Intents = discordgo.IntentsAll

	return bot, nil
}

func (b *Bot) Start(ctx context.Context) error {
	b.log.Debugf("starting bot")
	// Open the websocket and begin listening.
	if err := b.dg.Open(); err != nil {
		return fmt.Errorf("Error opening Discord session: %w", err)
	}

	select {
	case <-ctx.Done():
		b.log.Infof("shutting down bot")
	}

	// Cleanly close down the Discord session.
	return b.dg.Close()
}

// This function will be called (due to AddHandler above) when the bot receives
// the "ready" event from Discord.
func (b *Bot) ready(s *discordgo.Session, event *discordgo.Ready) {
	// Set the playing status.
	s.UpdateGameStatus(0, "!joinery")
}

// Update the game the user is playing
func (b *Bot) presenceUpdate(s *discordgo.Session, m *discordgo.PresenceUpdate) {
	watch, _ := b.db.WatchesByUserID(m.User.ID)
	if watch == nil {
		// not watching
		return
	}

	game := ""
	for _, a := range m.Activities {
		if a.Type == discordgo.ActivityTypeGame && a.Name != "" {
			game = a.Name
			break
		}
	}

	if err := b.db.WatchesGame(m.User.ID, game); err != nil {
		b.log.Errorf("Error updating user presence: %v", err)
		return
	}
	b.log.Debugf("User %s is playing %s", m.User.ID, game)

	channels, err := b.db.ChannelsList(m.GuildID)
	if err != nil {
		b.log.Errorf("Error getting guild channels: %v", err)
		return
	}

	gameChannel := ""
	lobbyChannel := ""
	move := true

	for _, c := range channels {
		if c.Name == game {
			gameChannel = c.ChannelId
		}
		if c.Name == "Lobby" {
			lobbyChannel = c.ChannelId
		}
		channel, err := s.Channel(c.ChannelId)
		if err != nil {
			b.log.Errorf("Error getting channel: %v", err)
			return
		}

		if channel.Type != discordgo.ChannelTypeGuildVoice {
			continue
		}

		for _, member := range channel.Members {
			if member.UserID == m.User.ID {
				move = false
			}
		}
	}

	// join channel corresponding to game
	if move && gameChannel != "" {
		if err := s.GuildMemberMove(m.GuildID, m.User.ID, &gameChannel); err != nil {
			b.log.Errorf("Error moving user: %v", err)
			return
		}
	}
	if move && gameChannel == "" && lobbyChannel != "" {
		if err := s.GuildMemberMove(m.GuildID, m.User.ID, &lobbyChannel); err != nil {
			b.log.Errorf("Error moving user: %v", err)
			return
		}
	}
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func (b *Bot) messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}

	// spew.Dump(m)
	switch m.Content {
	case "!joinery ping":
		s.ChannelMessageSend(m.ChannelID, "Pong!")
	case "!joinery", "!joinery help":
		s.ChannelMessageSend(m.ChannelID, b.helpText)
	case "!joinery watch", "!joinery me":
		b.watch(s, m)
	case "!joinery unwatch", "!joinery stop":
		b.unwatch(s, m)
	case "!joinery who", "!joinery list":
		b.list(s, m)
	case "!joinery channels":
		b.channels(s, m)
	default:
		// nothing
	}
}

// TODO: add guild model
func (b *Bot) guildCreate(s *discordgo.Session, m *discordgo.GuildCreate) {
	// b.log.Debugf("guild: [%s] %s", m.ID, m.Name)
	lobby := ""

	guild, err := s.State.Guild(m.Guild.ID)
	if err != nil {
		b.log.Errorf("Error getting guild: %v", err)
		return
	}

	for _, c := range guild.Channels {
		if c.Type == discordgo.ChannelTypeGuildVoice {
			// b.log.Debugf("guild: [%s] %s channel: [%s] %s", m.ID, m.Name, c.ID, c.Name)
			if c.Name == "Lobby" {
				lobby = c.ID
			}
			if err := b.db.ChannelsCreateUpdate(c.ID, c.Name, m.Guild.ID); err != nil {
				b.log.Errorf("Error creating or updating channel: %v", err)
				return
			}
		}
	}

	if lobby == "" {
		// Create Category
		cat, err := s.GuildChannelCreate(m.Guild.ID, "Joinery", discordgo.ChannelTypeGuildCategory)
		if err != nil {
			b.log.Errorf("Error creating channel: %v", err)
			return
		}

		// Create Lobby
		lob, err := s.GuildChannelCreate(m.Guild.ID, "Lobby", discordgo.ChannelTypeGuildVoice)
		if err != nil {
			b.log.Errorf("Error creating channel: %v", err)
			return
		}

		// Move Lobby to Category
		if _, err := s.ChannelEditComplex(lob.ID, &discordgo.ChannelEdit{ParentID: cat.ID}); err != nil {
			b.log.Errorf("Error editing channel: %v", err)
			return
		}

		b.log.Debugf("guild: created category: [%s] %s channel: [%s] %s", cat.ID, cat.Name, lob.ID, lob.Name)
	}
}

func (b *Bot) voiceChannelUpdate(s *discordgo.Session, m *discordgo.VoiceStateUpdate) {
	if m.ChannelID == "" {
		return
	}

	watch, _ := b.db.WatchesByUserID(m.UserID)
	if watch == nil {
		// not watching
		return
	}

	channel, err := s.Channel(m.ChannelID)
	if err != nil {
		b.log.Errorf("Error getting channel: %v", err)
		return
	}

	if channel.Type != discordgo.ChannelTypeGuildVoice {
		return
	}

	// if joining Lobby and playing game, switch to game channel
	if channel.Name == "Lobby" {
		game := watch.Game
		if game == "" {
			return
		}

		channels, err := b.db.ChannelsList(m.GuildID)
		if err != nil {
			b.log.Errorf("Error getting guild channels: %v", err)
			return
		}

		gameChannel := ""
		for _, c := range channels {
			if c.Name == game {
				gameChannel = c.ChannelId
				break
			}
		}

		if gameChannel != "" {
			if err := s.GuildMemberMove(m.GuildID, m.UserID, &gameChannel); err != nil {
				b.log.Errorf("Error moving user: %v", err)
				return
			}
		}
	}
}

func (b *Bot) channelCreated(s *discordgo.Session, m *discordgo.ChannelCreate) {
	if m.Type == discordgo.ChannelTypeGuildVoice {
		b.log.Debugf("channel created: [%s] %s", m.ID, m.Name)
		if err := b.db.ChannelsCreateUpdate(m.ID, m.Name, m.GuildID); err != nil {
			b.log.Errorf("Error creating or updating channel: %v", err)
			return
		}
	}
}

func (b *Bot) channelDeleted(s *discordgo.Session, m *discordgo.ChannelDelete) {
	if m.Type == discordgo.ChannelTypeGuildVoice {
		b.log.Debugf("channel deleted: [%s] %s", m.ID, m.Name)
		if err := b.db.ChannelsDelete(m.ID); err != nil {
			b.log.Errorf("Error deleting channel: %v", err)
			return
		}
	}
}

func (b *Bot) watch(s *discordgo.Session, m *discordgo.MessageCreate) {
	if err := b.db.WatchesWatch(m.Author.ID, m.Author.Username); err != nil {
		s.ChannelMessageSend(m.ChannelID, "Error watching user.")
		b.log.Errorf("Error watching user: %v", err)
		return
	}
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Joinery is now watching %s for games.", m.Author.Username))

	channels, err := s.GuildChannels(m.GuildID)
	if err != nil {
		b.log.Errorf("Error getting guild channels: %v", err)
		return
	}

	for _, c := range channels {
		if c.Type == discordgo.ChannelTypeGuildVoice {
			b.log.Debugf("channel: [%s] %s", c.ID, c.Name)
			if err := b.db.ChannelsCreateUpdate(c.ID, c.Name, m.GuildID); err != nil {
				b.log.Errorf("Error creating or updating channel: %v", err)
				return
			}
		}
	}
}

func (b *Bot) unwatch(s *discordgo.Session, m *discordgo.MessageCreate) {
	if err := b.db.WatchesUnwatch(m.Author.ID); err != nil {
		s.ChannelMessageSend(m.ChannelID, "Error unwatching user.")
		b.log.Errorf("Error unwatching user: %v", err)
		return
	}

	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Joinery is no longer watching %s for games.", m.Author.Username))
}

func (b *Bot) playing(s *discordgo.Session, userID, game string) {
}

func (b *Bot) list(s *discordgo.Session, m *discordgo.MessageCreate) {
	users, err := b.db.WatchesList()
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "Error listing users.")
		b.log.Errorf("Error listing users: %v", err)
		return
	}

	if len(users) == 0 {
		s.ChannelMessageSend(m.ChannelID, "Joinery is not watching anyone for games.")
		return
	}

	out := []string{"Joinery is watching the following users for games:"}
	for _, u := range users {
		game := "nothing"
		if u.Game != "" {
			game = u.Game
		}
		out = append(out, fmt.Sprintf("* `%s` playing: `%s`", u.Username, game))
	}

	s.ChannelMessageSend(m.ChannelID, strings.Join(out, "\n"))
}

func (b *Bot) channels(s *discordgo.Session, m *discordgo.MessageCreate) {
	channels, err := b.db.ChannelsList(m.GuildID)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "Error listing channels.")
		b.log.Errorf("Error listing channels: %w", err)
		return
	}

	if len(channels) == 0 {
		s.ChannelMessageSend(m.ChannelID, "Joinery is not watching any voice channels.")
		return
	}

	out := []string{"Joinery is watching the following voice channels:"}
	for _, c := range channels {
		out = append(out, fmt.Sprintf("* `%s`", c.Name))
	}

	s.ChannelMessageSend(m.ChannelID, strings.Join(out, "\n"))
}

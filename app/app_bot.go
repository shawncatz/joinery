package app

import (
	"context"
	"fmt"
)

func init() {
	initializers = append(initializers, setupBot)
	starters = append(starters, startBot)
}

func setupBot(a *Application) error {
	token := a.Config.Token
	if token == "" {
		return fmt.Errorf("No token provided")
	}

	bot, err := NewBot(token, a.Config.HelpText, a.Log.Named("bot"))
	if err != nil {
		return fmt.Errorf("Error creating bot: %w", err)
	}

	a.Bot = bot
	return nil
}

func startBot(ctx context.Context, a *Application) error {
	a.Bot.db = a.DB // race during initialization, do it here so we know its ready
	go func() {
		if err := a.Bot.Start(ctx); err != nil {
			a.Log.Errorf("Bot error: %v", err)
		}
	}()
	return nil
}

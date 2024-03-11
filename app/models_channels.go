package app

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
)

func (c *Connector) ChannelsGet(id string) (*Channels, error) {
	m := &Channels{}
	err := c.Channels.Find(id, m)
	if err != nil {
		return nil, err
	}

	// post process here

	return m, nil
}

func (c *Connector) ChannelsList(guildID string) ([]*Channels, error) {
	list, err := c.Channels.Query().Limit(-1).Where("guild_id", guildID).Run()
	if err != nil {
		return nil, err
	}

	return list, nil
}

func (c *Connector) ChannelsByChannelID(channel_id string) (*Channels, error) {
	list, err := c.Channels.Query().Where("channel_id", channel_id).Run()
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, fmt.Errorf("channel not found")
	}
	if len(list) > 1 {
		return nil, fmt.Errorf("too many channels found")
	}

	return list[0], nil
}

func (c *Connector) ChannelsCreateUpdate(channel_id, channel_name, guild_id string) error {
	channel, _ := c.ChannelsByChannelID(channel_id)
	if channel == nil {
		channel = &Channels{}
	}

	channel.Name = channel_name
	channel.ChannelId = channel_id
	channel.GuildId = guild_id

	return c.Channels.Save(channel)
}

func (c *Connector) ChannelsDelete(channel_id string) error {
	_, err := c.Channels.Collection.DeleteOne(context.Background(), bson.M{"channel_id": channel_id})
	return err
}

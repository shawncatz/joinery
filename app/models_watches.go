package app

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
)

func (c *Connector) WatchesGet(id string) (*Watches, error) {
	m := &Watches{}
	err := c.Watches.Find(id, m)
	if err != nil {
		return nil, err
	}

	// post process here

	return m, nil
}

func (c *Connector) WatchesList() ([]*Watches, error) {
	list, err := c.Watches.Query().Limit(-1).Run()
	if err != nil {
		return nil, err
	}

	return list, nil
}

func (c *Connector) WatchesByUserID(user_id string) (*Watches, error) {
	list, err := c.Watches.Query().Where("user_id", user_id).Run()
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, fmt.Errorf("user not found")
	}
	if len(list) > 1 {
		return nil, fmt.Errorf("too many users found")
	}
	return list[0], nil
}

func (c *Connector) WatchesWatching(user_id string) (bool, error) {
	count, err := c.Watches.Query().Where("user_id", user_id).Count()
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (c *Connector) WatchesWatch(user_id, username string) error {
	watching, err := c.WatchesWatching(user_id)
	if err != nil {
		c.Log.Errorf("Error watching user: %v", err)
		return err
	}
	if watching {
		return nil
	}

	err = c.Watches.Save(&Watches{
		UserId:   user_id,
		Username: username,
	})
	return err
}

func (c *Connector) WatchesUnwatch(user_id string) error {
	// watching, err := c.WatchesWatching(user_id)
	// if err != nil {
	// 	c.Log.Errorf("Error watching user: %v", err)
	// 	return err
	// }
	// if watching {
	// 	return nil
	// }

	_, err := c.Watches.Collection.DeleteOne(context.Background(), bson.M{"user_id": user_id})
	return err
}

func (c *Connector) WatchesGame(user_id, game string) error {
	list, err := c.Watches.Query().Where("user_id", user_id).Run()
	if err != nil {
		return err
	}
	if len(list) == 0 {
		return nil
	}
	if len(list) > 1 {
		return fmt.Errorf("found more than one watch for user %s", user_id)
	}

	list[0].Game = game
	if err := c.Watches.Save(list[0]); err != nil {
		return err
	}

	return nil
}

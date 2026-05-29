package database

import (
	"context"
	"discord-bot/internal/models"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (d *Database) GetProfile(userID string) (*models.UserProfile, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := d.DB.Collection("profiles")
	var profile models.UserProfile

	err := collection.FindOne(ctx, bson.M{"user_id": userID}).Decode(&profile)
	if err != nil {
		if err == mongo.ErrNoDocuments {

			return &models.UserProfile{
				UserID:       userID,
				Badges:       []string{},
				Tasks:        []models.TaskProgress{},
				TempAccesses: []models.TempAccess{},
				LastResetAt:  time.Now(),
			}, nil
		}
		return nil, err
	}

	if profile.Badges == nil {
		profile.Badges = []string{}
	}
	if profile.Tasks == nil {
		profile.Tasks = []models.TaskProgress{}
	}
	if profile.TempAccesses == nil {
		profile.TempAccesses = []models.TempAccess{}
	}

	now := time.Now()
	if profile.LastResetAt.Year() != now.Year() || profile.LastResetAt.YearDay() != now.YearDay() {
		profile.DailyMessageCount = 0
		profile.DailyShareCount = 0
		profile.LastResetAt = now

		d.UpsertProfile(&profile)
	}

	return &profile, nil
}

func (d *Database) UpsertProfile(profile *models.UserProfile) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := d.DB.Collection("profiles")
	profile.UpdatedAt = time.Now()

	opts := options.Update().SetUpsert(true)
	filter := bson.M{"user_id": profile.UserID}
	update := bson.M{"$set": profile}

	_, err := collection.UpdateOne(ctx, filter, update, opts)
	return err
}

func (d *Database) IncrementStats(userID string, messages, shares int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	profile, err := d.GetProfile(userID)
	if err != nil {
		return err
	}

	collection := d.DB.Collection("profiles")
	filter := bson.M{"user_id": userID}
	update := bson.M{
		"$inc": bson.M{
			"message_count":       messages,
			"share_count":         shares,
			"daily_message_count": messages,
			"daily_share_count":   shares,
		},
		"$set": bson.M{
			"daily_message_count": profile.DailyMessageCount + messages,
			"daily_share_count":   profile.DailyShareCount + shares,
			"last_reset_at":       profile.LastResetAt,
			"updated_at":          time.Now(),
		},
	}

	update = bson.M{
		"$inc": bson.M{
			"message_count":       messages,
			"share_count":         shares,
			"daily_message_count": messages,
			"daily_share_count":   shares,
		},
		"$set": bson.M{
			"last_reset_at": profile.LastResetAt,
			"updated_at":    time.Now(),
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err = collection.UpdateOne(ctx, filter, update, opts)
	return err
}

func (d *Database) AddBadge(userID, badgeID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := d.DB.Collection("profiles")
	filter := bson.M{"user_id": userID}
	update := bson.M{
		"$addToSet": bson.M{"badges": badgeID},
		"$set":      bson.M{"updated_at": time.Now()},
	}

	_, err := collection.UpdateOne(ctx, filter, update)
	return err
}

func (d *Database) RemoveBadge(userID, badgeID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := d.DB.Collection("profiles")
	filter := bson.M{"user_id": userID}
	update := bson.M{
		"$pull": bson.M{"badges": badgeID},
		"$set":  bson.M{"updated_at": time.Now()},
	}

	_, err := collection.UpdateOne(ctx, filter, update)
	return err
}

func (d *Database) CompleteTask(userID, taskID string, currentValue int, isRepeatable bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := d.DB.Collection("profiles")
	filter := bson.M{
		"user_id":       userID,
		"tasks.task_id": taskID,
	}

	completed := !isRepeatable
	var setFields bson.M
	if isRepeatable {
		setFields = bson.M{
			"tasks.$.completed":      completed,
			"tasks.$.current_value":  currentValue,
			"tasks.$.last_completed": time.Now(),
			"updated_at":             time.Now(),
		}
	} else {
		setFields = bson.M{
			"tasks.$.completed":      completed,
			"tasks.$.last_completed": time.Now(),
			"updated_at":             time.Now(),
		}
	}

	update := bson.M{
		"$set": setFields,
	}

	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		filter = bson.M{"user_id": userID}
		update = bson.M{
			"$push": bson.M{
				"tasks": models.TaskProgress{
					TaskID:        taskID,
					CurrentValue:  currentValue,
					Completed:     completed,
					LastCompleted: time.Now(),
				},
			},
			"$set": bson.M{"updated_at": time.Now()},
		}
		_, err = collection.UpdateOne(ctx, filter, update)
	}

	return err
}

func (d *Database) AddXP(userID string, amount int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := d.DB.Collection("profiles")
	filter := bson.M{"user_id": userID}
	update := bson.M{
		"$inc": bson.M{"xp": amount},
		"$set": bson.M{"updated_at": time.Now()},
	}

	_, err := collection.UpdateOne(ctx, filter, update)
	return err
}

func (d *Database) UpdateLastMessageAt(userID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := d.DB.Collection("profiles")
	filter := bson.M{"user_id": userID}
	update := bson.M{
		"$set": bson.M{
			"last_message_at": time.Now(),
			"updated_at":      time.Now(),
		},
	}

	_, err := collection.UpdateOne(ctx, filter, update)
	return err
}

func (d *Database) AddTempAccess(userID string, access models.TempAccess) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := d.DB.Collection("profiles")
	filter := bson.M{"user_id": userID}

	profile, err := d.GetProfile(userID)
	if err != nil {
		return err
	}

	update := bson.M{
		"$push": bson.M{"temp_accesses": access},
		"$set":  bson.M{"updated_at": time.Now()},
	}

	_, err = collection.UpdateOne(ctx, filter, update)
	if err != nil {

		profile.TempAccesses = append(profile.TempAccesses, access)
		return d.UpsertProfile(profile)
	}
	return nil
}

func (d *Database) RemoveTempAccess(userID string, targetID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := d.DB.Collection("profiles")
	filter := bson.M{"user_id": userID}
	update := bson.M{
		"$pull": bson.M{"temp_accesses": bson.M{"target_id": targetID}},
		"$set":  bson.M{"updated_at": time.Now()},
	}

	_, err := collection.UpdateOne(ctx, filter, update)
	return err
}

func (d *Database) GetAllProfiles() ([]models.UserProfile, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	collection := d.DB.Collection("profiles")
	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var profiles []models.UserProfile
	if err = cursor.All(ctx, &profiles); err != nil {
		return nil, err
	}

	return profiles, nil
}

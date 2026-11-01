package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/delaneyj/toolbelt/embeddednats"
	"github.com/nats-io/nats.go/jetstream"
)

type EmojiCounter struct {
	ThumbsUpEmojiCount int64
	ConfettiEmojiCount int64
	CryLaughEmojiCount int64
	FireEmojiCount     int64
	HeartEmojiCount    int64
}

type EmojiBalloonService struct {
	KV      jetstream.KeyValue
	Counter *EmojiCounter
}

func NewEmojiBalloonService(ns *embeddednats.Server) (*EmojiBalloonService, error) {
	nc, err := ns.Client()
	if err != nil {
		return nil, fmt.Errorf("error creating nats client: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		return nil, fmt.Errorf("error creating jetstream client: %w", err)
	}

	ctx := context.Background()

	kv, err := js.CreateOrUpdateKeyValue(ctx, jetstream.KeyValueConfig{
		Bucket:      "emoji_counter",
		Description: "Emoji Counter",
		Compression: true,
		TTL:         time.Hour,
		MaxBytes:    16 * 1024 * 1024,
	})

	if err != nil {
		return nil, fmt.Errorf("error creating key value: %w", err)
	}

	var counter EmojiCounter
	ebs := EmojiBalloonService{KV: kv, Counter: &counter}

	b, err := ebs.KV.Get(ctx, "emoji_counter")
	if err != nil {
		fmt.Printf("error: could not ebs.KV.Get emoji_counter: %s\n", err.Error())
		fmt.Println("attempting to create emoji_counter...")
		fmt.Println("_, err = ebs.KV.Put(...)")
		fmt.Println("b")
		fmt.Println(b)
		counterBytes, err := json.Marshal(counter)
		if err != nil {
			return nil, err
		}
		_, err = ebs.KV.Put(ctx, "emoji_counter", counterBytes)

		if err != nil {
			return nil, fmt.Errorf("error creating initial ebs.KV. %w", err)
		}

		b, err = ebs.KV.Get(ctx, "emoji_counter")

		if err != nil {
			return nil, fmt.Errorf("could not ebs.KV.Get emoji_counter: %w", err)
		}
	}

	err = json.Unmarshal(b.Value(), &counter)
	if err != nil {
		fmt.Printf("error: failed to json.Unmarshal emoji_counter %s", err.Error())
	}

	return &ebs, nil
}

func (ebs *EmojiBalloonService) AddBalloon(ctx context.Context, emoji string) error {
	if emoji == "👍" {
		ebs.Counter.ThumbsUpEmojiCount = ebs.Counter.ThumbsUpEmojiCount + 1
	}
	if emoji == "🎉" {
		ebs.Counter.ConfettiEmojiCount = ebs.Counter.ConfettiEmojiCount + 1
	}
	if emoji == "😂" {
		ebs.Counter.CryLaughEmojiCount = ebs.Counter.CryLaughEmojiCount + 1
	}
	if emoji == "🔥" {
		ebs.Counter.FireEmojiCount = ebs.Counter.FireEmojiCount + 1
	}
	if emoji == "❤️" {
		ebs.Counter.HeartEmojiCount = ebs.Counter.HeartEmojiCount + 1
	}

	b, err := json.Marshal(ebs.Counter)
	if err != nil {
		return fmt.Errorf("failed to marshal ebs.counter: %w", err)
	}

	_, err = ebs.KV.Put(ctx, "emoji_counter", b)
	if err != nil {
		return fmt.Errorf("failed to put key value: %w", err)
	}

	return nil
}

func (s *EmojiBalloonService) WatchUpdates(ctx context.Context) (jetstream.KeyWatcher, error) {
	return s.KV.Watch(ctx, "emoji_counter")
}

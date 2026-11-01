package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/delaneyj/toolbelt/embeddednats"
	"github.com/nats-io/nats.go/jetstream"
)

type demoState struct{}

type DemoService struct {
	KV                  jetstream.KeyValue
	emojiBalloonService *EmojiBalloonService
	state               *demoState
}

const BUCKET_NAME = "DEMO_STATE"

func NewDemoService(ns *embeddednats.Server, emojiBalloonService *EmojiBalloonService) (*DemoService, error) {
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
		Bucket:      BUCKET_NAME,
		Description: "Datastar Demo State",
		Compression: true,
		TTL:         time.Hour,
		MaxBytes:    16 * 1024 * 1024,
	})

	if err != nil {
		return nil, fmt.Errorf("error creating key value: %w", err)
	}

	var demoState demoState
	demoService := DemoService{KV: kv, state: &demoState, emojiBalloonService: emojiBalloonService}

	b, err := demoService.KV.Get(ctx, BUCKET_NAME)
	if err != nil {
		fmt.Printf("error: could not ebs.KV.Get demo: %s\n", err.Error())
		fmt.Println("attempting to create demo...")
		demoStateBytes, err := json.Marshal(demoState)
		if err != nil {
			return nil, err
		}
		_, err = demoService.KV.Put(ctx, BUCKET_NAME, demoStateBytes)

		if err != nil {
			return nil, fmt.Errorf("error creating initial ebs.KV. %w", err)
		}

		b, err = demoService.KV.Get(ctx, BUCKET_NAME)

		if err != nil {
			return nil, fmt.Errorf("could not ebs.KV.Get BUCKET_NAME: %w", err)
		}
	}

	err = json.Unmarshal(b.Value(), &demoState)
	if err != nil {
		fmt.Printf("error: failed to json.Unmarshal BUCKET_NAME %s", err.Error())
	}

	return &demoService, nil
}

// func (ebs *EmojiBalloonService) AddBalloon(ctx context.Context, emoji string) error  {
// 	if (emoji == "👍") {
// 		ebs.Counter.ThumbsUpEmojiCount = ebs.Counter.ThumbsUpEmojiCount + 1
// 	}
// 	if (emoji == "🎉") {
// 		ebs.Counter.ConfettiEmojiCount = ebs.Counter.ConfettiEmojiCount + 1
// 	}
// 	if (emoji == "😂") {
// 		ebs.Counter.CryLaughEmojiCount = ebs.Counter.CryLaughEmojiCount + 1
// 	}
// 	if (emoji == "🔥") {
// 		ebs.Counter.FireEmojiCount = ebs.Counter.FireEmojiCount + 1
// 	}
// 	if (emoji == "❤️") {
// 		ebs.Counter.HeartEmojiCount = ebs.Counter.HeartEmojiCount + 1
// 	}
//
// 	b, err := json.Marshal(ebs.Counter)
// 	if err != nil {
// 		return fmt.Errorf("failed to marshal ebs.counter: %w", err)
// 	}
//
// 	_, err = ebs.KV.Put(ctx, "emoji_counter", b)
// 	if err != nil {
// 		return fmt.Errorf("failed to put key value: %w", err)
// 	}
//
// 	return nil
// }

func (ds *DemoService) WatchUpdates(ctx context.Context) (jetstream.KeyWatcher, error) {
	return ds.KV.Watch(ctx, BUCKET_NAME)
}

func (ds *DemoService) WatchEmojiCounter(ctx context.Context) (jetstream.KeyWatcher, error) {
	return ds.emojiBalloonService.WatchUpdates(ctx)
}

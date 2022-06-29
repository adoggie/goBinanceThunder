package main

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/shopspring/decimal"
	"time"

	//"github.com/zeromq/goczmq"

	//"github.com/zeromq/goczmq"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

/*
redis
https://redis.uptrace.dev/guide/go-redis-pubsub.html

*/

//var(
//	posReceiver PositionReceiverRedis
//)

type PositionReceiverRedis struct {
	config        *Config
	mtx           sync.RWMutex
	positionCache map[string]decimal.Decimal
	rdb           *redis.Client
}

func (p *PositionReceiverRedis) init(config *Config, redisDB *redis.Client) *PositionReceiverRedis {
	p.positionCache = make(map[string]decimal.Decimal)
	p.config = config
	p.rdb = redisDB
	return p
}

func (p *PositionReceiverRedis) open(ctx context.Context) (stopC <-chan struct{}, err error) {
	stopC = make(chan struct{})
	err = nil
	go func() {
		topic := fmt.Sprintf("position_%s", config.ThunderId)
		pubsub := p.rdb.Subscribe(ctx, topic)
		defer pubsub.Close()

		for {

			msg, err := pubsub.ReceiveMessage(ctx)
			if err != nil {
				logger.Error("PositionRecv Failed:", err.Error())
				break
			}

			fmt.Printf("%s\n", msg.Payload)
			text := msg.Payload
			fs := strings.Split(text, ",")
			// thunder_id,symbol,pos
			if len(fs) != 3 {
				continue
			}
			if fs[0] != p.config.ThunderId {
				continue
			}
			name := fs[1]
			pos, err := decimal.NewFromString(fs[2])
			if err != nil {
				continue
			}

			p.mtx.RLock()
			if vol, ok := p.positionCache[name]; ok {
				if vol.Equal(pos) {
					continue
				}
			}
			p.positionCache[name] = pos
			p.mtx.RUnlock()

			event := &PositionEvent{symbol: name, volume: pos}
			controller.onRecvPosition(event)

			fn := filepath.Join(p.config.PsLogDir, name+".SIG")
			if fp, err := os.OpenFile(fn, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0660); err == nil {
				line := fmt.Sprintf("%s %s\n", time.Now().Format(time.RFC3339), text)
				fp.WriteString(line)
				fp.Close()
			}
		}

	}()
	return
}

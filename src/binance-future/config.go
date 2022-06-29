package main

import (
	"github.com/adshao/go-binance/v2/futures"
	"github.com/shopspring/decimal"
	"sync"
	"time"
)

type RedisServerAddress struct {
	Addr     string `toml:"addr"`
	Password string `toml:"password"`
	DB       int    `toml:"db"`
}

type Symbol struct {
	Name          string  `toml:"name"`
	Enable        bool    `toml:"enable"`         //
	LimitPosition float64 `toml:"limit_position"` //最大持仓
	Timeout       int64   `toml:"timeout"`
	OrderType     string  `toml:"order_type"`
	Ticks         int64   `toml:"ticks"`
	Leverage      int     `toml:"leverage"` // 开仓杠杆

	target   *decimal.Decimal
	expect   *decimal.Decimal
	position Position
	impl     interface{}              //
	aggTrade *futures.WsAggTradeEvent // 聚合成交记录
	detail   futures.Symbol

	slices     []*OrderSlice
	orderStart int64 // 报单开始时间

	mtx sync.RWMutex

	// binance.com/zh-CN/futures/trading-rules/perpetual/
	maxPrice    decimal.Decimal
	minPrice    decimal.Decimal
	tickSize    decimal.Decimal
	minNotional decimal.Decimal // 最小名义价值
	maxQty      decimal.Decimal
	minQty      decimal.Decimal
}

func (s Symbol) LastPrice() *decimal.Decimal {
	if s.aggTrade == nil {
		return nil
	}
	if price, e := decimal.NewFromString(s.aggTrade.Price); e == nil {
		return &price
	}
	return nil
}

type AccountConfig struct {
	Name      string `toml:"name"`
	ApiKey    string `toml:"api_key"`
	SecretKey string `toml:"secret_key"`
}

type Config struct {
	ThunderId                string             `toml:"thunder_id"`
	ServiceType              string             `toml:"service_type"`
	PositionBroadcastAddress string             `toml:"pos_mx_addr"`
	RedisServer              RedisServerAddress `toml:"redis_server"`
	LogFile                  string             `toml:"logfile"`
	Symbols                  []*Symbol          `toml:"symbols"`
	Accounts                 []*AccountConfig   `toml:"accounts"`
	//ApiKey                   string             `toml:"api_key"`
	//SecretKey                string             `toml:"secret_key"`
	PsLogDir               string `toml:"pslog_dir"`              // 仓位信号日志
	ReportStatusInterval   int64  `toml:"report_status_interval"` // 上报服务运行状态的时间间隔
	Account                string `toml:"account"`
	UserStreamKeepInterval int64  `toml:"userstream_keep_interval"`
	MessagePubChan         string `toml:"message_pub_chan"`
}

type Position struct {
	direction string          // long ,short
	volume    decimal.Decimal //
	impl      *futures.AccountPosition
}

type PositionEvent struct {
	thunderId string          // 下单程序id
	symbol    string          // 交易对
	volume    decimal.Decimal // 仓位
}

type Account struct {
	thunderId string  `json:"thunder_id"`
	balance   float64 `json:"balance"`
}

type LogEvent interface {
}

type ServiceSource struct {
	Type      string `json:"type"`
	Id        string `json:"id"`
	Ip        string `json:"ip"`
	Datetime  string `json:"datetime"`
	Timestamp int64  `json:"timestamp"`
}

type ServiceDelta struct {
	DB         string   `json:"db"`
	Table      string   `json:"table"`
	UpdateKeys []string `json:"update_keys"`
}

type ServiceMessage struct {
	Type    string                 `json:"type"`
	Source  ServiceSource          `json:"source"`
	Content map[string]interface{} `json:"content"`
	Delta   ServiceDelta           `json:"delta"`
}

type Timer struct {
	lastTs  int64
	period  int64
	onTimer func(timer *Timer)
	delta   interface{}
}

func NewTimer(period int64, onTimer func(timer *Timer), delta interface{}) *Timer {
	return &Timer{lastTs: time.Now().Unix(), period: period, onTimer: onTimer, delta: delta}
}

func (t *Timer) tick() {
	if t.lastTs+t.period < time.Now().Unix() {
		t.onTimer(t)
		t.lastTs = time.Now().Unix()
	}
}

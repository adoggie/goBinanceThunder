package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/adshao/go-binance/v2"
	"github.com/adshao/go-binance/v2/futures"
	"github.com/go-redis/redis/v8"
	"github.com/gookit/goutil/netutil"
	"github.com/shopspring/decimal"
	"math"
	"sync"
	"time"
)

const (
	UntradablePositionVolume = math.MaxInt32
)

type Controller struct {
	posChanList    map[string]chan PositionEvent
	symbols        map[string]*Symbol
	longPositions  map[string]Position
	shortPositions map[string]Position
	//positions      map[string]float64

	config           *Config
	rdb              *redis.Client
	logChan          chan LogEvent
	mtxSelf          sync.RWMutex
	ctx              context.Context
	clientFuture     *futures.Client
	exchangeInfo     *futures.ExchangeInfo
	accountConfig    *AccountConfig
	account          *futures.Account
	listenKey        string
	positionReceiver PositionReceiverRedis
}

func (c *Controller) RedisDB() *redis.Client {
	return c.rdb
}

func (c *Controller) init(config *Config) *Controller {
	c.posChanList = make(map[string]chan PositionEvent)
	c.symbols = make(map[string]*Symbol)
	c.longPositions = make(map[string]Position)
	c.shortPositions = make(map[string]Position)

	c.config = config
	c.logChan = make(chan LogEvent, 1000)

	for _, symbol := range c.config.Symbols {
		logger.Debug("load ", symbol.Name, " ..")
		c.symbols[symbol.Name] = symbol
		c.longPositions[symbol.Name] = Position{volume: decimal.NewFromInt32(0)}
		c.shortPositions[symbol.Name] = Position{volume: decimal.NewFromInt32(0)}
	}

	c.initRedis()
	c.updatePositionFromRedis()
	c.initBinance()
	c.initLeverage()
	c.positionReceiver.init(c.config, c.rdb)

	return c
}

func (c *Controller) initLeverage() {
	// 杠杆设置
	for name, sym := range c.symbols {
		if _, err := c.clientFuture.NewChangeLeverageService().Symbol(name).Leverage(sym.Leverage).Do(c.ctx); err != nil {
			logger.Error("setLeverage Failed:", name, " leverage:", sym.Leverage)
			panic("setLeverage failed!")
		}
		logger.Info("setLeverage okay:", name, " leverage:", sym.Leverage, " ..")
	}
}

func (c *Controller) initRedis() bool {
	c.rdb = redis.NewClient(&redis.Options{
		Addr:     c.config.RedisServer.Addr,
		Password: c.config.RedisServer.Password,
		DB:       c.config.RedisServer.DB,
	})
	return true
}

func (c *Controller) onAggTradeEvent(event *futures.WsAggTradeEvent) {
	if sym, ok := c.symbols[event.Symbol]; ok {
		//fmt.Println(event.Time, event.Symbol, event.Price, event.Quantity)
		sym.mtx.RLock()
		//sym.impl = event
		sym.aggTrade = event
		sym.mtx.RUnlock()

	}

}

// 1. account prepare
// 2. load symbols
// 3. user stream set

func (c *Controller) initBinance() {
	c.ctx = context.Background()
	for _, acc := range c.config.Accounts {
		if acc.Name == c.config.Account {
			c.accountConfig = acc
		}
	}
	if c.accountConfig == nil {
		panic("Please Corrrect Account!")
	}

	logger.Info("Select Account: ", c.accountConfig.Name)

	c.clientFuture = binance.NewFuturesClient(c.accountConfig.ApiKey, c.accountConfig.SecretKey)

	// 账户余额
	if acc, e := c.clientFuture.NewGetAccountService().Do(c.ctx); e == nil {
		c.account = acc
		fmt.Println("账户余额:", acc.TotalWalletBalance) // 可用余额
		//for _, asset := range acc.Assets {
		//	fmt.Println("  ", asset.Asset, asset.WalletBalance) // 持有资产
		//}
	}
	// User Stream
	if listenKey, e := c.clientFuture.NewStartUserStreamService().Do(c.ctx); e == nil {
		c.listenKey = listenKey
	} else {
		fmt.Println(e.Error())
		panic("Start User Stream Failed!")
	}

	// 交易所信息
	if exinfo, err := c.clientFuture.NewExchangeInfoService().Do(c.ctx); err == nil {
		c.exchangeInfo = exinfo
		for _, syn := range exinfo.Symbols {
			//fmt.Println("symbol:",syn.Symbol , " quotePrecision:" ,syn.QuotePrecision )

			if symbol, ok := c.symbols[syn.Symbol]; ok {
				symbol.detail = syn
				fmt.Println(syn)
				for _, filter := range syn.Filters {
					if filter["filterType"] == "PRICE_FILTER" {
						symbol.maxPrice, _ = decimal.NewFromString(filter["maxPrice"].(string))
						symbol.minPrice, _ = decimal.NewFromString(filter["minPrice"].(string))
						symbol.tickSize, _ = decimal.NewFromString(filter["tickSize"].(string))
						fmt.Println("    maxPrice:", symbol.maxPrice, " minPrice:", symbol.minPrice, " tickSize:", symbol.tickSize)
					}
				}
			}
		}
		fmt.Println("Total Symbols :", len(exinfo.Symbols))
	} else {
		fmt.Println(err.Error())
		return
	}

}

func (c *Controller) open() {
	c.positionReceiver.open(c.ctx)
	c.run()
}

func (c *Controller) newServiceMessage(type_ string) *ServiceMessage {
	message := &ServiceMessage{Type: type_}
	message.Source.Type = c.config.ServiceType
	message.Source.Id = c.config.ThunderId
	message.Source.Ip = netutil.InternalIP()
	message.Source.Datetime = time.Now().Format(time.RFC3339)
	message.Source.Timestamp = time.Now().Unix()
	message.Content = make(map[string]interface{})
	//message.Delta.UpdateKeys = make(map[string]interface{})
	return message
}

func (m *ServiceMessage) SetValue(key string, value interface{}) *ServiceMessage {
	m.Content[key] = value
	return m
}

func (m *ServiceMessage) Database(dbname string) *ServiceMessage {
	m.Delta.DB = dbname
	return m
}

func (m *ServiceMessage) Table(name string) *ServiceMessage {
	m.Delta.Table = name
	return m
}

func (m *ServiceMessage) UpdateKeys(keys ...string) *ServiceMessage {
	m.Delta.UpdateKeys = keys
	return m
}

// 投递到redis-pub
func (c *Controller) sendMessage(m *ServiceMessage) {
	if detail, err := json.Marshal(m); err == nil {
		c.rdb.Publish(c.ctx, c.config.MessagePubChan, detail)
	} else {
		fmt.Println(err.Error())
	}
}

//Balance和Position更新推送
func (c *Controller) onUserData(event *futures.WsUserDataEvent) {
	fmt.Println("onUserData: ", event.Event)
}

func (c *Controller) queryAccountAndPosition(justOne bool) {
	for {
		if acc, e := c.clientFuture.NewGetAccountService().Do(c.ctx); e == nil {
			c.mtxSelf.RLock()
			defer c.mtxSelf.RUnlock()

			c.account = acc

			fmt.Println("查询账户余额:", acc.TotalWalletBalance) // 可用余额
			for _, asset := range acc.Assets {
				_ = asset
				//fmt.Println("  资产:", asset.Asset, asset.WalletBalance) // 持有资产
				// 更新仓位
				for _, p := range acc.Positions {
					if _, ok := c.symbols[p.Symbol]; !ok {
						continue
					}
					if p.PositionSide == "LONG" {
						if volume, e := decimal.NewFromString(p.PositionAmt); e != nil {
							continue
						} else {
							position := Position{direction: "long", volume: volume.Abs(), impl: p}
							c.longPositions[p.Symbol] = position
						}
					}
					if p.PositionSide == "SHORT" {
						if volume, e := decimal.NewFromString(p.PositionAmt); e != nil {
							continue
						} else {
							position := Position{direction: "short", volume: volume.Abs(), impl: p}
							c.shortPositions[p.Symbol] = position
						}
					}

				}
			}
		} else {
			logger.Warn("Query Account failed:", e.Error())
			time.Sleep(time.Second * 5)
			if justOne {
				panic("query account fail, stop trader.")
			}
			continue // 查询失败从新执行
		}
		if justOne {
			return
		}
		select {
		case <-c.ctx.Done():
			logger.Info("账户查询退出 ..")
			return
		case <-time.After(time.Second * 2):

		}
	}
}

func (c *Controller) subUserStreamData() {
START:
	var (
		doneC, stopC chan struct{}
		err          error
	)
	for {
		doneC, stopC, err = futures.WsUserDataServe(c.listenKey, c.onUserData, func(err error) {})
		if err != nil {
			logger.Warn("WsUserDataServe Error , wait and retry..", err.Error())
			time.Sleep(time.Second * 5)
		}
		break
	}
	for {
		select {
		case <-c.ctx.Done():
			stopC <- struct{}{} //系统终止，准备全部退出
			return
		case <-doneC:
			// ws 断开需要重新连接
			logger.Warn("WsUserDataServe:: connection lost , wait and retry..")
			time.Sleep(time.Second * 5)
			goto START
		case <-time.After(time.Second * time.Duration(c.config.UserStreamKeepInterval)):
			logger.Info("UserStream Keep ..")
			if err := c.clientFuture.NewKeepaliveUserStreamService().Do(c.ctx); err != nil {

			}
		}
	}

}

func (c *Controller) subMarketData(name string) {
START:
	var (
		doneC, stopC chan struct{}
		err          error
	)

	for {
		doneC, stopC, err = futures.WsAggTradeServe(name, c.onAggTradeEvent, func(err error) {})
		if err != nil {
			logger.Warn("symbol:", name, " connect Rejected!", err.Error())
			time.Sleep(time.Second * 5)
		}
		break
	}
	for {
		select {
		case <-c.ctx.Done():
			stopC <- struct{}{} //系统终止，准备全部退出
			return
		case <-doneC:
			// ws 断开需要重新连接
			logger.Warn("symbol:", name, " Ws Disconnected! Sleep awhile and reconnect...")
			time.Sleep(time.Second * 5)
			goto START
		}
	}
}

func (c *Controller) run() {
	c.discardAllOrders()
	c.queryAccountAndPosition(true)
	// 连续订阅 userstream
	go c.subUserStreamData()

	// 定时查询账户信息（包括资金和仓位情况)
	go c.queryAccountAndPosition(false)

	// 启动所有交易对行情接收服务,ws 断开时等待并继续连接
	for name, _ := range c.symbols {
		go c.subMarketData(name)
	}

	// 进入报单循环
	for name, _ := range c.symbols {
		go func(name string) {
			timer := NewTimer(c.config.ReportStatusInterval, func(timer *Timer) {
				// 定时触发，发送状态信息（对于每一个交易symbol个性化状态上报)
				m := c.newServiceMessage("onThunderLive").
					Database("BinanceThunder").Table(c.config.ThunderId).
					UpdateKeys("Symbol").SetValue("Symbol", name).
					SetValue("UpdateTime", time.Now().Format(time.RFC3339))
				c.sendMessage(m)
			}, c)

			for {
				select {
				case <-c.ctx.Done():
					return
				case <-time.After(time.Millisecond * 100):
					c.doTrade(name)
				}
				timer.tick() //
			}
		}(name)
	}

}

// 保持仓位一致
func (c *Controller) doTrade(name string) {
	sym := c.symbols[name]
	sym.mtx.RLock()
	defer sym.mtx.RUnlock()
	if sym.expect != nil {
		sym.target = sym.expect
	}

	if sym.target == nil {
		return
	}
	slices := CreateOrderSlice(*sym.target, c.longPositions[name].volume, c.shortPositions[name].volume)
	if len(slices) == 0 {
		return
	}
	for _, slice := range slices {
		slice.symbol = sym
		orderSvc := c.clientFuture.NewCreateOrderService().Symbol(name).
			Side(slice.buysell).
			PositionSide(slice.direction).
			Type(futures.OrderType(sym.OrderType)).
			TimeInForce(futures.TimeInForceTypeGTC).
			Quantity(slice.quantity.String())
		//Price(slice.price.String()) //.Do(c.ctx)
		newprice := decimal.NewFromInt32(0)
		lastPrice := decimal.NewFromInt32(0)
		if sym.OrderType == string(futures.OrderTypeLimit) {

			if sym.LastPrice() == nil {
				logger.Warn("Price Not Present :", sym.Name)
				return
			}
			lastPrice = *sym.LastPrice()
			// lastprice + tick_size * ticks
			if slice.buysell == "BUY" {
				newprice = lastPrice.Add(sym.tickSize.Mul(decimal.NewFromInt(sym.Ticks)))
			}
			if slice.buysell == "SELL" {
				newprice = lastPrice.Sub(sym.tickSize.Mul(decimal.NewFromInt(sym.Ticks)))
			}

			orderSvc.Price(newprice.String())
		}
		logger.Info("createOrder:", sym.Name, " side:", slice.buysell,
			" positionSide:", slice.direction, " orderType:", sym.OrderType,
			" quantity:", slice.quantity.String(), " lastPrice:", lastPrice,
			" orderPrice:", newprice.String())

		order, err := orderSvc.Do(c.ctx)

		if err != nil {
			logger.Error(err.Error(), " sleep and retry..")
			time.Sleep(time.Second * 3)
			return
		}
		slice.orderRequest = order
	}
	sym.slices = slices
	sym.orderStart = time.Now().Unix()
	for {
		// mtx.Lock()
		present := c.longPositions[name].volume.Sub(c.shortPositions[name].volume)
		if sym.target.Equal(present) {
			logger.Info(name, "Trade Okay!")
			break
		}
		if sym.Timeout+sym.orderStart <= time.Now().Unix() {
			// timeout , cancel orders
			logger.Warn("Order Timeout ..")
			c.cancelOrder(sym.slices)
			break
		}
		time.Sleep(time.Millisecond * 100)
	}
}

func (c *Controller) discardAllOrders() {
	// 撤销所有挂单
	for symbol, _ := range c.symbols {
		orders, _ := c.clientFuture.NewListOpenOrdersService().Symbol(symbol).Do(c.ctx)
		var orderIds []int64
		orderIds = Map(orders, func(from *futures.Order) int64 {
			return from.OrderID
		})
		c.clientFuture.NewCancelMultipleOrdersService().Symbol(symbol).OrderIDList(orderIds).Do(c.ctx)
	}
}

func (c *Controller) cancelOrder(slices []*OrderSlice) {
	for _, slice := range slices {
		logger.Info("cancelOrder:", slice.symbol.Name, " side:", slice.buysell,
			" positionSide:", slice.orderRequest.PositionSide, " OrderID:", slice.orderRequest.OrderID,
			" price:", slice.orderRequest.Price, " OrigQuantity:", slice.orderRequest.OrigQuantity)
		_, err := c.clientFuture.NewCancelOrderService().Symbol(slice.orderRequest.Symbol).OrderID(slice.orderRequest.OrderID).Do(c.ctx)
		if err != nil {
			logger.Error("cancelOrder Failed : ", err.Error())
		}
	}
}

// 仓位信号到达
func (c *Controller) onRecvPosition(event *PositionEvent) {
	c.mtxSelf.RLock()
	defer c.mtxSelf.RUnlock()
	if symbol, ok := c.symbols[event.symbol]; !ok {
		return
	} else {
		symbol.expect = &event.volume
	}
}

func (c *Controller) updatePositionFromRedis() {

	redis := c.rdb
	ctx := context.Background()
	var (
		val string
		err error
	)
	for name, _ := range c.symbols {
		if val, err = redis.HGet(ctx, c.config.ThunderId, name).Result(); err != nil {
			continue
		}

		if pos, e := decimal.NewFromString(val); e == nil {
			fmt.Println(pos.String())
			if sym, ok := c.symbols[name]; ok {
				sym.target = &pos
			}
		} else {
			fmt.Print(e.Error())
		}

	}
}

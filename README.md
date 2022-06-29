

#GoBinanceThunder 
go版本的币安下单程序

**币安u本位下单**

- 接收策略信号
- 维持仓位一致的买卖
- 接收行情
- 资金和仓位回报

# Install & Enviroment

**binanceFuture** 

```
cd $project/src/binance-future 
go mod tidy 
go build 
```
**tests**
```
pip install fire redis 
```

# RUN 

修改 `settings.toml `配置

1. `redis_server` : 信号暂存和转发服务地址s
2. `accounts`: 币安交易账户 api_key
3. `symbols` : 参与交易的交易对

## 测试仓位

```
cd scripts
python elps-post.py ps_send 'thunder_lch' 'XRPUSDT' -15
报单XRPUSDT 15手单位
```


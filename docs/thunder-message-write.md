
# Write Message Into  NosqlDB 
---

## Message

```json
{
  "type": "dbwrite/log",
  "source": {
    "type": "thunder",
    "id": "scott",
    "ip": "x.x.x.x",
    "datetime": "20220101 12:01:00"
  },
  "content": {
    "k1": "1",
    "k2": "2"
  },
  "delta": {
    "db": "thunder_db",
    "table": "10001",
    "update_keys": {
      "k1": 1,
      "k2": 2
    }
  }
}
```

* type  消息类型
* source /  消息发送者属性
  * type   发送服务类型 
  * id     发送服务id   
  * ip     发送服务ip
  * datetime 发送服务系统时间
* content

* delta
  * db
  * table
  * update_keys  

** 默认执行记录插入 除非 delta/update_keys 存在 ** 


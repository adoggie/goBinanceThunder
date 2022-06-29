import os 
import os.path 
import datetime
import time
import traceback
import threading
import redis as Redis
import json 
import fire
import zmq
import random
from collections import defaultdict



"""
elps-post.py 
发送 仓位信号到 mx ，等待 thunder接收
"""

redis_server = dict(
#     host = '172.16.20.21' ,
    host = '127.0.0.1' ,
    port=6379,db = 0
)
redis = Redis.StrictRedis( **redis_server)

# pub_addr = "tcp://172.16.20.21:15551"
# ctx = zmq.Context()
# sock = ctx.socket(zmq.PUB)
# sock.connect(pub_addr )
# time.sleep(.5)

def ps_send(thunder,symbol,ps):
    redis.hset(thunder,symbol,ps)
    text = f"{thunder},{symbol},{ps}"
#     sock.send(text.encode())
    redis.publish(f"position_{thunder}",text)
    print(text)


def test():
    ps_send('thunder_lch','XRPUSDT',-15)

if __name__ == '__main__':
    fire.Fire() 
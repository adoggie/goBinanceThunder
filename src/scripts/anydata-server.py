# coding:utf-8
import json

VERSION = '0.1'

import datetime
import time
import traceback

import fire
import pymongo
import redis as Redis

import settings


class Server:
    def __init__(self):
        self.conn = pymongo.MongoClient(**settings.MONGODB)
        self.redis = Redis.StrictRedis(**settings.REDIS)
        self.pubsub = self.redis.pubsub()
        self.pubsub.psubscribe(settings.MESSAGE_PUB_CHAN)

    def on_any_data(self, data):
        source = data.get('source', {})
        delta = data.get('delta', {})

        service_type = source.get('type', '')
        service_id = source.get('id')
        dbname = delta.get('db', '')  #

        db = self.conn[dbname]
        cname = delta.get('table', service_type)
        coll = db[cname]

        content = data.get('content', {})
        content['_source'] = source
        content['_delta'] = delta

        keys = delta.get('update_keys', [])

        filters = {}
        for key in keys:
            if key in content:
                filters[key] = content[key]

        if filters:
            content['_update_time'] = datetime.datetime.now()
            coll.update_one(filter=filters, update={'$set': content}, upsert=True)
        else:
            content['_insert_time'] = datetime.datetime.now()
            content['_update_time'] = datetime.datetime.now()
            coll.insert_one(content)

    def message_recv(self):
        while True:
            try:
                message = self.pubsub.get_message(timeout=.5)
                data = None
                if message:
                    if message['type'] == 'pmessage':
                        # ctx['name'] = message['channel']
                        data = message['data']
                        self.on_any_data( json.loads(data))
            except:
                traceback.print_exc()
                time.sleep(1)

    def run(self):
        print('AnyServer Started ')
        self.message_recv()


def run():
    Server().run()


if __name__ == '__main__':
    fire.Fire()

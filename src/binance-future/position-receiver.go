package main

/*
import (
	"context"
	"fmt"
	"github.com/zeromq/goczmq"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)


var(
	//posReceiver PositionReceiver
)

type PositionReceiver struct {
	config *Config
	mtx 	sync.RWMutex
	positionCache 	map[string] float64
}

func ( p *PositionReceiver) init( config *Config) *PositionReceiver {
	p.config = config
	return p
}

func (p *PositionReceiver) open(ctx context.Context) (stopC <-chan struct{}, err error){
	stopC = make(chan struct{})
	err = nil
	go func() {
		sub:=goczmq.NewSubChanneler(p.config.PositionBroadcastAddress,p.config.ThunderId)
		Loop:
		for {
			select {
			case data := <-sub.RecvChan:
				for _, bytes:= range data{
					fmt.Printf("%s\n",string(bytes))
					text := string(bytes)
					fs:=strings.Split(text,",")
					// thunder_id,symbol,pos
					if len(fs) !=3{
						continue
					}
					if fs[0] != p.config.ThunderId{
						continue
					}
					name:=fs[1]
					pos ,err:=strconv.ParseFloat(fs[2],64)
					if err!=nil{
						continue
					}

					p.mtx.RLock()
					if vol,ok:=p.positionCache[name];ok{
						if vol == pos{
							continue
						}
					}
					p.positionCache[name] = pos
					p.mtx.RUnlock()

					event := &PositionEvent{symbol:name,volume:pos}
					controller.onRecvPosition(*event)
					//
					//, err := filepath.Abs(filepath.Dir(os.Args[0]))
					//if err != nil {
					//	continue
					//	//log.Fatal(err)
					//}

					//p ,_:=os.Getwd()
					fn :=filepath.Join( p.config.PsLogDir ,name)
					if fp ,err := os.OpenFile(fn,os.O_APPEND|os.O_WRONLY|os.O_CREATE,0660); err==nil{
						fp.WriteString(text+"\n")
						fp.Close()
					}

				}
			case <-ctx.Done() :
				fmt.Println("Recv Closed Exit..")
				break Loop
			}
		}
	}()
	return
}


*/

package main

import (
	"fmt"
	"testing"
	"time"
)

func TestChanOver(t *testing.T) {
	doneC := make(chan int, 2)
	doneC <- 1
	doneC <- 2
	doneC <- 3
	fmt.Println("end")
}

func TestStructMap(t *testing.T) {
	type Demo struct {
		name string
		age  int
	}

	datas := map[string]*Demo{"a": {"scott", 10}}

	for k, v := range datas {
		fmt.Println(k, v)
		v.age = 11
	}
	fmt.Println(datas["a"].age)
	fmt.Println(time.Now().Format(time.RFC3339))
}

func TestChanSelect(t *testing.T) {
	doneC := make(chan struct{}, 2)
	stopC := make(chan struct{}, 2)

	go func() {
		select {
		case <-stopC:
			fmt.Println("stop")
		case <-doneC:
			fmt.Println("done")
		}

		fmt.Println("end..")
	}()

	timer := time.After(time.Second * 5)
	go func() {
		fmt.Println("start timer..")
		<-timer
		fmt.Println("timeout!")
		//close(doneC)
		close(stopC)
	}()

	select {}
}

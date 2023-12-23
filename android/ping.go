package main

import (
	"context"
	"time"

	"github.com/fforchino/vector-go-sdk/pkg/vector"
	"github.com/fforchino/vector-go-sdk/pkg/vectorpb"
	"github.com/kercre123/wire-pod/chipper/pkg/logger"
	"github.com/kercre123/wire-pod/chipper/pkg/vars"
)

var PingingStarted bool
var TimeChannel chan int
var TimeUntilNextPing int
var DoReset bool
var PingNowDone chan bool
var PingNowDoneBool bool

func GetTimeChannel() chan int {
	return TimeChannel
}

func GetPingNowDoneChannel() chan bool {
	return PingNowDone
}

func PingNow() {
	DoReset = true
	ctx := context.Background()
	PingAllBots(ctx)
	PingNowDoneBool = true
	select {
	case PingNowDone <- true:
	default:
		PingNowDoneBool = true
	}
}

func UpdateChan(time int) {
	select {
	case TimeChannel <- time:
	default:
	}
}

func Wait() {
	for {
		TimeUntilNextPing = 60
		UpdateChan(TimeUntilNextPing)
		for {
			time.Sleep(time.Second * 1)
			TimeUntilNextPing = TimeUntilNextPing - 1
			UpdateChan(TimeUntilNextPing)
			if TimeUntilNextPing == 0 {
				return
			}
			if DoReset {
				DoReset = false
				TimeUntilNextPing = 60
				if !PingNowDoneBool {
					// hopefully hangs until sent
					for _ = range PingNowDone {
					}
					time.Sleep(time.Second)
				}
				PingNowDoneBool = false
				UpdateChan(TimeUntilNextPing)
			}
		}
	}
}

func PingAllBots(ctx context.Context) {
	for _, robot := range vars.BotInfo.Robots {
		if robot.Activated {
			v, err := vector.New(vector.WithSerialNo(robot.Esn), vector.WithTarget(robot.IPAddress+":443"), vector.WithToken(robot.GUID))
			if err != nil {
				logger.Println(err)
			} else {
				v.Conn.PullJdocs(ctx, &vectorpb.PullJdocsRequest{
					JdocTypes: []vectorpb.JdocType{vectorpb.JdocType_ROBOT_SETTINGS},
				})
			}
		}
	}
}

func PingJdocsInit() {
	TimeChannel = make(chan int, 1)
	PingNowDone = make(chan bool, 1)
}

func PingJdocsStart() {
	ctx, _ := context.WithTimeout(context.Background(), time.Second*5)
	if PingingStarted {
		return
	}
	PingingStarted = true
	for {
		PingAllBots(ctx)
		Wait()
	}
}

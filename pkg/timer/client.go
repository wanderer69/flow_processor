package timer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/wanderer69/flow_processor/pkg/entity"
)

type internalTimer struct {
	processName    string
	timerID        string
	processID      string
	msgs           []*entity.Message
	vars           []*entity.Variable
	activationTime time.Time
}

type TimerValue struct {
	ToTime   string
	Duration *time.Duration
	IsCycle  bool
}

type internalTimerValue struct {
	duration time.Duration
	isCycle  bool
}

type timerHandler struct {
	topicName string
	processID string

	//internalTimer *internalTimer

	fn func(processName, processId, timerID string, t time.Time, msgs []*entity.Message, vars []*entity.Variable) error

	timerValue *internalTimerValue
	ticker     *time.Ticker
	stop       chan bool
}

type Timer struct {
	activateTimerHandlersByTopicNameAndProcessName map[string]*timerHandler

	activate chan *internalTimer

	mu              *sync.Mutex
	msgsRoot        *internalEvent
	msgsCurrent     *internalEvent
	processDuration int
}

const (
	eventTypeActivate string = "activate"
)

type internalEvent struct {
	eventType string
	timer     *internalTimer
	next      *internalEvent
}

func NewTimer(processDuration int) *Timer {
	et := &Timer{
		activateTimerHandlersByTopicNameAndProcessName: make(map[string]*timerHandler),
		activate:        make(chan *internalTimer),
		mu:              &sync.Mutex{},
		processDuration: processDuration,
	}

	go func() {
		ctx := context.Background()
		for {
			time.Sleep(time.Duration(et.processDuration) * time.Millisecond)
			//				for {
			var msg *internalEvent
			msg = nil
			et.mu.Lock()
			if et.msgsRoot != nil {
				msg = et.msgsRoot
				et.msgsRoot = msg.next
				if et.msgsRoot == nil {
					et.msgsCurrent = nil
				}
			}
			et.mu.Unlock()
			if msg != nil {
				switch msg.eventType {
				case eventTypeActivate:
					err := et.ActivateTimer(ctx, msg.timer.processName, msg.timer.processID, msg.timer.timerID, msg.timer.activationTime, msg.timer.msgs, msg.timer.vars)
					if err != nil {
						fmt.Printf("failed call ActivateTopic: %v\r\n", err)
					}
				}
			}
		}
		//			}
	}()

	return et
}

// server
func (et *Timer) Set(ctx context.Context, processName, processID string, timerID string, timerValue *TimerValue, msgs []*entity.Message, vars []*entity.Variable) error {
	currentTimerHandler, ok := et.activateTimerHandlersByTopicNameAndProcessName[timerID+processName]
	if !ok {
		return fmt.Errorf("timer %v in process %v not found", timerID, processName)
	}
	if timerValue == nil {
		return fmt.Errorf("bad timer value")
	}
	if currentTimerHandler.ticker == nil {
		switch {
		case len(timerValue.ToTime) > 0:
			toTime, err := time.Parse("2006-01-02T15:04:05Z", timerValue.ToTime)
			if err != nil {
				return fmt.Errorf("failed parse time %v: %w", timerValue.ToTime, err)
			}

			currentTimerHandler.timerValue = &internalTimerValue{
				duration: time.Since(toTime),
				isCycle:  timerValue.IsCycle,
			}
		case timerValue.Duration != nil:
			/*
				duration, err := durationISO.ParseString(timerValue.Duration)
				if err != nil {
					return fmt.Errorf("failed parse duration %v: %w", timerValue.Duration, err)
				}
			*/
			currentTimerHandler.timerValue = &internalTimerValue{
				duration: *timerValue.Duration,
				isCycle:  timerValue.IsCycle,
			}
		}

		currentTimerHandler.ticker = time.NewTicker(currentTimerHandler.timerValue.duration)
		currentTimerHandler.stop = make(chan bool)
		go func() {
			isStopped := false
			for {
				time.Sleep(time.Duration(et.processDuration) * time.Millisecond)
				select {
				case t := <-currentTimerHandler.ticker.C:
					event := &internalEvent{
						timer: &internalTimer{
							processName:    processName,
							processID:      processID,
							timerID:        timerID,
							activationTime: t,
						},
						eventType: eventTypeActivate,
					}
					et.mu.Lock()
					if et.msgsRoot == nil {
						et.msgsRoot = event
						et.msgsCurrent = et.msgsRoot
					} else {
						event = et.msgsCurrent
						et.msgsCurrent = event
					}
					et.mu.Unlock()
					if !currentTimerHandler.timerValue.isCycle {
						isStopped = true
					}
				case <-currentTimerHandler.stop:
					isStopped = true
				}
				if isStopped {
					break
				}
			}
		}()
	}

	return nil
}

func (et *Timer) SetTimerResponse(ctx context.Context, processName, timerID string, fn func(processName, processId, timerID string, t time.Time, msgs []*entity.Message, vars []*entity.Variable) error) error {
	_, ok := et.activateTimerHandlersByTopicNameAndProcessName[timerID+processName]
	if ok {
		return fmt.Errorf("timer %v found", timerID)
	}

	if fn != nil {
		et.activateTimerHandlersByTopicNameAndProcessName[timerID+processName] = &timerHandler{
			topicName: timerID,
			processID: processName,
			fn:        fn,
		}
	}

	return nil
}

func (et *Timer) ActivateTimer(ctx context.Context, processName, processID, timerID string, t time.Time, msgs []*entity.Message, vars []*entity.Variable) error {
	currentTimerHandler, ok := et.activateTimerHandlersByTopicNameAndProcessName[timerID+processName]
	if !ok {
		return fmt.Errorf("timer %v found", timerID)
	}

	if currentTimerHandler.fn != nil {
		return currentTimerHandler.fn(processName, processID, timerID, t, msgs, vars)
	}
	return nil
}

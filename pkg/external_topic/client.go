package externaltopic

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/wanderer69/flow_processor/pkg/entity"
)

type topic struct {
	processName string
	topicName   string
	processID   string
	msgs        []*entity.Message
	vars        []*entity.Variable
}

type topicHandler struct {
	topicName   string
	processName string
	processID   string

	fn func(processName, processId, topicName string, msgs []*entity.Message, vars []*entity.Variable) error
}

type ExternalTopic struct {
	topicsByProcessName                           map[string][]string
	sendTopicHandlersByTopicNameAndProcessName    map[string]*topicHandler
	recieveTopicHandlersByTopicNameAndProcessName map[string]*topicHandler

	send    chan *topic
	recieve chan *topic

	mu          *sync.Mutex
	msgsRoot    *internalEvent
	msgsCurrent *internalEvent

	processDuration int
}

const (
	eventTypeSend    string = "send"
	eventTypeRecieve string = "recieve"
)

type internalEvent struct {
	eventType string
	topic     *topic
	next      *internalEvent
}

func NewExternalTopic(processDuration int) *ExternalTopic {
	et := &ExternalTopic{
		topicsByProcessName:                           make(map[string][]string),
		sendTopicHandlersByTopicNameAndProcessName:    make(map[string]*topicHandler),
		recieveTopicHandlersByTopicNameAndProcessName: make(map[string]*topicHandler),
		send:            make(chan *topic),
		recieve:         make(chan *topic),
		mu:              &sync.Mutex{},
		processDuration: processDuration,
	}

	go func() {
		for {
			time.Sleep(time.Duration(et.processDuration) * time.Millisecond)
			select {
			case t := <-et.send:
				event := &internalEvent{
					topic:     t,
					eventType: eventTypeSend,
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
			case t := <-et.recieve:
				event := &internalEvent{
					topic:     t,
					eventType: eventTypeRecieve,
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
			}
		}
	}()

	go func() {
		ctx := context.Background()
		for {
			time.Sleep(time.Duration(et.processDuration) * time.Millisecond)
			//			for {
			//				time.Sleep(time.Duration(et.processDuration) * time.Millisecond)
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
				case eventTypeSend:
					err := et.ActivateTopic(ctx, msg.topic.processName, msg.topic.processID, msg.topic.topicName, msg.topic.msgs, msg.topic.vars)
					if err != nil {
						fmt.Printf("failed call ActivateTopic: %v\r\n", err)
					}
				case eventTypeRecieve:
					err := et.RecieveTopic(ctx, msg.topic.processName, msg.topic.processID, msg.topic.topicName, msg.topic.msgs, msg.topic.vars)
					if err != nil {
						fmt.Printf("failed call RecieveTopic: %v\r\n", err)
					}
				}
			}
			//			}
		}
	}()

	return et
}

// server
func (et *ExternalTopic) Init(ctx context.Context, processName string, topicName string) error {
	topics, ok := et.topicsByProcessName[processName]
	if !ok {
		topics = []string{}
	}
	topics = append(topics, topicName)
	et.topicsByProcessName[processName] = topics
	return nil
}

func (et *ExternalTopic) Send(ctx context.Context, processName, processID, topicName string, msgs []*entity.Message, vars []*entity.Variable) error {
	topics, ok := et.topicsByProcessName[processName]
	if !ok {
		return fmt.Errorf("Send: failed get process %v", processName)
	}
	isFound := false
	for i := range topics {
		if topics[i] == topicName {
			isFound = true
		}
	}
	if !isFound {
		return fmt.Errorf("Send: topic %v not found", topicName)
	}
	t := &topic{
		processName: processName,
		topicName:   topicName,
		processID:   processID,
		msgs:        msgs,
		vars:        vars,
	}
	et.send <- t

	return nil
}

func (et *ExternalTopic) SetTopicResponse(ctx context.Context, processName, topicName string, fn func(processName, processId, topicName string, msgs []*entity.Message, vars []*entity.Variable) error) error {
	// устанавливает обработчик топика
	topics, ok := et.topicsByProcessName[processName]
	if !ok {
		return fmt.Errorf("SetTopicResponse: failed get process %v", processName)
	}
	isFound := false
	for i := range topics {
		if topics[i] == topicName {
			_, ok = et.recieveTopicHandlersByTopicNameAndProcessName[topicName+processName]
			if ok {
				isFound = true
			}
		}
	}
	if isFound {
		return fmt.Errorf("SetTopicResponse: topic %v found", topicName)
	}

	if fn != nil {
		et.recieveTopicHandlersByTopicNameAndProcessName[topicName+processName] = &topicHandler{
			topicName:   topicName,
			processName: processName,
			fn:          fn,
		}
	}

	return nil
}

func (et *ExternalTopic) ActivateTopic(ctx context.Context, processName, processID, topicName string, msgs []*entity.Message, vars []*entity.Variable) error {
	// активирует топик
	topics, ok := et.topicsByProcessName[processName]
	if !ok {
		return fmt.Errorf("ActivateTopic: failed get process %v", processName)
	}
	isFound := false
	var currentTopicHandler *topicHandler = nil
	for i := range topics {
		if topics[i] == topicName {
			currentTopicHandler, ok = et.sendTopicHandlersByTopicNameAndProcessName[topicName+processName]
			if ok {
				isFound = true
			}
		}
	}
	if !isFound {
		return fmt.Errorf("ActivateTopic: topic %v found", topicName)
	}

	if currentTopicHandler.fn != nil {
		return currentTopicHandler.fn(processName, processID, topicName, msgs, vars)
	}
	return nil
}

func (et *ExternalTopic) RecieveTopic(ctx context.Context, processName, processID, topicName string, msgs []*entity.Message, vars []*entity.Variable) error {
	// заканчивает топик
	topics, ok := et.topicsByProcessName[processName]
	if !ok {
		return fmt.Errorf("RecieveTopic: failed get process %v", processName)
	}
	isFound := false
	var currentTopicHandler *topicHandler = nil
	for i := range topics {
		if topics[i] == topicName {
			currentTopicHandler, ok = et.recieveTopicHandlersByTopicNameAndProcessName[topicName+processName]
			if ok {
				isFound = true
			}
		}
	}
	if !isFound {
		return fmt.Errorf("RecieveTopic: topic %v found", topicName)
	}

	if currentTopicHandler.fn != nil {
		return currentTopicHandler.fn(processName, processID, topicName, msgs, vars)
	}

	return nil
}

// client
func (et *ExternalTopic) Connect(ctx context.Context, processName string) error {
	// возвращает список топиков у процесса
	return nil
}

func (et *ExternalTopic) SetTopicHandler(ctx context.Context, processName, topicName string, fn func(processName, processId, topicName string, msgs []*entity.Message, vars []*entity.Variable) error) error {
	// устанавливает обработчик топика
	topics, ok := et.topicsByProcessName[processName]
	if !ok {
		return fmt.Errorf("SetTopicHandler: failed get process %v", processName)
	}
	isFound := false
	for i := range topics {
		if topics[i] == topicName {
			_, ok = et.sendTopicHandlersByTopicNameAndProcessName[topicName+processName]
			if ok {
				isFound = true
			}
		}
	}
	if isFound {
		return fmt.Errorf("SetTopicHandler: topic %v found", topicName)
	}

	if fn != nil {
		et.sendTopicHandlersByTopicNameAndProcessName[topicName+processName] = &topicHandler{
			topicName:   topicName,
			processName: processName,
			fn:          fn,
		}
	}
	return nil
}

func (et *ExternalTopic) CompleteTopic(ctx context.Context, processName, processID, topicName string, msgs []*entity.Message, vars []*entity.Variable) error {
	// заканчивает топик
	topics, ok := et.topicsByProcessName[processName]
	if !ok {
		return fmt.Errorf("CompleteTopic: failed get process %v", processName)
	}
	isFound := false
	for i := range topics {
		if topics[i] == topicName {
			isFound = true
		}
	}
	if !isFound {
		return fmt.Errorf("CompleteTopic: topic %v found", topicName)
	}

	t := &topic{
		processName: processName,
		topicName:   topicName,
		processID:   processID,
		msgs:        msgs,
		vars:        vars,
	}
	et.recieve <- t

	return nil
}

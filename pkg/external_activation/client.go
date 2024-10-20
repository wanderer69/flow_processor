package externalactivation

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/wanderer69/flow_processor/pkg/entity"
)

type task struct {
	processName string
	taskName    string
	processID   string
	msgs        []*entity.Message
	vars        []*entity.Variable
}

type taskHandler struct {
	taskName    string
	processName string
	processID   string

	fn func(processName, processId, taskName string, msgs []*entity.Message, vars []*entity.Variable) error
}

type ExternalActivation struct {
	taskNamesByProcessName map[string][]string
	//sendTopicHandlersByTopicNameAndProcessName    map[string]*topicHandler
	recieveTopicHandlersByTopicNameAndProcessName map[string]*taskHandler

	//send    chan *topic
	recieve chan *task

	mu          *sync.Mutex
	msgsRoot    *internalEvent
	msgsCurrent *internalEvent
}

const (
	//eventTypeSend    string = "send"
	eventTypeRecieve string = "recieve"
)

type internalEvent struct {
	eventType string
	topic     *task
	next      *internalEvent
}

func NewExternalActivation() *ExternalActivation {
	et := &ExternalActivation{
		taskNamesByProcessName: make(map[string][]string),
		//sendTopicHandlersByTopicNameAndProcessName:    make(map[string]*topicHandler),
		recieveTopicHandlersByTopicNameAndProcessName: make(map[string]*taskHandler),
		//send:    make(chan *topic),
		recieve: make(chan *task),
		mu:      &sync.Mutex{},
	}

	go func() {
		for {
			select {
			/*
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
			*/
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
			time.Sleep(time.Duration(1) * time.Millisecond)
			for {
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
					/*
						case eventTypeSend:
							err := et.ActivateTopic(ctx, msg.topic.processName, msg.topic.processID, msg.topic.topicName, msg.topic.msgs, msg.topic.vars)
							if err != nil {
								fmt.Printf("failed call ActivateTopic: %v\r\n", err)
							}
					*/
					case eventTypeRecieve:
						err := et.RecieveTopic(ctx, msg.topic.processName, msg.topic.processID, msg.topic.taskName, msg.topic.msgs, msg.topic.vars)
						if err != nil {
							fmt.Printf("failed call RecieveTopic: %v\r\n", err)
						}
					}
				}
			}
		}
	}()

	return et
}

// server
func (et *ExternalActivation) Init(ctx context.Context, processName string, topicName string) error {
	taskNames, ok := et.taskNamesByProcessName[processName]
	if !ok {
		taskNames = []string{}
	}
	taskNames = append(taskNames, topicName)
	et.taskNamesByProcessName[processName] = taskNames
	return nil
}

/*
func (et *ExternalActivation) Send(ctx context.Context, processID, topicName string, msgs []*entity.Message, vars []*entity.Variable) error {
	topics, ok := et.topicsByProcessName[processID]
	if !ok {
		return fmt.Errorf("failed get process %v", processID)
	}
	isFound := false
	for i := range topics {
		if topics[i] == topicName {
			isFound = true
		}
	}
	if !isFound {
		return fmt.Errorf("topic %v not found", topicName)
	}
	t := &topic{
		topicName: topicName,
		processID: processID,
		msgs:      msgs,
		vars:      vars,
	}
	et.send <- t

	return nil
}
*/

func (et *ExternalActivation) SetActivationResponse(ctx context.Context, processName, taskName string, fn func(processName, processId, taskName string, msgs []*entity.Message, vars []*entity.Variable) error) error {
	// устанавливает обработчик топика
	taskNames, ok := et.taskNamesByProcessName[processName]
	if !ok {
		return fmt.Errorf("SetActivationResponse: failed get process %v", processName)
	}
	isFound := false
	for i := range taskNames {
		if taskNames[i] == taskName {
			_, ok = et.recieveTopicHandlersByTopicNameAndProcessName[taskName+processName]
			if ok {
				isFound = true
			}
		}
	}
	if isFound {
		return fmt.Errorf("SetActivationResponse: topic %v found", taskName)
	}

	if fn != nil {
		et.recieveTopicHandlersByTopicNameAndProcessName[taskName+processName] = &taskHandler{
			taskName:    taskName,
			processName: processName,
			fn:          fn,
		}
	}

	return nil
}

/*
	func (et *ExternalActivation) ActivateTopic(ctx context.Context, processName, processID, topicName string, msgs []*entity.Message, vars []*entity.Variable) error {
		// активирует топик
		topics, ok := et.topicsByProcessName[processName]
		if !ok {
			return fmt.Errorf("failed get process %v", processName)
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
			return fmt.Errorf("topic %v found", topicName)
		}

		if currentTopicHandler.fn != nil {
			return currentTopicHandler.fn(processName, processID, topicName, msgs, vars)
		}
		return nil
	}
*/
func (et *ExternalActivation) RecieveTopic(ctx context.Context, processName, processID, topicName string, msgs []*entity.Message, vars []*entity.Variable) error {
	// заканчивает топик
	topics, ok := et.taskNamesByProcessName[processName]
	if !ok {
		return fmt.Errorf("RecieveTopic: failed get process %v", processName)
	}
	isFound := false
	var currentTopicHandler *taskHandler = nil
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
func (et *ExternalActivation) Connect(ctx context.Context, processName string) error {
	// возвращает список топиков у процесса
	return nil
}

/*
	func (et *ExternalActivation) SetTopicHandler(ctx context.Context, processName, topicName string, fn func(processName, processId, topicName string, msgs []*entity.Message, vars []*entity.Variable) error) error {
		// устанавливает обработчик топика
		topics, ok := et.topicsByProcessName[processName]
		if !ok {
			return fmt.Errorf("failed get process %v", processName)
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
			return fmt.Errorf("topic %v found", topicName)
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
*/
func (et *ExternalActivation) CompleteActivation(ctx context.Context, processName, processID, taskName string, msgs []*entity.Message, vars []*entity.Variable) error {
	// заканчивает топик
	topics, ok := et.taskNamesByProcessName[processName]
	if !ok {
		return fmt.Errorf("CompleteActivation: failed get process %v", processName)
	}
	isFound := false
	for i := range topics {
		if topics[i] == taskName {
			isFound = true
		}
	}
	if !isFound {
		return fmt.Errorf("CompleteActivation: topic %v found", taskName)
	}

	t := &task{
		processName: processName,
		taskName:    taskName,
		processID:   processID,
		msgs:        msgs,
		vars:        vars,
	}
	et.recieve <- t

	return nil
}

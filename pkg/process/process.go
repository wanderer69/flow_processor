package process

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/wanderer69/flow_processor/pkg/entity"
	"github.com/wanderer69/flow_processor/pkg/script"
	"github.com/wanderer69/flow_processor/pkg/timer"
)

type ChannelMessage struct {
	CurrentElement *entity.Element

	Variables         []*entity.Variable
	Messages          []*entity.Message
	activationTime    time.Time
	processID         string
	nextElementsNames []string
}

type ProcessElementData struct {
	nextElements []*entity.Element
	waitFlowCnt  int
}

type Process struct {
	UUID string

	Context *entity.Context

	mu                                          *sync.Mutex
	processElementDataByProcessIDAndElementUUID map[string]*ProcessElementData
}

func NewProcess(context *entity.Context) *Process {
	return &Process{
		UUID:    uuid.NewString(),
		mu:      &sync.Mutex{},
		Context: context,
		processElementDataByProcessIDAndElementUUID: make(map[string]*ProcessElementData),
	}
}

func (p *Process) SetData(elementUUID string, ped *ProcessElementData) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.processElementDataByProcessIDAndElementUUID[elementUUID+p.UUID] = ped
}

func (p *Process) GetData(elementUUID string) (*ProcessElementData, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	ped, ok := p.processElementDataByProcessIDAndElementUUID[elementUUID+p.UUID]
	return ped, ok
}

type ProcessExecutor struct {
	UUID                  string
	process               *entity.Process
	processes             []*Process
	executedProcessByUUID map[string]*Process

	internalActivation chan *ChannelMessage
	externalActivation chan *ChannelMessage

	activateTopic   chan *ChannelMessage
	activateTimer   chan *ChannelMessage
	activateMailBox chan *ChannelMessage

	fromTopic   chan *ChannelMessage
	fromTimer   chan *ChannelMessage
	fromMailBox chan *ChannelMessage

	broker  func()
	Stopped chan bool

	elementByTopics             map[string]*entity.Element
	elementByTimerID            map[string]*entity.Element
	elementByMailBoxID          map[string]*entity.Element
	elementByMessageName        map[string][]*entity.Element
	elementByUUID               map[string]*entity.Element
	elementByExternalName       map[string]*entity.Element
	elementByExternalActivation map[string]*entity.Element

	externalTopic           ExternalTopic
	timer                   Timer
	externalActivationAgent ExternalActivation

	fnDebug func(ctx context.Context, msg string) error
}

const (
	eventInternal      string = "internalActivation"
	eventExternal      string = "externalActivation"
	eventActivateTopic string = "activateTopic"
	eventActivateTimer string = "activateTimer"
	eventSendToMailBox string = "sendToMailBox"
	eventFromTopic     string = "fromTopic"
	eventFromTimer     string = "fromTimer"
	eventFromMailBox   string = "fromMailBox"
)

type internalEvent struct {
	eventType string
	msg       *ChannelMessage
	next      *internalEvent
}

func NewProcessExecutor(
	externalTopic ExternalTopic,
	timer Timer,
	externalActivationAgent ExternalActivation,
) *ProcessExecutor {
	ctx := context.Background()
	pe := &ProcessExecutor{
		UUID:               uuid.NewString(),
		internalActivation: make(chan *ChannelMessage, 10),
		externalActivation: make(chan *ChannelMessage),
		Stopped:            make(chan bool),
		activateTopic:      make(chan *ChannelMessage),
		activateTimer:      make(chan *ChannelMessage),
		activateMailBox:    make(chan *ChannelMessage),

		fromTopic:   make(chan *ChannelMessage),
		fromTimer:   make(chan *ChannelMessage),
		fromMailBox: make(chan *ChannelMessage),

		elementByTopics:             make(map[string]*entity.Element),
		elementByTimerID:            make(map[string]*entity.Element),
		elementByMailBoxID:          make(map[string]*entity.Element),
		elementByMessageName:        make(map[string][]*entity.Element),
		elementByUUID:               make(map[string]*entity.Element),
		elementByExternalActivation: make(map[string]*entity.Element),

		elementByExternalName: make(map[string]*entity.Element),
		executedProcessByUUID: make(map[string]*Process),

		externalTopic:           externalTopic,
		timer:                   timer,
		externalActivationAgent: externalActivationAgent,
	}
	broker := func() {
		muEventType := sync.Mutex{}
		var msgsRoot *internalEvent
		var msgsCurrent *internalEvent

		go func() {
			time.Sleep(time.Duration(10) * time.Microsecond)
			for {
				var event *internalEvent = nil
				select {
				case msg := <-pe.internalActivation:
					event = &internalEvent{
						eventType: eventInternal,
						msg:       msg,
					}
				case msg := <-pe.externalActivation:
					event = &internalEvent{
						eventType: eventExternal,
						msg:       msg,
					}
				case msg := <-pe.activateTopic:
					event = &internalEvent{
						eventType: eventActivateTopic,
						msg:       msg,
					}
				case msg := <-pe.activateTimer:
					event = &internalEvent{
						eventType: eventActivateTimer,
						msg:       msg,
					}
				case msg := <-pe.activateMailBox:
					event = &internalEvent{
						eventType: eventSendToMailBox,
						msg:       msg,
					}
				case msg := <-pe.fromTopic:
					event = &internalEvent{
						eventType: eventFromTopic,
						msg:       msg,
					}
				case msg := <-pe.fromTimer:
					event = &internalEvent{
						eventType: eventFromTimer,
						msg:       msg,
					}
				case msg := <-pe.fromMailBox:
					event = &internalEvent{
						eventType: eventFromMailBox,
						msg:       msg,
					}
				}
				//if event != nil {
				muEventType.Lock()
				if msgsRoot == nil {
					msgsRoot = event
					msgsCurrent = msgsRoot
				} else {
					msgsCurrent.next = event
					msgsCurrent = event
				}
				muEventType.Unlock()
				//}
			}
		}()
		go func() {
			for {
				time.Sleep(time.Duration(5) * time.Millisecond)
				for {
					var msg *internalEvent
					msg = nil
					muEventType.Lock()
					if msgsRoot != nil {
						msg = msgsRoot
						msgsRoot = msg.next
						if msgsRoot == nil {
							msgsCurrent = nil
						}
					}
					muEventType.Unlock()
					if msg != nil {
						switch msg.eventType {
						case eventInternal:
							go func() {
								err := pe.NextProcessStep(ctx, msg.msg, false)
								if err != nil {
									fmt.Printf("failed call NextProcessStep: %v\r\n", err)
									pe.Stopped <- true
								}
							}()
						case eventExternal:
							go func() {
								err := pe.NextProcessStep(ctx, msg.msg, true)
								if err != nil {
									fmt.Printf("failed call NextProcessStep: %v\r\n", err)
									pe.Stopped <- true
								}
							}()
						case eventActivateTopic:
							err := pe.SendToExternalTopic(msg.msg)
							if err != nil {
								fmt.Printf("failed call SendToExternalTopic: %v\r\n", err)
								pe.Stopped <- true
							}
						case eventActivateTimer:
							err := pe.ActivateTimer(msg.msg)
							if err != nil {
								fmt.Printf("failed call ActivateTimer: %v\r\n", err)
								pe.Stopped <- true
							}
						case eventFromTopic:
							err := pe.RecieveFromTopic(msg.msg)
							if err != nil {
								fmt.Printf("failed call RecieveFromTopic: %v\r\n", err)
								pe.Stopped <- true
							}
						case eventFromTimer:
							err := pe.RecieveFromTimer(msg.msg)
							if err != nil {
								fmt.Printf("failed call RecieveFromTimer: %v\r\n", err)
								pe.Stopped <- true
							}
						case eventSendToMailBox:
							err := pe.SendToMailBox(msg.msg)
							if err != nil {
								fmt.Printf("failed call SendToMailBox: %v\r\n", err)
								pe.Stopped <- true
							}
						case eventFromMailBox:
							err := pe.RecieveFromMail(msg.msg)
							if err != nil {
								fmt.Printf("failed call RecieveFromMail: %v\r\n", err)
								pe.Stopped <- true
							}
						}
					}
				}
			}
		}()
	}
	pe.broker = broker

	return pe
}

func (pe *ProcessExecutor) SetLogger(ctx context.Context, fn func(ctx context.Context, msg string) error) error {
	pe.fnDebug = fn
	return nil
}

func (pe *ProcessExecutor) CheckElement(element *entity.Element) error {
	switch element.ElementType {
	case entity.ElementTypeStartEvent:
		// может иметь только один выход
		if len(element.OutputsElementID) > 1 {
			return fmt.Errorf("must be only one output")
		}

		element := pe.GetElementByUUID(element.OutputsElementID[0])
		if element.ElementType != entity.ElementTypeFlow {
			return fmt.Errorf("output element must be flow")
		}
		if element.ActivationType == entity.ActivationTypeExternal {
			return fmt.Errorf("must be only internal activation")
		}

	case entity.ElementTypeUserTask:
		// может иметь только один выход
		if len(element.OutputsElementID) > 1 {
			return fmt.Errorf("must be only one output")
		}
		element := pe.GetElementByUUID(element.OutputsElementID[0])
		if element.ElementType != entity.ElementTypeFlow {
			return fmt.Errorf("output element must be flow")
		}

	case entity.ElementTypeServiceTask:
		// может иметь только один выход
		if len(element.OutputsElementID) > 1 {
			return fmt.Errorf("must be only one output")
		}
		element := pe.GetElementByUUID(element.OutputsElementID[0])
		if element.ElementType != entity.ElementTypeFlow {
			return fmt.Errorf("output element must be flow")
		}

	case entity.ElementTypeSendTask:
		// может иметь только один выход
		if len(element.OutputsElementID) > 1 {
			return fmt.Errorf("must be only one output")
		}
		element := pe.GetElementByUUID(element.OutputsElementID[0])
		if element.ElementType != entity.ElementTypeFlow {
			return fmt.Errorf("output element must be flow")
		}

	case entity.ElementTypeReceiveTask:
		// может иметь только один выход
		if len(element.OutputsElementID) > 1 {
			return fmt.Errorf("must be only one output")
		}
		element := pe.GetElementByUUID(element.OutputsElementID[0])
		if element.ElementType != entity.ElementTypeFlow {
			return fmt.Errorf("output element must be flow")
		}

	case entity.ElementTypeManualTask:
		// может иметь только один выход
		if len(element.OutputsElementID) > 1 {
			return fmt.Errorf("must be only one output")
		}
		element := pe.GetElementByUUID(element.OutputsElementID[0])
		if element.ElementType != entity.ElementTypeFlow {
			return fmt.Errorf("output element must be flow")
		}

	case entity.ElementTypeExclusiveGateway:
		if element.ActivationType == entity.ActivationTypeExternal {
			return fmt.Errorf("must be only internal activation")
		}
		// может быть только один вход
		if len(element.InputsElementID) > 1 {
			return fmt.Errorf("must be only one input")
		}
		// может быть несколько выходов, но один без условия, а остальные с условиями
		if len(element.OutputsElementID) < 2 {
			return fmt.Errorf("must be greate 1 outputs")
		}
		isEmptyScript := 0
		for i := range element.OutputsElementID {
			element := pe.GetElementByUUID(element.OutputsElementID[i])
			if element.ElementType != entity.ElementTypeFlow {
				return fmt.Errorf("output element must be flow")
			}

			if len(element.Script) == 0 {
				isEmptyScript++
			}
		}
		if isEmptyScript != 1 {
			return fmt.Errorf("must be only one empty script")
		}

	case entity.ElementTypeParallelGateway:
		if element.ActivationType == entity.ActivationTypeExternal {
			return fmt.Errorf("must be only internal activation")
		}

		for i := range element.OutputsElementID {
			element := pe.GetElementByUUID(element.OutputsElementID[i])
			if element.ElementType != entity.ElementTypeFlow {
				return fmt.Errorf("output element must be flow")
			}
		}

		// либо один вход и несколько выходов, либо несколько входов и один выход
		isCase2 := false
		// может быть только один вход
		if len(element.InputsElementID) > 1 {
			isCase2 = true
		}
		// может быть несколько выходов
		if len(element.OutputsElementID) < 2 {
			isCase2 = true
		}
		if isCase2 {
			// может быть несколько входов
			if len(element.InputsElementID) < 2 {
				isCase2 = false
			}
			// может быть  только один выход
			if len(element.OutputsElementID) > 1 {
				isCase2 = false
			}
			if !isCase2 {
				return fmt.Errorf("must be 1 input and many outputs or many inputs and one output")
			}
		}

	case entity.ElementTypeBusinessRuleTask:
		// может иметь только один выход
		if len(element.OutputsElementID) > 1 {
			return fmt.Errorf("must be only one output")
		}

	case entity.ElementTypeScriptTask:
		// может иметь только один выход
		if len(element.OutputsElementID) > 1 {
			return fmt.Errorf("must be only one output")
		}

	case entity.ElementTypeCallActivity:
		// может иметь только один выход
		if len(element.OutputsElementID) > 1 {
			return fmt.Errorf("must be only one output")
		}

	case entity.ElementTypeIntermediateCatchEvent:
		// может быть только один вход
		if len(element.InputsElementID) > 1 {
			return fmt.Errorf("must be only one input")
		}

		if len(element.OutputsElementID) > 1 {
			return fmt.Errorf("must be only one output")
		}

	case entity.ElementTypeIntermediateThrowEvent:
		// может быть только один вход
		if len(element.InputsElementID) > 1 {
			return fmt.Errorf("must be only one input")
		}

		if len(element.OutputsElementID) > 1 {
			return fmt.Errorf("must be only one output")
		}

	case entity.ElementTypeTask:
		// может иметь только один выход
		if len(element.OutputsElementID) > 1 {
			return fmt.Errorf("must be only one output")
		}

	case entity.ElementTypeEndEvent:
		// не может иметь выход
		if len(element.OutputsElementID) > 0 {
			return fmt.Errorf("must be not have output")
		}

	case entity.ElementTypeFlow:
		// может иметь один вход
		if len(element.InputsElementID) > 1 {
			return fmt.Errorf("must be only one input")
		}
		// может иметь один выход
		if len(element.OutputsElementID) > 1 {
			return fmt.Errorf("must be only one output")
		}

	default:
		return fmt.Errorf("bad element type")
	}
	return nil
}

func (pe *ProcessExecutor) SetProcess(ctx context.Context, process *entity.Process) error {
	pe.process = process
	for i := range pe.process.Elements {
		pe.elementByUUID[pe.process.Elements[i].UUID] = pe.process.Elements[i]
	}

	for i := range pe.process.Elements {
		err := pe.CheckElement(pe.process.Elements[i])
		if err != nil {
			return fmt.Errorf("%v %v %v: %w", pe.process.Elements[i].ElementType, pe.process.Elements[i].CamundaModelerID, pe.process.Elements[i].CamundaModelerName, err)
		}
	}
	// ищем элемент у которого есть только выходы
	for i := range pe.process.Elements {
		if pe.process.Elements[i].IsExternalByTopic {
			if len(pe.process.Elements[i].TopicName) > 0 {
				_, ok := pe.elementByTopics[pe.process.Elements[i].TopicName]
				if ok {
					return fmt.Errorf("topic %v duplicate, must be unique", pe.process.Elements[i].TopicName)
				}
				pe.elementByTopics[pe.process.Elements[i].TopicName] = pe.process.Elements[i]
			}
		}
		if pe.process.Elements[i].IsTimer {
			if len(pe.process.Elements[i].TimerID) > 0 {
				_, ok := pe.elementByTimerID[pe.process.Elements[i].TimerID]
				if ok {
					return fmt.Errorf("timer %v duplicate, must be unique", pe.process.Elements[i].TimerID)
				}
				pe.elementByTimerID[pe.process.Elements[i].TimerID] = pe.process.Elements[i]
			}
		}
		if pe.process.Elements[i].IsRecieveMail {
			if len(pe.process.Elements[i].MailBoxID) > 0 {
				_, ok := pe.elementByTopics[pe.process.Elements[i].MailBoxID]
				if ok {
					return fmt.Errorf("mail box %v duplicate, must be unique", pe.process.Elements[i].MailBoxID)
				}
				pe.elementByMailBoxID[pe.process.Elements[i].MailBoxID] = pe.process.Elements[i]
			}
		}
		if pe.process.Elements[i].IsExternal {
			if len(pe.process.Elements[i].CamundaModelerName) > 0 {
				_, ok := pe.elementByExternalActivation[pe.process.Elements[i].CamundaModelerName]
				if ok {
					return fmt.Errorf("mail box %v duplicate, must be unique", pe.process.Elements[i].CamundaModelerName)
				}
				pe.elementByExternalActivation[pe.process.Elements[i].CamundaModelerName] = pe.process.Elements[i]
			}
		}
		if pe.process.Elements[i].IsRecieveMail {
			for j := range pe.process.Elements[i].InputMessages {
				if len(pe.process.Elements[i].InputMessages[j].Name) > 0 {
					elements, ok := pe.elementByMessageName[pe.process.Elements[i].InputMessages[j].Name]
					if !ok {
						elements = []*entity.Element{}
					}
					elements = append(elements, pe.process.Elements[i])
					pe.elementByMessageName[pe.process.Elements[i].InputMessages[j].Name] = elements
				}
			}
		}
	}

	// инициализируем топики
	for t := range pe.elementByTopics {
		err := pe.externalTopic.Init(ctx, pe.process.Name, t)
		if err != nil {
			return fmt.Errorf("failed init topic %v: %w", t, err)
		}
		// добавляем в топик обработчик
		pe.externalTopic.SetTopicResponse(ctx, pe.process.Name, t, func(processName, processId, topicName string, msgs []*entity.Message, vars []*entity.Variable) error {
			if processName != pe.process.Name {
				return fmt.Errorf("bad process %v", processName)
			}
			if topicName != t {
				return fmt.Errorf("bad topic %v", topicName)
			}
			element, ok := pe.elementByTopics[topicName]
			if !ok {
				return fmt.Errorf("element by topic %v not found", topicName)
			}
			pe.fromTopic <- &ChannelMessage{
				CurrentElement: element,
				Messages:       msgs,
				Variables:      vars,
				processID:      processId,
			}

			return nil
		})
	}
	for t := range pe.elementByTimerID {
		err := pe.timer.SetTimerResponse(ctx, pe.process.Name, t, func(processName, processId, timerID string, tm time.Time, msgs []*entity.Message, vars []*entity.Variable) error {
			if processName != pe.process.Name {
				return fmt.Errorf("bad process %v", processName)
			}
			if timerID != t {
				return fmt.Errorf("bad topic %v", timerID)
			}
			element, ok := pe.elementByTimerID[timerID]
			if !ok {
				return fmt.Errorf("element by topic %v not found", timerID)
			}
			pe.fromTimer <- &ChannelMessage{
				CurrentElement: element,
				Messages:       msgs,
				Variables:      vars,
				activationTime: tm,
				processID:      processId,
			}

			return nil
		})
		if err != nil {
			return fmt.Errorf("failed set timer response: %w", err)
		}
	}

	// инициализируем обработчики внешней активации
	for t := range pe.elementByExternalActivation {
		err := pe.externalActivationAgent.Init(ctx, pe.process.Name, t)
		if err != nil {
			return fmt.Errorf("failed init topic %v: %w", t, err)
		}
		// добавляем в топик обработчик
		pe.externalActivationAgent.SetActivationResponse(ctx, pe.process.Name, t, func(processName, processId, elementName string, msgs []*entity.Message, vars []*entity.Variable) error {
			if processName != pe.process.Name {
				return fmt.Errorf("bad process %v", processName)
			}
			if elementName != t {
				return fmt.Errorf("bad element name %v", elementName)
			}
			element, ok := pe.elementByExternalActivation[elementName]
			if !ok {
				return fmt.Errorf("element by element name %v not found", elementName)
			}
			pe.fromTopic <- &ChannelMessage{
				CurrentElement: element,
				Messages:       msgs,
				Variables:      vars,
				processID:      processId,
			}

			return nil
		})
	}

	go pe.broker()

	return nil
}

func (pe *ProcessExecutor) StartProcess(ctx context.Context, vars []*entity.Variable) (string, error) {
	// ищем элемент у которого есть только выходы
	var currentElements []*entity.Element
	for i := range pe.process.Elements {
		if len(pe.process.Elements[i].InputsElementID) == 0 && len(pe.process.Elements[i].OutputsElementID) > 0 {
			currentElements = append(currentElements, pe.process.Elements[i])
		}
	}
	if currentElements == nil {
		return "", fmt.Errorf("start element not found")
	}

	pContext := &entity.Context{
		VariablesByName: make(map[string]*entity.Variable),
		MessagesByName:  make(map[string]*entity.Message),
	}
	for i := range vars {
		pContext.VariablesByName[vars[i].Name] = vars[i]
	}

	process := NewProcess(pContext)

	pe.processes = append(pe.processes, process)
	pe.executedProcessByUUID[process.UUID] = process

	for i := range currentElements {
		pe.internalActivation <- &ChannelMessage{
			processID:      process.UUID,
			CurrentElement: currentElements[i],
			Variables:      vars,
		}
	}

	return process.UUID, nil
}

func (pe *ProcessExecutor) GetElementByUUID(uuid string) *entity.Element {
	element, ok := pe.elementByUUID[uuid]
	if !ok {
		return nil
	}
	return element
}

func (pe *ProcessExecutor) GetNextElements(uuids []string) []*entity.Element {
	elements := []*entity.Element{}
	for i := range uuids {
		element, ok := pe.elementByUUID[uuids[i]]
		if !ok {
			return nil
		}
		elements = append(elements, element)
	}
	return elements
}

func (pe *ProcessExecutor) GetProcess(uuid string) *Process {
	process, ok := pe.executedProcessByUUID[uuid]
	if !ok {
		return nil
	}
	return process
}

func (pe *ProcessExecutor) StartExecuteElement(msg *ChannelMessage) (bool, error) {
	isExternal := false
	switch msg.CurrentElement.ElementType {
	case entity.ElementTypeStartEvent:
		process := pe.GetProcess(msg.processID)
		nextElements := pe.GetNextElements(msg.CurrentElement.OutputsElementID)
		ped, ok := process.GetData(msg.CurrentElement.UUID)
		if !ok {
			ped = &ProcessElementData{
				nextElements: nextElements,
			}
		}
		process.SetData(msg.CurrentElement.UUID, ped)

	case entity.ElementTypeUserTask:
		if msg.CurrentElement.IsExternal {
			// ожидаем инициации от внешнего процесса
			isExternal = true
		}

		process := pe.GetProcess(msg.processID)
		nextElements := pe.GetNextElements(msg.CurrentElement.OutputsElementID)
		ped, ok := process.GetData(msg.CurrentElement.UUID)
		if !ok {
			ped = &ProcessElementData{
				nextElements: nextElements,
			}
		}
		process.SetData(msg.CurrentElement.UUID, ped)

	case entity.ElementTypeServiceTask:
		process := pe.GetProcess(msg.processID)
		nextElements := pe.GetNextElements(msg.CurrentElement.OutputsElementID)
		ped, ok := process.GetData(msg.CurrentElement.UUID)
		if !ok {
			ped = &ProcessElementData{
				nextElements: nextElements,
			}
		}
		process.SetData(msg.CurrentElement.UUID, ped)

		if msg.CurrentElement.IsExternalByTopic {
			// запускаем внешний процесс
			isExternal = true
		}

	case entity.ElementTypeSendTask:
		process := pe.GetProcess(msg.processID)
		nextElements := pe.GetNextElements(msg.CurrentElement.OutputsElementID)
		ped, ok := process.GetData(msg.CurrentElement.UUID)
		if !ok {
			ped = &ProcessElementData{
				nextElements: nextElements,
			}
		}
		process.SetData(msg.CurrentElement.UUID, ped)

		if msg.CurrentElement.IsExternalByTopic {
			// запускаем внешний процесс
			isExternal = true
		}

	case entity.ElementTypeReceiveTask:
		if msg.CurrentElement.IsRecieveMail {
			// ожидаем инициации от внешнего процесса
			isExternal = true
		}
		process := pe.GetProcess(msg.processID)
		nextElements := pe.GetNextElements(msg.CurrentElement.OutputsElementID)
		ped, ok := process.GetData(msg.CurrentElement.UUID)
		if !ok {
			ped = &ProcessElementData{
				nextElements: nextElements,
			}
		}
		process.SetData(msg.CurrentElement.UUID, ped)

	case entity.ElementTypeManualTask:
		process := pe.GetProcess(msg.processID)
		nextElements := pe.GetNextElements(msg.CurrentElement.OutputsElementID)
		ped, ok := process.GetData(msg.CurrentElement.UUID)
		if !ok {
			ped = &ProcessElementData{
				nextElements: nextElements,
			}
		}
		process.SetData(msg.CurrentElement.UUID, ped)

	case entity.ElementTypeExclusiveGateway:
		isEmptyScript := 0
		for i := range msg.CurrentElement.OutputsElementID {
			element := pe.GetElementByUUID(msg.CurrentElement.OutputsElementID[i])
			if element.ElementType != entity.ElementTypeFlow {
				return false, fmt.Errorf("output element must be flow")
			}

			if len(element.Script) == 0 {
				isEmptyScript++
			}
		}
		if isEmptyScript != 1 {
			return false, fmt.Errorf("must be only one empty script")
		}
		process := pe.GetProcess(msg.processID)
		nextElements := pe.GetNextElements(msg.CurrentElement.OutputsElementID)
		ped, ok := process.GetData(msg.CurrentElement.UUID)
		if !ok {
			ped = &ProcessElementData{
				nextElements: nextElements,
			}
		}
		process.SetData(msg.CurrentElement.UUID, ped)

	case entity.ElementTypeParallelGateway:
		// либо один вход и несколько выходов, либо несколько входов и один выход
		isCase2 := false
		// может быть только один вход
		if len(msg.CurrentElement.InputsElementID) > 1 {
			isCase2 = true
		}
		// может быть несколько выходов
		if len(msg.CurrentElement.OutputsElementID) < 2 {
			isCase2 = true
		}
		if isCase2 {
			// может быть несколько входов
			if len(msg.CurrentElement.InputsElementID) < 2 {
				isCase2 = false
			}
			// может быть  только один выход
			if len(msg.CurrentElement.OutputsElementID) > 1 {
				isCase2 = false
			}
			if !isCase2 {
				return false, fmt.Errorf("must be 1 input and many outputs or many inputs and one output")
			}
		}
		process := pe.GetProcess(msg.processID)
		nextElements := pe.GetNextElements(msg.CurrentElement.OutputsElementID)
		_, ok := process.GetData(msg.CurrentElement.UUID)
		if !ok {
			ped := &ProcessElementData{}
			if isCase2 {
				ped.waitFlowCnt = len(msg.CurrentElement.InputsElementID)
			} else {
				ped.nextElements = nextElements
			}
			process.SetData(msg.CurrentElement.UUID, ped)
		}

	case entity.ElementTypeBusinessRuleTask:
		process := pe.GetProcess(msg.processID)
		nextElements := pe.GetNextElements(msg.CurrentElement.OutputsElementID)
		ped, ok := process.GetData(msg.CurrentElement.UUID)
		if !ok {
			ped = &ProcessElementData{
				nextElements: nextElements,
			}
		}
		process.SetData(msg.CurrentElement.UUID, ped)

		if msg.CurrentElement.IsExternalByTopic {
			// запускаем внешний процесс
			isExternal = true
		}

	case entity.ElementTypeScriptTask:
		process := pe.GetProcess(msg.processID)
		nextElements := pe.GetNextElements(msg.CurrentElement.OutputsElementID)
		ped, ok := process.GetData(msg.CurrentElement.UUID)
		if !ok {
			ped = &ProcessElementData{
				nextElements: nextElements,
			}
		}
		process.SetData(msg.CurrentElement.UUID, ped)

	case entity.ElementTypeCallActivity:
		// может иметь только один выход
		if len(msg.CurrentElement.OutputsElementID) > 1 {
			return false, fmt.Errorf("must be only one output")
		}

	case entity.ElementTypeIntermediateCatchEvent:
		process := pe.GetProcess(msg.processID)
		if msg.CurrentElement.IsTimer {
			// ожидаем инициации от внешнего процесса
			isExternal = true
		}

		if msg.CurrentElement.IsRecieveMail {
			// ожидаем инициации от внешнего процесса
			isExternal = true
		}
		nextElements := pe.GetNextElements(msg.CurrentElement.OutputsElementID)
		ped, ok := process.GetData(msg.CurrentElement.UUID)
		if !ok {
			ped = &ProcessElementData{
				nextElements: nextElements,
			}
		}
		process.SetData(msg.CurrentElement.UUID, ped)

	case entity.ElementTypeIntermediateThrowEvent:
		if msg.CurrentElement.IsSendMail {
			// отправка
			// запускаем инициализацию отправки сообщения
			pe.activateMailBox <- &ChannelMessage{
				CurrentElement: msg.CurrentElement,
				processID:      msg.processID,
			}
		}
		if msg.CurrentElement.IsExternalByTopic {
			isExternal = true
		}
		process := pe.GetProcess(msg.processID)
		nextElements := pe.GetNextElements(msg.CurrentElement.OutputsElementID)
		ped, ok := process.GetData(msg.CurrentElement.UUID)
		if !ok {
			ped = &ProcessElementData{
				nextElements: nextElements,
			}
		}
		process.SetData(msg.CurrentElement.UUID, ped)

	case entity.ElementTypeTask:
		process := pe.GetProcess(msg.processID)
		nextElements := pe.GetNextElements(msg.CurrentElement.OutputsElementID)
		ped, ok := process.GetData(msg.CurrentElement.UUID)
		if !ok {
			ped = &ProcessElementData{
				nextElements: nextElements,
			}
		}
		process.SetData(msg.CurrentElement.UUID, ped)

	case entity.ElementTypeEndEvent:
		// не может иметь выход
		if len(msg.CurrentElement.OutputsElementID) > 0 {
			return false, fmt.Errorf("must be not have output")
		}

	case entity.ElementTypeFlow:
		process := pe.GetProcess(msg.processID)
		nextElements := pe.GetNextElements(msg.CurrentElement.OutputsElementID)
		ped, ok := process.GetData(msg.CurrentElement.UUID)
		if !ok {
			ped = &ProcessElementData{
				nextElements: nextElements,
			}
		}
		process.SetData(msg.CurrentElement.UUID, ped)

	default:
		return false, fmt.Errorf("bad element type")
	}
	return isExternal, nil
}

func (pe *ProcessExecutor) FinishExecuteElement(msg *ChannelMessage) ([]*entity.Element, bool, error) {
	//nextElements := []*entity.Element{}
	currentElement := []*entity.Element{}
	process := pe.GetProcess(msg.processID)
	// логика обработки результата элемента
	switch msg.CurrentElement.ElementType {
	case entity.ElementTypeExclusiveGateway:
		// смотрим у всех выходов скрипт, вычисляем и если никто не выдает true - значит вызываем тот который без
		elements := pe.GetNextElements(msg.CurrentElement.OutputsElementID)
		var defaultElement *entity.Element
		var executedElements []*entity.Element
		for i := range elements {
			if len(elements[i].Script) > 0 {
				lexemas, err := script.ParserLexema(elements[i].Script)
				if err != nil {
					return nil, false, err
				}
				vars, err := script.TranslateLexemaList(lexemas, process.Context)
				if err != nil {
					return nil, false, err
				}
				isExecuted := true
				for j := range vars {
					if vars[j].Name == script.ExecuteResult {
						if vars[j].Value != "true" {
							isExecuted = false
						}
					}
				}
				if isExecuted {
					executedElements = append(executedElements, elements[i])
				}
			}
			if elements[i].IsDefault {
				defaultElement = elements[i]
			}
		}
		if len(executedElements) == 0 {
			ped, ok := process.GetData(msg.CurrentElement.UUID)
			if !ok {
				ped = &ProcessElementData{
					nextElements: []*entity.Element{defaultElement},
				}
			}
			process.SetData(msg.CurrentElement.UUID, ped)
		} else {
			ped, ok := process.GetData(msg.CurrentElement.UUID)
			if !ok {
				ped = &ProcessElementData{
					nextElements: executedElements,
				}
			}
			process.SetData(msg.CurrentElement.UUID, ped)
		}

	case entity.ElementTypeParallelGateway:
		// смотрим сколько входов и ожидаем вызова от всех этих входов. длоя ожидания ставим снова в очередь этот элемент
		for i := range msg.CurrentElement.OutputsElementID {
			element := pe.GetElementByUUID(msg.CurrentElement.OutputsElementID[i])
			if element.ElementType != entity.ElementTypeFlow {
				return nil, false, fmt.Errorf("output element must be flow")
			}
		}

		// либо один вход и несколько выходов, либо несколько входов и один выход
		isCase2 := false
		// может быть только один вход
		if len(msg.CurrentElement.InputsElementID) > 1 {
			isCase2 = true
		}
		// может быть несколько выходов
		if len(msg.CurrentElement.OutputsElementID) < 2 {
			isCase2 = true
		}
		if isCase2 {
			// может быть несколько входов
			if len(msg.CurrentElement.InputsElementID) < 2 {
				isCase2 = false
			}
			// может быть  только один выход
			if len(msg.CurrentElement.OutputsElementID) > 1 {
				isCase2 = false
			}
		}
		if !isCase2 {
			// несколько выходов - значит запускаем несколько процессов
			//elements := pe.GetNextElements(msg.CurrentElement.OutputsElementID)
			//nextElements = elements
			nextElements := pe.GetNextElements(msg.CurrentElement.OutputsElementID)
			ped, ok := process.GetData(msg.CurrentElement.UUID)
			if !ok {
				ped = &ProcessElementData{
					nextElements: nextElements,
				}
			}
			process.SetData(msg.CurrentElement.UUID, ped)
		} else {
			ped, ok := process.GetData(msg.CurrentElement.UUID)
			if !ok {
				ped = &ProcessElementData{
					waitFlowCnt: len(msg.CurrentElement.OutputsElementID),
				}
			}

			// несколько входов - это ожидание окончания
			if ped.waitFlowCnt > 0 {
				ped.waitFlowCnt -= 1
				if ped.waitFlowCnt == 0 {
					elements := pe.GetNextElements(msg.CurrentElement.OutputsElementID)
					ped.nextElements = elements
				}
				process.SetData(msg.CurrentElement.UUID, ped)
			}
		}

	default:
		nextElements := pe.GetNextElements(msg.CurrentElement.OutputsElementID)
		ped, ok := process.GetData(msg.CurrentElement.UUID)
		if !ok {
			ped = &ProcessElementData{
				nextElements: nextElements,
			}
		}
		process.SetData(msg.CurrentElement.UUID, ped)
	}
	if len(msg.CurrentElement.OutputsElementID) == 0 {
		// останавливаем исполнение
		msg.CurrentElement = nil
		pe.Stopped <- true
		return nil, false, nil
	}

	return currentElement, false, nil
}

func (pe *ProcessExecutor) NextProcessStep(ctx context.Context, msg *ChannelMessage, isFinish bool) error {
	isWait := false
	var err error
	if pe.fnDebug != nil {
		pe.fnDebug(ctx, fmt.Sprintf("Current element %v %v %v", msg.CurrentElement.ElementType, msg.CurrentElement.CamundaModelerID, msg.CurrentElement.CamundaModelerName))
	}
	if !isFinish {
		isWait, err = pe.StartExecuteElement(msg)
		if err != nil {
			return err
		}
		if isWait {
			if msg.CurrentElement.IsExternalByTopic {
				// запускаем отдачу в топик
				pe.activateTopic <- &ChannelMessage{
					CurrentElement: msg.CurrentElement,
					processID:      msg.processID,
				}
			}
			if msg.CurrentElement.IsTimer {
				// запускаем инициализацию таймера
				pe.activateTimer <- &ChannelMessage{
					CurrentElement: msg.CurrentElement,
					processID:      msg.processID,
				}
			}
		}
	}
	if !isWait {
		_, _, err := pe.FinishExecuteElement(msg)
		if err != nil {
			return err
		}

		if msg.CurrentElement != nil {
			//fmt.Printf("---> %v %v\r\n", msg.CurrentElement.ElementType, msg.CurrentElement.CamundaModelerName)
			if len(msg.nextElementsNames) > 0 {
				for i := range msg.nextElementsNames {
					element := pe.GetElementByUUID(msg.nextElementsNames[i])
					pe.internalActivation <- &ChannelMessage{
						CurrentElement: element,
						processID:      msg.nextElementsNames[i],
					}
				}
				return nil
			}
			process := pe.GetProcess(msg.processID)
			ped, ok := process.GetData(msg.CurrentElement.UUID)
			if ok {
				for i := range ped.nextElements {
					//fmt.Printf("----> %v %v\r\n", ped.nextElements[i].ElementType, ped.nextElements[i].CamundaModelerName)
					pe.internalActivation <- &ChannelMessage{
						CurrentElement: ped.nextElements[i],
						processID:      process.UUID,
					}
				}
			}
		}
	}
	return nil
}

func (pe *ProcessExecutor) SendToExternalTopic(msg *ChannelMessage) error {
	ctx := context.Background()
	process := pe.GetProcess(msg.processID)

	if pe.fnDebug != nil {
		pe.fnDebug(ctx, fmt.Sprintf("process %v send to topic %v ", process.UUID, msg.CurrentElement.TopicName))
	}

	return pe.externalTopic.Send(ctx, pe.process.Name, process.UUID, msg.CurrentElement.TopicName, msg.Messages, msg.Variables)
}

func (pe *ProcessExecutor) ActivateTimer(msg *ChannelMessage) error {
	ctx := context.Background()
	process := pe.GetProcess(msg.processID)

	if pe.fnDebug != nil {
		pe.fnDebug(ctx, fmt.Sprintf("process %v set timer %v ", process.UUID, msg.CurrentElement.TimerID))
	}

	timerValue := &timer.TimerValue{
		ToTime:   msg.CurrentElement.TimerTime,
		Duration: &msg.CurrentElement.TimerDuration,
		IsCycle:  msg.CurrentElement.TimerIsCycle,
	}

	err := pe.timer.Set(ctx, pe.process.Name, process.UUID, msg.CurrentElement.TimerID, timerValue, msg.CurrentElement.OutputMessages, msg.CurrentElement.InputVars)
	if err != nil {
		return fmt.Errorf("failed activate timer: %w", err)
	}

	return nil
}

func (pe *ProcessExecutor) SendToMailBox(msg *ChannelMessage) error {
	ctx := context.Background()
	process := pe.GetProcess(msg.processID)

	if pe.fnDebug != nil {
		pe.fnDebug(ctx, fmt.Sprintf("process %v send email %v ", process.UUID, msg.CurrentElement.OutputMessages))
	}

	// проходим по подписчикам и кидаем им в цикле сообщения
	for i := range msg.CurrentElement.OutputMessages {
		elements, ok := pe.elementByMessageName[msg.CurrentElement.OutputMessages[i].Name]
		if ok {
			for j := range elements {
				// надо проверить, что у элемента есть возможность получения
				if elements[j].IsRecieveMail {
					if pe.fnDebug != nil {
						pe.fnDebug(ctx, fmt.Sprintf("process %v send mail msg %v to %v %v", process.UUID, msg.Messages, elements[j].ElementType, elements[j].CamundaModelerName))
					}
					pe.fromMailBox <- &ChannelMessage{
						CurrentElement: elements[j],
						Messages:       msg.Messages,
						Variables:      msg.Variables,
						processID:      msg.processID,
					}
				}
			}
		}
	}
	return nil
}

func (pe *ProcessExecutor) RecieveFromTopic(msg *ChannelMessage) error {
	ctx := context.Background()
	process := pe.GetProcess(msg.processID)
	currentElement := msg.CurrentElement

	if pe.fnDebug != nil {
		pe.fnDebug(ctx, fmt.Sprintf("process %v recive from topic %v ", process.UUID, msg.CurrentElement.TopicName))
	}

	for i := range msg.Messages {
		process.Context.MessagesByName[msg.Messages[i].Name] = msg.Messages[i]
	}
	for i := range msg.Variables {
		process.Context.VariablesByName[msg.Variables[i].Name] = msg.Variables[i]
	}

	pe.externalActivation <- &ChannelMessage{
		CurrentElement: currentElement,
		processID:      msg.processID,
	}
	return nil
}

func (pe *ProcessExecutor) RecieveFromTimer(msg *ChannelMessage) error {
	//
	ctx := context.Background()
	process := pe.GetProcess(msg.processID)

	if pe.fnDebug != nil {
		pe.fnDebug(ctx, fmt.Sprintf("process %v recive from timer", process.UUID))
	}

	currentElement := msg.CurrentElement
	for i := range msg.Messages {
		process.Context.MessagesByName[msg.Messages[i].Name] = msg.Messages[i]
	}
	for i := range msg.Variables {
		process.Context.VariablesByName[msg.Variables[i].Name] = msg.Variables[i]
	}

	pe.externalActivation <- &ChannelMessage{
		CurrentElement: currentElement,
		processID:      msg.processID,
	}
	return nil
}

func (pe *ProcessExecutor) RecieveFromMail(msg *ChannelMessage) error {
	//
	ctx := context.Background()
	process := pe.GetProcess(msg.processID)

	currentElement := msg.CurrentElement

	if pe.fnDebug != nil {
		pe.fnDebug(ctx, fmt.Sprintf("process %v recieve from mail", process.UUID))
	}

	for i := range msg.Messages {
		process := pe.GetProcess(msg.processID)
		process.Context.MessagesByName[msg.Messages[i].Name] = msg.Messages[i]
	}
	for i := range msg.Variables {
		process := pe.GetProcess(msg.processID)
		process.Context.VariablesByName[msg.Variables[i].Name] = msg.Variables[i]
	}

	pe.externalActivation <- &ChannelMessage{
		CurrentElement: currentElement,
		processID:      msg.processID,
	}
	return nil
}

func (pe *ProcessExecutor) Set(ctx context.Context, processId string, mailBoxID string, duration string, msgTemplates []*entity.Message) error {

	return nil
}

func (pe *ProcessExecutor) TimerResponse(ctx context.Context, fn func(processId, mailBoxID, msgs *entity.Message) error) error {

	return nil
}

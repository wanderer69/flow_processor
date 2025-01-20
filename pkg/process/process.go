package process

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/wanderer69/flow_processor/pkg/entity"
	"github.com/wanderer69/flow_processor/pkg/script"
	"github.com/wanderer69/flow_processor/pkg/timer"
)

type Process struct {
	UUID          string
	process       *entity.Process
	elementByUUID map[string]*entity.Element

	Context              *entity.Context
	elementByMessageName map[string][]*entity.Element

	Stopped         chan bool
	InternalStopped chan bool

	mu                                          *sync.Mutex
	ProcessElementDataByProcessIDAndElementUUID map[string]*entity.ProcessElementData
	ProcessEndCnt                               int
}

func NewProcess(context *entity.Context) *Process {
	process := &Process{
		UUID:            uuid.NewString(),
		Stopped:         make(chan bool),
		InternalStopped: make(chan bool),
		mu:              &sync.Mutex{},
		Context:         context,
		ProcessElementDataByProcessIDAndElementUUID: make(map[string]*entity.ProcessElementData),
		elementByUUID:        make(map[string]*entity.Element),
		elementByMessageName: make(map[string][]*entity.Element),
	}
	go func() {
		<-process.InternalStopped
		process.Stopped <- true
	}()
	return process
}

func (p *Process) SetData(elementUUID string, ped *entity.ProcessElementData) {
	p.mu.Lock()
	defer p.mu.Unlock()
	//fmt.Printf("set elementUUID %v p.UUID %v\r\n", elementUUID, p.UUID)
	p.ProcessElementDataByProcessIDAndElementUUID[elementUUID+p.UUID] = ped
}

func (p *Process) GetData(elementUUID string) (*entity.ProcessElementData, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	//fmt.Printf("get elementUUID %v p.UUID %v\r\n", elementUUID, p.UUID)
	ped, ok := p.ProcessElementDataByProcessIDAndElementUUID[elementUUID+p.UUID]
	return ped, ok
}

type FinishedProcessData struct {
	ProcessID string
	Error     string
}
type ProcessExecutor struct {
	UUID string
	//	process               *entity.Process
	processByProcessName map[string]*entity.Process

	processes             []*Process
	executedProcessByUUID map[string]*Process

	internalActivation chan *entity.ChannelMessage
	externalActivation chan *entity.ChannelMessage

	activateTopic   chan *entity.ChannelMessage
	activateTimer   chan *entity.ChannelMessage
	activateMailBox chan *entity.ChannelMessage

	fromTopic   chan *entity.ChannelMessage
	fromTimer   chan *entity.ChannelMessage
	fromMailBox chan *entity.ChannelMessage

	Stopped         chan *FinishedProcessData
	InternalStopped chan *FinishedProcessData

	externalTopic           ExternalTopic
	timer                   Timer
	externalActivationAgent ExternalActivation
	storeClient             StoreClient

	fnDebug     func(ctx context.Context, msg string) error
	broker      func()
	mu          *sync.Mutex
	msgsRoot    *InternalEvent
	msgsCurrent *InternalEvent
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

type InternalEvent struct {
	eventType string
	msg       *entity.ChannelMessage
	next      *InternalEvent
}

func NewProcessExecutor(
	externalTopic ExternalTopic,
	timer Timer,
	externalActivationAgent ExternalActivation,
	storeClient StoreClient,
	stop chan struct{},
) *ProcessExecutor {
	ctx := context.Background()
	pe := &ProcessExecutor{
		UUID:               uuid.NewString(),
		internalActivation: make(chan *entity.ChannelMessage, 10),
		externalActivation: make(chan *entity.ChannelMessage),
		Stopped:            make(chan *FinishedProcessData),
		InternalStopped:    make(chan *FinishedProcessData),
		activateTopic:      make(chan *entity.ChannelMessage),
		activateTimer:      make(chan *entity.ChannelMessage),
		activateMailBox:    make(chan *entity.ChannelMessage),

		fromTopic:   make(chan *entity.ChannelMessage),
		fromTimer:   make(chan *entity.ChannelMessage),
		fromMailBox: make(chan *entity.ChannelMessage),

		executedProcessByUUID: make(map[string]*Process),

		externalTopic:           externalTopic,
		timer:                   timer,
		externalActivationAgent: externalActivationAgent,
		storeClient:             storeClient,
		processByProcessName:    make(map[string]*entity.Process),
		mu:                      &sync.Mutex{},
	}
	broker := func() {
		muEventType := sync.Mutex{}
		doubleStop := make(chan struct{})
		go func() {
			time.Sleep(time.Duration(10) * time.Microsecond)
			for {
				var event *InternalEvent = nil
				select {
				case <-stop:
					doubleStop <- struct{}{}
					return
				case msg := <-pe.internalActivation:
					event = &InternalEvent{
						eventType: eventInternal,
						msg:       msg,
					}
				case msg := <-pe.externalActivation:
					event = &InternalEvent{
						eventType: eventExternal,
						msg:       msg,
					}
				case msg := <-pe.activateTopic:
					event = &InternalEvent{
						eventType: eventActivateTopic,
						msg:       msg,
					}
				case msg := <-pe.activateTimer:
					event = &InternalEvent{
						eventType: eventActivateTimer,
						msg:       msg,
					}
				case msg := <-pe.activateMailBox:
					event = &InternalEvent{
						eventType: eventSendToMailBox,
						msg:       msg,
					}
				case msg := <-pe.fromTopic:
					event = &InternalEvent{
						eventType: eventFromTopic,
						msg:       msg,
					}
				case msg := <-pe.fromTimer:
					event = &InternalEvent{
						eventType: eventFromTimer,
						msg:       msg,
					}
				case msg := <-pe.fromMailBox:
					event = &InternalEvent{
						eventType: eventFromMailBox,
						msg:       msg,
					}
				case pes := <-pe.InternalStopped:
					pe.ProcessFinished(ctx, pes.ProcessID, pes.Error)
				}
				//if event != nil {
				muEventType.Lock()
				pe.mu.Lock()
				if pe.msgsRoot == nil {
					pe.msgsRoot = event
					pe.msgsCurrent = pe.msgsRoot
				} else {
					pe.msgsCurrent.next = event
					pe.msgsCurrent = event
				}
				pe.mu.Unlock()
				muEventType.Unlock()
				//}
			}
		}()
		go func() {
			for {
				select {
				case <-doubleStop:
					pe.ProcessExecutorFinished(ctx, "process executor interrupted")
					return
				default:
				}
				time.Sleep(time.Duration(5) * time.Millisecond)
				//	for {
				var msg *InternalEvent
				msg = nil
				muEventType.Lock()
				pe.mu.Lock()
				if pe.msgsRoot != nil {
					msg = pe.msgsRoot
					pe.msgsRoot = msg.next
					if pe.msgsRoot == nil {
						pe.msgsCurrent = nil
					}
				}
				pe.mu.Unlock()
				muEventType.Unlock()
				if msg != nil {
					switch msg.eventType {
					case eventInternal:
						go func() {
							err := pe.NextProcessStep(ctx, msg.msg, false)
							if err != nil {
								errorData := fmt.Sprintf("failed call NextProcessStep: %v", err)
								fmt.Printf("%v\r\n", errorData)
								pe.InternalStopped <- &FinishedProcessData{
									ProcessID: msg.msg.ProcessID,
									Error:     errorData,
								}
							}
						}()
					case eventExternal:
						go func() {
							err := pe.NextProcessStep(ctx, msg.msg, true)
							if err != nil {
								errorData := fmt.Sprintf("failed call NextProcessStep: %v", err)
								fmt.Printf("%v\r\n", errorData)
								pe.InternalStopped <- &FinishedProcessData{
									ProcessID: msg.msg.ProcessID,
									Error:     errorData,
								}
							}
						}()
					case eventActivateTopic:
						err := pe.SendToExternalTopic(msg.msg)
						if err != nil {
							errorData := fmt.Sprintf("failed call SendToExternalTopic: %v", err)
							fmt.Printf("%v\r\n", errorData)
							pe.InternalStopped <- &FinishedProcessData{
								ProcessID: msg.msg.ProcessID,
								Error:     errorData,
							}
						}
					case eventActivateTimer:
						err := pe.ActivateTimer(msg.msg)
						if err != nil {
							errorData := fmt.Sprintf("failed call ActivateTimer: %v", err)
							fmt.Printf("%v\r\n", errorData)
							pe.InternalStopped <- &FinishedProcessData{
								ProcessID: msg.msg.ProcessID,
								Error:     errorData,
							}
						}
					case eventFromTopic:
						err := pe.RecieveFromTopic(msg.msg)
						if err != nil {
							errorData := fmt.Sprintf("failed call RecieveFromTopic: %v", err)
							fmt.Printf("%v\r\n", errorData)
							pe.InternalStopped <- &FinishedProcessData{
								ProcessID: msg.msg.ProcessID,
								Error:     errorData,
							}
						}
					case eventFromTimer:
						err := pe.RecieveFromTimer(msg.msg)
						if err != nil {
							errorData := fmt.Sprintf("failed call RecieveFromTimer: %v", err)
							fmt.Printf("%v\r\n", errorData)
							pe.InternalStopped <- &FinishedProcessData{
								ProcessID: msg.msg.ProcessID,
								Error:     errorData,
							}
						}
					case eventSendToMailBox:
						err := pe.SendToMailBox(msg.msg)
						if err != nil {
							errorData := fmt.Sprintf("failed call SendToMailBox: %v", err)
							fmt.Printf("%v\r\n", errorData)
							pe.InternalStopped <- &FinishedProcessData{
								ProcessID: msg.msg.ProcessID,
								Error:     errorData,
							}
						}
					case eventFromMailBox:
						err := pe.RecieveFromMail(msg.msg)
						if err != nil {
							errorData := fmt.Sprintf("failed call RecieveFromMail: %v", err)
							fmt.Printf("%v\r\n", errorData)
							pe.InternalStopped <- &FinishedProcessData{
								ProcessID: msg.msg.ProcessID,
								Error:     errorData,
							}
						}
					}
				}
				//				}
			}
		}()
	}
	pe.broker = broker
	go broker()

	return pe
}

func (pe *ProcessExecutor) ProcessFinished(ctx context.Context, processID string, errorMsg string) error {
	process := pe.GetProcess(processID)
	if process == nil {
		return fmt.Errorf("process %v not found", processID)
	}
	process.ProcessEndCnt -= 1
	if process.ProcessEndCnt != 0 {
		return nil
	}
	pe.mu.Lock()
	if pe.msgsRoot != nil {
		currentMsg := pe.msgsRoot
		prevMsg := pe.msgsRoot
		for {
			if currentMsg.msg.ProcessID == processID {
				if prevMsg != nil {
					prevMsg.next = currentMsg.next
					prevMsg = prevMsg.next
				}
				if currentMsg == pe.msgsRoot {
					pe.msgsCurrent = currentMsg.next
				}
				if currentMsg == pe.msgsCurrent {
					pe.msgsCurrent = currentMsg.next
				}
			}
			currentMsg = currentMsg.next
			if currentMsg == nil {
				break
			}
		}
	}
	pe.mu.Unlock()

	process.InternalStopped <- true

	delete(pe.executedProcessByUUID, processID)

	pe.storeClient.StoreFinishProcessState(ctx, pe.UUID, processID, "")
	pe.Stopped <- &FinishedProcessData{
		ProcessID: processID,
		Error:     errorMsg,
	}
	return nil
}

func (pe *ProcessExecutor) ProcessExecutorFinished(ctx context.Context, errorMsg string) error {
	pe.Stopped <- &FinishedProcessData{
		Error: errorMsg,
	}
	return nil
}

/*
	func (pe *ProcessExecutor) Load(ctx context.Context) error {
		lst, err := pe.storeClient.LoadStoredProcessesList()
		if err != nil {
			if !strings.Contains(err.Error(), "not found") {
				return err
			}
		}

		for i := range lst {
			processName, processID, elementUUID, state, processContext, err := pe.storeClient.LoadProcessState(lst[i].ProcessName, lst[i].ProcessID)
			if err != nil {
				// skip error
				continue
			}
			// загружаем процесс
			processRaw, err := pe.storeClient.LoadProcessDiagramm(processName)
			if err != nil {
				// skip error
				continue
			}

			err = pe.AddProcess(ctx, processRaw)
			if err != nil {
				// skip error
				continue
			}

			// формируем текущие значения
			_, err = pe.ContinueProcess(ctx, processName, processID, elementUUID, state, processContext)
			if err != nil {
				// skip error
				continue
			}

			// в зависимости от состояния отправляем соответствующие события
		}
		return nil
	}
*/
func (pe *ProcessExecutor) SetLogger(ctx context.Context, fn func(ctx context.Context, msg string) error) error {
	pe.fnDebug = fn
	return nil
}

func (pe *ProcessExecutor) GetStopped() chan *FinishedProcessData {
	return pe.Stopped
}

func (pe *ProcessExecutor) CheckElement(elementByUUID map[string]*entity.Element, element *entity.Element) error {
	switch element.ElementType {
	case entity.ElementTypeStartEvent:
		// может иметь только один выход
		if len(element.OutputsElementID) > 1 {
			return fmt.Errorf("must be only one output")
		}

		element := pe.getElementByUUID(elementByUUID, element.OutputsElementID[0])
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
		element := pe.getElementByUUID(elementByUUID, element.OutputsElementID[0])
		if element.ElementType != entity.ElementTypeFlow {
			return fmt.Errorf("output element must be flow")
		}

	case entity.ElementTypeServiceTask:
		// может иметь только один выход
		if len(element.OutputsElementID) > 1 {
			return fmt.Errorf("must be only one output")
		}
		element := pe.getElementByUUID(elementByUUID, element.OutputsElementID[0])
		if element.ElementType != entity.ElementTypeFlow {
			return fmt.Errorf("output element must be flow")
		}

	case entity.ElementTypeSendTask:
		// может иметь только один выход
		if len(element.OutputsElementID) > 1 {
			return fmt.Errorf("must be only one output")
		}
		element := pe.getElementByUUID(elementByUUID, element.OutputsElementID[0])
		if element.ElementType != entity.ElementTypeFlow {
			return fmt.Errorf("output element must be flow")
		}

	case entity.ElementTypeReceiveTask:
		// может иметь только один выход
		if len(element.OutputsElementID) > 1 {
			return fmt.Errorf("must be only one output")
		}
		element := pe.getElementByUUID(elementByUUID, element.OutputsElementID[0])
		if element.ElementType != entity.ElementTypeFlow {
			return fmt.Errorf("output element must be flow")
		}

	case entity.ElementTypeManualTask:
		// может иметь только один выход
		if len(element.OutputsElementID) > 1 {
			return fmt.Errorf("must be only one output")
		}
		element := pe.getElementByUUID(elementByUUID, element.OutputsElementID[0])
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
			element := pe.getElementByUUID(elementByUUID, element.OutputsElementID[i])
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
			element := pe.getElementByUUID(elementByUUID, element.OutputsElementID[i])
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

func (pe *ProcessExecutor) AddProcess(ctx context.Context, process *entity.Process) error {
	elementByUUID := make(map[string]*entity.Element)
	for i := range process.Elements {
		elementByUUID[process.Elements[i].UUID] = process.Elements[i]
	}

	for i := range process.Elements {
		err := pe.CheckElement(elementByUUID, process.Elements[i])
		if err != nil {
			return fmt.Errorf("%v %v %v: %w", process.Elements[i].ElementType, process.Elements[i].CamundaModelerID, process.Elements[i].CamundaModelerName, err)
		}
	}

	elementByTopics := make(map[string]*entity.Element)
	elementByTimerID := make(map[string]*entity.Element)
	elementByMailBoxID := make(map[string]*entity.Element)
	elementByMessageName := make(map[string][]*entity.Element)
	elementByExternalActivation := make(map[string]*entity.Element)

	// ищем элемент у которого есть только выходы
	for i := range process.Elements {
		if process.Elements[i].IsExternalByTopic {
			if len(process.Elements[i].TopicName) > 0 {
				_, ok := elementByTopics[process.Elements[i].TopicName]
				if ok {
					return fmt.Errorf("topic %v duplicate, must be unique", process.Elements[i].TopicName)
				}
				elementByTopics[process.Elements[i].TopicName] = process.Elements[i]
			}
		}
		if process.Elements[i].IsTimer {
			if len(process.Elements[i].TimerID) > 0 {
				_, ok := elementByTimerID[process.Elements[i].TimerID]
				if ok {
					return fmt.Errorf("timer %v duplicate, must be unique", process.Elements[i].TimerID)
				}
				elementByTimerID[process.Elements[i].TimerID] = process.Elements[i]
			}
		}
		if process.Elements[i].IsRecieveMail {
			if len(process.Elements[i].MailBoxID) > 0 {
				_, ok := elementByTopics[process.Elements[i].MailBoxID]
				if ok {
					return fmt.Errorf("mail box %v duplicate, must be unique", process.Elements[i].MailBoxID)
				}
				elementByMailBoxID[process.Elements[i].MailBoxID] = process.Elements[i]
			}
		}
		if process.Elements[i].IsExternal {
			if len(process.Elements[i].CamundaModelerName) > 0 {
				_, ok := elementByExternalActivation[process.Elements[i].CamundaModelerName]
				if ok {
					return fmt.Errorf("mail box %v duplicate, must be unique", process.Elements[i].CamundaModelerName)
				}
				elementByExternalActivation[process.Elements[i].CamundaModelerName] = process.Elements[i]
			}
		}
		if process.Elements[i].IsRecieveMail {
			for j := range process.Elements[i].InputMessages {
				if len(process.Elements[i].InputMessages[j].Name) > 0 {
					elements, ok := elementByMessageName[process.Elements[i].InputMessages[j].Name]
					if !ok {
						elements = []*entity.Element{}
					}
					elements = append(elements, process.Elements[i])
					elementByMessageName[process.Elements[i].InputMessages[j].Name] = elements
				}
			}
		}
	}

	// инициализируем топики
	for t := range elementByTopics {
		err := pe.externalTopic.Init(ctx, process.Name, t)
		if err != nil {
			return fmt.Errorf("failed init topic %v: %w", t, err)
		}
		// добавляем в топик обработчик
		pe.externalTopic.SetTopicResponse(ctx, process.Name, t, func(processName, processId, topicName string, msgs []*entity.Message, vars []*entity.Variable) error {
			if processName != process.Name {
				return fmt.Errorf("bad process %v", processName)
			}
			if topicName != t {
				return fmt.Errorf("bad topic %v", topicName)
			}
			element, ok := elementByTopics[topicName]
			if !ok {
				return fmt.Errorf("element by topic %v not found", topicName)
			}
			pe.fromTopic <- &entity.ChannelMessage{
				CurrentElement: element,
				Messages:       msgs,
				Variables:      vars,
				ProcessID:      processId,
			}

			return nil
		})
	}
	for t := range elementByTimerID {
		err := pe.timer.SetTimerResponse(ctx, process.Name, t, func(processName, processId, timerID string, tm time.Time, msgs []*entity.Message, vars []*entity.Variable) error {
			if processName != process.Name {
				return fmt.Errorf("bad process %v", processName)
			}
			if timerID != t {
				return fmt.Errorf("bad topic %v", timerID)
			}
			element, ok := elementByTimerID[timerID]
			if !ok {
				return fmt.Errorf("element by topic %v not found", timerID)
			}
			pe.fromTimer <- &entity.ChannelMessage{
				CurrentElement: element,
				Messages:       msgs,
				Variables:      vars,
				ActivationTime: tm,
				ProcessID:      processId,
			}

			return nil
		})
		if err != nil {
			return fmt.Errorf("failed set timer response: %w", err)
		}
	}

	// инициализируем обработчики внешней активации
	for t := range elementByExternalActivation {
		err := pe.externalActivationAgent.Init(ctx, process.Name, t)
		if err != nil {
			return fmt.Errorf("failed init topic %v: %w", t, err)
		}
		// добавляем в топик обработчик
		pe.externalActivationAgent.SetActivationResponse(ctx, process.Name, t, func(processName, processId, elementName string, msgs []*entity.Message, vars []*entity.Variable) error {
			if processName != process.Name {
				return fmt.Errorf("bad process %v", processName)
			}
			if elementName != t {
				return fmt.Errorf("bad element name %v", elementName)
			}
			element, ok := elementByExternalActivation[elementName]
			if !ok {
				return fmt.Errorf("element by element name %v not found", elementName)
			}
			pe.fromTopic <- &entity.ChannelMessage{
				CurrentElement: element,
				Messages:       msgs,
				Variables:      vars,
				ProcessID:      processId,
			}

			return nil
		})
	}

	if len(process.Name) == 0 {
		return fmt.Errorf("process has empty name")
	}
	pe.processByProcessName[process.Name] = process

	return nil
}

func (pe *ProcessExecutor) StartProcess(ctx context.Context, processName string, vars []*entity.Variable) (*Process, error) {
	currentProcess, ok := pe.processByProcessName[processName]
	if !ok {
		return nil, fmt.Errorf("process %v not found", processName)
	}
	// ищем элемент у которого есть только выходы
	var currentElements []*entity.Element
	endCnt := 0
	elementByMessageName := make(map[string][]*entity.Element)
	elementByUUID := make(map[string]*entity.Element)
	for i := range currentProcess.Elements {
		if len(currentProcess.Elements[i].InputsElementID) == 0 && len(currentProcess.Elements[i].OutputsElementID) > 0 {
			currentElements = append(currentElements, currentProcess.Elements[i])
		}
		if currentProcess.Elements[i].IsRecieveMail {
			for j := range currentProcess.Elements[i].InputMessages {
				if len(currentProcess.Elements[i].InputMessages[j].Name) > 0 {
					elements, ok := elementByMessageName[currentProcess.Elements[i].InputMessages[j].Name]
					if !ok {
						elements = []*entity.Element{}
					}
					elements = append(elements, currentProcess.Elements[i])
					elementByMessageName[currentProcess.Elements[i].InputMessages[j].Name] = elements
				}
			}
		}
		elementByUUID[currentProcess.Elements[i].UUID] = currentProcess.Elements[i]
		if currentProcess.Elements[i].ElementType == entity.ElementTypeEndEvent {
			endCnt += 1
		}
	}
	if currentElements == nil {
		return nil, fmt.Errorf("start element not found")
	}

	pContext := &entity.Context{
		VariablesByName: make(map[string]*entity.Variable),
		MessagesByName:  make(map[string]*entity.Message),
	}
	for i := range vars {
		pContext.VariablesByName[vars[i].Name] = vars[i]
	}

	process := NewProcess(pContext)

	process.process = currentProcess
	process.ProcessEndCnt = endCnt
	process.elementByUUID = elementByUUID
	process.elementByMessageName = elementByMessageName

	pe.processes = append(pe.processes, process)
	pe.executedProcessByUUID[process.UUID] = process

	spc := &entity.StoreProcessContext{
		ProcessID:   process.UUID,
		Ctx:         process.Context,
		ProcessName: processName,
	}
	dataRaw, err := json.Marshal(spc)
	if err != nil {
		return nil, err
	}
	err = pe.storeClient.StoreStartProcessState(ctx, pe.UUID, process.UUID, string(dataRaw))
	if err != nil {
		if !strings.Contains(err.Error(), "not found") {
			return nil, err
		}
	}
	for i := range currentElements {
		pe.internalActivation <- &entity.ChannelMessage{
			ProcessID:      process.UUID,
			CurrentElement: currentElements[i],
			Variables:      vars,
		}
	}

	return process, nil
}

func (pe *ProcessExecutor) ContinueProcessExecutor(
	ctx context.Context,
	processExecutor string,
	//processExecutorStateItems []*entity.ProcessExecutorStateItem,
) error {
	/*
		var processID string
		var elementUUID string
		var state string
		var processContext *entity.Context
	*/
	processExecutorStateItems, err := pe.storeClient.LoadProcessStates(ctx, processExecutor)
	if err != nil {
		return err
	}

	for i := range processExecutorStateItems {
		if len(processExecutorStateItems[i].ProcessName) == 0 {
			continue
		}
		currentProcess, ok := pe.processByProcessName[processExecutorStateItems[i].ProcessName]
		if !ok {
			process, err := pe.storeClient.LoadProcessDiagramm(ctx, processExecutorStateItems[i].ProcessName)
			if err != nil {
				return fmt.Errorf("process %v not found", processExecutorStateItems[i].ProcessName)
			}
			pe.processByProcessName[processExecutorStateItems[i].ProcessName] = process
			currentProcess = process
		}
		// ищем последний контекст
		var processContext *entity.Context
		var lastSPC *entity.StoreProcessContext
		var msg *entity.ChannelMessage
		state := ""
		for j := range processExecutorStateItems[i].ProcessStates {
			if len(processExecutorStateItems[i].ProcessStates[j]) == 0 {
				continue
			}
			var spc entity.StoreProcessContext
			err := json.Unmarshal([]byte(processExecutorStateItems[i].ProcessStates[j]), &spc)
			if err != nil {
				return fmt.Errorf("failed unmarshal state %v process %v %v: %w", processExecutorStateItems[i].ProcessStates[j],
					processExecutorStateItems[i].ProcessName, processExecutorStateItems[i].ProcessID, err)
			}
			if spc.Ctx != nil {
				processContext = spc.Ctx
			}
			lastSPC = &spc
			if spc.Msg != nil {
				msg = spc.Msg
			}
		}
		if processContext == nil {
			continue
		}
		if msg == nil {
			continue
		}

		process := NewProcess(processContext)
		process.process = currentProcess

		for i := range currentProcess.Elements {
			process.elementByUUID[currentProcess.Elements[i].UUID] = currentProcess.Elements[i]
		}

		pe.processes = append(pe.processes, process)
		pe.executedProcessByUUID[process.UUID] = process

		state = processExecutorStateItems[i].Execute

		err := pe.ProcessLoad(ctx, state, msg, lastSPC)
		if err != nil {
			return fmt.Errorf("failed load process %v %v: %w", processExecutorStateItems[i].ProcessName,
				processExecutorStateItems[i].ProcessID, err)
		}
	}

	go pe.broker()
	/*
		currentProcess, ok := pe.processByProcessName[processName]
		if !ok {
			return fmt.Errorf("process %v not found", processName)
		}

		// ищем элемент у которого есть только выходы
		var currentElement *entity.Element
		for i := range currentProcess.Elements {
			if currentProcess.Elements[i].UUID == elementUUID {
				currentElement = currentProcess.Elements[i]
			}
		}
		if currentElement == nil {
			return fmt.Errorf("element not found")
		}
	*/

	/*
		for i := range currentElements {
			pe.internalActivation <- &entity.ChannelMessage{
				processID:      process.UUID,
				CurrentElement: currentElements[i],
				Variables:      vars,
			}
		}
	*/

	return nil
}

/*
type ProcessContext struct {
	ProcessID string
	ElementID string
	Ctx       *entity.Context
	State     string
}

type ProcessExecutorContext struct {
	MsgsRoots []*InternalEvent
	Processes []*ProcessContext
}

func (pe *ProcessExecutor) StoreProcessExecutorContext(
	ctx context.Context,
) error {
	pe.mu.Lock()
	// сохраняем очередь
	msgsRoots := []*InternalEvent{}
	if pe.msgsRoot != nil {
		currentMsg := pe.msgsRoot
		for {
			msgsRoots = append(msgsRoots, currentMsg)
			currentMsg = currentMsg.next
			if currentMsg == nil {
				break
			}
		}
	}
	pcs := []*ProcessContext{}
	for i := range pe.processes {
		pc := &ProcessContext{
			ProcessID: pe.processes[i].UUID,
			Ctx:       pe.processes[i].Context,
		}
		pcs = append(pcs, pc)
	}
	pe.mu.Unlock()
	pec := &ProcessExecutorContext{
		MsgsRoots: msgsRoots,
		Processes: pcs,
	}
	dataRaw, err := json.Marshal(pec)
	if err != nil {
		return err
	}
	return pe.storeClient.StoreProcessExecutorState(pe.UUID, string(dataRaw))
}
*/

func (p *Process) GetElementByUUID(uuid string) *entity.Element {
	element, ok := p.elementByUUID[uuid]
	if !ok {
		return nil
	}
	return element
}

func (pe *ProcessExecutor) getElementByUUID(elementByUUID map[string]*entity.Element, uuid string) *entity.Element {
	element, ok := elementByUUID[uuid]
	if !ok {
		return nil
	}
	return element
}

func (p *Process) GetNextElements(uuids []string) []*entity.Element {
	elements := []*entity.Element{}
	for i := range uuids {
		element, ok := p.elementByUUID[uuids[i]]
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

func (pe *ProcessExecutor) GetProcessExecutor() *ProcessExecutor {
	return pe
}

func (pe *ProcessExecutor) StartExecuteElement(msg *entity.ChannelMessage) (bool, error) {
	isExternal := false
	switch msg.CurrentElement.ElementType {
	case entity.ElementTypeStartEvent:
		process := pe.GetProcess(msg.ProcessID)
		nextElements := process.GetNextElements(msg.CurrentElement.OutputsElementID)
		ped, ok := process.GetData(msg.CurrentElement.UUID)
		if !ok {
			ped = &entity.ProcessElementData{
				NextElements: nextElements,
			}
		}
		process.SetData(msg.CurrentElement.UUID, ped)

	case entity.ElementTypeUserTask:
		if msg.CurrentElement.IsExternal {
			// ожидаем инициации от внешнего процесса
			isExternal = true
		}

		process := pe.GetProcess(msg.ProcessID)
		nextElements := process.GetNextElements(msg.CurrentElement.OutputsElementID)
		ped, ok := process.GetData(msg.CurrentElement.UUID)
		if !ok {
			ped = &entity.ProcessElementData{
				NextElements: nextElements,
			}
		}
		process.SetData(msg.CurrentElement.UUID, ped)

	case entity.ElementTypeServiceTask:
		process := pe.GetProcess(msg.ProcessID)
		nextElements := process.GetNextElements(msg.CurrentElement.OutputsElementID)
		ped, ok := process.GetData(msg.CurrentElement.UUID)
		if !ok {
			ped = &entity.ProcessElementData{
				NextElements: nextElements,
			}
		}
		process.SetData(msg.CurrentElement.UUID, ped)

		if msg.CurrentElement.IsExternalByTopic {
			// запускаем внешний процесс
			isExternal = true
		}

	case entity.ElementTypeSendTask:
		process := pe.GetProcess(msg.ProcessID)
		nextElements := process.GetNextElements(msg.CurrentElement.OutputsElementID)
		ped, ok := process.GetData(msg.CurrentElement.UUID)
		if !ok {
			ped = &entity.ProcessElementData{
				NextElements: nextElements,
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
		process := pe.GetProcess(msg.ProcessID)
		nextElements := process.GetNextElements(msg.CurrentElement.OutputsElementID)
		ped, ok := process.GetData(msg.CurrentElement.UUID)
		if !ok {
			ped = &entity.ProcessElementData{
				NextElements: nextElements,
			}
		}
		process.SetData(msg.CurrentElement.UUID, ped)

	case entity.ElementTypeManualTask:
		process := pe.GetProcess(msg.ProcessID)
		nextElements := process.GetNextElements(msg.CurrentElement.OutputsElementID)
		ped, ok := process.GetData(msg.CurrentElement.UUID)
		if !ok {
			ped = &entity.ProcessElementData{
				NextElements: nextElements,
			}
		}
		process.SetData(msg.CurrentElement.UUID, ped)

	case entity.ElementTypeExclusiveGateway:
		isEmptyScript := 0
		process := pe.GetProcess(msg.ProcessID)
		for i := range msg.CurrentElement.OutputsElementID {
			element := process.GetElementByUUID(msg.CurrentElement.OutputsElementID[i])
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
		/*
			nextElements := process.GetNextElements(msg.CurrentElement.OutputsElementID)
			ped, ok := process.GetData(msg.CurrentElement.UUID)
			if !ok {
				ped = &entity.ProcessElementData{
					NextElements: nextElements,
				}
			}
			process.SetData(msg.CurrentElement.UUID, ped)
		*/

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
		/*
			process := pe.GetProcess(msg.ProcessID)
			nextElements := process.GetNextElements(msg.CurrentElement.OutputsElementID)
			_, ok := process.GetData(msg.CurrentElement.UUID)
			if !ok {
				ped := &entity.ProcessElementData{}
				if isCase2 {
					ped.WaitFlowCnt = len(msg.CurrentElement.InputsElementID)
				} else {
									ped.NextElements = nextElements
				}
				process.SetData(msg.CurrentElement.UUID, ped)
			}
		*/
	case entity.ElementTypeBusinessRuleTask:
		process := pe.GetProcess(msg.ProcessID)
		nextElements := process.GetNextElements(msg.CurrentElement.OutputsElementID)
		ped, ok := process.GetData(msg.CurrentElement.UUID)
		if !ok {
			ped = &entity.ProcessElementData{
				NextElements: nextElements,
			}
		}
		process.SetData(msg.CurrentElement.UUID, ped)

		if msg.CurrentElement.IsExternalByTopic {
			// запускаем внешний процесс
			isExternal = true
		}

	case entity.ElementTypeScriptTask:
		process := pe.GetProcess(msg.ProcessID)
		nextElements := process.GetNextElements(msg.CurrentElement.OutputsElementID)
		ped, ok := process.GetData(msg.CurrentElement.UUID)
		if !ok {
			ped = &entity.ProcessElementData{
				NextElements: nextElements,
			}
		}
		process.SetData(msg.CurrentElement.UUID, ped)

	case entity.ElementTypeCallActivity:
		// может иметь только один выход
		if len(msg.CurrentElement.OutputsElementID) > 1 {
			return false, fmt.Errorf("must be only one output")
		}

	case entity.ElementTypeIntermediateCatchEvent:
		process := pe.GetProcess(msg.ProcessID)
		if msg.CurrentElement.IsTimer {
			// ожидаем инициации от внешнего процесса
			isExternal = true
		}

		if msg.CurrentElement.IsRecieveMail {
			// ожидаем инициации от внешнего процесса
			isExternal = true
		}
		nextElements := process.GetNextElements(msg.CurrentElement.OutputsElementID)
		ped, ok := process.GetData(msg.CurrentElement.UUID)
		if !ok {
			ped = &entity.ProcessElementData{
				NextElements: nextElements,
			}
		}
		process.SetData(msg.CurrentElement.UUID, ped)

	case entity.ElementTypeIntermediateThrowEvent:
		if msg.CurrentElement.IsSendMail {
			// отправка
			// запускаем инициализацию отправки сообщения
			pe.activateMailBox <- &entity.ChannelMessage{
				CurrentElement: msg.CurrentElement,
				ProcessID:      msg.ProcessID,
			}
		}
		if msg.CurrentElement.IsExternalByTopic {
			isExternal = true
		}
		process := pe.GetProcess(msg.ProcessID)
		nextElements := process.GetNextElements(msg.CurrentElement.OutputsElementID)
		ped, ok := process.GetData(msg.CurrentElement.UUID)
		if !ok {
			ped = &entity.ProcessElementData{
				NextElements: nextElements,
			}
		}
		process.SetData(msg.CurrentElement.UUID, ped)

	case entity.ElementTypeTask:
		process := pe.GetProcess(msg.ProcessID)
		nextElements := process.GetNextElements(msg.CurrentElement.OutputsElementID)
		ped, ok := process.GetData(msg.CurrentElement.UUID)
		if !ok {
			ped = &entity.ProcessElementData{
				NextElements: nextElements,
			}
		}
		process.SetData(msg.CurrentElement.UUID, ped)

	case entity.ElementTypeEndEvent:
		// не может иметь выход
		if len(msg.CurrentElement.OutputsElementID) > 0 {
			return false, fmt.Errorf("must be not have output")
		}

	case entity.ElementTypeFlow:
		// не может иметь больше одного выхода
		if len(msg.CurrentElement.InputsElementID) > 1 {
			return false, fmt.Errorf("must be not have input")
		}
		if len(msg.CurrentElement.OutputsElementID) > 1 {
			return false, fmt.Errorf("must be not have output")
		}

		process := pe.GetProcess(msg.ProcessID)
		nextElements := process.GetNextElements(msg.CurrentElement.OutputsElementID)
		ped, ok := process.GetData(msg.CurrentElement.UUID)
		if !ok {
			ped = &entity.ProcessElementData{
				NextElements: nextElements,
			}
		}
		process.SetData(msg.CurrentElement.UUID, ped)

	default:
		return false, fmt.Errorf("bad element type")
	}
	return isExternal, nil
}

func (pe *ProcessExecutor) FinishExecuteElement(msg *entity.ChannelMessage) ([]*entity.Element, bool, error) {
	currentElement := []*entity.Element{}
	process := pe.GetProcess(msg.ProcessID)
	// логика обработки результата элемента
	switch msg.CurrentElement.ElementType {
	case entity.ElementTypeExclusiveGateway:
		// смотрим у всех выходов скрипт, вычисляем и если никто не выдает true - значит вызываем тот который без
		elements := process.GetNextElements(msg.CurrentElement.OutputsElementID)
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
				ped = &entity.ProcessElementData{
					NextElements: []*entity.Element{defaultElement},
				}
			}
			process.SetData(msg.CurrentElement.UUID, ped)
		} else {
			ped, ok := process.GetData(msg.CurrentElement.UUID)
			if !ok {
				ped = &entity.ProcessElementData{
					NextElements: executedElements,
				}
			}
			process.SetData(msg.CurrentElement.UUID, ped)
		}

	case entity.ElementTypeParallelGateway:
		// смотрим сколько входов и ожидаем вызова от всех этих входов. для ожидания ставим снова в очередь этот элемент
		process := pe.GetProcess(msg.ProcessID)
		for i := range msg.CurrentElement.OutputsElementID {
			element := process.GetElementByUUID(msg.CurrentElement.OutputsElementID[i])
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
			if true {
				nextElements := process.GetNextElements(msg.CurrentElement.OutputsElementID)
				ped, ok := process.GetData(msg.CurrentElement.UUID)
				if !ok {
					ped = &entity.ProcessElementData{
						NextElements: nextElements,
					}
				}
				process.SetData(msg.CurrentElement.UUID, ped)
			}
		} else {
			ped, ok := process.GetData(msg.CurrentElement.UUID)
			if !ok {
				ped = &entity.ProcessElementData{
					WaitFlowCnt: len(msg.CurrentElement.InputsElementID),
				}
				process.SetData(msg.CurrentElement.UUID, ped)
			}

			// несколько входов - это ожидание окончания
			if ped.WaitFlowCnt > 0 {
				ped.WaitFlowCnt -= 1
				if ped.WaitFlowCnt == 0 {
					elements := process.GetNextElements(msg.CurrentElement.OutputsElementID)
					ped.NextElements = elements
				}
				process.SetData(msg.CurrentElement.UUID, ped)
			}
		}

	default:
		process := pe.GetProcess(msg.ProcessID)
		nextElements := process.GetNextElements(msg.CurrentElement.OutputsElementID)
		ped, ok := process.GetData(msg.CurrentElement.UUID)
		if !ok {
			ped = &entity.ProcessElementData{
				NextElements: nextElements,
			}
		}
		process.SetData(msg.CurrentElement.UUID, ped)
	}
	if len(msg.CurrentElement.OutputsElementID) == 0 {
		//process.Stopped <- true
		return nil, false, nil
	}

	return currentElement, false, nil
}

const (
	CPStateStart      string = "start"
	CPStateIsWait     string = "is_wait"
	CPStateFinish     string = "finish"
	CPStateFinishPost string = "finish_post"
)

func (pe *ProcessExecutor) NextProcessStep(ctx context.Context, msg *entity.ChannelMessage, isFinish bool) error {
	isWait := false
	var err error
	var dataRaw []byte
	if pe.fnDebug != nil {
		pe.fnDebug(ctx, fmt.Sprintf("Current element %v %v %v", msg.CurrentElement.ElementType, msg.CurrentElement.CamundaModelerID, msg.CurrentElement.CamundaModelerName))
	}
	process := pe.GetProcess(msg.ProcessID)
	if !isFinish {
		spc := &entity.StoreProcessContext{
			ProcessID: msg.ProcessID,
			Ctx:       process.Context,
			Msg:       msg,
			IsFinish:  isFinish,
		}

		dataRaw, err = json.Marshal(spc)
		if err != nil {
			return err
		}

		err = pe.storeClient.StoreChangeProcessState(ctx, pe.UUID, msg.ProcessID, CPStateStart, string(dataRaw))
		if err != nil {
			if !strings.Contains(err.Error(), "not found") {
				return err
			}
		}
		isWait, err = pe.StartExecuteElement(msg)
		if err != nil {
			return err
		}
		if isWait {
			spc := &entity.StoreProcessContext{
				ProcessID: msg.ProcessID,
				Ctx:       process.Context,
				//Msg: msg,
				IsWait: isWait,
			}
			dataRaw, err = json.Marshal(spc)
			if err != nil {
				return err
			}

			err = pe.storeClient.StoreChangeProcessState(ctx, pe.UUID, msg.ProcessID, CPStateIsWait, string(dataRaw))
			if err != nil {
				if !strings.Contains(err.Error(), "not found") {
					return err
				}
			}
			if msg.CurrentElement.IsExternalByTopic {
				// запускаем отдачу в топик
				pe.activateTopic <- &entity.ChannelMessage{
					CurrentElement: msg.CurrentElement,
					ProcessID:      msg.ProcessID,
				}
			}
			if msg.CurrentElement.IsTimer {
				// запускаем инициализацию таймера
				pe.activateTimer <- &entity.ChannelMessage{
					CurrentElement: msg.CurrentElement,
					ProcessID:      msg.ProcessID,
				}
			}
		}
	}
	if !isWait {
		spc := &entity.StoreProcessContext{
			ProcessID: msg.ProcessID,
			Ctx:       process.Context,
			Msg:       msg,
			IsWait:    isWait,
		}
		dataRaw, err = json.Marshal(spc)
		if err != nil {
			return err
		}

		err = pe.storeClient.StoreChangeProcessState(ctx, pe.UUID, msg.ProcessID, CPStateFinish, string(dataRaw))
		if err != nil {
			if !strings.Contains(err.Error(), "not found") {
				return err
			}
		}
		_, _, err := pe.FinishExecuteElement(msg)
		if err != nil {
			return err
		}
		if msg.CurrentElement.ElementType == entity.ElementTypeEndEvent {
			//		// останавливаем исполнение
			msg.CurrentElement = nil
			pe.InternalStopped <- &FinishedProcessData{
				ProcessID: process.UUID,
			}
		}

		if msg.CurrentElement != nil {
			spc := &entity.StoreProcessContext{
				ProcessID: msg.ProcessID,
				Ctx:       process.Context,
				//Msg: msg,
			}
			dataRaw, err = json.Marshal(spc)
			if err != nil {
				return err
			}
			ped, isPresent := process.GetData(msg.CurrentElement.UUID)
			if isPresent {
				spc.ProcessElementData = ped
			}
			err = pe.storeClient.StoreChangeProcessState(ctx, pe.UUID, msg.ProcessID, CPStateFinishPost, string(dataRaw))
			if err != nil {
				if !strings.Contains(err.Error(), "not found") {
					return err
				}
			}

			//fmt.Printf("---> %v %v\r\n", msg.CurrentElement.ElementType, msg.CurrentElement.CamundaModelerName)
			process := pe.GetProcess(msg.ProcessID)
			if len(msg.NextElementsNames) > 0 {
				for i := range msg.NextElementsNames {
					element := process.GetElementByUUID(msg.NextElementsNames[i])
					pe.internalActivation <- &entity.ChannelMessage{
						CurrentElement: element,
						ProcessID:      msg.NextElementsNames[i],
					}
				}
				return nil
			}
			//ped, ok := process.GetData(msg.CurrentElement.UUID)
			if isPresent {
				for i := range ped.NextElements {
					//fmt.Printf("----> %v %v\r\n", ped.nextElements[i].ElementType, ped.nextElements[i].CamundaModelerName)
					pe.internalActivation <- &entity.ChannelMessage{
						CurrentElement: ped.NextElements[i],
						ProcessID:      process.UUID,
					}
				}
			}
		}
	}
	return nil
}

func (pe *ProcessExecutor) ProcessLoad(ctx context.Context, state string, msg *entity.ChannelMessage, spc *entity.StoreProcessContext) error {
	isWait := false
	var err error
	var dataRaw []byte

	process := pe.GetProcess(spc.ProcessID)
	// msg := spc.Msg
	if pe.fnDebug != nil {
		pe.fnDebug(ctx, fmt.Sprintf("Current element %v %v %v", msg.CurrentElement.ElementType, msg.CurrentElement.CamundaModelerID, msg.CurrentElement.CamundaModelerName))
	}

	switch state {
	case CPStateStart:
		if isWait {
			spc := &entity.StoreProcessContext{
				ProcessID: msg.ProcessID,
				Ctx:       process.Context,
				//Msg: msg,
			}
			dataRaw, err = json.Marshal(spc)
			if err != nil {
				return err
			}

			err = pe.storeClient.StoreChangeProcessState(ctx, pe.UUID, msg.ProcessID, CPStateIsWait, string(dataRaw))
			if err != nil {
				if !strings.Contains(err.Error(), "not found") {
					return err
				}
			}
			if msg.CurrentElement.IsExternalByTopic {
				// запускаем отдачу в топик
				pe.activateTopic <- &entity.ChannelMessage{
					CurrentElement: msg.CurrentElement,
					ProcessID:      msg.ProcessID,
				}
			}
			if msg.CurrentElement.IsTimer {
				// запускаем инициализацию таймера
				pe.activateTimer <- &entity.ChannelMessage{
					CurrentElement: msg.CurrentElement,
					ProcessID:      msg.ProcessID,
				}
			}
		}
	case CPStateIsWait:

	case CPStateFinish:
		_, _, err := pe.FinishExecuteElement(msg)
		if err != nil {
			return err
		}
		if msg.CurrentElement.ElementType == entity.ElementTypeEndEvent {
			//		// останавливаем исполнение
			msg.CurrentElement = nil
			pe.InternalStopped <- &FinishedProcessData{
				ProcessID: process.UUID,
			}
		}

		if msg.CurrentElement != nil {
			spc := &entity.StoreProcessContext{
				ProcessID: msg.ProcessID,
				Ctx:       process.Context,
				//Msg: msg,
			}
			dataRaw, err = json.Marshal(spc)
			if err != nil {
				return err
			}
			ped, isPresent := process.GetData(msg.CurrentElement.UUID)
			if isPresent {
				spc.ProcessElementData = ped
			}
			err = pe.storeClient.StoreChangeProcessState(ctx, pe.UUID, msg.ProcessID, CPStateFinishPost, string(dataRaw))
			if err != nil {
				if !strings.Contains(err.Error(), "not found") {
					return err
				}
			}

			//fmt.Printf("---> %v %v\r\n", msg.CurrentElement.ElementType, msg.CurrentElement.CamundaModelerName)
			process := pe.GetProcess(msg.ProcessID)
			if len(msg.NextElementsNames) > 0 {
				for i := range msg.NextElementsNames {
					element := process.GetElementByUUID(msg.NextElementsNames[i])
					pe.internalActivation <- &entity.ChannelMessage{
						CurrentElement: element,
						ProcessID:      msg.NextElementsNames[i],
					}
				}
				return nil
			}
			//ped, ok := process.GetData(msg.CurrentElement.UUID)
			if isPresent {
				for i := range ped.NextElements {
					//fmt.Printf("----> %v %v\r\n", ped.nextElements[i].ElementType, ped.nextElements[i].CamundaModelerName)
					pe.internalActivation <- &entity.ChannelMessage{
						CurrentElement: ped.NextElements[i],
						ProcessID:      process.UUID,
					}
				}
			}
		}
	case CPStateFinishPost:
		//fmt.Printf("---> %v %v\r\n", msg.CurrentElement.ElementType, msg.CurrentElement.CamundaModelerName)
		process := pe.GetProcess(msg.ProcessID)
		if len(msg.NextElementsNames) > 0 {
			for i := range msg.NextElementsNames {
				element := process.GetElementByUUID(msg.NextElementsNames[i])
				pe.internalActivation <- &entity.ChannelMessage{
					CurrentElement: element,
					ProcessID:      msg.NextElementsNames[i],
				}
			}
			return nil
		}
		ped, isPresent := process.GetData(msg.CurrentElement.UUID)
		if isPresent {
			for i := range ped.NextElements {
				//fmt.Printf("----> %v %v\r\n", ped.nextElements[i].ElementType, ped.nextElements[i].CamundaModelerName)
				pe.internalActivation <- &entity.ChannelMessage{
					CurrentElement: ped.NextElements[i],
					ProcessID:      process.UUID,
				}
			}
		}
	}
	return nil
}

func (pe *ProcessExecutor) SendToExternalTopic(msg *entity.ChannelMessage) error {
	ctx := context.Background()
	process := pe.GetProcess(msg.ProcessID)

	if pe.fnDebug != nil {
		pe.fnDebug(ctx, fmt.Sprintf("process %v send to topic %v ", process.UUID, msg.CurrentElement.TopicName))
	}

	return pe.externalTopic.Send(ctx, process.process.Name, process.UUID, msg.CurrentElement.TopicName, msg.Messages, msg.Variables)
}

func (pe *ProcessExecutor) ActivateTimer(msg *entity.ChannelMessage) error {
	ctx := context.Background()
	process := pe.GetProcess(msg.ProcessID)

	if pe.fnDebug != nil {
		pe.fnDebug(ctx, fmt.Sprintf("process %v set timer %v ", process.UUID, msg.CurrentElement.TimerID))
	}

	timerValue := &timer.TimerValue{
		ToTime:   msg.CurrentElement.TimerTime,
		Duration: &msg.CurrentElement.TimerDuration,
		IsCycle:  msg.CurrentElement.TimerIsCycle,
	}

	err := pe.timer.Set(ctx, process.process.Name, process.UUID, msg.CurrentElement.TimerID, timerValue, msg.CurrentElement.OutputMessages, msg.CurrentElement.InputVars)
	if err != nil {
		return fmt.Errorf("failed activate timer: %w", err)
	}

	return nil
}

func (pe *ProcessExecutor) SendToMailBox(msg *entity.ChannelMessage) error {
	ctx := context.Background()
	process := pe.GetProcess(msg.ProcessID)

	if pe.fnDebug != nil {
		pe.fnDebug(ctx, fmt.Sprintf("process %v send email %v ", process.UUID, msg.CurrentElement.OutputMessages))
	}

	eventsByElement := make(map[string]*entity.ChannelMessage)
	// проходим по подписчикам и кидаем им в цикле сообщения
	for i := range msg.CurrentElement.OutputMessages {
		elements, ok := process.elementByMessageName[msg.CurrentElement.OutputMessages[i].Name]
		if ok {
			for j := range elements {
				// надо проверить, что у элемента есть возможность получения
				if elements[j].IsRecieveMail {
					if pe.fnDebug != nil {
						pe.fnDebug(ctx, fmt.Sprintf("process %v send mail msg %v to %v '%v'", process.UUID, msg.CurrentElement.OutputMessages[i], elements[j].ElementType, elements[j].CamundaModelerName))
					}
					v, ok := eventsByElement[elements[j].UUID]
					if !ok {
						v = &entity.ChannelMessage{
							CurrentElement: elements[j],
							Variables:      msg.Variables,
							ProcessID:      msg.ProcessID,
						}
					}
					v.Messages = append(v.Messages, msg.CurrentElement.OutputMessages[i])
					eventsByElement[elements[j].UUID] = v
				}
			}
		}
	}
	for _, v := range eventsByElement {
		pe.fromMailBox <- v
	}
	return nil
}

func (pe *ProcessExecutor) RecieveFromTopic(msg *entity.ChannelMessage) error {
	ctx := context.Background()
	process := pe.GetProcess(msg.ProcessID)
	currentElement := msg.CurrentElement

	if pe.fnDebug != nil {
		pe.fnDebug(ctx, fmt.Sprintf("process %v recieve from topic %v ", process.UUID, msg.CurrentElement.TopicName))
	}

	for i := range msg.Messages {
		process.Context.MessagesByName[msg.Messages[i].Name] = msg.Messages[i]
	}
	for i := range msg.Variables {
		process.Context.VariablesByName[msg.Variables[i].Name] = msg.Variables[i]
	}

	pe.externalActivation <- &entity.ChannelMessage{
		CurrentElement: currentElement,
		ProcessID:      msg.ProcessID,
	}
	return nil
}

func (pe *ProcessExecutor) RecieveFromTimer(msg *entity.ChannelMessage) error {
	//
	ctx := context.Background()
	process := pe.GetProcess(msg.ProcessID)

	if pe.fnDebug != nil {
		pe.fnDebug(ctx, fmt.Sprintf("process %v recieve from timer", process.UUID))
	}

	currentElement := msg.CurrentElement
	for i := range msg.Messages {
		process.Context.MessagesByName[msg.Messages[i].Name] = msg.Messages[i]
	}
	for i := range msg.Variables {
		process.Context.VariablesByName[msg.Variables[i].Name] = msg.Variables[i]
	}

	pe.externalActivation <- &entity.ChannelMessage{
		CurrentElement: currentElement,
		ProcessID:      msg.ProcessID,
	}
	return nil
}

func (pe *ProcessExecutor) RecieveFromMail(msg *entity.ChannelMessage) error {
	//
	ctx := context.Background()
	process := pe.GetProcess(msg.ProcessID)

	currentElement := msg.CurrentElement

	if pe.fnDebug != nil {
		pe.fnDebug(ctx, fmt.Sprintf("process %v recieve mail", process.UUID))
	}

	for i := range msg.Messages {
		process := pe.GetProcess(msg.ProcessID)
		process.Context.MessagesByName[msg.Messages[i].Name] = msg.Messages[i]
	}
	for i := range msg.Variables {
		process := pe.GetProcess(msg.ProcessID)
		process.Context.VariablesByName[msg.Variables[i].Name] = msg.Variables[i]
	}

	pe.externalActivation <- &entity.ChannelMessage{
		CurrentElement: currentElement,
		ProcessID:      msg.ProcessID,
	}
	return nil
}

func (pe *ProcessExecutor) Set(ctx context.Context, processId string, mailBoxID string, duration string, msgTemplates []*entity.Message) error {

	return nil
}

func (pe *ProcessExecutor) TimerResponse(ctx context.Context, fn func(processId, mailBoxID, msgs *entity.Message) error) error {

	return nil
}

func (pe *ProcessExecutor) ExternalSendToMailBox(processName, processID, topicName string, msgs []*entity.Message) error {
	ctx := context.Background()
	process := pe.GetProcess(processID)

	if pe.fnDebug != nil {
		pe.fnDebug(ctx, fmt.Sprintf("process %v send email %v ", process.UUID, msgs))
	}

	eventsByElement := make(map[string]*entity.ChannelMessage)
	// проходим по подписчикам и кидаем им в цикле сообщения
	for i := range msgs {
		elements, ok := process.elementByMessageName[msgs[i].Name]
		if ok {
			for j := range elements {
				// надо проверить, что у элемента есть возможность получения
				if elements[j].IsRecieveMail {
					if pe.fnDebug != nil {
						pe.fnDebug(ctx, fmt.Sprintf("process %v send mail msg %v to %v '%v'", process.UUID, msgs[i], elements[j].ElementType, elements[j].CamundaModelerName))
					}
					v, ok := eventsByElement[elements[j].UUID]
					if !ok {
						v = &entity.ChannelMessage{
							CurrentElement: elements[j],
							//Messages:       msg.CurrentElement.OutputMessages,
							//Variables: msg.Variables,
							ProcessID: processID,
						}
					}
					v.Messages = append(v.Messages, msgs[i])
					eventsByElement[elements[j].UUID] = v
				}
			}
		}
	}
	for _, v := range eventsByElement {
		pe.fromMailBox <- v
	}
	return nil
}

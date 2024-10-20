package entity

import "time"

/*
элемент может иметь активацию внутреннюю или внешнюю
элемент может иметь несколько входных переходов и несколько выходных
элемент может получать на входе сообщение и отдавать сообщение на выходе
элемент может забирать переменные из глобального контекста и выдавать переменные в глобальный контекст
элемент может вызывать внешний обработчик и получать результат от внешнего обработчика
*/

type ActivationType string

const (
	ActivationTypeInternal ActivationType = "Internal"
	ActivationTypeExternal ActivationType = "External"
)

type ElementType string

const (
	ElementTypeStartEvent             ElementType = "startEvent"
	ElementTypeServiceTask            ElementType = "serviceTask"
	ElementTypeUserTask               ElementType = "userTask"
	ElementTypeSendTask               ElementType = "sendTask"
	ElementTypeReceiveTask            ElementType = "receiveTask"
	ElementTypeManualTask             ElementType = "manualTask"
	ElementTypeExclusiveGateway       ElementType = "exclusiveGateway"
	ElementTypeParallelGateway        ElementType = "parallelGateway"
	ElementTypeBusinessRuleTask       ElementType = "businessRuleTask"
	ElementTypeScriptTask             ElementType = "scriptTask"
	ElementTypeCallActivity           ElementType = "callActivity"
	ElementTypeIntermediateCatchEvent ElementType = "intermediateCatchEvent"
	ElementTypeIntermediateThrowEvent ElementType = "intermediateThrowEvent"
	ElementTypeTask                   ElementType = "task"
	ElementTypeEndEvent               ElementType = "endEvent"
	ElementTypeFlow                   ElementType = "flow"
)

type Field struct {
	Name  string
	Type  string
	Value string
}

type Message struct {
	Name   string
	Fields []*Field
}

type Element struct {
	UUID               string         `json:"uuid,omitempty"`
	CamundaModelerID   string         `json:"camunda_modeler_id,omitempty"`
	CamundaModelerName string         `json:"camunda_modeler_name,omitempty"`
	ActivationType     ActivationType `json:"activation_type,omitempty"`
	InputsElementID    []string       `json:"inputs,omitempty"`
	OutputsElementID   []string       `json:"outputs,omitempty"`
	InputMessages      []*Message     `json:"input_messages,omitempty"`
	OutputMessages     []*Message     `json:"output_messages,omitempty"`
	IsExternal         bool           `json:"is_external,omitempty"`
	IsTimer            bool           `json:"is_timer,omitempty"`
	IsSendMail         bool           `json:"is_send_mail,omitempty"`
	IsRecieveMail      bool           `json:"is_recive_mail,omitempty"`
	IsExternalByTopic  bool           `json:"is_external_by_topic,omitempty"`
	TopicName          string         `json:"topic_name,omitempty"`
	ElementType        ElementType    `json:"element_type,omitempty"`
	Script             string         `json:"script,omitempty"`
	TimerID            string         `json:"timer_id,omitempty"`
	TimerTime          string         `json:"timer_time,omitempty"`
	TimerDuration      time.Duration  `json:"timer_duration,omitempty"`
	TimerIsCycle       bool           `json:"timer_is_cycle,omitempty"`
	MailBoxID          string         `json:"mail_box_id,omitempty"`
	CamundaID          string         `json:"camunda_id,omitempty"`
	CamundaType        string         `json:"camunda_type,omitempty"`
	Documentation      string         `json:"documentation,omitempty"`
	InputVars          []*Variable    `json:"input_vars,omitempty"`
	OutputVars         []*Variable    `json:"output_vars,omitempty"`
	IsDefault          bool           `json:"is_default,omitempty"`
	//Inputs             []*Element     `json:"inputs,omitempty"`
	//Outputs            []*Element     `json:"outputs,omitempty"`
}

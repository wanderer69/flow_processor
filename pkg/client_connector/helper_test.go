package clientconnector

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/wanderer69/flow_processor/pkg/entity"
	internalformat "github.com/wanderer69/flow_processor/pkg/internal_format"
)

func makeDiagrammI(t *testing.T, processName, topic1, topic2 string) string {
	/*
	   тестовая последовательность
	   1. старт -> подаем переменные a b
	   2. service task -> получает переменные, вызывает внешнюю таску, передает переменные
	   4. service task -> получает переменные, вызывает внешнюю таску, передает переменные
	   5. стоп
	*/
	e1 := &entity.Element{
		UUID:           uuid.NewString(),
		ActivationType: entity.ActivationTypeInternal,
		ElementType:    entity.ElementTypeStartEvent,
	}

	f1 := &entity.Element{
		UUID:           uuid.NewString(),
		ActivationType: entity.ActivationTypeInternal,
		ElementType:    entity.ElementTypeFlow,
	}

	e2 := &entity.Element{
		UUID:              uuid.NewString(),
		ActivationType:    entity.ActivationTypeInternal,
		ElementType:       entity.ElementTypeServiceTask,
		IsExternalByTopic: true,
		TopicName:         topic1,
	}

	f2 := &entity.Element{
		UUID:           uuid.NewString(),
		ActivationType: entity.ActivationTypeInternal,
		ElementType:    entity.ElementTypeFlow,
	}

	e3 := &entity.Element{
		UUID:           uuid.NewString(),
		ActivationType: entity.ActivationTypeExternal,
		ElementType:    entity.ElementTypeUserTask,
	}

	f3 := &entity.Element{
		UUID:           uuid.NewString(),
		ActivationType: entity.ActivationTypeInternal,
		ElementType:    entity.ElementTypeFlow,
	}

	e4 := &entity.Element{
		UUID:              uuid.NewString(),
		ActivationType:    entity.ActivationTypeInternal,
		ElementType:       entity.ElementTypeServiceTask,
		IsExternalByTopic: true,
		TopicName:         topic2,
	}

	f4 := &entity.Element{
		UUID:           uuid.NewString(),
		ActivationType: entity.ActivationTypeInternal,
		ElementType:    entity.ElementTypeFlow,
	}

	e5 := &entity.Element{
		UUID:           uuid.NewString(),
		ActivationType: entity.ActivationTypeInternal,
		ElementType:    entity.ElementTypeEndEvent,
	}

	e1.OutputsElementID = append(e1.OutputsElementID, f1.UUID)

	f1.InputsElementID = append(f1.InputsElementID, e1.UUID)
	f1.OutputsElementID = append(f1.OutputsElementID, e2.UUID)

	e2.InputsElementID = append(e2.InputsElementID, f1.UUID)
	e2.OutputsElementID = append(e2.OutputsElementID, f2.UUID)

	f2.InputsElementID = append(f2.InputsElementID, e2.UUID)
	f2.OutputsElementID = append(f2.OutputsElementID, e3.UUID)

	e3.InputsElementID = append(e3.InputsElementID, f2.UUID)
	e3.OutputsElementID = append(e3.OutputsElementID, f3.UUID)

	f3.InputsElementID = append(f3.InputsElementID, e3.UUID)
	f3.OutputsElementID = append(f3.OutputsElementID, e4.UUID)

	e4.InputsElementID = append(e4.InputsElementID, f3.UUID)
	e4.OutputsElementID = append(e4.OutputsElementID, f4.UUID)

	f4.InputsElementID = append(f4.InputsElementID, e4.UUID)
	f4.OutputsElementID = append(f4.OutputsElementID, e5.UUID)

	e5.InputsElementID = append(e5.InputsElementID, f4.UUID)

	p := &entity.Process{
		Name: processName,
		Elements: []*entity.Element{
			e1,
			f1,
			e2,
			f2,
			e3,
			f3,
			e4,
			f4,
			e5,
		},
	}

	internalFormatClient := internalformat.NewInternalFormat()
	processRaw, err := json.Marshal(p)
	require.NoError(t, err)
	processRawFull, err := internalFormatClient.Store(string(processRaw))
	require.NoError(t, err)
	return processRawFull
}

func makeDiagrammII(t *testing.T, processName, topic1, topic2, taskName3 string) string {
	/*
	   тестовая последовательность
	   1. старт -> подаем переменные a b
	   2. service task -> получает переменные, вызывает внешнюю таску, передает переменные
	   3. user task -> передает полученные переменные
	   4. service task -> получает переменные, вызывает внешнюю таску, передает переменные
	   5. стоп
	*/
	e1 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeInternal,
		ElementType:        entity.ElementTypeStartEvent,
		CamundaModelerID:   "start_1",
		CamundaModelerName: "element_start_1",
	}

	f1 := &entity.Element{
		UUID:           uuid.NewString(),
		ActivationType: entity.ActivationTypeInternal,
		ElementType:    entity.ElementTypeFlow,
	}

	e2 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeInternal,
		ElementType:        entity.ElementTypeServiceTask,
		IsExternalByTopic:  true,
		TopicName:          topic1,
		CamundaModelerID:   "service_task_1",
		CamundaModelerName: "element_service_task_1",
	}

	f2 := &entity.Element{
		UUID:           uuid.NewString(),
		ActivationType: entity.ActivationTypeInternal,
		ElementType:    entity.ElementTypeFlow,
	}

	e3 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeExternal,
		ElementType:        entity.ElementTypeUserTask,
		IsExternal:         true,
		CamundaModelerID:   "user_task_1",
		CamundaModelerName: taskName3,
	}

	f3 := &entity.Element{
		UUID:           uuid.NewString(),
		ActivationType: entity.ActivationTypeInternal,
		ElementType:    entity.ElementTypeFlow,
	}

	e4 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeInternal,
		ElementType:        entity.ElementTypeServiceTask,
		IsExternalByTopic:  true,
		TopicName:          topic2,
		CamundaModelerID:   "service_task_2",
		CamundaModelerName: "element_service_task_2",
	}

	f4 := &entity.Element{
		UUID:           uuid.NewString(),
		ActivationType: entity.ActivationTypeInternal,
		ElementType:    entity.ElementTypeFlow,
	}

	e5 := &entity.Element{
		UUID:               uuid.NewString(),
		ActivationType:     entity.ActivationTypeInternal,
		ElementType:        entity.ElementTypeEndEvent,
		CamundaModelerID:   "end_1",
		CamundaModelerName: "element_end_1",
	}

	e1.OutputsElementID = append(e1.OutputsElementID, f1.UUID)

	f1.InputsElementID = append(f1.InputsElementID, e1.UUID)
	f1.OutputsElementID = append(f1.OutputsElementID, e2.UUID)

	e2.InputsElementID = append(e2.InputsElementID, f1.UUID)
	e2.OutputsElementID = append(e2.OutputsElementID, f2.UUID)

	f2.InputsElementID = append(f2.InputsElementID, e2.UUID)
	f2.OutputsElementID = append(f2.OutputsElementID, e3.UUID)

	e3.InputsElementID = append(e3.InputsElementID, f2.UUID)
	e3.OutputsElementID = append(e3.OutputsElementID, f3.UUID)

	f3.InputsElementID = append(f3.InputsElementID, e3.UUID)
	f3.OutputsElementID = append(f3.OutputsElementID, e4.UUID)

	e4.InputsElementID = append(e4.InputsElementID, f3.UUID)
	e4.OutputsElementID = append(e4.OutputsElementID, f4.UUID)

	f4.InputsElementID = append(f4.InputsElementID, e4.UUID)
	f4.OutputsElementID = append(f4.OutputsElementID, e5.UUID)

	e5.InputsElementID = append(e5.InputsElementID, f4.UUID)

	p := &entity.Process{
		Name: processName,
		Elements: []*entity.Element{
			e1,
			f1,
			e2,
			f2,
			e3,
			f3,
			e4,
			f4,
			e5,
		},
	}

	internalFormatClient := internalformat.NewInternalFormat()
	processRaw, err := json.Marshal(p)
	require.NoError(t, err)
	processRawFull, err := internalFormatClient.Store(string(processRaw))
	require.NoError(t, err)
	return processRawFull
}

func makeDiagrammIII(t *testing.T, processName, topic1, topic2, taskName3 string) string {
	return process2
}

var process2 string = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:bpmndi="http://www.omg.org/spec/BPMN/20100524/DI" xmlns:dc="http://www.omg.org/spec/DD/20100524/DC" xmlns:camunda="http://camunda.org/schema/1.0/bpmn" xmlns:di="http://www.omg.org/spec/DD/20100524/DI" xmlns:modeler="http://camunda.org/schema/modeler/1.0" id="Definitions_036p21f" targetNamespace="http://bpmn.io/schema/bpmn" exporter="Camunda Modeler" exporterVersion="5.26.0" modeler:executionPlatform="Camunda Platform" modeler:executionPlatformVersion="7.21.0">
  <bpmn:process id="Process_0ul3sfu" name="Тест1" isExecutable="true" camunda:versionTag="0.1" camunda:historyTimeToLive="PT1M">
    <bpmn:startEvent id="StartEvent_1">
      <bpmn:outgoing>Flow_1kcirj0</bpmn:outgoing>
    </bpmn:startEvent>
    <bpmn:serviceTask id="Activity_0ad6ndl" name="процесс1" camunda:type="external" camunda:topic="ExecProcess1">
      <bpmn:incoming>Flow_1kcirj0</bpmn:incoming>
      <bpmn:outgoing>Flow_0u3u62p</bpmn:outgoing>
    </bpmn:serviceTask>
    <bpmn:sequenceFlow id="Flow_1kcirj0" sourceRef="StartEvent_1" targetRef="Activity_0ad6ndl" />
    <bpmn:serviceTask id="Activity_0l1xmnl" name="процесс2" camunda:type="external" camunda:topic="ExecProcess2">
      <bpmn:extensionElements>
        <camunda:inputOutput>
          <camunda:outputParameter name="Output_12k9dai" />
        </camunda:inputOutput>
      </bpmn:extensionElements>
      <bpmn:incoming>Flow_11p59rc</bpmn:incoming>
      <bpmn:outgoing>Flow_0109zje</bpmn:outgoing>
    </bpmn:serviceTask>
    <bpmn:sequenceFlow id="Flow_0u3u62p" sourceRef="Activity_0ad6ndl" targetRef="Event_1mfzqut" />
    <bpmn:intermediateCatchEvent id="Event_0usaqoj" name="принять сообщение">
      <bpmn:outgoing>Flow_11p59rc</bpmn:outgoing>
      <bpmn:messageEventDefinition id="MessageEventDefinition_0f31sk3" messageRef="Message_1kij1k3" />
    </bpmn:intermediateCatchEvent>
    <bpmn:intermediateThrowEvent id="Event_1mfzqut" name="отправить сообщение">
      <bpmn:incoming>Flow_0u3u62p</bpmn:incoming>
      <bpmn:messageEventDefinition id="MessageEventDefinition_0fs07p6" messageRef="Message_1kij1k3" camunda:type="external" camunda:topic="SendMessage">
        <bpmn:extensionElements>
          <camunda:field name="Test1">
            <camunda:string>test1_value</camunda:string>
          </camunda:field>
          <camunda:field name="Test2">
            <camunda:string>value_test2</camunda:string>
          </camunda:field>
          <camunda:field name="Test3">
            <camunda:expression>${1}</camunda:expression>
          </camunda:field>
        </bpmn:extensionElements>
      </bpmn:messageEventDefinition>
    </bpmn:intermediateThrowEvent>
    <bpmn:sequenceFlow id="Flow_11p59rc" sourceRef="Event_0usaqoj" targetRef="Activity_0l1xmnl" />
    <bpmn:endEvent id="Event_0xdunvq">
      <bpmn:incoming>Flow_0109zje</bpmn:incoming>
    </bpmn:endEvent>
    <bpmn:sequenceFlow id="Flow_0109zje" sourceRef="Activity_0l1xmnl" targetRef="Event_0xdunvq" />
  </bpmn:process>
  <bpmn:message id="Message_1kij1k3" name="Message_1kij1k3" />
  <bpmndi:BPMNDiagram id="BPMNDiagram_1">
    <bpmndi:BPMNPlane id="BPMNPlane_1" bpmnElement="Process_0ul3sfu">
      <bpmndi:BPMNShape id="Event_0xdunvq_di" bpmnElement="Event_0xdunvq">
        <dc:Bounds x="962" y="99" width="36" height="36" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="BPMNShape_1yw978j" bpmnElement="Activity_0l1xmnl">
        <dc:Bounds x="770" y="77" width="100" height="80" />
        <bpmndi:BPMNLabel />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="Activity_111f528_di" bpmnElement="Activity_0ad6ndl">
        <dc:Bounds x="270" y="77" width="100" height="80" />
        <bpmndi:BPMNLabel />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="_BPMNShape_StartEvent_2" bpmnElement="StartEvent_1">
        <dc:Bounds x="152" y="99" width="36" height="36" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="Event_053sx49_di" bpmnElement="Event_1mfzqut">
        <dc:Bounds x="442" y="99" width="36" height="36" />
        <bpmndi:BPMNLabel>
          <dc:Bounds x="431" y="142" width="58" height="27" />
        </bpmndi:BPMNLabel>
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="Event_0ryiixu_di" bpmnElement="Event_0usaqoj">
        <dc:Bounds x="642" y="99" width="36" height="36" />
        <bpmndi:BPMNLabel>
          <dc:Bounds x="631" y="145" width="58" height="27" />
        </bpmndi:BPMNLabel>
      </bpmndi:BPMNShape>
      <bpmndi:BPMNEdge id="Flow_11p59rc_di" bpmnElement="Flow_11p59rc">
        <di:waypoint x="678" y="117" />
        <di:waypoint x="770" y="117" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_0109zje_di" bpmnElement="Flow_0109zje">
        <di:waypoint x="870" y="117" />
        <di:waypoint x="962" y="117" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_1kcirj0_di" bpmnElement="Flow_1kcirj0">
        <di:waypoint x="188" y="117" />
        <di:waypoint x="270" y="117" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_0u3u62p_di" bpmnElement="Flow_0u3u62p">
        <di:waypoint x="370" y="117" />
        <di:waypoint x="442" y="117" />
      </bpmndi:BPMNEdge>
    </bpmndi:BPMNPlane>
  </bpmndi:BPMNDiagram>
</bpmn:definitions>`

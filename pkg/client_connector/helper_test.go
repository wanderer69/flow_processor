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

var diagramm1 string = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:bpmndi="http://www.omg.org/spec/BPMN/20100524/DI" xmlns:dc="http://www.omg.org/spec/DD/20100524/DC" xmlns:camunda="http://camunda.org/schema/1.0/bpmn" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:di="http://www.omg.org/spec/DD/20100524/DI" xmlns:bioc="http://bpmn.io/schema/bpmn/biocolor/1.0" xmlns:color="http://www.omg.org/spec/BPMN/non-normative/color/1.0" xmlns:modeler="http://camunda.org/schema/modeler/1.0" id="Definitions_1j2zht9" targetNamespace="http://bpmn.io/schema/bpmn" exporter="Camunda Modeler" exporterVersion="5.26.0" modeler:executionPlatform="Camunda Platform" modeler:executionPlatformVersion="7.21.0">
  <bpmn:process id="Process_0qsg7id" name="Процесс выполнения задачи" isExecutable="true" camunda:versionTag="0.2">
    <bpmn:startEvent id="StartEvent_1">
      <bpmn:outgoing>Flow_06dy3gj</bpmn:outgoing>
    </bpmn:startEvent>
    <bpmn:sequenceFlow id="Flow_06dy3gj" sourceRef="StartEvent_1" targetRef="Activity_1ns79tb" />
    <bpmn:exclusiveGateway id="Gateway_0nycwe1" default="Flow_11f0pfi">
      <bpmn:incoming>Flow_0r3970h</bpmn:incoming>
      <bpmn:outgoing>Flow_1k6lsbw</bpmn:outgoing>
      <bpmn:outgoing>Flow_11f0pfi</bpmn:outgoing>
    </bpmn:exclusiveGateway>
    <bpmn:sequenceFlow id="Flow_1xjnpn6" sourceRef="Activity_1ns79tb" targetRef="Gateway_1g9jza3" />
    <bpmn:sequenceFlow id="Flow_0r3970h" sourceRef="Activity_0jkjpcm" targetRef="Gateway_0nycwe1" />
    <bpmn:endEvent id="Event_0v7ilwi">
      <bpmn:incoming>Flow_1ixcdyp</bpmn:incoming>
    </bpmn:endEvent>
    <bpmn:sequenceFlow id="Flow_1k6lsbw" sourceRef="Gateway_0nycwe1" targetRef="Activity_1w6f5j0">
      <bpmn:conditionExpression xsi:type="bpmn:tFormalExpression">${isConditionError}</bpmn:conditionExpression>
    </bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="Flow_1ixcdyp" sourceRef="Activity_1w6f5j0" targetRef="Event_0v7ilwi" />
    <bpmn:exclusiveGateway id="Gateway_0x1tej4" default="Flow_0f65im9">
      <bpmn:incoming>Flow_11f0pfi</bpmn:incoming>
      <bpmn:outgoing>Flow_0f65im9</bpmn:outgoing>
      <bpmn:outgoing>Flow_097m7iv</bpmn:outgoing>
    </bpmn:exclusiveGateway>
    <bpmn:sequenceFlow id="Flow_11f0pfi" sourceRef="Gateway_0nycwe1" targetRef="Gateway_0x1tej4" />
    <bpmn:userTask id="Activity_0olwk38" name="Ожидание факта оплаты">
      <bpmn:incoming>Flow_0a0vipb</bpmn:incoming>
      <bpmn:outgoing>Flow_1j01kwk</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:sequenceFlow id="Flow_0f65im9" sourceRef="Gateway_0x1tej4" targetRef="Activity_1m9c3b5" />
    <bpmn:sequenceFlow id="Flow_0a0vipb" sourceRef="Activity_1m9c3b5" targetRef="Activity_0olwk38" />
    <bpmn:serviceTask id="Activity_1ns79tb" name="Создание задачи" camunda:type="external" camunda:topic="CreateTask">
      <bpmn:extensionElements />
      <bpmn:incoming>Flow_06dy3gj</bpmn:incoming>
      <bpmn:outgoing>Flow_1xjnpn6</bpmn:outgoing>
    </bpmn:serviceTask>
    <bpmn:serviceTask id="Activity_0jkjpcm" name="Проверка условий задачи" camunda:type="external" camunda:topic="CheckTaskCondition">
      <bpmn:incoming>Flow_0yxffkl</bpmn:incoming>
      <bpmn:incoming>Flow_0s3fjbv</bpmn:incoming>
      <bpmn:incoming>Flow_1u3rl74</bpmn:incoming>
      <bpmn:outgoing>Flow_0r3970h</bpmn:outgoing>
    </bpmn:serviceTask>
    <bpmn:serviceTask id="Activity_1w6f5j0" name="Завершение процесса по ошибке 1" camunda:type="external" camunda:topic="FinishedWithError">
      <bpmn:documentation>Завершение процесса по причине ошибок условий</bpmn:documentation>
      <bpmn:extensionElements>
        <camunda:inputOutput>
          <camunda:inputParameter name="Error">check_condition_error</camunda:inputParameter>
        </camunda:inputOutput>
      </bpmn:extensionElements>
      <bpmn:incoming>Flow_1k6lsbw</bpmn:incoming>
      <bpmn:outgoing>Flow_1ixcdyp</bpmn:outgoing>
    </bpmn:serviceTask>
    <bpmn:serviceTask id="Activity_1m9c3b5" name="Оплата задачи" camunda:type="external" camunda:topic="PaymentTask">
      <bpmn:incoming>Flow_0f65im9</bpmn:incoming>
      <bpmn:outgoing>Flow_0a0vipb</bpmn:outgoing>
    </bpmn:serviceTask>
    <bpmn:serviceTask id="Activity_16bzrna" name="Запуск задачи" camunda:type="external" camunda:topic="SendStartTask">
      <bpmn:incoming>Flow_097m7iv</bpmn:incoming>
      <bpmn:incoming>Flow_0fiarqt</bpmn:incoming>
      <bpmn:outgoing>Flow_0fygq6x</bpmn:outgoing>
    </bpmn:serviceTask>
    <bpmn:sequenceFlow id="Flow_1j01kwk" sourceRef="Activity_0olwk38" targetRef="Gateway_0zr8n60" />
    <bpmn:sequenceFlow id="Flow_097m7iv" sourceRef="Gateway_0x1tej4" targetRef="Activity_16bzrna">
      <bpmn:conditionExpression xsi:type="bpmn:tFormalExpression">${isDemo}</bpmn:conditionExpression>
    </bpmn:sequenceFlow>
    <bpmn:exclusiveGateway id="Gateway_0zr8n60" default="Flow_0fiarqt">
      <bpmn:incoming>Flow_1j01kwk</bpmn:incoming>
      <bpmn:outgoing>Flow_0fiarqt</bpmn:outgoing>
      <bpmn:outgoing>Flow_0pubmzz</bpmn:outgoing>
    </bpmn:exclusiveGateway>
    <bpmn:sequenceFlow id="Flow_0fiarqt" sourceRef="Gateway_0zr8n60" targetRef="Activity_16bzrna" />
    <bpmn:endEvent id="Event_17uaw5g">
      <bpmn:incoming>Flow_0l5sp8h</bpmn:incoming>
    </bpmn:endEvent>
    <bpmn:serviceTask id="Activity_0eudc9d" name="Завершение процесса по ошибке 2" camunda:type="external" camunda:topic="FinishedWithError">
      <bpmn:documentation>Завершение процесса из за ошибки оплаты</bpmn:documentation>
      <bpmn:extensionElements>
        <camunda:inputOutput>
          <camunda:inputParameter name="Error">payment_error</camunda:inputParameter>
        </camunda:inputOutput>
      </bpmn:extensionElements>
      <bpmn:incoming>Flow_0pubmzz</bpmn:incoming>
      <bpmn:outgoing>Flow_0l5sp8h</bpmn:outgoing>
    </bpmn:serviceTask>
    <bpmn:sequenceFlow id="Flow_0pubmzz" sourceRef="Gateway_0zr8n60" targetRef="Activity_0eudc9d">
      <bpmn:conditionExpression xsi:type="bpmn:tFormalExpression">${isPaymentError}</bpmn:conditionExpression>
    </bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="Flow_0l5sp8h" sourceRef="Activity_0eudc9d" targetRef="Event_17uaw5g" />
    <bpmn:sequenceFlow id="Flow_0fygq6x" sourceRef="Activity_16bzrna" targetRef="Activity_1h0w2mg" />
    <bpmn:userTask id="Activity_1h0w2mg" name="Ожидание отправки в кластер 1">
      <bpmn:incoming>Flow_0fygq6x</bpmn:incoming>
      <bpmn:outgoing>Flow_1gtjk2f</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:sequenceFlow id="Flow_1gtjk2f" sourceRef="Activity_1h0w2mg" targetRef="Gateway_1c7kods" />
    <bpmn:exclusiveGateway id="Gateway_1c7kods">
      <bpmn:incoming>Flow_1gtjk2f</bpmn:incoming>
      <bpmn:outgoing>Flow_1s5oqef</bpmn:outgoing>
      <bpmn:outgoing>Flow_0v8vwkj</bpmn:outgoing>
    </bpmn:exclusiveGateway>
    <bpmn:serviceTask id="Activity_0paergu" name="Завершение процесса по ошибке 3" camunda:type="external" camunda:topic="FinishedWithError">
      <bpmn:documentation>Завершение процесса из за ошибки оправки в кластер</bpmn:documentation>
      <bpmn:extensionElements>
        <camunda:inputOutput>
          <camunda:inputParameter name="Error">start_send_error</camunda:inputParameter>
        </camunda:inputOutput>
      </bpmn:extensionElements>
      <bpmn:incoming>Flow_1s5oqef</bpmn:incoming>
      <bpmn:outgoing>Flow_15od6mt</bpmn:outgoing>
    </bpmn:serviceTask>
    <bpmn:endEvent id="Event_0h5f8k8">
      <bpmn:incoming>Flow_15od6mt</bpmn:incoming>
    </bpmn:endEvent>
    <bpmn:sequenceFlow id="Flow_15od6mt" sourceRef="Activity_0paergu" targetRef="Event_0h5f8k8" />
    <bpmn:sequenceFlow id="Flow_1s5oqef" sourceRef="Gateway_1c7kods" targetRef="Activity_0paergu">
      <bpmn:conditionExpression xsi:type="bpmn:tFormalExpression">${isSendError}</bpmn:conditionExpression>
    </bpmn:sequenceFlow>
    <bpmn:userTask id="Activity_00zpmb4" name="Ожидание события от кластера или от пользователя">
      <bpmn:incoming>Flow_1hyt4ec</bpmn:incoming>
      <bpmn:incoming>Flow_096mo8m</bpmn:incoming>
      <bpmn:incoming>Flow_0dx6eax</bpmn:incoming>
      <bpmn:incoming>Flow_0jz0iep</bpmn:incoming>
      <bpmn:outgoing>Flow_0h4ksb5</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:exclusiveGateway id="Gateway_1xw251t">
      <bpmn:incoming>Flow_0h4ksb5</bpmn:incoming>
      <bpmn:outgoing>Flow_1t6me9c</bpmn:outgoing>
      <bpmn:outgoing>Flow_1ambyox</bpmn:outgoing>
      <bpmn:outgoing>Flow_1pxdfdk</bpmn:outgoing>
      <bpmn:outgoing>Flow_0k4wjh2</bpmn:outgoing>
      <bpmn:outgoing>Flow_1ymjm8y</bpmn:outgoing>
      <bpmn:outgoing>Flow_1ejesd3</bpmn:outgoing>
    </bpmn:exclusiveGateway>
    <bpmn:serviceTask id="Activity_1h47nnc" name="Задача выполнена" camunda:type="external" camunda:topic="ChangeTaskStateFinishedToNeedPay">
      <bpmn:incoming>Flow_1t6me9c</bpmn:incoming>
      <bpmn:outgoing>Flow_1bm4kfb</bpmn:outgoing>
    </bpmn:serviceTask>
    <bpmn:sequenceFlow id="Flow_1t6me9c" sourceRef="Gateway_1xw251t" targetRef="Activity_1h47nnc">
      <bpmn:conditionExpression xsi:type="bpmn:tFormalExpression">${answerState == "finished"}</bpmn:conditionExpression>
    </bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="Flow_1ambyox" sourceRef="Gateway_1xw251t" targetRef="Activity_0sqydhn">
      <bpmn:conditionExpression xsi:type="bpmn:tFormalExpression">${answerState == "executionError"}</bpmn:conditionExpression>
    </bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="Flow_0h4ksb5" sourceRef="Activity_00zpmb4" targetRef="Gateway_1xw251t" />
    <bpmn:serviceTask id="Activity_0sqydhn" name="Выполнение задачи прервано из-за ошибки" camunda:type="external" camunda:topic="ChangeTaskStateErrorToNeedPay">
      <bpmn:incoming>Flow_1ambyox</bpmn:incoming>
      <bpmn:outgoing>Flow_0pf5r01</bpmn:outgoing>
    </bpmn:serviceTask>
    <bpmn:exclusiveGateway id="Gateway_0dqgtav" default="Flow_1elh8jk">
      <bpmn:incoming>Flow_1bm4kfb</bpmn:incoming>
      <bpmn:incoming>Flow_0pf5r01</bpmn:incoming>
      <bpmn:incoming>Flow_0bws7s0</bpmn:incoming>
      <bpmn:outgoing>Flow_1elh8jk</bpmn:outgoing>
      <bpmn:outgoing>Flow_0dqjff1</bpmn:outgoing>
    </bpmn:exclusiveGateway>
    <bpmn:userTask id="Activity_0xc6f45" name="Ожидание факта выплаты">
      <bpmn:incoming>Flow_0t0rw1l</bpmn:incoming>
      <bpmn:outgoing>Flow_0ya3xhb</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:serviceTask id="Activity_1lvyab6" name="Выплата по задаче" camunda:type="external" camunda:topic="PayTask">
      <bpmn:incoming>Flow_1elh8jk</bpmn:incoming>
      <bpmn:outgoing>Flow_0t0rw1l</bpmn:outgoing>
    </bpmn:serviceTask>
    <bpmn:serviceTask id="Activity_0apgv3s" name="Завершение задачи" camunda:type="external" camunda:topic="ChangeTaskStateToFinishedOrErrorOrStoppedOrTerminated">
      <bpmn:incoming>Flow_0dqjff1</bpmn:incoming>
      <bpmn:incoming>Flow_12hi27d</bpmn:incoming>
      <bpmn:outgoing>Flow_0959lh3</bpmn:outgoing>
    </bpmn:serviceTask>
    <bpmn:exclusiveGateway id="Gateway_15mhbb9" default="Flow_12hi27d">
      <bpmn:incoming>Flow_0ya3xhb</bpmn:incoming>
      <bpmn:outgoing>Flow_12hi27d</bpmn:outgoing>
      <bpmn:outgoing>Flow_09s5t9z</bpmn:outgoing>
    </bpmn:exclusiveGateway>
    <bpmn:endEvent id="Event_1dvyawq">
      <bpmn:incoming>Flow_06pilas</bpmn:incoming>
    </bpmn:endEvent>
    <bpmn:serviceTask id="Activity_18aokfd" name="Завершение процесса по ошибке 2" camunda:type="external" camunda:topic="FinishedWithError">
      <bpmn:documentation>Завершение процесса из за ошибки оплаты</bpmn:documentation>
      <bpmn:extensionElements>
        <camunda:inputOutput>
          <camunda:inputParameter name="Error">pay_error</camunda:inputParameter>
        </camunda:inputOutput>
      </bpmn:extensionElements>
      <bpmn:incoming>Flow_09s5t9z</bpmn:incoming>
      <bpmn:outgoing>Flow_06pilas</bpmn:outgoing>
    </bpmn:serviceTask>
    <bpmn:sequenceFlow id="Flow_1elh8jk" sourceRef="Gateway_0dqgtav" targetRef="Activity_1lvyab6" />
    <bpmn:sequenceFlow id="Flow_0dqjff1" sourceRef="Gateway_0dqgtav" targetRef="Activity_0apgv3s">
      <bpmn:conditionExpression xsi:type="bpmn:tFormalExpression">${isDemo}</bpmn:conditionExpression>
    </bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="Flow_0t0rw1l" sourceRef="Activity_1lvyab6" targetRef="Activity_0xc6f45" />
    <bpmn:sequenceFlow id="Flow_0ya3xhb" sourceRef="Activity_0xc6f45" targetRef="Gateway_15mhbb9" />
    <bpmn:sequenceFlow id="Flow_12hi27d" sourceRef="Gateway_15mhbb9" targetRef="Activity_0apgv3s" />
    <bpmn:sequenceFlow id="Flow_09s5t9z" sourceRef="Gateway_15mhbb9" targetRef="Activity_18aokfd">
      <bpmn:conditionExpression xsi:type="bpmn:tFormalExpression">${isPayError}</bpmn:conditionExpression>
    </bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="Flow_06pilas" sourceRef="Activity_18aokfd" targetRef="Event_1dvyawq" />
    <bpmn:sequenceFlow id="Flow_1bm4kfb" sourceRef="Activity_1h47nnc" targetRef="Gateway_0dqgtav" />
    <bpmn:endEvent id="Event_0cw1rx3">
      <bpmn:incoming>Flow_0959lh3</bpmn:incoming>
    </bpmn:endEvent>
    <bpmn:sequenceFlow id="Flow_0959lh3" sourceRef="Activity_0apgv3s" targetRef="Event_0cw1rx3" />
    <bpmn:sequenceFlow id="Flow_0pf5r01" sourceRef="Activity_0sqydhn" targetRef="Gateway_0dqgtav" />
    <bpmn:serviceTask id="Activity_03sjseg" name="Пользователь останавливает выполнение задачи" camunda:type="external" camunda:topic="UserHasStopTask">
      <bpmn:incoming>Flow_1pxdfdk</bpmn:incoming>
      <bpmn:outgoing>Flow_1fee3l1</bpmn:outgoing>
    </bpmn:serviceTask>
    <bpmn:sequenceFlow id="Flow_1pxdfdk" sourceRef="Gateway_1xw251t" targetRef="Activity_03sjseg">
      <bpmn:conditionExpression xsi:type="bpmn:tFormalExpression">${answerState == "userHasStop"}</bpmn:conditionExpression>
    </bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="Flow_1fee3l1" sourceRef="Activity_03sjseg" targetRef="Activity_15sx01p" />
    <bpmn:userTask id="Activity_0r9f7nb" name="Ожидание отправки в кластер 2">
      <bpmn:incoming>Flow_11hlmef</bpmn:incoming>
      <bpmn:outgoing>Flow_0kksag7</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:exclusiveGateway id="Gateway_0h6uyeh" default="Flow_1231whk">
      <bpmn:incoming>Flow_0kksag7</bpmn:incoming>
      <bpmn:outgoing>Flow_1231whk</bpmn:outgoing>
      <bpmn:outgoing>Flow_0i2idsf</bpmn:outgoing>
    </bpmn:exclusiveGateway>
    <bpmn:serviceTask id="Activity_15f7bys" name="Завершение процесса по ошибке 3" camunda:type="external" camunda:topic="FinishedWithError">
      <bpmn:documentation>Завершение процесса из за ошибки оправки в кластер</bpmn:documentation>
      <bpmn:extensionElements>
        <camunda:inputOutput>
          <camunda:inputParameter name="Error">send_stop_error</camunda:inputParameter>
        </camunda:inputOutput>
      </bpmn:extensionElements>
      <bpmn:incoming>Flow_0i2idsf</bpmn:incoming>
      <bpmn:outgoing>Flow_1byhbqv</bpmn:outgoing>
    </bpmn:serviceTask>
    <bpmn:endEvent id="Event_0gb7lbe">
      <bpmn:incoming>Flow_1byhbqv</bpmn:incoming>
    </bpmn:endEvent>
    <bpmn:sequenceFlow id="Flow_0kksag7" sourceRef="Activity_0r9f7nb" targetRef="Gateway_0h6uyeh" />
    <bpmn:sequenceFlow id="Flow_1231whk" sourceRef="Gateway_0h6uyeh" targetRef="Activity_0p4vwfe" />
    <bpmn:sequenceFlow id="Flow_0i2idsf" sourceRef="Gateway_0h6uyeh" targetRef="Activity_15f7bys">
      <bpmn:conditionExpression xsi:type="bpmn:tFormalExpression">${isError}</bpmn:conditionExpression>
    </bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="Flow_1byhbqv" sourceRef="Activity_15f7bys" targetRef="Event_0gb7lbe" />
    <bpmn:serviceTask id="Activity_15sx01p" name="Отравка сообщения" camunda:type="external" camunda:topic="SendStopTask">
      <bpmn:incoming>Flow_1fee3l1</bpmn:incoming>
      <bpmn:outgoing>Flow_11hlmef</bpmn:outgoing>
    </bpmn:serviceTask>
    <bpmn:sequenceFlow id="Flow_11hlmef" sourceRef="Activity_15sx01p" targetRef="Activity_0r9f7nb" />
    <bpmn:sequenceFlow id="Flow_1hyt4ec" sourceRef="Activity_0p4vwfe" targetRef="Activity_00zpmb4" />
    <bpmn:serviceTask id="Activity_054cafe" name="Уведомление об остановке задачи" camunda:type="external" camunda:topic="ChangeTaskStateStoppedToNeedPay">
      <bpmn:incoming>Flow_0k4wjh2</bpmn:incoming>
      <bpmn:outgoing>Flow_0bws7s0</bpmn:outgoing>
    </bpmn:serviceTask>
    <bpmn:sequenceFlow id="Flow_0k4wjh2" sourceRef="Gateway_1xw251t" targetRef="Activity_054cafe">
      <bpmn:conditionExpression xsi:type="bpmn:tFormalExpression">${answerState == "stopped"}</bpmn:conditionExpression>
    </bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="Flow_0bws7s0" sourceRef="Activity_054cafe" targetRef="Gateway_0dqgtav" />
    <bpmn:serviceTask id="Activity_1sw7fkp" name="Задача поставлена в очередь" camunda:type="external" camunda:topic="ChangeTaskStateToQueued">
      <bpmn:incoming>Flow_1ymjm8y</bpmn:incoming>
      <bpmn:outgoing>Flow_096mo8m</bpmn:outgoing>
    </bpmn:serviceTask>
    <bpmn:serviceTask id="Activity_1116qx0" name="Задача в состоянии выполнения" camunda:type="external" camunda:topic="ChangeTaskStateToPerformed">
      <bpmn:incoming>Flow_1ejesd3</bpmn:incoming>
      <bpmn:outgoing>Flow_0dx6eax</bpmn:outgoing>
    </bpmn:serviceTask>
    <bpmn:sequenceFlow id="Flow_1ymjm8y" sourceRef="Gateway_1xw251t" targetRef="Activity_1sw7fkp">
      <bpmn:conditionExpression xsi:type="bpmn:tFormalExpression">${answerState == "inQueued"}</bpmn:conditionExpression>
    </bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="Flow_1ejesd3" sourceRef="Gateway_1xw251t" targetRef="Activity_1116qx0">
      <bpmn:conditionExpression xsi:type="bpmn:tFormalExpression">${answerState == "performed"}</bpmn:conditionExpression>
    </bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="Flow_096mo8m" sourceRef="Activity_1sw7fkp" targetRef="Activity_00zpmb4" />
    <bpmn:sequenceFlow id="Flow_0dx6eax" sourceRef="Activity_1116qx0" targetRef="Activity_00zpmb4" />
    <bpmn:sequenceFlow id="Flow_0v8vwkj" sourceRef="Gateway_1c7kods" targetRef="Activity_00uwg2b" />
    <bpmn:serviceTask id="Activity_00uwg2b" name="Задача в статусе инициализирована" camunda:type="external" camunda:topic="ChangeTaskStateToInitialized">
      <bpmn:incoming>Flow_0v8vwkj</bpmn:incoming>
      <bpmn:outgoing>Flow_0jz0iep</bpmn:outgoing>
    </bpmn:serviceTask>
    <bpmn:sequenceFlow id="Flow_0jz0iep" sourceRef="Activity_00uwg2b" targetRef="Activity_00zpmb4" />
    <bpmn:serviceTask id="Activity_0p4vwfe" name="Задача в статусе готова к останоке" camunda:type="external" camunda:topic="ChangeTaskStateToMustBeStopped">
      <bpmn:incoming>Flow_1231whk</bpmn:incoming>
      <bpmn:outgoing>Flow_1hyt4ec</bpmn:outgoing>
    </bpmn:serviceTask>
    <bpmn:userTask id="Activity_0ct74oz" name="Ожидание передачи параметров задачи">
      <bpmn:incoming>Flow_0hwsc4t</bpmn:incoming>
      <bpmn:outgoing>Flow_0s3fjbv</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:sequenceFlow id="Flow_0s3fjbv" sourceRef="Activity_0ct74oz" targetRef="Activity_0jkjpcm" />
    <bpmn:intermediateCatchEvent id="Event_0wo8z7w" name="Таймер ожидания продолжения">
      <bpmn:incoming>Flow_0oyz1sf</bpmn:incoming>
      <bpmn:outgoing>Flow_0yxffkl</bpmn:outgoing>
      <bpmn:timerEventDefinition id="TimerEventDefinition_0rfl65h">
        <bpmn:timeDuration xsi:type="bpmn:tFormalExpression">PT15M</bpmn:timeDuration>
      </bpmn:timerEventDefinition>
    </bpmn:intermediateCatchEvent>
    <bpmn:sequenceFlow id="Flow_0hwsc4t" sourceRef="Gateway_1hyospr" targetRef="Activity_0ct74oz" />
    <bpmn:parallelGateway id="Gateway_1hyospr">
      <bpmn:incoming>Flow_1mzl3ve</bpmn:incoming>
      <bpmn:outgoing>Flow_0hwsc4t</bpmn:outgoing>
      <bpmn:outgoing>Flow_0oyz1sf</bpmn:outgoing>
    </bpmn:parallelGateway>
    <bpmn:sequenceFlow id="Flow_0oyz1sf" sourceRef="Gateway_1hyospr" targetRef="Event_0wo8z7w" />
    <bpmn:sequenceFlow id="Flow_0yxffkl" sourceRef="Event_0wo8z7w" targetRef="Activity_0jkjpcm" />
    <bpmn:exclusiveGateway id="Gateway_1g9jza3" default="Flow_1mzl3ve">
      <bpmn:incoming>Flow_1xjnpn6</bpmn:incoming>
      <bpmn:outgoing>Flow_1mzl3ve</bpmn:outgoing>
      <bpmn:outgoing>Flow_1u3rl74</bpmn:outgoing>
    </bpmn:exclusiveGateway>
    <bpmn:sequenceFlow id="Flow_1mzl3ve" sourceRef="Gateway_1g9jza3" targetRef="Gateway_1hyospr" />
    <bpmn:sequenceFlow id="Flow_1u3rl74" sourceRef="Gateway_1g9jza3" targetRef="Activity_0jkjpcm">
      <bpmn:conditionExpression xsi:type="bpmn:tFormalExpression">${isUseOldCreateMethod}</bpmn:conditionExpression>
    </bpmn:sequenceFlow>
  </bpmn:process>
  <bpmndi:BPMNDiagram id="BPMNDiagram_1">
    <bpmndi:BPMNPlane id="BPMNPlane_1" bpmnElement="Process_0qsg7id">
      <bpmndi:BPMNShape id="BPMNShape_1ong3us" bpmnElement="Activity_00zpmb4" bioc:stroke="#5b176d" bioc:fill="#e1bee7" color:background-color="#e1bee7" color:border-color="#5b176d">
        <dc:Bounds x="2470" y="667" width="100" height="80" />
        <bpmndi:BPMNLabel />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="BPMNShape_1fmohe2" bpmnElement="Gateway_1xw251t" isMarkerVisible="true">
        <dc:Bounds x="2645" y="682" width="50" height="50" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="BPMNShape_0h8vydj" bpmnElement="Activity_1h47nnc" bioc:stroke="#205022" bioc:fill="#c8e6c9" color:background-color="#c8e6c9" color:border-color="#205022">
        <dc:Bounds x="2780" y="667" width="100" height="80" />
        <bpmndi:BPMNLabel />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="Activity_0u1p0fk_di" bpmnElement="Activity_0sqydhn" bioc:stroke="#205022" bioc:fill="#c8e6c9" color:background-color="#c8e6c9" color:border-color="#205022">
        <dc:Bounds x="2780" y="820" width="100" height="80" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="BPMNShape_03gnjyn" bpmnElement="Gateway_0dqgtav" isMarkerVisible="true">
        <dc:Bounds x="2965" y="682" width="50" height="50" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="BPMNShape_1wo9c05" bpmnElement="Activity_0xc6f45" bioc:stroke="#5b176d" bioc:fill="#e1bee7" color:background-color="#e1bee7" color:border-color="#5b176d">
        <dc:Bounds x="3250" y="667" width="100" height="80" />
        <bpmndi:BPMNLabel />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="BPMNShape_02k6d4p" bpmnElement="Activity_1lvyab6" bioc:stroke="#205022" bioc:fill="#c8e6c9" color:background-color="#c8e6c9" color:border-color="#205022">
        <dc:Bounds x="3070" y="667" width="100" height="80" />
        <bpmndi:BPMNLabel />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="BPMNShape_175u7z8" bpmnElement="Activity_0apgv3s" bioc:stroke="#205022" bioc:fill="#c8e6c9" color:background-color="#c8e6c9" color:border-color="#205022">
        <dc:Bounds x="3560" y="667" width="100" height="80" />
        <bpmndi:BPMNLabel />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="BPMNShape_15psrnt" bpmnElement="Gateway_15mhbb9" isMarkerVisible="true">
        <dc:Bounds x="3425" y="682" width="50" height="50" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="BPMNShape_0j3wb4z" bpmnElement="Event_1dvyawq">
        <dc:Bounds x="3432" y="932" width="36" height="36" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="BPMNShape_0evbv43" bpmnElement="Activity_18aokfd" bioc:stroke="#831311" bioc:fill="#ffcdd2" color:background-color="#ffcdd2" color:border-color="#831311">
        <dc:Bounds x="3400" y="790" width="100" height="80" />
        <bpmndi:BPMNLabel />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="Event_0cw1rx3_di" bpmnElement="Event_0cw1rx3">
        <dc:Bounds x="3742" y="689" width="36" height="36" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="BPMNShape_0kqd3gf" bpmnElement="Activity_03sjseg" bioc:stroke="#205022" bioc:fill="#c8e6c9" color:background-color="#c8e6c9" color:border-color="#205022">
        <dc:Bounds x="2780" y="1130" width="100" height="80" />
        <bpmndi:BPMNLabel />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="BPMNShape_1he2urh" bpmnElement="Activity_054cafe" bioc:stroke="#205022" bioc:fill="#c8e6c9" color:background-color="#c8e6c9" color:border-color="#205022">
        <dc:Bounds x="2780" y="970" width="100" height="80" />
        <bpmndi:BPMNLabel />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="BPMNShape_1b2b057" bpmnElement="Activity_1sw7fkp" bioc:stroke="#205022" bioc:fill="#c8e6c9" color:background-color="#c8e6c9" color:border-color="#205022">
        <dc:Bounds x="2790" y="320" width="100" height="80" />
        <bpmndi:BPMNLabel />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="BPMNShape_1pgidgr" bpmnElement="Activity_1116qx0" bioc:stroke="#205022" bioc:fill="#c8e6c9" color:background-color="#c8e6c9" color:border-color="#205022">
        <dc:Bounds x="2790" y="440" width="100" height="80" />
        <bpmndi:BPMNLabel />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="Gateway_0nycwe1_di" bpmnElement="Gateway_0nycwe1" isMarkerVisible="true">
        <dc:Bounds x="1025" y="682" width="50" height="50" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="Event_0v7ilwi_di" bpmnElement="Event_0v7ilwi">
        <dc:Bounds x="1032" y="932" width="36" height="36" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="Gateway_0x1tej4_di" bpmnElement="Gateway_0x1tej4" isMarkerVisible="true">
        <dc:Bounds x="1155" y="682" width="50" height="50" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="Activity_1l9t32f_di" bpmnElement="Activity_0olwk38" bioc:stroke="#5b176d" bioc:fill="#e1bee7" color:background-color="#e1bee7" color:border-color="#5b176d">
        <dc:Bounds x="1440" y="667" width="100" height="80" />
        <bpmndi:BPMNLabel />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="Activity_06xwlqo_di" bpmnElement="Activity_0jkjpcm" bioc:stroke="#205022" bioc:fill="#c8e6c9" color:background-color="#c8e6c9" color:border-color="#205022">
        <dc:Bounds x="860" y="667" width="100" height="80" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="Activity_1fbfsl0_di" bpmnElement="Activity_1w6f5j0" bioc:stroke="#831311" bioc:fill="#ffcdd2" color:background-color="#ffcdd2" color:border-color="#831311">
        <dc:Bounds x="1000" y="790" width="100" height="80" />
        <bpmndi:BPMNLabel />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="Activity_1gpmyev_di" bpmnElement="Activity_1m9c3b5" bioc:stroke="#205022" bioc:fill="#c8e6c9" color:background-color="#c8e6c9" color:border-color="#205022">
        <dc:Bounds x="1260" y="667" width="100" height="80" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="Activity_12u4f4l_di" bpmnElement="Activity_16bzrna" bioc:stroke="#205022" bioc:fill="#c8e6c9" color:background-color="#c8e6c9" color:border-color="#205022">
        <dc:Bounds x="1750" y="667" width="100" height="80" />
        <bpmndi:BPMNLabel />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="Gateway_0zr8n60_di" bpmnElement="Gateway_0zr8n60" isMarkerVisible="true">
        <dc:Bounds x="1615" y="682" width="50" height="50" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="Event_17uaw5g_di" bpmnElement="Event_17uaw5g">
        <dc:Bounds x="1622" y="932" width="36" height="36" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="Activity_1eailry_di" bpmnElement="Activity_0eudc9d" bioc:stroke="#831311" bioc:fill="#ffcdd2" color:background-color="#ffcdd2" color:border-color="#831311">
        <dc:Bounds x="1590" y="790" width="100" height="80" />
        <bpmndi:BPMNLabel />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="BPMNShape_062az00" bpmnElement="Activity_1h0w2mg" bioc:stroke="#5b176d" bioc:fill="#e1bee7" color:background-color="#e1bee7" color:border-color="#5b176d">
        <dc:Bounds x="1930" y="667" width="100" height="80" />
        <bpmndi:BPMNLabel />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="Gateway_1c7kods_di" bpmnElement="Gateway_1c7kods" isMarkerVisible="true">
        <dc:Bounds x="2105" y="682" width="50" height="50" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="BPMNShape_0cbww6o" bpmnElement="Activity_0paergu" bioc:stroke="#831311" bioc:fill="#ffcdd2" color:background-color="#ffcdd2" color:border-color="#831311">
        <dc:Bounds x="2080" y="790" width="100" height="80" />
        <bpmndi:BPMNLabel />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="Event_0h5f8k8_di" bpmnElement="Event_0h5f8k8">
        <dc:Bounds x="2112" y="932" width="36" height="36" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="BPMNShape_0zzbkf2" bpmnElement="Activity_00uwg2b" bioc:stroke="#205022" bioc:fill="#c8e6c9" color:background-color="#c8e6c9" color:border-color="#205022">
        <dc:Bounds x="2270" y="667" width="100" height="80" />
        <bpmndi:BPMNLabel />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="BPMNShape_0845mfz" bpmnElement="Activity_0r9f7nb" bioc:stroke="#5b176d" bioc:fill="#e1bee7" color:background-color="#e1bee7" color:border-color="#5b176d">
        <dc:Bounds x="3250" y="1130" width="100" height="80" />
        <bpmndi:BPMNLabel />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="BPMNShape_0o4w89f" bpmnElement="Gateway_0h6uyeh" isMarkerVisible="true">
        <dc:Bounds x="3425" y="1145" width="50" height="50" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="BPMNShape_0j6sw84" bpmnElement="Activity_15f7bys" bioc:stroke="#831311" bioc:fill="#ffcdd2" color:background-color="#ffcdd2" color:border-color="#831311">
        <dc:Bounds x="3400" y="1253" width="100" height="80" />
        <bpmndi:BPMNLabel />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="BPMNShape_1u3mkhh" bpmnElement="Event_0gb7lbe">
        <dc:Bounds x="3432" y="1395" width="36" height="36" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="BPMNShape_1k7j60c" bpmnElement="Activity_15sx01p" bioc:stroke="#205022" bioc:fill="#c8e6c9" color:background-color="#c8e6c9" color:border-color="#205022">
        <dc:Bounds x="3070" y="1130" width="100" height="80" />
        <bpmndi:BPMNLabel />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="BPMNShape_1vnvuuh" bpmnElement="Activity_0p4vwfe" bioc:stroke="#205022" bioc:fill="#c8e6c9" color:background-color="#c8e6c9" color:border-color="#205022">
        <dc:Bounds x="3570" y="1130" width="100" height="80" />
        <bpmndi:BPMNLabel />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="BPMNShape_1s4n9se" bpmnElement="Activity_0ct74oz" bioc:stroke="#5b176d" bioc:fill="#e1bee7" color:background-color="#e1bee7" color:border-color="#5b176d">
        <dc:Bounds x="690" y="667" width="100" height="80" />
        <bpmndi:BPMNLabel />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="Gateway_1vnqt0k_di" bpmnElement="Gateway_1hyospr">
        <dc:Bounds x="575" y="682" width="50" height="50" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="Event_01e6hlc_di" bpmnElement="Event_0wo8z7w">
        <dc:Bounds x="582" y="512" width="36" height="36" />
        <bpmndi:BPMNLabel>
          <dc:Bounds x="565" y="462" width="69" height="40" />
        </bpmndi:BPMNLabel>
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="Gateway_1g9jza3_di" bpmnElement="Gateway_1g9jza3" isMarkerVisible="true">
        <dc:Bounds x="445" y="682" width="50" height="50" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="_BPMNShape_StartEvent_2" bpmnElement="StartEvent_1">
        <dc:Bounds x="159" y="689" width="36" height="36" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="Activity_0102i0f_di" bpmnElement="Activity_1ns79tb" bioc:stroke="#205022" bioc:fill="#c8e6c9" color:background-color="#c8e6c9" color:border-color="#205022">
        <dc:Bounds x="270" y="667" width="100" height="80" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNEdge id="Flow_1hyt4ec_di" bpmnElement="Flow_1hyt4ec">
        <di:waypoint x="3670" y="1170" />
        <di:waypoint x="3910" y="1170" />
        <di:waypoint x="3910" y="80" />
        <di:waypoint x="2520" y="80" />
        <di:waypoint x="2520" y="667" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_096mo8m_di" bpmnElement="Flow_096mo8m">
        <di:waypoint x="2890" y="360" />
        <di:waypoint x="2920" y="360" />
        <di:waypoint x="2920" y="270" />
        <di:waypoint x="2520" y="270" />
        <di:waypoint x="2520" y="667" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_0dx6eax_di" bpmnElement="Flow_0dx6eax">
        <di:waypoint x="2890" y="480" />
        <di:waypoint x="2950" y="480" />
        <di:waypoint x="2950" y="220" />
        <di:waypoint x="2520" y="220" />
        <di:waypoint x="2520" y="667" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_0jz0iep_di" bpmnElement="Flow_0jz0iep">
        <di:waypoint x="2370" y="707" />
        <di:waypoint x="2470" y="707" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_0h4ksb5_di" bpmnElement="Flow_0h4ksb5">
        <di:waypoint x="2570" y="707" />
        <di:waypoint x="2645" y="707" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="BPMNEdge_04kj4y0" bpmnElement="Flow_1t6me9c">
        <di:waypoint x="2695" y="707" />
        <di:waypoint x="2780" y="707" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="BPMNEdge_0xymk33" bpmnElement="Flow_1ambyox">
        <di:waypoint x="2670" y="732" />
        <di:waypoint x="2670" y="860" />
        <di:waypoint x="2780" y="860" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_1pxdfdk_di" bpmnElement="Flow_1pxdfdk">
        <di:waypoint x="2670" y="732" />
        <di:waypoint x="2670" y="1170" />
        <di:waypoint x="2780" y="1170" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_0k4wjh2_di" bpmnElement="Flow_0k4wjh2">
        <di:waypoint x="2670" y="732" />
        <di:waypoint x="2670" y="1010" />
        <di:waypoint x="2780" y="1010" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_1ymjm8y_di" bpmnElement="Flow_1ymjm8y">
        <di:waypoint x="2670" y="682" />
        <di:waypoint x="2670" y="360" />
        <di:waypoint x="2790" y="360" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_1ejesd3_di" bpmnElement="Flow_1ejesd3">
        <di:waypoint x="2670" y="682" />
        <di:waypoint x="2670" y="480" />
        <di:waypoint x="2790" y="480" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_1bm4kfb_di" bpmnElement="Flow_1bm4kfb">
        <di:waypoint x="2880" y="707" />
        <di:waypoint x="2965" y="707" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_0pf5r01_di" bpmnElement="Flow_0pf5r01">
        <di:waypoint x="2880" y="860" />
        <di:waypoint x="2990" y="860" />
        <di:waypoint x="2990" y="732" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_0bws7s0_di" bpmnElement="Flow_0bws7s0">
        <di:waypoint x="2880" y="1010" />
        <di:waypoint x="2990" y="1010" />
        <di:waypoint x="2990" y="732" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="BPMNEdge_11gp8jf" bpmnElement="Flow_1elh8jk">
        <di:waypoint x="3015" y="707" />
        <di:waypoint x="3070" y="707" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="BPMNEdge_15wuj9i" bpmnElement="Flow_0dqjff1">
        <di:waypoint x="2990" y="682" />
        <di:waypoint x="2990" y="540" />
        <di:waypoint x="3610" y="540" />
        <di:waypoint x="3610" y="667" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="BPMNEdge_11jczk0" bpmnElement="Flow_0t0rw1l">
        <di:waypoint x="3170" y="707" />
        <di:waypoint x="3250" y="707" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="BPMNEdge_0zvmt74" bpmnElement="Flow_0ya3xhb">
        <di:waypoint x="3350" y="707" />
        <di:waypoint x="3425" y="707" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="BPMNEdge_171gz1s" bpmnElement="Flow_12hi27d">
        <di:waypoint x="3475" y="707" />
        <di:waypoint x="3560" y="707" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_0959lh3_di" bpmnElement="Flow_0959lh3">
        <di:waypoint x="3660" y="707" />
        <di:waypoint x="3742" y="707" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="BPMNEdge_0uvfxgf" bpmnElement="Flow_09s5t9z">
        <di:waypoint x="3450" y="732" />
        <di:waypoint x="3450" y="790" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="BPMNEdge_1si5mhu" bpmnElement="Flow_06pilas">
        <di:waypoint x="3450" y="870" />
        <di:waypoint x="3450" y="932" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_1fee3l1_di" bpmnElement="Flow_1fee3l1">
        <di:waypoint x="2880" y="1170" />
        <di:waypoint x="3070" y="1170" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_0r3970h_di" bpmnElement="Flow_0r3970h">
        <di:waypoint x="960" y="707" />
        <di:waypoint x="1025" y="707" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_1k6lsbw_di" bpmnElement="Flow_1k6lsbw">
        <di:waypoint x="1050" y="732" />
        <di:waypoint x="1050" y="790" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_11f0pfi_di" bpmnElement="Flow_11f0pfi">
        <di:waypoint x="1075" y="707" />
        <di:waypoint x="1155" y="707" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_1ixcdyp_di" bpmnElement="Flow_1ixcdyp">
        <di:waypoint x="1050" y="870" />
        <di:waypoint x="1050" y="932" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_0f65im9_di" bpmnElement="Flow_0f65im9">
        <di:waypoint x="1205" y="707" />
        <di:waypoint x="1260" y="707" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_097m7iv_di" bpmnElement="Flow_097m7iv">
        <di:waypoint x="1180" y="682" />
        <di:waypoint x="1180" y="540" />
        <di:waypoint x="1800" y="540" />
        <di:waypoint x="1800" y="667" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_0a0vipb_di" bpmnElement="Flow_0a0vipb">
        <di:waypoint x="1360" y="707" />
        <di:waypoint x="1440" y="707" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_1j01kwk_di" bpmnElement="Flow_1j01kwk">
        <di:waypoint x="1540" y="707" />
        <di:waypoint x="1615" y="707" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_1xjnpn6_di" bpmnElement="Flow_1xjnpn6">
        <di:waypoint x="370" y="707" />
        <di:waypoint x="445" y="707" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_0fiarqt_di" bpmnElement="Flow_0fiarqt">
        <di:waypoint x="1665" y="707" />
        <di:waypoint x="1750" y="707" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_0fygq6x_di" bpmnElement="Flow_0fygq6x">
        <di:waypoint x="1850" y="707" />
        <di:waypoint x="1930" y="707" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_0pubmzz_di" bpmnElement="Flow_0pubmzz">
        <di:waypoint x="1640" y="732" />
        <di:waypoint x="1640" y="790" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_0l5sp8h_di" bpmnElement="Flow_0l5sp8h">
        <di:waypoint x="1640" y="870" />
        <di:waypoint x="1640" y="932" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_1gtjk2f_di" bpmnElement="Flow_1gtjk2f">
        <di:waypoint x="2030" y="707" />
        <di:waypoint x="2105" y="707" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_1s5oqef_di" bpmnElement="Flow_1s5oqef">
        <di:waypoint x="2130" y="732" />
        <di:waypoint x="2130" y="790" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_0v8vwkj_di" bpmnElement="Flow_0v8vwkj">
        <di:waypoint x="2155" y="707" />
        <di:waypoint x="2270" y="707" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_15od6mt_di" bpmnElement="Flow_15od6mt">
        <di:waypoint x="2130" y="870" />
        <di:waypoint x="2130" y="932" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_11hlmef_di" bpmnElement="Flow_11hlmef">
        <di:waypoint x="3170" y="1170" />
        <di:waypoint x="3250" y="1170" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="BPMNEdge_1br892n" bpmnElement="Flow_0kksag7">
        <di:waypoint x="3350" y="1170" />
        <di:waypoint x="3425" y="1170" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="BPMNEdge_0w97suh" bpmnElement="Flow_1231whk">
        <di:waypoint x="3475" y="1170" />
        <di:waypoint x="3570" y="1170" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="BPMNEdge_0e8mn36" bpmnElement="Flow_0i2idsf">
        <di:waypoint x="3450" y="1195" />
        <di:waypoint x="3450" y="1253" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="BPMNEdge_10tz649" bpmnElement="Flow_1byhbqv">
        <di:waypoint x="3450" y="1333" />
        <di:waypoint x="3450" y="1395" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_0s3fjbv_di" bpmnElement="Flow_0s3fjbv">
        <di:waypoint x="790" y="707" />
        <di:waypoint x="860" y="707" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_0hwsc4t_di" bpmnElement="Flow_0hwsc4t">
        <di:waypoint x="625" y="707" />
        <di:waypoint x="690" y="707" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_0oyz1sf_di" bpmnElement="Flow_0oyz1sf">
        <di:waypoint x="600" y="682" />
        <di:waypoint x="600" y="548" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_0yxffkl_di" bpmnElement="Flow_0yxffkl">
        <di:waypoint x="618" y="530" />
        <di:waypoint x="910" y="530" />
        <di:waypoint x="910" y="667" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_1mzl3ve_di" bpmnElement="Flow_1mzl3ve">
        <di:waypoint x="495" y="707" />
        <di:waypoint x="575" y="707" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_06dy3gj_di" bpmnElement="Flow_06dy3gj">
        <di:waypoint x="195" y="707" />
        <di:waypoint x="270" y="707" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_1u3rl74_di" bpmnElement="Flow_1u3rl74">
        <di:waypoint x="470" y="732" />
        <di:waypoint x="470" y="890" />
        <di:waypoint x="910" y="890" />
        <di:waypoint x="910" y="747" />
      </bpmndi:BPMNEdge>
    </bpmndi:BPMNPlane>
  </bpmndi:BPMNDiagram>
</bpmn:definitions>`

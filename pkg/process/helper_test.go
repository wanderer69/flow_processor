package process

var process1 string = `<?xml version="1.0" encoding="UTF-8"?>
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
      <bpmn:incoming>Flow_0u3u62p</bpmn:incoming>
      <bpmn:outgoing>Flow_0rycqr4</bpmn:outgoing>
    </bpmn:serviceTask>
    <bpmn:sequenceFlow id="Flow_0u3u62p" sourceRef="Activity_0ad6ndl" targetRef="Activity_0l1xmnl" />
    <bpmn:intermediateCatchEvent id="Event_0usaqoj" name="отправить сообщение">
      <bpmn:incoming>Flow_0rycqr4</bpmn:incoming>
      <bpmn:messageEventDefinition id="MessageEventDefinition_0f31sk3" messageRef="Message_1kij1k3" />
    </bpmn:intermediateCatchEvent>
    <bpmn:sequenceFlow id="Flow_0rycqr4" sourceRef="Activity_0l1xmnl" targetRef="Event_0usaqoj" />
    <bpmn:intermediateThrowEvent id="Event_1mfzqut" name="принять сообщение">
      <bpmn:outgoing>Flow_11p59rc</bpmn:outgoing>
      <bpmn:messageEventDefinition id="MessageEventDefinition_0fs07p6" messageRef="Message_1kij1k3" camunda:type="external" camunda:topic="" />
    </bpmn:intermediateThrowEvent>
    <bpmn:serviceTask id="Activity_05lfthg" name="процесс3" camunda:type="external" camunda:topic="">
      <bpmn:incoming>Flow_11p59rc</bpmn:incoming>
      <bpmn:outgoing>Flow_06a5pw4</bpmn:outgoing>
    </bpmn:serviceTask>
    <bpmn:sequenceFlow id="Flow_11p59rc" sourceRef="Event_1mfzqut" targetRef="Activity_05lfthg" />
    <bpmn:endEvent id="Event_0xdunvq">
      <bpmn:incoming>Flow_06a5pw4</bpmn:incoming>
    </bpmn:endEvent>
    <bpmn:sequenceFlow id="Flow_06a5pw4" sourceRef="Activity_05lfthg" targetRef="Event_0xdunvq" />
  </bpmn:process>
  <bpmn:message id="Message_1kij1k3" name="Message_1kij1k3" />
  <bpmndi:BPMNDiagram id="BPMNDiagram_1">
    <bpmndi:BPMNPlane id="BPMNPlane_1" bpmnElement="Process_0ul3sfu">
      <bpmndi:BPMNShape id="_BPMNShape_StartEvent_2" bpmnElement="StartEvent_1">
        <dc:Bounds x="179" y="99" width="36" height="36" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="Activity_111f528_di" bpmnElement="Activity_0ad6ndl">
        <dc:Bounds x="330" y="77" width="100" height="80" />
        <bpmndi:BPMNLabel />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="BPMNShape_1yw978j" bpmnElement="Activity_0l1xmnl">
        <dc:Bounds x="520" y="77" width="100" height="80" />
        <bpmndi:BPMNLabel />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="Event_0ryiixu_di" bpmnElement="Event_0usaqoj">
        <dc:Bounds x="702" y="99" width="36" height="36" />
        <bpmndi:BPMNLabel>
          <dc:Bounds x="691" y="142" width="58" height="27" />
        </bpmndi:BPMNLabel>
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="Event_053sx49_di" bpmnElement="Event_1mfzqut">
        <dc:Bounds x="902" y="99" width="36" height="36" />
        <bpmndi:BPMNLabel>
          <dc:Bounds x="891" y="142" width="58" height="27" />
        </bpmndi:BPMNLabel>
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="BPMNShape_0yc0frt" bpmnElement="Activity_05lfthg">
        <dc:Bounds x="1020" y="77" width="100" height="80" />
        <bpmndi:BPMNLabel />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="Event_0xdunvq_di" bpmnElement="Event_0xdunvq">
        <dc:Bounds x="1212" y="99" width="36" height="36" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNEdge id="Flow_1kcirj0_di" bpmnElement="Flow_1kcirj0">
        <di:waypoint x="215" y="117" />
        <di:waypoint x="330" y="117" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_0u3u62p_di" bpmnElement="Flow_0u3u62p">
        <di:waypoint x="430" y="117" />
        <di:waypoint x="520" y="117" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_0rycqr4_di" bpmnElement="Flow_0rycqr4">
        <di:waypoint x="620" y="117" />
        <di:waypoint x="702" y="117" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_11p59rc_di" bpmnElement="Flow_11p59rc">
        <di:waypoint x="938" y="117" />
        <di:waypoint x="1020" y="117" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_06a5pw4_di" bpmnElement="Flow_06a5pw4">
        <di:waypoint x="1120" y="117" />
        <di:waypoint x="1212" y="117" />
      </bpmndi:BPMNEdge>
    </bpmndi:BPMNPlane>
  </bpmndi:BPMNDiagram>
</bpmn:definitions>`

var process2 string = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:bpmndi="http://www.omg.org/spec/BPMN/20100524/DI" xmlns:dc="http://www.omg.org/spec/DD/20100524/DC" xmlns:camunda="http://camunda.org/schema/1.0/bpmn" xmlns:di="http://www.omg.org/spec/DD/20100524/DI" xmlns:modeler="http://camunda.org/schema/modeler/1.0" id="Definitions_0kcmz1z" targetNamespace="http://bpmn.io/schema/bpmn" exporter="Camunda Modeler" exporterVersion="5.26.0" modeler:executionPlatform="Camunda Platform" modeler:executionPlatformVersion="7.21.0">
  <bpmn:process id="Process_0m0z3lq" isExecutable="true">
    <bpmn:startEvent id="StartEvent_1">
      <bpmn:outgoing>Flow_0ncjqzi</bpmn:outgoing>
    </bpmn:startEvent>
    <bpmn:serviceTask id="Activity_0ad6ndl" name="процесс1" camunda:type="external" camunda:topic="ExecProcess1">
      <bpmn:incoming>Flow_0ncjqzi</bpmn:incoming>
      <bpmn:outgoing>Flow_0i1g0wr</bpmn:outgoing>
    </bpmn:serviceTask>
    <bpmn:userTask id="Activity_1keabte" name="ожидание ввода" camunda:formKey="">
      <bpmn:documentation>ожидание ввода</bpmn:documentation>
      <bpmn:extensionElements>
        <camunda:inputOutput>
          <camunda:inputParameter name="Input_30h5s0b">qwerqwewqe</camunda:inputParameter>
          <camunda:outputParameter name="Output_028bsmh" />
        </camunda:inputOutput>
      </bpmn:extensionElements>
      <bpmn:incoming>Flow_0i1g0wr</bpmn:incoming>
      <bpmn:outgoing>Flow_1q15m3y</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:serviceTask id="Activity_05lfthg" name="процесс3" camunda:type="external" camunda:topic="ExecProcess2">
      <bpmn:incoming>Flow_1q15m3y</bpmn:incoming>
      <bpmn:outgoing>Flow_18su0qa</bpmn:outgoing>
    </bpmn:serviceTask>
    <bpmn:endEvent id="Event_1cpipct">
      <bpmn:incoming>Flow_18su0qa</bpmn:incoming>
    </bpmn:endEvent>
    <bpmn:sequenceFlow id="Flow_0ncjqzi" sourceRef="StartEvent_1" targetRef="Activity_0ad6ndl" />
    <bpmn:sequenceFlow id="Flow_0i1g0wr" sourceRef="Activity_0ad6ndl" targetRef="Activity_1keabte" />
    <bpmn:sequenceFlow id="Flow_1q15m3y" sourceRef="Activity_1keabte" targetRef="Activity_05lfthg" />
    <bpmn:sequenceFlow id="Flow_18su0qa" sourceRef="Activity_05lfthg" targetRef="Event_1cpipct" />
  </bpmn:process>
  <bpmndi:BPMNDiagram id="BPMNDiagram_1">
    <bpmndi:BPMNPlane id="BPMNPlane_1" bpmnElement="Process_0m0z3lq">
      <bpmndi:BPMNShape id="_BPMNShape_StartEvent_2" bpmnElement="StartEvent_1">
        <dc:Bounds x="179" y="99" width="36" height="36" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="Activity_111f528_di" bpmnElement="Activity_0ad6ndl">
        <dc:Bounds x="280" y="77" width="100" height="80" />
        <bpmndi:BPMNLabel />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="Activity_14ddhl2_di" bpmnElement="Activity_1keabte">
        <dc:Bounds x="450" y="77" width="100" height="80" />
        <bpmndi:BPMNLabel />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="BPMNShape_0yc0frt" bpmnElement="Activity_05lfthg">
        <dc:Bounds x="620" y="77" width="100" height="80" />
        <bpmndi:BPMNLabel />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="Event_1cpipct_di" bpmnElement="Event_1cpipct">
        <dc:Bounds x="792" y="99" width="36" height="36" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNEdge id="Flow_0ncjqzi_di" bpmnElement="Flow_0ncjqzi">
        <di:waypoint x="215" y="117" />
        <di:waypoint x="280" y="117" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_0i1g0wr_di" bpmnElement="Flow_0i1g0wr">
        <di:waypoint x="380" y="117" />
        <di:waypoint x="450" y="117" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_1q15m3y_di" bpmnElement="Flow_1q15m3y">
        <di:waypoint x="550" y="117" />
        <di:waypoint x="620" y="117" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_18su0qa_di" bpmnElement="Flow_18su0qa">
        <di:waypoint x="720" y="117" />
        <di:waypoint x="792" y="117" />
      </bpmndi:BPMNEdge>
    </bpmndi:BPMNPlane>
  </bpmndi:BPMNDiagram>
</bpmn:definitions>`

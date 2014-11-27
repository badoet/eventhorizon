// Copyright (c) 2014 - Max Persson <max@looplab.se>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package eventhorizon

import (
	"fmt"
	"reflect"

	. "gopkg.in/check.v1"

	t "github.com/looplab/eventhorizon/testing"
)

var _ = Suite(&ReflectDispatcherSuite{})

type ReflectDispatcherSuite struct {
	store *MockEventStore
	bus   *MockEventBus
	disp  *ReflectDispatcher
}

func (s *ReflectDispatcherSuite) SetUpTest(c *C) {
	s.store = &MockEventStore{
		events: make([]Event, 0),
	}
	s.bus = &MockEventBus{
		events: make([]Event, 0),
	}
	s.disp = NewReflectDispatcher(s.store, s.bus)
}

func (s *ReflectDispatcherSuite) Test_NewReflectAggregate(c *C) {
	store := &MockEventStore{
		events: make([]Event, 0),
	}
	bus := &MockEventBus{
		events: make([]Event, 0),
	}
	disp := NewReflectDispatcher(store, bus)
	c.Assert(disp, Not(Equals), nil)
	c.Assert(disp.eventStore, Equals, store)
	c.Assert(disp.eventBus, Equals, bus)
	c.Assert(disp.commandHandlers, Not(Equals), nil)
}

var dispatchedCommand Command

type TestSource struct {
	Aggregate
}

func (t *TestSource) HandleTestCommand(command TestCommand) ([]Event, error) {
	dispatchedCommand = command
	if command.Content == "error" {
		return nil, fmt.Errorf("command error")
	}
	return []Event{TestEvent{command.TestID, command.Content}}, nil
}

func (t *TestSource) HandleCommandOther2(command TestCommandOther2, invalidParam string) ([]Event, error) {
	return nil, nil
}

func (s *ReflectDispatcherSuite) Test_Dispatch_Simple(c *C) {
	source := &TestSource{}
	sourceType := reflect.ValueOf(source).Elem().Type()
	method, _ := reflect.TypeOf(source).MethodByName("HandleTestCommand")
	s.disp.commandHandlers[reflect.TypeOf(TestCommand{})] = handler{
		sourceType: sourceType,
		method:     method,
	}
	command1 := TestCommand{NewUUID(), "command1"}
	err := s.disp.Dispatch(command1)
	c.Assert(dispatchedCommand, Equals, command1)
	c.Assert(err, Equals, nil)
}

func (s *ReflectDispatcherSuite) Test_Dispatch_ErrorInHandler(c *C) {
	source := &TestSource{}
	sourceType := reflect.ValueOf(source).Elem().Type()
	method, _ := reflect.TypeOf(source).MethodByName("HandleTestCommand")
	s.disp.commandHandlers[reflect.TypeOf(TestCommand{})] = handler{
		sourceType: sourceType,
		method:     method,
	}
	commandError := TestCommand{NewUUID(), "error"}
	err := s.disp.Dispatch(commandError)
	c.Assert(err, ErrorMatches, "command error")
	c.Assert(dispatchedCommand, Equals, commandError)
}

func (s *ReflectDispatcherSuite) Test_Dispatch_NoHandlers(c *C) {
	command1 := TestCommand{NewUUID(), "command1"}
	err := s.disp.Dispatch(command1)
	c.Assert(err, ErrorMatches, "no handlers for command")
}

func (s *ReflectDispatcherSuite) Test_AddHandler_Simple(c *C) {
	source := &TestSource{}
	s.disp.AddHandler(TestCommand{}, source)
	c.Assert(len(s.disp.commandHandlers), Equals, 1)
	commandType := reflect.TypeOf(TestCommand{})
	c.Assert(s.disp.commandHandlers, t.HasKey, commandType)
	sourceType := reflect.ValueOf(source).Elem().Type()
	method, _ := reflect.TypeOf(source).MethodByName("HandleTestCommand")
	sourceHandler := handler{
		sourceType: sourceType,
		method:     method,
	}
	c.Assert(s.disp.commandHandlers[commandType], Equals, sourceHandler)
}

func (s *ReflectDispatcherSuite) Test_AddHandler_Duplicate(c *C) {
	source := &TestSource{}
	s.disp.AddHandler(TestCommand{}, source)
	source2 := &TestSource{}
	s.disp.AddHandler(TestCommand{}, source2)
	c.Assert(len(s.disp.commandHandlers), Equals, 1)
	commandType := reflect.TypeOf(TestCommand{})
	c.Assert(s.disp.commandHandlers, t.HasKey, commandType)
	sourceType := reflect.ValueOf(source).Elem().Type()
	method, _ := reflect.TypeOf(source).MethodByName("HandleTestCommand")
	sourceHandler := handler{
		sourceType: sourceType,
		method:     method,
	}
	c.Assert(s.disp.commandHandlers[commandType], Equals, sourceHandler)
}

func (s *ReflectDispatcherSuite) Test_AddHandler_MissingMethod(c *C) {
	source := &TestSource{}
	s.disp.AddHandler(TestCommandOther{}, source)
	c.Assert(len(s.disp.commandHandlers), Equals, 0)
}

func (s *ReflectDispatcherSuite) Test_AddHandler_IncorrectMethod(c *C) {
	source := &TestSource{}
	s.disp.AddHandler(TestCommandOther2{}, source)
	c.Assert(len(s.disp.commandHandlers), Equals, 0)
}

type TestGlobalSubscriber struct {
	handledEvent Event
}

func (t *TestGlobalSubscriber) HandleEvent(event Event) {
	t.handledEvent = event
}

func (s *ReflectDispatcherSuite) Test_HandleCommand_Simple(c *C) {
	source := &TestSource{}
	s.disp.AddHandler(TestCommand{}, source)
	command1 := TestCommand{NewUUID(), "command1"}
	err := s.disp.Dispatch(command1)
	c.Assert(err, Equals, nil)
	c.Assert(dispatchedCommand, Equals, command1)
	c.Assert(len(s.store.events), Equals, 1)
	c.Assert(s.store.events[0], DeepEquals, TestEvent{command1.TestID, command1.Content})
	c.Assert(len(s.bus.events), Equals, 1)
	c.Assert(s.bus.events[0], DeepEquals, TestEvent{command1.TestID, command1.Content})
}

func (s *ReflectDispatcherSuite) Test_HandleCommand_ErrorInHandler(c *C) {
	source := &TestSource{}
	s.disp.AddHandler(TestCommand{}, source)
	commandError := TestCommand{NewUUID(), "error"}
	err := s.disp.Dispatch(commandError)
	c.Assert(dispatchedCommand, Equals, commandError)
	c.Assert(err, ErrorMatches, "command error")
	c.Assert(len(s.store.events), Equals, 0)
	c.Assert(len(s.bus.events), Equals, 0)
}

var callCount int

type BenchmarkAggregate struct {
	Aggregate
}

func (t *BenchmarkAggregate) HandleTestCommand(command TestCommand) ([]Event, error) {
	callCount++
	return nil, nil
}

func (s *ReflectDispatcherSuite) Benchmark_ReflectDispatcher(c *C) {
	store := &MockEventStore{
		events: make([]Event, 0),
	}
	bus := &MockEventBus{
		events: make([]Event, 0),
	}
	disp := NewReflectDispatcher(store, bus)
	agg := &BenchmarkAggregate{}
	disp.AddHandler(TestCommand{}, agg)

	callCount = 0
	command1 := TestCommand{NewUUID(), "command1"}
	for i := 0; i < c.N; i++ {
		disp.Dispatch(command1)
	}
	c.Assert(callCount, Equals, c.N)
}

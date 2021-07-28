/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package gstop

import "sync"

type Task func()

// Stopper the stop status holder.
type Stopper struct {
	// channel to control stop status, stop it by calling Stop().
	C chan struct{}

	once  sync.Once
	tasks []Task
}

// Defer add task called in desc order when stopper stopped.
func (s *Stopper) Defer(task Task) {
	s.tasks = append(s.tasks, task)
}

// Stop stop the chan and call all tasks.
func (s *Stopper) Stop() {
	s.once.Do(func() {
		close(s.C)

		// call in desc order, like defer.
		for i := len(s.tasks) - 1; i >= 0; i-- {
			s.tasks[i]()
		}

		// help gc
		s.tasks = nil
	})
}

// Loop run task util stopped.
func (s *Stopper) Loop(task Task) {
	go func() {
		for {
			select {
			case <-s.C:
				return
			default:
				task()
			}
		}
	}()
}

// New create a new Stopper.
func New() *Stopper {
	return &Stopper{
		once: sync.Once{},
		C:    make(chan struct{}),
	}
}

// NewChild create a new Stopper as child of the exist chan, when which is closed the child will be stopped too.
func NewChild(stop chan struct{}) *Stopper {
	child := &Stopper{
		once: sync.Once{},
		C:    make(chan struct{}),
	}

	go func() {
		select {
		case <-stop:
			child.Stop()
		case <-child.C:
		}
	}()

	return child
}

// NewChild create a new Stopper as child of the exist one, when which is stopped the child will be stopped too.
func (s *Stopper) NewChild() *Stopper {
	return NewChild(s.C)
}

// NewParent create a new Stopper as parent of the exist one, which will be stopped when the new parent stopped.
func (s *Stopper) NewParent() *Stopper {
	parent := &Stopper{
		once: sync.Once{},
		C:    make(chan struct{}),
	}

	go func() {
		select {
		case <-parent.C:
			s.Stop()
		case <-s.C:
		}
	}()

	return parent
}

// Package linkedlist provides an expandeble and generic
// interface to linkedlists. It uses two-way non-cirkular
// linkedlist and is thread-safe.
package linkedlist

import (
	"bytes"
	"encoding/gob"
	"errors"
	"reflect"
	"sync"
)

type ILinkedList interface {
	Size() int
	Len() int
	Empty() bool
	AddFirst(V Elem)
	AddLast(V Elem)
	Contains(V Elem) bool
	Index(V Elem) int
	Get(i int) Elem
	Set(i int, V Elem) error
	First() Elem
	Last() Elem
	RemoveFirst() (Elem, error)
	RemoveLast() (Elem, error)
	Remove(V Elem) error
	FastRemove(V Elem) error
	RemoveAll(V Elem) error
	Iter() chan Elem
	ToSlice() []Elem
	Conc(other *LinkedList)
	Append(other *LinkedList)
	Reduce(f func(interface{}, interface{}) interface{}) interface{}
	Filter(f func(interface{}) bool)
	ParFilter(f func(interface{}) bool)
	Map(f func(interface{}) interface{})
	ParMap(f func(interface{}) interface{})
	Reverse()
	Serialize() []byte
}

var (
	ErrListEmpty   = errors.New("list is empty")
	ErrNotFound    = errors.New("item not found in list")
	ErrOutOfBounds = errors.New("index out of bounds")
)

// A linkedlist has a size, a pointer to the first node, and
// a pointer to the last node in the list.
//
// e.g.
//     first -> 1
//              2
//              3
//      last -> 4
//
type LinkedList struct {
	size   int
	first  *node
	last   *node
	synced bool
	lock   sync.RWMutex
}

// The linkedlist's chain is made up of nodes with an element,
// a pointer to the previous node, and a pointer to the next node.
//
// e.g. 1<->2<->3<->4
type node struct {
	Value Elem
	next  *node
	prev  *node
}

// Elem is used as a generic for any type of value.
type Elem interface{}

// New creates a synchronized linked list
//
// e.g. mylist := linkedlist.New()
//
func New() *LinkedList {
	return NewSynced()
}

// New creates a synchronized linked list
//
// e.g. mylist := linkedlist.NewSynced()
//
func NewSynced() *LinkedList {
	return &LinkedList{synced: true}
}

// New creates an unsynchronized linked list
//
// e.g. mylist := linkedlist.NewUnsynced()
//
func NewUnsynced() *LinkedList {
	return &LinkedList{synced: false}
}

// Size returns the size of the list.
//
// e.g. (1,2,3).Size() => 3
//
func (L *LinkedList) Size() int {
	if L.synced {
		L.lock.RLock()
		defer L.lock.RUnlock()
	}

	return L.size
}

// Len is an alias for Size().
func (L *LinkedList) Len() int {
	return L.Size()
}

// Empty returns true if the list is empty.
//
// e.g. ().Empty() => true
//
func (L *LinkedList) Empty() bool {
	return L.Size() == 0
}

// AddFirst adds a node at the start of the list with the given element.
//
// e.g. (1,2,3).AddFirst(0) => (0,1,2,3)
//
func (L *LinkedList) AddFirst(V Elem) {
	if L.synced {
		L.lock.Lock()
	}
	n := node{V, nil, nil}

	if L.size == 0 {
		L.last = &n
	} else {
		n.next = L.first
		L.first.prev = &n
	}

	L.first = &n
	L.size += 1
	if L.synced {
		L.lock.Unlock()
	}
}

// AddLast adds a node at the end of the list with the given element.
//
// e.g. (1,2,3).AddLast(4) => (1,2,3,4)
//
func (L *LinkedList) AddLast(V Elem) {
	if L.synced {
		L.lock.Lock()
	}
	n := node{V, nil, L.last}

	if L.size == 0 {
		L.first = &n
	} else {
		L.last.next = &n
	}

	L.last = &n
	L.size += 1
	if L.synced {
		L.lock.Unlock()
	}
}

// Contains returns true if the list has at least one
// node with the given element.
//
// e.g. (1,2,3).Contains(2) => true
//
func (L *LinkedList) Contains(V Elem) bool {
	if L.synced {
		L.lock.RLock()
		defer L.lock.RUnlock()
	}

	res := L.fastGet(V)
	if res == nil {
		return false
	}
	return true
}

// Index returns the index of the first occurrence of the
// a node with the given element.
//
// e.g. (1,2,1).Index(1) => 0
//
func (L *LinkedList) Index(V Elem) int {
	if L.synced {
		L.lock.RLock()
		defer L.lock.RUnlock()
	}

	i := 0
	for n := range L.iter() {
		if n == V {
			return i
		}
		i++
	}
	return -1
}

// Get returns the node's element at the given index.
//
// e.g. (1,2,3).Get(2) => 3
//
func (L *LinkedList) Get(i int) Elem {
	if L.synced {
		L.lock.RLock()
		defer L.lock.RUnlock()
	}

	node, err := L.getNode(i)
	if err != nil {
		return nil
	}
	return node.Value
}

// Set updates a node's element given its index.
//
// e.g. (1,2,3).Set(1, 8) => (1,8,3)
//
func (L *LinkedList) Set(i int, V Elem) error {
	if L.synced {
		L.lock.Lock()
		defer L.lock.Unlock()
	}

	node, err := L.getNode(i)
	if err == nil {
		node.Value = V
	}
	return err
}

// First returns the first nodes' element.
//
// e.g. (1,2,3).First() => 1
//
func (L *LinkedList) First() Elem {
	if L.synced {
		L.lock.RLock()
		defer L.lock.RUnlock()
	}

	if L.size == 0 {
		return nil
	}
	return L.first.Value
}

// Last returns the last node's element.
//
// e.g. (1,2,3).Last() => 3
//
func (L *LinkedList) Last() Elem {
	if L.synced {
		L.lock.RLock()
		defer L.lock.RUnlock()
	}

	if L.size == 0 {
		return nil
	}
	return L.last.Value
}

// RemoveFirst deletes the first node in the list.
//
// e.g. (1,2,3).RemoveFirst() => (2,3)
//
func (L *LinkedList) RemoveFirst() (Elem, error) {
	if L.synced {
		L.lock.Lock()
		defer L.lock.Unlock()
	}

	if L.size == 0 {
		return nil, ErrListEmpty
	}
	result := L.first
	L.removeNode(L.first)
	return result, nil
}

// RemoveLast deletes the last node in the list.
//
// e.g. (1,2,3).RemoveLast() => (1,2)
//
func (L *LinkedList) RemoveLast() (Elem, error) {
	if L.synced {
		L.lock.Lock()
		defer L.lock.Unlock()
	}

	if L.size == 0 {
		return nil, ErrListEmpty
	}
	result := L.last
	L.removeNode(L.last)
	return result, nil
}

// Remove deletes the first occurrence of a node in
// the LinkedList by its value.
//
// e.g. (1,2,1).Remove(1) => (2,1)
//
func (L *LinkedList) Remove(V Elem) error {
	if L.synced {
		L.lock.Lock()
		defer L.lock.Unlock()
	}

	res := L.slowGet(V)
	if res == nil {
		return ErrNotFound
	}
	L.removeNode(res)
	return nil
}

// FastRemove deletes an occurrence of a node in the LinkedList by its value.
//
// e.g. (1,2,1).FastRemove(1) => (2,1)
// e.g. (1,2,1).FastRemove(1) => (1,2)
//
func (L *LinkedList) FastRemove(V Elem) error {
	if L.synced {
		L.lock.Lock()
		defer L.lock.Unlock()
	}

	res := L.fastGet(V)
	if res == nil {
		return ErrNotFound
	}
	L.removeNode(res)
	return nil
}

// FastRemove deletes all occurrences of nodes with the input value.
//
// e.g. (1,2,1).RemoveAll(1) => (2)
//
func (L *LinkedList) RemoveAll(V Elem) error {
	if L.synced {
		L.lock.Lock()
		defer L.lock.Unlock()
	}

	s := L.size
	for n := L.first; n != nil; n = n.next {
		if n.Value == V {
			L.removeNode(n)
		}
	}
	if L.size == s {
		return ErrNotFound
	}
	return nil
}

// Iter is an iterator to be used for iterate the linkedlist (front first) easily.
//
// e.g. for x := range list.Iter() { }
//
func (L *LinkedList) Iter() chan Elem {
	if L.synced {
		L.lock.RLock()
		defer L.lock.RUnlock()
	}

	return L.iter()
}

// ToSlice returns a slice representation of the LinkedList.
func (L *LinkedList) ToSlice() []Elem {
	if L.synced {
		L.lock.RLock()
		defer L.lock.RUnlock()
	}

	res := make([]Elem, L.size)
	i := 0
	for x := range L.iter() {
		res[i] = x
		i++
	}
	return res
}

// FromSlice creates a LinkedList from a go slice.
func FromSlice(slc interface{}) *LinkedList {
	v := reflect.ValueOf(slc)
	newl := New()

	for i := 0; i < v.Len(); i++ {
		newl.AddLast(v.Index(i).Interface())
	}

	return newl
}

// Conc concatenates two linkedlists.
//
// e.g. (1,2,3).Conc((4,5,6)) => (1,2,3,4,5,6)
//
func (L *LinkedList) Conc(other *LinkedList) {
	if L.synced {
		L.lock.Lock()
		defer L.lock.Unlock()
	}

	if L.size == 0 {
		L.first = other.first
		L.last = other.last
		L.size = other.size
		return
	} else if other.size == 0 {
		return
	}

	L.last.next = other.first
	other.first.prev = L.last
	L.last = other.last
	L.size = L.size + other.size
}

// Append is an alias for Conc()
func (L *LinkedList) Append(other *LinkedList) {
	L.Conc(other)
}

// Reduce compiles the list with an input function.
//
// e.g. (1,2,3).Reduce(+) => 6
//
func (L *LinkedList) Reduce(f func(interface{}, interface{}) interface{}) interface{} {
	if L.synced {
		L.lock.RLock()
		defer L.lock.RUnlock()
	}

	if L.size == 0 {
		return nil
	}
	if L.size == 1 {
		return L.first.Value
	}
	if L.size == 2 {
		return f(L.first.Value, L.first.next.Value)
	}

	e := f(L.first.Value, L.first.next.Value)
	for n := L.first.next.next; n != nil; n = n.next {
		e = f(e, n.Value)
	}

	return e
}

// Filter filters the list with regards to an input function.
//
// e.g. (1,2,3).Filter(>= 2) => (2,3)
//
func (L *LinkedList) Filter(f func(interface{}) bool) {
	if L.synced {
		L.lock.Lock()
		defer L.lock.Unlock()
	}

	for n := L.first; n != nil; n = n.next {
		if !f(n.Value) {
			L.removeNode(n)
		}
	}
}

// ParFilter filters the list in parallel with regards to an input function.
//
// e.g. (1,2,3).Filter(>= 2) => (2,3)
//
// TODO: This method is not thread safe!
//
func (L *LinkedList) ParFilter(f func(interface{}) bool) {
	if L.synced {
		L.lock.Lock()
		defer L.lock.Unlock()
	}

	c := make(chan bool, L.size)
	for n := L.first; n != nil; n = n.next {
		go func(n *node) {
			if !f(n.Value) {
				L.removeNode(n)
			}
			c <- true
		}(n)
	}

	/* drain the channel */
	for i := 0; i < L.size; i++ {
		<-c
	}
}

// Map performs a function on every element in the list.
//
// e.g. (1,2,3).Map(f) => (f(1),f(2),f(3))
//
func (L *LinkedList) Map(f func(interface{}) interface{}) {
	if L.synced {
		L.lock.Lock()
		defer L.lock.Unlock()
	}

	for n := L.first; n != nil; n = n.next {
		n.Value = f(n.Value)
	}
}

// ParMap performs a function on every element in the list, in parallel.
//
// e.g. (1,2,3).ParMap(f) => (f(1),f(2),f(3))
//
func (L *LinkedList) ParMap(f func(interface{}) interface{}) {
	if L.synced {
		L.lock.Lock()
		defer L.lock.Unlock()
	}

	c := make(chan bool, L.size)
	for n := L.first; n != nil; n = n.next {
		go func(n *node) {
			n.Value = f(n.Value)
			c <- true
		}(n)
	}

	/* drain the channel */
	for i := 0; i < L.size; i++ {
		<-c
	}
}

// Reverse reverses the list.
//
// e.g. (1,2,3).Reverse() => (3,2,1)
//
func (L *LinkedList) Reverse() {
	if L.synced {
		L.lock.Lock()
		defer L.lock.Unlock()
	}

	if L.size == 0 || L.size == 1 {
		return
	}

	start := L.first
	for start != nil {
		temp := start.next
		start.next = start.prev
		start.prev = temp

		if start.prev == nil {
			L.first = start
		}

		if start.next == nil {
			L.last = start
		}

		start = start.prev
	}

}

// Serialize
func (L *LinkedList) Serialize() []byte {
	if L.synced {
		L.lock.RLock()
		defer L.lock.RUnlock()
	}

	m := new(bytes.Buffer)
	gob.NewEncoder(m).Encode(L.ToSlice())

	return m.Bytes()
}

// Deserialize
func Deserialize(bt []byte) *LinkedList {
	p := bytes.NewBuffer(bt)
	dec := gob.NewDecoder(p)

	var slc []Elem
	dec.Decode(&slc)

	return FromSlice(slc)
}

// iter is used internally and is not locked.
func (L *LinkedList) iter() chan Elem {
	ch := make(chan Elem, L.size)
	for n := L.first; n != nil; n = n.next {
		ch <- n.Value
	}
	close(ch)
	return ch
}

// get searches the list from start to end for the node with
// the given element. This returns the first instance.
func (L *LinkedList) slowGet(E Elem) *node {
	for n := L.first; n != nil; n = n.next {
		if n.Value == E {
			return n
		}
	}
	return nil
}

// fastGet searches the list from both ends concurrently.
// This returns any instance. O(n/2)
func (L *LinkedList) fastGet(E Elem) *node {

	/* Delegate to slower get if the list is small enough */
	if L.size < 100 {
		return L.slowGet(E)
	}

	found := make(chan *node, 1)
	done := make(chan bool, 2)
	half := L.size / 2

	go func() {
		cur := L.first
		for n := 0; n < half; n++ {
			if E == cur.Value {
				found <- cur
				break
			}
			cur = cur.next
		}
		done <- true
	}()

	go func() {
		cur := L.last
		for n := L.size; n >= half; n-- {
			if E == cur.Value {
				found <- cur
				break
			}
			cur = cur.prev
		}
		done <- true
	}()

	/* If we receive done twice, return nil. */
	select {
	case <-done:
		select {
		case <-done:
			return nil
		case fnd := <-found:
			return fnd
		}
	case fnd := <-found:
		return fnd
	}

	return nil
}

// getNode retrieves a node given an index.
func (L *LinkedList) getNode(i int) (*node, error) {
	if L.size == 0 || i > L.size-1 {
		return nil, ErrOutOfBounds
	}

	var n *node

	if i <= L.size/2 {
		n = L.first
		for p := 0; p != i; p++ {
			n = n.next
		}
	} else {
		n = L.last
		for p := L.size - 1; p != i; p-- {
			n = n.prev
		}
	}

	return n, nil
}

// removeNode deletes the node from the given list.
// The function is considered to be used internally.
func (L *LinkedList) removeNode(N *node) {
	if L.size == 1 { // Only node
		L.first = nil
		L.last = nil
		L.size--
		return
	}

	if N.prev == nil { // First node
		N.next.prev = nil
		L.first = N.next
		L.size--
		return
	}

	if N.next == nil { //Last node
		N.prev.next = nil
		L.last = N.prev
		L.size--
		return
	}

	// Node in middle of chain
	N.next.prev = N.prev
	N.prev.next = N.next
	L.size--
	return
}

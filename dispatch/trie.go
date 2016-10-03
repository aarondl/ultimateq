package dispatch

import (
	"strings"
	"sync"
)

// errTrieNotUnique is a sentinel id value that occurs when an id was
// not consumed and therefore nothing was inserted
const errTrieNotUnique = 0

var handlerList = sync.Pool{
	New: func() interface{} {
		return make([]interface{}, 0)
	},
}

func getHandlerList() []interface{} {
	list := handlerList.Get().([]interface{})
	return list[:0]
}

func putHandlerList(list []interface{}) {
	handlerList.Put(list)
}

// trie is a prefix tree, not goroutine safe
type trie struct {
	counter  uint64
	isUnique bool
	root     *trieNode
}

type trieNode struct {
	subtrees map[string]*trieNode
	handlers map[uint64]interface{}
}

func newTrie(isUnique bool) *trie {
	return &trie{
		root:     newTrieNode(),
		isUnique: isUnique,
	}
}

func newTrieNode() *trieNode {
	return &trieNode{
		subtrees: make(map[string]*trieNode),
	}
}

func (t *trie) register(network, channel, event string, handler interface{}) uint64 {
	toInsert := []string{
		strings.ToLower(network),
		strings.ToLower(channel),
		strings.ToLower(event),
	}
	return t.insert(t.root, toInsert, handler)
}

func (t *trie) insert(node *trieNode, toInsert []string, handler interface{}) uint64 {
	insert := toInsert[0]

	nextNode, ok := node.subtrees[insert]
	if !ok {
		nextNode = newTrieNode()
		node.subtrees[insert] = nextNode
	}

	if len(toInsert) == 1 {
		if t.isUnique && ok {
			return errTrieNotUnique
		}

		if nextNode.handlers == nil {
			nextNode.handlers = make(map[uint64]interface{})
		}
		t.counter++
		nextNode.handlers[t.counter] = handler
		return t.counter
	}

	toInsert = toInsert[1:]
	return t.insert(nextNode, toInsert, handler)
}

func (t *trie) handlers(network, channel, event string) []interface{} {
	toFind := []string{
		strings.ToLower(network),
		strings.ToLower(channel),
		strings.ToLower(event),
	}
	list := getHandlerList()

	t.find(t.root, toFind, &list)

	retList := make([]interface{}, len(list))
	copy(retList, list)
	putHandlerList(list)

	return retList
}

func (t *trie) find(node *trieNode, toFind []string, list *[]interface{}) {
	if len(toFind) == 0 {
		for _, h := range node.handlers {
			*list = append(*list, h)
		}
		return
	}

	find := toFind[0]

	if nextNode, ok := node.subtrees[""]; ok {
		t.find(nextNode, toFind[1:], list)
	}

	// This can happen if "channel" is nil, and in which case we don't want
	// to look ourselves up twice.
	if len(find) == 0 {
		return
	}
	if nextNode, ok := node.subtrees[find]; ok {
		t.find(nextNode, toFind[1:], list)
	}
}

func (t *trie) unregister(id uint64) bool {
	found, _ := t.unregisterHelper(t.root, id)
	return found
}

func (t *trie) unregisterHelper(node *trieNode, toFind uint64) (found, empty bool) {
	for id := range node.handlers {
		if id == toFind {
			delete(node.handlers, toFind)
			return true, len(node.handlers) == 0
		}
	}

	for k, n := range node.subtrees {
		f, e := t.unregisterHelper(n, toFind)
		if f {
			if e {
				delete(node.subtrees, k)
			}
			return true, len(node.subtrees) == 0
		}
	}

	return false, false
}

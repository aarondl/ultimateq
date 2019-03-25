package dispatch

import (
	"bytes"
	"fmt"
	"sync"
	"testing"

	"github.com/aarondl/ultimateq/irc"
)

type testDispatchHandler struct {
	n, c, e string
}

func (t *testDispatchHandler) Handle(_ irc.Writer, e *irc.Event) {
	_ = e
}

func TestTrieRegister(t *testing.T) {
	t.Parallel()

	tr := newTrie(false)
	id := tr.register("net", "chan", "privmsg", testDispatchHandler{})
	if id != 1 {
		t.Error("id was wrong", id)
	}

	id = tr.register("net", "chan", "privmsg", testDispatchHandler{})
	if id != 2 {
		t.Error("id was wrong", id)
	}
}

func TestTrieRegisterUnique(t *testing.T) {
	t.Parallel()

	tr := newTrie(true)
	id := tr.register("net", "chan", "privmsg", testDispatchHandler{})
	if id != 1 {
		t.Error("id was wrong", id)
	}

	id = tr.register("net", "chan", "privmsg", testDispatchHandler{})
	if id != errTrieNotUnique {
		t.Error("id was wrong", id)
	}
}

func TestTrieUnregister(t *testing.T) {
	t.Parallel()

	tr := newTrie(false)
	id1 := tr.register("n1", "c1", "e1", 1)
	id2 := tr.register("n2", "c2", "e2", 2)
	id3 := tr.register("n2", "c2", "e2", 3)

	check1 := func() bool {
		_, ok := tr.root.subtrees["n1"].subtrees["c1"].subtrees["e1"].handlers[id1]
		return ok
	}
	check2 := func() bool {
		_, ok := tr.root.subtrees["n2"].subtrees["c2"].subtrees["e2"].handlers[id2]
		return ok
	}
	check3 := func() bool {
		_, ok := tr.root.subtrees["n2"].subtrees["c2"].subtrees["e2"].handlers[id3]
		return ok
	}

	if !check1() {
		t.Error("want to find handler 1")
	}
	if !check2() {
		t.Error("want to find handler 2")
	}
	if !check3() {
		t.Error("want to find handler 3")
	}

	if !tr.unregister(id1) {
		t.Error("should delete handler 1")
	}
	if tr.unregister(id1) {
		t.Error("should only delete handler 1 once")
	}

	if _, ok := tr.root.subtrees["n1"]; ok {
		t.Error("should not find a subtree for n1")
	}

	if !check2() {
		t.Error("want to find handler 2")
	}
	if !check3() {
		t.Error("want to find handler 3")
	}

	if !tr.unregister(id2) {
		t.Error("should delete handler 2")
	}
	if tr.unregister(id2) {
		t.Error("should only delete handler 2 once")
	}

	if 1 != len(tr.root.subtrees["n2"].subtrees["c2"].subtrees["e2"].handlers) {
		t.Error("didn't remove handler 2 properly")
	}

	if !tr.unregister(id3) {
		t.Error("should delete handler 3")
	}
	if tr.unregister(id3) {
		t.Error("should only delete handler 3 once")
	}

	if _, ok := tr.root.subtrees["n2"]; ok {
		t.Error("should not find a subtree for n2")
	}
}

func TestTrieHandler(t *testing.T) {
	t.Parallel()

	nets := []string{"n1", "n2"}
	chans := []string{"c1", "c2"}
	events := []string{"e1", "e2"}

	tr := newTrie(false)

	tr.register("", "", "", &testDispatchHandler{"", "", ""})
	for _, n := range nets {
		tr.register(n, "", "", &testDispatchHandler{n, "", ""})
		for _, c := range chans {
			tr.register(n, c, "", &testDispatchHandler{n, c, ""})
			tr.register("", c, "", &testDispatchHandler{"", c, ""})
			for _, e := range events {
				tr.register(n, c, e, &testDispatchHandler{n, c, e})
				tr.register(n, "", e, &testDispatchHandler{n, "", e})
				tr.register("", c, e, &testDispatchHandler{"", c, e})
				tr.register("", "", e, &testDispatchHandler{"", "", e})
			}
		}
	}

	tests := []struct {
		Net, Chan, Event string
		Handlers         int
	}{
		{"n1", "c1", "e1", 14},
		{"n2", "c2", "e2", 14},
		{"n", "c", "e", 1},
		{"n", "c1", "e", 3},
		{"n", "c1", "e1", 9},
	}

	for i, test := range tests {
		got := tr.handlers(test.Net, test.Chan, test.Event)
		if test.Handlers != len(got) {
			t.Errorf("%d) want: %d, got %d", i, test.Handlers, len(got))
		}

		//t.Log(i, "event:", test.Net, test.Chan, test.Event)
		for _, handler := range got {
			h := handler.(*testDispatchHandler)
			//t.Log(i, "dispatch:", wc(h.n), wc(h.c), wc(h.e))
			if (h.n != "" && h.n != test.Net) ||
				(h.c != "" && h.c != test.Chan) ||
				(h.e != "" && h.e != test.Event) {
				t.Error("event:", test.Net, test.Chan, test.Event)
				t.Error("dispatch:", wc(h.n), wc(h.c), wc(h.e))
			}
		}
	}
}

func wc(a string) string {
	if len(a) == 0 {
		return "*"
	}
	return a
}

func showGraph(t *trie) string {
	b := &bytes.Buffer{}
	fmt.Fprintln(b, "digraph name {")
	showGraphHelper(b, t.root, "root")
	fmt.Fprintln(b, "}")

	return b.String()
}

func showGraphHelper(b *bytes.Buffer, t *trieNode, name string) {
	if len(t.subtrees) > 0 {
		for str, s := range t.subtrees {
			if str == "" {
				str = name + "*"
			}
			fmt.Fprintf(b, "    \"%s\" -> \"%s\" [label=\" %s\"]\n", name, str, str)

			showGraphHelper(b, s, str)
		}
	}
}

func BenchmarkTrie(b *testing.B) {
	nets := []string{"n1", "n2"}
	chans := []string{"c1", "c2"}
	events := []string{"e1", "e2"}

	tr := newTrie(false)

	tr.register("", "", "", &testDispatchHandler{"", "", ""})
	for _, n := range nets {
		tr.register(n, "", "", &testDispatchHandler{n, "", ""})
		for _, c := range chans {
			tr.register(n, c, "", &testDispatchHandler{n, c, ""})
			tr.register("", c, "", &testDispatchHandler{"", c, ""})
			for _, e := range events {
				tr.register(n, c, e, &testDispatchHandler{n, c, e})
				tr.register(n, "", e, &testDispatchHandler{n, "", e})
				tr.register("", c, e, &testDispatchHandler{"", c, e})
				tr.register("", "", e, &testDispatchHandler{"", "", e})
			}
		}
	}

	ev := irc.NewEvent("", nil, "PRIVMSG", "server", "none")

	i, j, k := 0, 0, 0
	var wg sync.WaitGroup

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		handlers := tr.handlers(nets[i], chans[j], events[k])
		ev.NetworkID = nets[i]
		for _, h := range handlers {
			d := h.(*testDispatchHandler)
			wg.Add(1)
			go func() {
				d.Handle(nil, ev)
				wg.Done()
			}()
		}

		wg.Wait()

		k++
		if k == 2 {
			k = 0
			j++
		}
		if j == 2 {
			j = 0
			i++
		}
		if i == 2 {
			i, j, k = 0, 0, 0
		}
	}
}

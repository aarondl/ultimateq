package registrar

import (
	"errors"
	"testing"

	"github.com/aarondl/ultimateq/dispatch/cmd"
)

type mockReg struct {
	regs   int
	unregs int
	cmds   int
	uncmds int

	id  uint64
	err error
	ret bool
}

func (m *mockReg) Register(_, _, _ string, _ interface{}) uint64 {
	m.regs++
	m.id++
	return m.id
}

func (m *mockReg) RegisterCmd(_, _ string, _ *cmd.Command) (uint64, error) {
	m.cmds++
	return 0, m.err
}

func (m *mockReg) Unregister(_ uint64) bool {
	m.unregs++
	return m.ret
}

func (m *mockReg) UnregisterCmd(_ uint64) bool {
	m.uncmds++
	return m.ret
}

func (m *mockReg) verifyMock(t *testing.T, regs, unregs, cmds, uncmds int) {
	t.Helper()
	if regs != m.regs {
		t.Errorf("regs wrong, want: %d, got: %d", regs, m.regs)
	}
	if unregs != m.unregs {
		t.Errorf("unregs wrong, want: %d, got: %d", unregs, m.unregs)
	}
	if cmds != m.cmds {
		t.Errorf("cmds wrong, want: %d, got: %d", cmds, m.cmds)
	}
	if uncmds != m.uncmds {
		t.Errorf("uncmds wrong, want: %d, got: %d", uncmds, m.uncmds)
	}
}

func TestHolder_New(t *testing.T) {
	t.Parallel()

	m := &mockReg{}
	h := newHolder(m)

	if h.registrar != m {
		t.Error("registrar is wrong")
	}
	if h.events == nil {
		t.Error("events is nil")
	}
	if h.commands == nil {
		t.Error("commands is nil")
	}

	m.verifyMock(t, 0, 0, 0, 0)
}

func TestHolder_Register(t *testing.T) {
	t.Parallel()

	m := &mockReg{}
	h := newHolder(m)

	id := h.Register("n", "c", "e", nil)
	if _, ok := h.events[id]; !ok {
		t.Error("did not record the registration")
	}

	m.verifyMock(t, 1, 0, 0, 0)
}

func TestHolder_Unregister(t *testing.T) {
	t.Parallel()

	m := &mockReg{ret: true}
	h := newHolder(m)

	id := h.Register("n", "c", "e", nil)
	if !h.Unregister(id) {
		t.Error("should be true")
	}
	if _, ok := h.events[id]; ok {
		t.Error("did not delete the registration")
	}

	m.verifyMock(t, 1, 1, 0, 0)
}

func TestHolder_UnregisterFail(t *testing.T) {
	t.Parallel()

	m := &mockReg{ret: false}
	h := newHolder(m)

	id := h.Register("n", "c", "e", nil)
	if h.Unregister(id) {
		t.Error("should be false")
	}

	m.verifyMock(t, 1, 1, 0, 0)
}

func TestHolder_RegisterCmd(t *testing.T) {
	t.Parallel()

	m := &mockReg{}
	h := newHolder(m)

	id, err := h.RegisterCmd("n", "c", &cmd.Command{Name: "cmd", Extension: "e"})
	if err != nil {
		t.Error(err)
	}
	if _, ok := h.commands[id]; !ok {
		t.Error("did not record the registration")
	}

	m.verifyMock(t, 0, 0, 1, 0)
}

func TestHolder_RegisterCmdFail(t *testing.T) {
	t.Parallel()

	e := errors.New("failure")
	m := &mockReg{err: e}
	h := newHolder(m)

	id, err := h.RegisterCmd("n", "c", &cmd.Command{Name: "cmd", Extension: "e"})
	if err != e {
		t.Error("wrong error:", err, "want:", e)
	}
	if _, ok := h.commands[id]; ok {
		t.Error("should not record the registration")
	}

	m.verifyMock(t, 0, 0, 1, 0)
}

func TestHolder_UnregisterCmd(t *testing.T) {
	t.Parallel()

	m := &mockReg{}
	h := newHolder(m)

	id, err := h.RegisterCmd("n", "c", &cmd.Command{Name: "cmd", Extension: "e"})
	if err != nil {
		t.Error(err)
	}

	m.ret = true
	if ok := h.UnregisterCmd(id); !ok {
		t.Error("command not found")
	}

	if _, ok := h.commands[id]; ok {
		t.Error("did not delete the registration")
	}

	m.verifyMock(t, 0, 0, 1, 1)
}

func TestHolder_UnregisterCmdFail(t *testing.T) {
	t.Parallel()

	m := &mockReg{ret: false}
	h := newHolder(m)

	id, err := h.RegisterCmd("n", "c", &cmd.Command{Name: "cmd", Extension: "e"})
	if err != nil {
		t.Error(err)
	}

	m.ret = false
	if ok := h.UnregisterCmd(id); ok {
		t.Error("command was found, should not have been found")
	}
	if _, ok := h.commands[id]; ok {
		t.Error("did not delete the registration")
	}

	m.verifyMock(t, 0, 0, 1, 1)
}

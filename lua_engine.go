package lua_engine

/*
#cgo CFLAGS: -Ilua
#cgo LDFLAGS: -llua

#include "lua.h"
#include "lauxlib.h"
#include "lualib.h"

void halt(lua_State *L, lua_Debug *ar);
void luaHaltHook(lua_State *L);

void halt(lua_State *L, lua_Debug *ar) {
    lua_sethook(L, halt, LUA_MASKLINE, 0);
    luaL_error(L,"");
}

void luaHaltHook(lua_State *L) {
    lua_sethook(L,halt,LUA_MASKCOUNT, 1);
}
*/
import "C"

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/aarzilli/golua/lua"
	"github.com/stevedonovan/luar"
)

type LuaEngine struct {
	state     *lua.State
	source    string
	timestamp time.Time
	lmap      luar.Map
	doneCh    chan bool
	mu        sync.Mutex
}

func NewLuaEngine() *LuaEngine {
	state := luar.Init()

	l := &LuaEngine{
		state:  state,
		lmap:   make(map[string]interface{}),
		doneCh: make(chan bool),
	}

	// Default mappings
	l.lmap["Sleep"] = l.Sleep

	return l
}

func (l *LuaEngine) LoadFirmware(source string) error {
	l.source = source
	luar.Register(l.state, "", l.lmap)

	err := l.state.DoString(l.source)
	if err != nil {
		return err
	}

	return nil
}

func (l *LuaEngine) Process() error {
	// initialize the source
	err := l.initialize()
	if err != nil {
		return err
	}

	go l.process()

	return nil
}

func (l *LuaEngine) RegisterMap(m luar.Map) {
	for k, v := range m {
		l.lmap[k] = v
	}
}

func (l *LuaEngine) Sleep(n int64) {
	time.Sleep(time.Duration(n) * time.Millisecond)
}

func (l *LuaEngine) Halt() {
	l.mu.Lock()
	defer l.mu.Unlock()

	C.luaHaltHook((*C.struct_lua_State)(l.state.GetState()))
	l.doneCh <- true
}

func (l *LuaEngine) Validate(fn string) error {
	if l.validateFn(fn) == false {
		if l.validateCoreFn(fn) == true {
			return nil
		}
		return errors.New(fmt.Sprintf("missing %s method in lua firmware", fn))
	}

	return nil
}

func (l *LuaEngine) Close() {
	l.state.Close()
}

func (l *LuaEngine) validateCoreFn(fn string) bool {
	l.state.GetGlobal(fn)
	if l.state.IsGoFunction(-1) == false {
		return false
	}

	return true
}

func (l *LuaEngine) validateFn(fn string) bool {
	l.state.PushString(fn)
	l.state.RawGet(lua.LUA_GLOBALSINDEX)

	if l.state.IsFunction(-1) == false {
		l.state.Pop(1)
		return false
	}
	l.state.Pop(1)
	return true
}

func (l *LuaEngine) initialize() error {
	l.state.GetGlobal("Initialize")
	defer l.state.Pop(0)

	if err := l.Validate("Initialize"); err == nil {
		if err = l.state.Call(0, 0); err != nil {
			return err
		}
	}
	return nil
}

func (l *LuaEngine) process() error {
	l.state.GetGlobal("Process")
	defer l.state.Pop(0)

	if err := l.Validate("Process"); err == nil {
		go func() {
			err := l.state.Call(0, 0)
			if err != nil {
				log.Printf("Error: %s", err)
			}
			l.doneCh <- true
		}()
	} else {
		log.Println(err)
	}

	for {
		select {
		case <-l.doneCh:
			return nil
		}
	}
}

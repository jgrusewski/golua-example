package lua_engine

import (
	"log"
	"testing"
	"time"

	"github.com/stevedonovan/luar"
)

func TestLoadFirmware(t *testing.T) {
	source := `
		function Initialize()
		end

		function Process()
		end
	`
	e := NewLuaEngine()
	err := e.LoadFirmware(source)
	if err != nil {
		t.Error(err)
	}
}

func TestRegisterMap(t *testing.T) {
	source := `
		function Process()
			print "Hello from lua"
			SimpleTest()
			Sleep(1000)
			print "Done closing..."
		end
	`

	e := NewLuaEngine()

	e.RegisterMap(luar.Map{
		"SimpleTest": func() {
			var i int
			for i = 0; i < 10; i++ {
				log.Printf("Hello from Go (%d)", i)
				time.Sleep(500)
			}
		},
	})

	err := e.LoadFirmware(source)
	if err != nil {
		t.Error(err)
	}

	err = e.Validate("SimpleTest")
	if err != nil {
		t.Error(err)
	}

	e.Process()
}

func TestHaltHook(t *testing.T) {
	source := `
		function Process()
			while true do
				print "Halt Hook"
				Sleep(500)
			end
		end
	`
	e := NewLuaEngine()
	if err := e.LoadFirmware(source); err != nil {
		t.Error(err)
	}
	e.Process()

	time.Sleep(time.Duration(5 * time.Second))
	e.Halt()
}

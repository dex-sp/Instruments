package instruments

import (
	"fmt"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/jpoirier/visa"
)

func TestAgilent34980a(t *testing.T) {

	// loads values from .env into the system
	if err := godotenv.Load(); err != nil {
		t.Errorf("no .env file found")
	}

	// Get the SWITCH_IP_ADDR environment variable
	addr, exists := os.LookupEnv("SWITCH_IP_ADDR")
	if exists == false {
		t.Errorf("SWITCH_IP_ADDR not exists")
	}

	var fullAddr = fmt.Sprintf("TCPIP0::%s::INSTR", addr)
	var manufacturer = "Agilent Technologies"
	var model = "34980A"

	rm, visaStatus := visa.OpenDefaultRM()
	if visaStatus != visa.SUCCESS {
		t.Errorf("resource manager error")
	}
	defer rm.Close()

	mtrxHandler := VisaObjectWrapper{ResourceName: fullAddr, ResourceManager: &rm}
	err := mtrxHandler.Init()
	if err != nil {
		t.Errorf(err.Error())
	}

	instrInfo := mtrxHandler.GetInfo()
	if instrInfo["Manufacturer"] != manufacturer &&
		instrInfo["Model"] != model {
		t.Errorf("instrument \"%s\" is not %s %s", fullAddr, manufacturer, model)
	}

	mtrx := Agilent34980A{}
	for pins := pinsInModule; pins <= 256; pins += pinsInModule {
		err = mtrx.Init(&mtrxHandler, pins)
		if err != nil {
			t.Errorf(err.Error())
		}
	}
}

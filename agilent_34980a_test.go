package instruments

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/joho/godotenv"
)

func TestAgilent34980a(t *testing.T) {

	// loads values from .env into the system
	if err := godotenv.Load(); err != nil {
		t.Errorf("no .env file found")
	}

	// Get the AG34980A_IP_ADDR environment variable
	addr, exists := os.LookupEnv("AG34980A_IP_ADDR")
	if exists == false {
		t.Errorf("AG34980A_IP_ADDR not exists")
	}

	var fullAddr = fmt.Sprintf("TCPIP0::%s::INSTR", addr)
	var manufacturer = "Agilent Technologies"
	var model = "34980A"

	rm, err := GetResourceManager()
	if err != nil {
		t.Errorf(err.Error())
	}
	defer rm.Close()

	mtrxHandler := VisaObjectWrapper{ResourceName: fullAddr, ResourceManager: &rm}
	err = mtrxHandler.Init()
	if err != nil {
		t.Errorf(err.Error())
	}

	instrInfo := mtrxHandler.GetInfo()
	if instrInfo["Manufacturer"] != manufacturer ||
		instrInfo["Model"] != model {
		t.Errorf("instrument \"%s\" is not %s %s", fullAddr, manufacturer, model)
	}

	var moduleCnt int
	for i := 1; i <= 8; i++ {
		result, err := mtrxHandler.Query(fmt.Sprintf("SYST:CTYP? %d", i))
		if err != nil {
			t.Errorf(err.Error())
		}

		if len(result) != 0 {
			queryResultSplit := strings.Split(result, ",")
			if queryResultSplit[1] == moduleDual4x16 {
				moduleCnt++
			}
		}
	}

	mtrx := Agilent34980A{}
	err = mtrx.Init(&mtrxHandler, moduleCnt*pinsInModule)
	if err != nil {
		t.Errorf(err.Error())
	}

	keys := make([]int, len(mtrx.pinsMap))
	i := 0
	for k := range mtrx.pinsMap {
		keys[i] = k
		i++
	}
	sort.Ints(keys)

	var rows [][]int
	var end int
	keysLen := len(keys)
	chunkSize := keysLen / moduleRowNum

	for i := 0; i <= keysLen; i += chunkSize {
		end = i + chunkSize
		if end > keysLen {
			end = keysLen
		}
		rows = append(rows, keys[i:end])
	}

	for i := 0; i < moduleRowNum; i++ {
		pins := rows[i]
		err = mtrx.SetCommutation(pins, true)
		if err != nil {
			t.Errorf(err.Error())
		}
	}
	mtrx.OpenAllRelays()
}

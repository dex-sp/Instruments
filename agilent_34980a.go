// Управление Agilent 34980A Multifunction Switch/Measure Mainframe
// https://www.keysight.com/ru/ru/assets/9018-02146/user-manuals/9018-02146.pdf
//
// Модули Agilent 34932A Dual 4x16 Armature Matrix установленные в 34980A
// https://www.keysight.com/ru/ru/assets/9018-02148/user-manuals/9018-02148.pdf

package instruments

import (
	"bytes"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

const (
	moduleDual4x16 = "34932A"
	moduleRowNum   = 4
	moduleColNum   = 16
	relayRatio     = 1000
	pinsInModule   = 2 * moduleColNum
)

type Agilent34980A struct {
	instr     *VisaObjectWrapper
	pinsMap   map[int]int
	relaysMap map[int]int
}

// Инициализация коммутатора
func (sw *Agilent34980A) Init(instr *VisaObjectWrapper, pinsNum int) error {

	sw.instr = instr
	sw.instr.SetErrorQuery("SYST:ERR?")
	err := sw.instr.Write("*RST")
	if err != nil {
		return err
	}
	sw.pinsMap = make(map[int]int, pinsNum*moduleRowNum)
	sw.relaysMap = make(map[int]int, pinsNum*moduleRowNum)
	err = sw.fillPinArray(pinsNum)
	if err != nil {
		return err
	}
	return nil
}

// Проверка слотов коммутатора на наличие модулей внутри.
func (sw *Agilent34980A) CheckSlots() [8]string {

	var moduleList [8]string
	for i := 1; i <= len(moduleList); i++ {
		result, _ := sw.instr.Query(fmt.Sprintf("SYSTem:CTYPe? %d", i))
		if len(result) == 0 {
			moduleList[i-1] = "empty"
		} else {
			queryResultSplit := strings.Split(result, ",")
			moduleList[i-1] = queryResultSplit[1]
		}
	}
	return moduleList
}

// Создание перекодировочной таблицы для измерительной оснастки.
func (sw *Agilent34980A) fillPinArray(totalPinsNum int) error {

	requiredNumOfModules := int(math.Ceil(float64(totalPinsNum) / pinsInModule))
	excessPins := totalPinsNum % pinsInModule
	finishPins := 0

	if excessPins != 0 {
		finishPins = totalPinsNum
		totalPinsNum = totalPinsNum - excessPins + pinsInModule
	}

	// Get number of installed Agilent 34932A (Dual 4x16 Armature Matrix)
	installedModules := sw.CheckSlots()
	var moduleDual4x16Number []int
	moduleDual4x16Counter := 0

	for i := 0; i < len(installedModules); i++ {
		if moduleDual4x16Counter >= requiredNumOfModules {
			break
		}

		if installedModules[i] == moduleDual4x16 {
			moduleDual4x16Number = append(moduleDual4x16Number, i+1)
			moduleDual4x16Counter++
		}
	}

	// Проверка, хватает ли модулей 34932A для создания таблицы с количеством выводов "totalPinsNum"
	// если не хватает - вывести предупреждение
	var maxPossiblePinNum int

	if (moduleDual4x16Counter < requiredNumOfModules) && (moduleDual4x16Counter > 0) {

		maxPossiblePinNum = moduleDual4x16Counter * pinsInModule

		fmt.Printf("to create a mapping table for %d pins, you need %d pieces of %s modules, "+
			"Agilent 34980A has only %d installed %s modules, "+
			"maximum possible number of pins for mapping table is %d",
			totalPinsNum, requiredNumOfModules, moduleDual4x16,
			moduleDual4x16Counter, moduleDual4x16, maxPossiblePinNum)

		requiredNumOfModules = moduleDual4x16Counter
		totalPinsNum = maxPossiblePinNum
	}

	if moduleDual4x16Counter == 0 {
		return fmt.Errorf("no %s module found in Agilent 34980A slots", moduleDual4x16)
	}

	// Get pin numbers array
	pins := make([]int, totalPinsNum*moduleRowNum)
	pinsCounter := 0
	for i := 1; i <= moduleRowNum; i++ {
		for j := 1; j <= totalPinsNum; j++ {
			pins[pinsCounter] = i*relayRatio + j
			pinsCounter++
		}
	}

	// Get 34932A relay numbers array
	relayNumbersArray := make([]int, totalPinsNum*moduleRowNum)
	involvedModules := moduleDual4x16Number[0:requiredNumOfModules]
	relayArrCounter := 0

	for _, module := range involvedModules {
		for row := 1; row <= 2*moduleRowNum; row++ {
			for column := 1; column <= moduleColNum; column++ {
				relayNumbersArray[relayArrCounter] = 1000*module + 100*row + column
				relayArrCounter++
			}
		}
	}

	// Get 2 elder digits of relay number
	highDigitsInRelayNum := make([]int, len(relayNumbersArray))
	for i := 0; i < len(highDigitsInRelayNum); i++ {
		highDigitsInRelayNum[i] = relayNumbersArray[i] / 100
	}

	var data []int
	for _, module := range involvedModules {
		for i := module*10 + 1; i <= module*10+moduleColNum/2; i++ {
			data = append(data, i)
		}
	}

	//Матрица индексов
	var rowIndexes [moduleColNum][moduleRowNum]int
	for i := 0; i < totalPinsNum/moduleColNum; i++ {
		for j := 0; j < moduleRowNum; j++ {
			rowIndexes[i][j] = data[i*moduleRowNum+j]
		}
	}

	// Добавочное значение
	addIndexes := make([]int, totalPinsNum/moduleColNum)
	for i, add := 0, 0; i < len(addIndexes) && add <= totalPinsNum; i, add = i+1, add+moduleColNum {
		addIndexes[i] = add
	}

	// Перекомпановка массива реле
	relaysBlank := make([]int, moduleRowNum*relayRatio+totalPinsNum)
	for i := 0; i < len(relayNumbersArray); i++ {

		// Индексы по столбцам
		myColumn := 0
	outerLoopColumn:
		for r := 0; r < moduleRowNum; r++ {
			for c := 0; c < moduleColNum; c++ {
				if rowIndexes[c][r] == highDigitsInRelayNum[i] {
					myColumn = r + 1
					break outerLoopColumn
				}
			}
		}

		// Индексы по строкам
		myRow := 0
		commonPart := (relayNumbersArray[i] - highDigitsInRelayNum[i]*100)
	outerLoopRow:
		for r := 0; r < moduleColNum; r++ {
			for c := 0; c < moduleRowNum; c++ {
				if rowIndexes[r][c] == highDigitsInRelayNum[i] {
					myRow = commonPart + addIndexes[r]
					break outerLoopRow
				}
			}
		}
		relaysBlank[myColumn*relayRatio+myRow-1] = relayNumbersArray[i]
	}

	relays := make([]int, moduleRowNum*totalPinsNum)
	relayCounter := 0
	for _, rel := range relaysBlank {
		if rel != 0 {
			relays[relayCounter] = rel
			relayCounter++
		}
	}

	// Требуется удалить строки с выводами, которые добавлены для кратности таблицы
	if excessPins != 0 {
		for i := 0; i < len(pins); i++ {
			if pins[i]%relayRatio <= finishPins {
				sw.pinsMap[pins[i]] = relays[i]
				sw.relaysMap[relays[i]] = pins[i]
			}
		}
	} else {
		for i := 0; i < len(pins); i++ {
			sw.pinsMap[pins[i]] = relays[i]
			sw.relaysMap[relays[i]] = pins[i]
		}
	}
	return nil
}

// Конвертация номера вывода измерительной оснастки в номер реле Agilent 34932A.
func (sw *Agilent34980A) PinsToRelays(pins []int) ([]int, error) {

	relays := make([]int, len(pins))
	wrongPins := make([]int, 0)

	for i, pin := range pins {
		relay, relayExist := sw.pinsMap[pin]
		if !relayExist {
			wrongPins = append(wrongPins, pin)
			continue
		}
		relays[i] = relay
	}
	if len(wrongPins) > 0 {
		wrongPinsStr := strings.Trim(strings.Replace(fmt.Sprint(wrongPins), " ", ",", -1), "[]")
		return pins, fmt.Errorf("%s are not pin numbers for the current configuration of Agilent 34980A (%d row by %d pins)",
			wrongPinsStr, moduleRowNum, len(sw.pinsMap)/moduleRowNum)
	}
	return relays, nil
}

// Конвертация номера вывода измерительной оснастки в номер реле Agilent 34932A (представление в виде строки).
func (sw *Agilent34980A) PinsToRelaysString(pins []int) (string, error) {

	var buffer bytes.Buffer
	var relaySeries bool
	var previousRelay int

	sort.Ints(pins)
	relays, err := sw.PinsToRelays(pins)
	if err != nil {
		return "", err
	}
	lastRelay := relays[len(relays)-1]

	for i, relay := range relays {

		if relay-previousRelay > 1 {
			if buffer.Len() > 0 && !relaySeries {
				buffer.WriteString(",")
			}

			if i > 1 && previousRelay-relays[i-2] == 1 {
				buffer.WriteString(fmt.Sprintf("%d,%d", previousRelay, relay))
				relaySeries = false

			} else {
				buffer.WriteString(fmt.Sprintf("%d", relay))
			}
		} else {
			if relay == lastRelay {
				buffer.WriteString(fmt.Sprintf("%d", relay))
				break
			}

			if !relaySeries {
				buffer.WriteString(":")
				relaySeries = true
			}
		}
		previousRelay = relay
	}
	return buffer.String(), nil
}

// Конвертация номера реле Agilent 34932A в номер вывода измерительной оснастки.
func (sw *Agilent34980A) RelaysToPins(relays []int) ([]int, error) {

	pins := make([]int, len(relays))
	wrongRelays := make([]int, 0)
	var pin int

	for i, relay := range relays {
		pin = sw.relaysMap[relay]
		if pin == 0 {
			wrongRelays = append(wrongRelays, relay)
			continue
		}
		pins[i] = pin
	}
	if len(wrongRelays) > 0 {
		wrongRelaysStr := strings.Trim(strings.Replace(fmt.Sprint(wrongRelays), " ", ",", -1), "[]")
		return pins, fmt.Errorf("%s are not relay numbers of Agilent 34980A", wrongRelaysStr)
	}
	return pins, nil
}

// Open/Close 34932A relays.
func (sw *Agilent34980A) SetCommutation(pins []int, state bool) error {

	var strState string
	if state {
		strState = "CLOSE"
	} else {
		strState = "OPEN"
	}
	relayStr, err := sw.PinsToRelaysString(pins)
	if err != nil {
		return errors.Wrap(err, "commutation failed")
	}
	err = sw.instr.Write(fmt.Sprintf("ROUT:%s (@%s)", strState, relayStr))
	if err != nil {
		return errors.Wrap(err, "commutation failed")
	}
	return nil
}

// Get 34932A relay states.
func (sw *Agilent34980A) GetCommutation(pins []int) ([]bool, error) {

	var errMsg = "failed to get pin states"
	var states = make([]bool, len(pins))
	var relayStr, stateStr string
	var err error

	relayStr, err = sw.PinsToRelaysString(pins)
	if err != nil {
		return nil, errors.Wrap(err, errMsg)
	}

	stateStr, err = sw.instr.Query(fmt.Sprintf("ROUTe:CLOSe? (@%s)", relayStr))
	if err != nil {
		return nil, errors.Wrap(err, errMsg)
	}

	stateStrSplit := strings.Split(stateStr, ",")

	for i, st := range stateStrSplit {

		state, err := strconv.ParseInt(st, 10, 8)
		if err != nil {
			return nil, errors.Wrap(err, errMsg)
		}

		if state == 1 {
			states[i] = true
		} else {
			states[i] = false
		}
	}
	return states, nil
}

// Open all relays of 34932A modules installed in 34980A multifunction switch/measure mainframe.
func (sw *Agilent34980A) OpenAllRelays() error {

	err := sw.instr.Write("ROUT:OPEN:ALL ALL;*OPC")
	if err != nil {
		return err
	}
	return nil
}

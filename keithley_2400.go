// Управление источником измерителем Keithley 2400
// https://download.tek.com/manual/2400S-900-01_K-Sep2011_User.pdf

package instruments

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

type Keithley2400 struct {
	instr         *VisaObjectWrapper
	voltageRanges []float64
	currentRanges []float64
}

// Инициализация источника-измерителя.
func (ke2400 *Keithley2400) Init(instr *VisaObjectWrapper) error {

	ke2400.instr = instr
	ke2400.instr.SetErrorQuery("SYST:ERR?")
	err := ke2400.instr.Write("*RST")
	if err != nil {
		return err
	}
	ke2400.voltageRanges = []float64{0.02, 0.2, 2, 20, 200}
	ke2400.currentRanges = []float64{10e-9, 100e-9, 1e-6, 10e-6, 100e-6, 1e-3, 0.01, 0.1, 1}
	return nil
}

// Считать значения тока и напряжения.
func (ke2400 *Keithley2400) ReadSrcData() (current float64, voltage float64, err error) {

	response, err := ke2400.instr.Query(":READ?")
	if err != nil {
		return 0, 0, errors.Wrap(err, "data read fail")
	}
	splitResponse := strings.Split(response, ",")
	current, err = strconv.ParseFloat(splitResponse[2], 64)
	if err != nil {
		return 0, 0, errors.Wrap(err, "conversion for current value failed")
	}
	voltage, err = strconv.ParseFloat(splitResponse[1], 64)
	if err != nil {
		return 0, 0, errors.Wrap(err, "conversion for voltage value failed")
	}
	return
}

// Сконфигурировать выход источника-измерителя как источник напряжения с автодиапазоном.
func (ke2400 *Keithley2400) SetAutoRangeVoltageSource(srcVoltage, limCurrent, nplc float64, remote bool) error {

	var err error
	errContext := "auto range voltage source init fail"

	err = ke2400.instr.WriteWithoutCheck("SOUR:FUNC VOLT")
	if err != nil {
		return errors.Wrap(err, errContext)
	}
	err = ke2400.instr.WriteWithoutCheck("OUTP:SMOD ZERO")
	if err != nil {
		return errors.Wrap(err, errContext)
	}
	err = ke2400.instr.WriteWithoutCheck("SOUR:VOLT:PROT:LEV 210")
	if err != nil {
		return errors.Wrap(err, errContext)
	}
	err = ke2400.instr.WriteWithoutCheck("SOUR:VOLT:MODE AUTO")
	if err != nil {
		return errors.Wrap(err, errContext)
	}
	err = ke2400.instr.WriteWithoutCheck("SOUR:DEL:AUTO ON")
	if err != nil {
		return errors.Wrap(err, errContext)
	}
	err = ke2400.instr.WriteWithoutCheck("SYST:AZER:STAT ONCE")
	if err != nil {
		return errors.Wrap(err, errContext)
	}
	err = ke2400.instr.WriteWithoutCheck("SENS:FUNC \"CURR:DC\"")
	if err != nil {
		return errors.Wrap(err, errContext)
	}
	err = ke2400.instr.WriteWithoutCheck("SENS:CURR:DC:RANG:AUTO ON")
	if err != nil {
		return errors.Wrap(err, errContext)
	}

	// Error checkable settings
	err = ke2400.instr.Write(fmt.Sprintf("SOUR:VOLT %f", srcVoltage))
	if err != nil {
		return errors.Wrap(err, errContext)
	}
	err = ke2400.instr.Write(fmt.Sprintf("SENS:CURR:PROT %f", limCurrent))
	if err != nil {
		return errors.Wrap(err, errContext)
	}
	err = ke2400.instr.Write(fmt.Sprintf("SENS:CURR:NPLC %f", nplc))
	if err != nil {
		return errors.Wrap(err, errContext)
	}

	if remote {
		ke2400.instr.WriteWithoutCheck(":SYST:RSEN ON")
	} else {
		ke2400.instr.WriteWithoutCheck(":SYST:RSEN OFF")
	}
	return nil
}

// Сконфигурировать выход источника-измерителя как источник напряжения с фиксированным диапазоном.
func (ke2400 *Keithley2400) SetFixedRangeVoltageSource(srcVoltage, limCurrent, nplc float64, remote bool) error {

	var err error
	errContext := "fixed range voltage source init fail"
	vltRng := ke2400.GetSuitableVoltageRange(srcVoltage)
	curRng := ke2400.GetSuitableCurrentRange(limCurrent)

	err = ke2400.instr.WriteWithoutCheck("SOUR:FUNC VOLT")
	if err != nil {
		return errors.Wrap(err, errContext)
	}
	err = ke2400.instr.WriteWithoutCheck("OUTP:SMOD ZERO")
	if err != nil {
		return errors.Wrap(err, errContext)
	}
	err = ke2400.instr.WriteWithoutCheck("SOUR:VOLT:PROT:LEV 210")
	if err != nil {
		return errors.Wrap(err, errContext)
	}
	err = ke2400.instr.WriteWithoutCheck("SOUR:VOLT:MODE FIX")
	if err != nil {
		return errors.Wrap(err, errContext)
	}
	err = ke2400.instr.WriteWithoutCheck("SOUR:DEL:AUTO ON")
	if err != nil {
		return errors.Wrap(err, errContext)
	}
	err = ke2400.instr.WriteWithoutCheck("SYST:AZER:STAT ONCE")
	if err != nil {
		return errors.Wrap(err, errContext)
	}
	err = ke2400.instr.WriteWithoutCheck("SENS:FUNC \"CURR:DC\"")
	if err != nil {
		return errors.Wrap(err, errContext)
	}
	err = ke2400.instr.WriteWithoutCheck("SENS:CURR:DC:RANG:AUTO OFF")
	if err != nil {
		return errors.Wrap(err, errContext)
	}

	// Error checkable settings
	err = ke2400.instr.Write(fmt.Sprintf("SOUR:VOLT:RANG %f", vltRng))
	if err != nil {
		return errors.Wrap(err, errContext)
	}
	err = ke2400.instr.Write(fmt.Sprintf("SOUR:VOLT %f", srcVoltage))
	if err != nil {
		return errors.Wrap(err, errContext)
	}
	err = ke2400.instr.Write(fmt.Sprintf("SENS:CURR:DC:RANG %f", curRng))
	if err != nil {
		return errors.Wrap(err, errContext)
	}
	err = ke2400.instr.Write(fmt.Sprintf("SENS:CURR:PROT %f", limCurrent))
	if err != nil {
		return errors.Wrap(err, errContext)
	}
	err = ke2400.instr.Write(fmt.Sprintf("SENS:CURR:NPLC %f", nplc))
	if err != nil {
		return errors.Wrap(err, errContext)
	}

	if remote {
		ke2400.instr.WriteWithoutCheck("SYST:RSEN ON")
	} else {
		ke2400.instr.WriteWithoutCheck("SYST:RSEN OFF")
	}
	return nil
}

// Сконфигурировать выход источника-измерителя как источник тока с автодиапазоном
func (ke2400 *Keithley2400) SetAutoRangeCurrentSource(srcCurrent, limVoltage, nplc float64, remote bool) error {

	var err error
	errContext := "auto range current source init fail"

	err = ke2400.instr.WriteWithoutCheck("SOUR:FUNC CURR")
	if err != nil {
		return errors.Wrap(err, errContext)
	}
	err = ke2400.instr.WriteWithoutCheck("OUTP:SMOD ZERO")
	if err != nil {
		return errors.Wrap(err, errContext)
	}
	err = ke2400.instr.WriteWithoutCheck("SOUR:CURR:MODE AUTO")
	if err != nil {
		return errors.Wrap(err, errContext)
	}
	err = ke2400.instr.WriteWithoutCheck("SOUR:DEL:AUTO ON")
	if err != nil {
		return errors.Wrap(err, errContext)
	}
	err = ke2400.instr.WriteWithoutCheck("SYST:AZER:STAT ONCE")
	if err != nil {
		return errors.Wrap(err, errContext)
	}
	err = ke2400.instr.WriteWithoutCheck("SENS:FUNC \"VOLT:DC\"")
	if err != nil {
		return errors.Wrap(err, errContext)
	}
	err = ke2400.instr.WriteWithoutCheck("SENS:VOLT:DC:RANG:AUTO ON")
	if err != nil {
		return errors.Wrap(err, errContext)
	}

	// Error checkable settings
	err = ke2400.instr.Write(fmt.Sprintf("SOUR:CURR %f", srcCurrent))
	if err != nil {
		return errors.Wrap(err, errContext)
	}
	err = ke2400.instr.Write(fmt.Sprintf("SENS:VOLT:PROT %f", limVoltage))
	if err != nil {
		return errors.Wrap(err, errContext)
	}
	err = ke2400.instr.Write(fmt.Sprintf("SENS:VOLT:NPLC %f", nplc))
	if err != nil {
		return errors.Wrap(err, errContext)
	}

	if remote {
		ke2400.instr.WriteWithoutCheck("SYST:RSEN ON")
	} else {
		ke2400.instr.WriteWithoutCheck("SYST:RSEN OFF")
	}
	return nil
}

// Сконфигурировать выход источника-измерителя как источник тока с фиксированным диапазоном.
func (ke2400 *Keithley2400) SetFixedRangeCurrentSource(srcCurrent, limVoltage, nplc float64, remote bool) error {

	var err error
	errContext := "fixed range current source init fail"
	curRng := ke2400.GetSuitableCurrentRange(srcCurrent)
	vltRng := ke2400.GetSuitableVoltageRange(limVoltage)

	err = ke2400.instr.WriteWithoutCheck("SOUR:FUNC CURR")
	if err != nil {
		return errors.Wrap(err, errContext)
	}
	err = ke2400.instr.WriteWithoutCheck("OUTP:SMOD ZERO")
	if err != nil {
		return errors.Wrap(err, errContext)
	}
	err = ke2400.instr.WriteWithoutCheck("SOUR:CURR:MODE FIX")
	if err != nil {
		return errors.Wrap(err, errContext)
	}
	err = ke2400.instr.WriteWithoutCheck("SOUR:DEL:AUTO ON")
	if err != nil {
		return errors.Wrap(err, errContext)
	}
	err = ke2400.instr.WriteWithoutCheck("SYST:AZER:STAT ONCE")
	if err != nil {
		return errors.Wrap(err, errContext)
	}
	err = ke2400.instr.WriteWithoutCheck("SENS:FUNC \"VOLT:DC\"")
	if err != nil {
		return errors.Wrap(err, errContext)
	}
	err = ke2400.instr.WriteWithoutCheck("SENS:VOLT:DC:RANG:AUTO OFF")
	if err != nil {
		return errors.Wrap(err, errContext)
	}

	// Error-checkable settings
	err = ke2400.instr.Write(fmt.Sprintf("SOUR:CURR:RANG %f", curRng))
	if err != nil {
		return errors.Wrap(err, errContext)
	}
	err = ke2400.instr.Write(fmt.Sprintf("SOUR:CURR %f", srcCurrent))
	if err != nil {
		return errors.Wrap(err, errContext)
	}
	err = ke2400.instr.Write(fmt.Sprintf("SENS:VOLT:DC:RANG %f", vltRng))
	if err != nil {
		return errors.Wrap(err, errContext)
	}
	err = ke2400.instr.Write(fmt.Sprintf("SENS:VOLT:PROT %f", limVoltage))
	if err != nil {
		return errors.Wrap(err, errContext)
	}
	err = ke2400.instr.Write(fmt.Sprintf("SENS:VOLT:NPLC %f", nplc))
	if err != nil {
		return errors.Wrap(err, errContext)
	}

	if remote {
		ke2400.instr.WriteWithoutCheck("SYST:RSEN ON")
	} else {
		ke2400.instr.WriteWithoutCheck("SYST:RSEN OFF")
	}
	return nil
}

// Подобрать ближайший допустимый диапазон источника-измерителя для текущего значения напряжения.
func (ke2400 *Keithley2400) GetSuitableVoltageRange(targetVoltage float64) float64 {
	return getSuitableRange(ke2400.voltageRanges, targetVoltage)
}

// Подобрать ближайший допустимый диапазон источника измерителя для текущего значения тока.
func (ke2400 *Keithley2400) GetSuitableCurrentRange(targetCurrent float64) float64 {
	return getSuitableRange(ke2400.currentRanges, targetCurrent)
}

func getSuitableRange(rangesArray []float64, target float64) float64 {

	arrLen := len(rangesArray)
	differences := make([]float64, arrLen)
	minDifference := float64(^uint64(0) >> 1)
	targetAbsValue := math.Abs(target)
	var minDifferenceIndex int
	var targetRange float64

	for i := 0; i < arrLen; i++ {
		differences[i] = math.Abs(rangesArray[i] - targetAbsValue)
		if differences[i] < minDifference {
			minDifference = differences[i]
			minDifferenceIndex = i
		}
	}
	if targetAbsValue > rangesArray[arrLen-1] {
		targetRange = rangesArray[arrLen-1]
	} else if targetAbsValue > rangesArray[minDifferenceIndex] {
		targetRange = rangesArray[minDifferenceIndex+1]
	} else {
		targetRange = rangesArray[minDifferenceIndex]
	}
	return targetRange
}

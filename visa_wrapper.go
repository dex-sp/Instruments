package instruments

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jpoirier/visa"
	"github.com/pkg/errors"
)

const bufferSize = 1024

type VisaObjectWrapper struct {
	ResourceName    string
	ResourceManager *visa.Session
	instr           *visa.Object
	errorQuery      string
	info            map[string]string
}

func (vw *VisaObjectWrapper) Init() error {

	instr, visaStatus := vw.ResourceManager.Open(vw.ResourceName, uint32(visa.NULL), uint32(visa.NULL))
	if visaStatus != visa.SUCCESS {
		statusDesc, _ := vw.instr.StatusDesc(visaStatus)
		visaErr := fmt.Errorf("%d, %s", visaStatus, statusDesc[0:strings.Index(statusDesc, ".")])
		context := fmt.Sprintf("an VISA error occurred while connect to \"%s\"", vw.ResourceName)
		return errors.Wrap(visaErr, context)
	}

	vw.instr = &instr
	response, err := vw.Query("*IDN?")
	if err != nil {
		return err
	}
	splitResponse := strings.Split(response, ",")
	vw.info = make(map[string]string, 4)
	vw.info["Manufacturer"] = splitResponse[0]
	vw.info["Model"] = splitResponse[1]
	vw.info["Serial"] = splitResponse[2]
	vw.info["Version"] = splitResponse[3]
	return nil
}

// Write command to instr and read response
func (vw *VisaObjectWrapper) Query(cmd string) (string, error) {

	_, visaStatus := vw.instr.Write([]byte(cmd), uint32(len(cmd)))
	if visaStatus != visa.SUCCESS {
		statusDesc, _ := vw.instr.StatusDesc(visaStatus)
		visaErr := fmt.Errorf("%d, %s", visaStatus, statusDesc[0:strings.Index(statusDesc, ".")])
		context := fmt.Sprintf("an VISA error occurred while writing \"%s\" command", cmd)
		return "", errors.Wrap(visaErr, context)
	}

	bytes, _, visaStatus := vw.instr.Read(bufferSize)
	if visaStatus != visa.SUCCESS {
		instrErr := vw.CheckErrors()
		if instrErr != nil {
			context := fmt.Sprintf("an instr error occurred while reading response after \"%s\" command", cmd)
			return "", errors.Wrap(instrErr, context)
		} else {
			statusDesc, _ := vw.instr.StatusDesc(visaStatus)
			visaErr := fmt.Errorf("%d, %s", visaStatus, statusDesc[0:strings.Index(statusDesc, ".")])
			context := fmt.Sprintf("an VISA error occurred while reading response after \"%s\" command", cmd)
			return "", errors.Wrap(visaErr, context)
		}
	}
	response := string(bytes)
	if len(response) == 0 {
		return response, fmt.Errorf("get empty response from instr after \"%s\" command", cmd)
	}
	return response[0:strings.Index(response, "\n")], nil
}

// Write command to instr
func (vw *VisaObjectWrapper) Write(cmd string) error {

	_, visaStatus := vw.instr.Write([]byte(cmd), uint32(len(cmd)))
	if visaStatus != visa.SUCCESS {
		statusDesc, _ := vw.instr.StatusDesc(visaStatus)
		visaErr := fmt.Errorf("%d, %s", visaStatus, statusDesc[0:strings.Index(statusDesc, ".")])
		context := fmt.Sprintf("an VISA error occurred while writing \"%s\" command", cmd)
		return errors.Wrap(visaErr, context)
	}
	instrErr := vw.CheckErrors()
	if instrErr != nil {
		context := fmt.Sprintf("an instr error occurred while writing \"%s\" command", cmd)
		return errors.Wrap(instrErr, context)
	}
	return nil
}

// Write command to instr without instr error check (not recommended)
func (vw *VisaObjectWrapper) WriteWithoutCheck(cmd string) error {

	_, visaStatus := vw.instr.Write([]byte(cmd), uint32(len(cmd)))
	if visaStatus != visa.SUCCESS {
		statusDesc, _ := vw.instr.StatusDesc(visaStatus)
		visaErr := fmt.Errorf("%d, %s", visaStatus, statusDesc[0:strings.Index(statusDesc, ".")])
		context := fmt.Sprintf("an VISA error occurred while writing \"%s\" command", cmd)
		return errors.Wrap(visaErr, context)
	}
	return nil
}

// Check instrument errors
func (vw *VisaObjectWrapper) CheckErrors() error {

	res, _ := vw.Query(vw.errorQuery + ";*CLS")
	res = strings.ReplaceAll(res, "\"", "")
	splitRes := strings.Split(res, ",")
	code, _ := strconv.Atoi(splitRes[0])

	if code != 0 {
		return fmt.Errorf(res)
	}
	return nil
}

// Cast instrument info to string
func (vw *VisaObjectWrapper) String() string {
	infoStr := fmt.Sprintf(
		"Manufacturer:\t%s\n"+
			"Model:\t\t%s\n"+
			"Serial:\t\t%s\n"+
			"Version:\t%s\n",
		vw.info["Manufacturer"], vw.info["Model"], vw.info["Serial"], vw.info["Version"])
	return infoStr
}

func (vw *VisaObjectWrapper) SetErrorQuery(query string) {
	vw.errorQuery = query
}

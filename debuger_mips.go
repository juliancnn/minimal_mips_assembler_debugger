package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/tarm/serial"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var serDev *serial.Port
var verbose bool
var asm string;

func main() {

	// Uart config
	var dev string;
	var baud int;
	// File input config
	var fileProgram string

	fmt.Println("\n\nGuelcome tu de debag for de MIPs\n")

	flag.StringVar(&dev, "d", "ttyUSB1", "Dispositivo serial en el /dev/<dispositivo>")
	flag.IntVar(&baud, "b", 115200, "Baud rate")
	flag.StringVar(&fileProgram, "l", "", "Program file to be load")
	flag.BoolVar(&verbose, "v", false, "Muestra lo enviado y lo recibido por la uart")
	flag.Parse()

	if "" == fileProgram {
		fmt.Println("Negri cargame el programa, mentime con el archivo aunque sea")
		flag.Usage()
		return
	} else {
		info, err := os.Stat(fileProgram)
		if os.IsNotExist(err) || info.IsDir() {
			fmt.Println("El archivo %s no existe o es un directorio\n", fileProgram)
		}
		asm = loadASM(fileProgram);
	}


	serDev = connect(dev, baud)
	defer serDev.Close()
	serDev.Flush() // Descarto datos en buffer i/o sin enviarlos ni lerlos

	getPrompt()

}

//================================ Manejo del serial ==================================================================

/**************************************************
				CONEXION SERIAL
***************************************************/

func connect(file string, baud int) *serial.Port {

	c := &serial.Config{Name: "/dev/" + file, Baud: baud}

	s, err := serial.OpenPort(c)
	if err != nil {
		log.Fatal("[Error en conexion] %v", err)
	}
	return s;
}

func sendBytes(buffer []byte, num int) {

	_, err := serDev.Write(buffer[:num])
	if err != nil {
		log.Fatal("[Error enviando datos]", err)
	}

}

func reciveBytes(num int) []byte {

	buf := make([]byte, 0, num)
	interBuf := make([]byte, 1)

	for nRecv := 0; nRecv < num; {
		n, err := serDev.Read(interBuf)
		if err != nil {
			log.Fatal(err)
		}
		buf = append(buf, interBuf[0])
		nRecv = nRecv + n
	}

	return buf;

}

//================================== Comunicacion serial con el debugger ==============================================

/**************************************************
				Dump of reg file
***************************************************/

func getPC() uint32 {
	const dumpPCread byte = byte(30)
	const dataToSend int = 6;
	const dataToRecv int = 4;

	var pc uint32;
	var requestPC []byte = []byte{dumpPCread, 0x00, 0x00}
	var requestRead []byte = []byte{0xff, 0xff, 0xff}

	var sendArray []byte = make([]byte, 0, dataToSend)

	sendArray = append(sendArray,requestPC...)
	sendArray = append(sendArray,requestRead...)

	var registerFile []byte

	sendBytes(sendArray, dataToSend)
	if verbose{
		fmt.Printf(">> [Dump PC] || % x || %d \n", sendArray, sendArray)
	}

	registerFile = reciveBytes(dataToRecv)
	if verbose{
		fmt.Printf("<< [RES PC]  || % x || %d \n", registerFile, registerFile)
	}

	pc = uint32(registerFile[3]) << 24 | uint32(registerFile[2]) << 16 | uint32(registerFile[1]) << 8 | uint32(registerFile[0])

	return pc
}

/**************************************************
				Just Do'it, ahr
***************************************************/
func runStep() {
	const doStepEnable byte = byte(11)
	const doStepDisable byte = byte(12)
	const dataToSend int = 6 // 4optimi

	var sendArray []byte = make([]byte, 0, dataToSend)

	sendArray = append(sendArray, doStepEnable, 0x00, 0x00)
	sendArray = append(sendArray, doStepDisable, 0x00, 0x00)

	sendBytes(sendArray, len(sendArray))
	if verbose{
		fmt.Printf(">> [duStep] || % x || %d \n", sendArray, sendArray)
	}

	return
}
/**************************************************
				Just Doit
***************************************************/
func runRun() {
	const doRunEnable byte = byte(13)
	const doRunDisable byte = byte(14)
	const dataToSend int = 6 // 4optimi

	var sendArray []byte = make([]byte, 0, dataToSend)

	sendArray = append(sendArray, doRunEnable, 0x00, 0x00)
	sendArray = append(sendArray, doRunDisable, 0x00, 0x00)

	sendBytes(sendArray, len(sendArray))
	if verbose{
		fmt.Printf(">> [duStep] || % x || %d \n", sendArray, sendArray)
	}

	return
}
/**************************************************
				Write one instrucction
***************************************************/

func writeInstruction(addr int32, instruccion string) {
	const writeInstructionAddrLow byte = byte(15)
	const writeInstructionAddrHigh byte = byte(16)
	const writeInstructionDataLow byte = byte(17)
	const writeInstructionDataHigh byte = byte(18)
	const writeInstructionEnable byte = byte(19)
	const writeInstructionDisable byte = byte(20)

	const dataToSend int = 18 // 4optimi

	var sendArray []byte = make([]byte, 0, dataToSend)

	addrByte := str32toByte(fmt.Sprintf("%032b", addr))
	instByte := str32toByte(instruccion);


	sendArray = append(sendArray, writeInstructionAddrHigh)
	sendArray = append(sendArray, addrByte[0:2]...)

	sendArray = append(sendArray, writeInstructionAddrLow)
	sendArray = append(sendArray, addrByte[2:4]...)

	sendArray = append(sendArray, writeInstructionDataHigh)
	sendArray = append(sendArray, instByte[0:2]...)

	sendArray = append(sendArray, writeInstructionDataLow)
	sendArray = append(sendArray, instByte[2:4]...)

	sendArray = append(sendArray, writeInstructionEnable, 0x00, 0x00)
	sendArray = append(sendArray, writeInstructionDisable, 0x00, 0x00)

	sendBytes(sendArray, len(sendArray))
	if verbose{
		fmt.Printf(">> [WrInst] || % x || %d \n", sendArray, sendArray)
	}


	return // Devolver el ack
}

/**************************************************
				Dump of reg file
***************************************************/

func dumRegFile() [32][]byte {
	const dumpRegWriteIndex byte = byte(21)
	const dumpRegRead byte = byte(24)
	const dataToSend int = 9;
	const dataToRecv int = 4;
	var requestRead []byte = []byte{0xff, 0xff, 0xff}

	var sendArray []byte = make([]byte, 0, dataToSend)
	var registerFile [32][]byte

	for i := 0; i < 32; i++ {
		sendArray = append(sendArray, dumpRegWriteIndex)
		sendArray = append(sendArray, 0x00)
		sendArray = append(sendArray, byte(i))
		sendArray = append(sendArray, dumpRegRead, 0x00, 0x00)
		sendArray = append(sendArray, requestRead...)

		sendBytes(sendArray, dataToSend)
		if verbose{
			fmt.Printf(">> [Dump R%d] || % x || %d \n", i, sendArray, sendArray)
		}


		registerFile[i] = reciveBytes(dataToRecv)
		if verbose{
			fmt.Printf("<< [Dump R%d]  || % x || %d \n", i, registerFile[i], registerFile[i])
		}
		sendArray = sendArray[:0] //Keep allocated memory
	}

	return registerFile
}

/**************************************************
				Dump de memory data
***************************************************/

func dumpMemData(start int, end int) [][]byte {
	const writeAddrMemLow byte = byte(22)
	const writeAddrMemHigh byte = byte(23)
	const readMemData byte = byte(29)
	var requestRead []byte = []byte{0xff, 0xff, 0xff}
	const dataToSend int = 12
	const dataToRecv int = 4

	var numRegs int = end - start + 1
	if numRegs < 0 {
		log.Printf("Negri el start(%v) no puede ser mayor que el end(%v)\n", start, end)
		return nil
	}
	var sendArray []byte = make([]byte, 0, dataToSend)
	var memData [][]byte = make([][]byte, 0, numRegs)
	var addrByte []byte

	for i := 0; i < numRegs; i++ {
		addrByte = str32toByte(fmt.Sprintf("%032b", start+i))
		sendArray = append(sendArray, writeAddrMemHigh)
		sendArray = append(sendArray, addrByte[0:2]...)
		sendArray = append(sendArray, writeAddrMemLow)
		sendArray = append(sendArray, addrByte[2:4]...)
		sendArray = append(sendArray, readMemData)
		sendArray = append(sendArray, 0x00, 0x00)
		sendArray = append(sendArray, requestRead...)

		sendBytes(sendArray, dataToSend)
		if verbose {
			fmt.Printf(">> [Dump Mem %d] || % x || %d \n", (start+i), sendArray, sendArray)
		}

		memData = append(memData, reciveBytes(dataToRecv))
		if verbose {
			fmt.Printf("<< [Dump Mem %d]  || % x || %d \n", start+i, memData[i], memData[i])
		}
		sendArray = sendArray[:0] //Keep allocated memory
	}

	return memData
}

//===================================== utils =========================================================================
/***************************************************
			str32bits to [4]byte

str:   "00000000  00000000  00000000  00000000"
       | byte[0] | byte[1] | byte[2] | byte[3]|
***************************************************/
func str32toByte(str string) []byte {

	var result []byte = make([]byte, 4)
	str = strings.TrimSuffix(str, "\n")
	if len(str) != 32 {
		log.Fatal("[%v], no es una cadena valida de 32 bits para transformar", len(str))
	}

	i, _ := strconv.ParseUint(str[0:8], 2, 8)
	result[0] = byte(i)
	i, _ = strconv.ParseUint(str[8:16], 2, 8)
	result[1] = byte(i)
	i, _ = strconv.ParseUint(str[16:24], 2, 8)
	result[2] = byte(i)
	i, _ = strconv.ParseUint(str[24:32], 2, 8)
	result[3] = byte(i)

	return result;

}

func loadASM(filename string) string {
	var contentFile string

	var rawFile []byte
	var err error
	rawFile, err = ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}
	contentFile = string(rawFile)

	return contentFile
}

func writeProgram() {
	lines := strings.Split(asm, "\n")

	for i, v := range lines[:len(lines)-1]{
		writeInstruction(int32(i),v)
	}
}

//======================================= command line ================================================================
func getPrompt() {
	var reDumpReg *regexp.Regexp
	var reStep *regexp.Regexp
	var reRun *regexp.Regexp
	var reExit *regexp.Regexp
	var reDumpMem *regexp.Regexp
	var reLoadRom *regexp.Regexp
	var rePC *regexp.Regexp

	reader := bufio.NewReader(os.Stdin)

	reExit = regexp.MustCompile(`(?m)exit$`)      // exit: sale del dumper
	reDumpReg = regexp.MustCompile(`(?m)dumprf$`) // dumprf: dumpea los regfiles
	reStep = regexp.MustCompile(`(?m)step$`)      // step: hace un step
	reRun = regexp.MustCompile(`(?m)run$`)      // run: hace un run run
	rePC = regexp.MustCompile(`(?m)pc$`)      // pc: hace un run run
	reDumpMem = regexp.MustCompile(`(?m)dumpmem\s+([0-9]+)\s+([0-9]+)\s*$`) // dumpmem start end
	reLoadRom = regexp.MustCompile(`(?m)load\s*$`) // dumpmem start end

	for {
		fmt.Print("#Debugger -> ")
		text, _ := reader.ReadString('\n')
		text = strings.Replace(text, "\n", "", -1)

		if reExit.MatchString(text) {
			return
		} else if reDumpReg.MatchString(text) {
			dump := dumRegFile()
			for i,v := range dump{
				fmt.Printf( " | R%02d %s \n", i, prettyReg(v) )
			}
		} else if reStep.MatchString(text) {
			runStep()
		} else if reRun.MatchString(text) {
			runRun()
		} else if rePC.MatchString(text) {
			fmt.Printf( " | PC: %4d \n", getPC() )
		} else if reLoadRom.MatchString(text) {
			writeProgram()
		} else if reDumpMem.MatchString(text) {
			match := reDumpMem.FindStringSubmatch(text)
			start, _ := strconv.Atoi(match[1])
			end, _ := strconv.Atoi(match[2])
			dump := dumpMemData(start, end)
			for i,v := range dump{
				fmt.Printf( " | Mem[%03d-%03d] %s \n", (i+start)*4, (i+start)*4+3, prettyReg(v) )
			}
		} else {
			fmt.Println("Comando no reconocido")
		}

	}

	return
}

func prettyReg(dump [] byte) string {
	var num uint32;

	if len(dump) !=4 {
		log.Fatal("Solo para 4 bytes\n");
	}

	num = uint32(dump[3]) << 24 | uint32(dump[2]) << 16 | uint32(dump[1]) << 8 | uint32(dump[0])

	return fmt.Sprintf("| % x |  %3d  | %6d |", dump, dump, num)
}

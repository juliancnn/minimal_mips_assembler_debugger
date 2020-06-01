package main

import (
	"flag"
	"fmt"
	"github.com/tarm/serial"
	"log"
	"os"
	"strconv"
	"strings"
)

var serDev *serial.Port

// @TODO chequear todos los cmd enviados y orden de bytes
func main() {

	// Uart config
	var dev string;
	var baud int;
	// File input config
	var fileProgram string

	fmt.Println("\n\nGuelcome tu de debag for de MIPs\n")

	flag.StringVar(&dev,"d", "ttyUSB1", "Dispositivo serial en el /dev/<dispositivo>")
	flag.IntVar(&baud,"b", 115200, "Baud rate")
	flag.StringVar(&fileProgram,"l", "", "Program file to be load")
	flag.Parse()


	if "" == fileProgram{
		fmt.Println("Negri cargame el programa, mentime con el archivo aunque sea")
		//flag.Usage()
		//return
	}else {
		info, err := os.Stat(fileProgram)
		if os.IsNotExist(err) || info.IsDir() {
			fmt.Println("El archivo %s no existe o es un directorio\n", fileProgram)
		}
	}


	serDev = connect(dev, baud)
	defer serDev.Close()
	serDev.Flush() // Descarto datos en buffer i/o sin enviarlos ni lerlos



}

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

//=====================================================================================================================
/**************************************************
				Write one instrucction
***************************************************/

func writeInstruction(addr int32, instruccion string) {
	const writeInstructionAddrLow byte = byte(15)
	const writeInstructionAddrHigh byte = byte(16)
	const writeInstructionDataLow byte = byte(17)
	const writeInstructionDataHigh byte = byte(18)
	const writeInstructionEnable  byte = byte(19)
	const writeInstructionDisable  byte = byte(20)

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
	fmt.Printf(">> [WrInst] || % x || %d \n", sendArray, sendArray)

	return // Devolver el ack
}

/**************************************************
				Dump of reg file
@TODO falta del lado del debugUnit setear para leer el regfile esto no va a andar hasta entonnces

***************************************************/

func dumRegFile() [32][]byte {
	const dumpRegIndex byte = byte(21)
	const dataToSend int = 6;
	const dataToRecv int = 4;
	var requestRead []byte = []byte{0xff,0xff,0xff}

	var sendArray []byte = make([]byte, 0, dataToSend)
	var registerFile [32][]byte

	for i := 0; i < 32; i++ {
		sendArray = append(sendArray, dumpRegIndex)
		sendArray = append(sendArray, 0x00)
		sendArray = append(sendArray, byte(i))
		sendArray = append(sendArray, requestRead...)

		sendBytes(sendArray, dataToSend)
		fmt.Printf(">> [Dump R%i] || % x || %d \n", i, sendArray, sendArray)

		registerFile[i] = reciveBytes(dataToRecv)
		fmt.Printf("<< [Dump R%i]  || % x || %d \n", i, registerFile[i], registerFile[i])
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
	var requestRead []byte = []byte{0xff,0xff,0xff}
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


	for i := 0 ; i < numRegs ; i++{
		addrByte = str32toByte(fmt.Sprintf("%032b", start + i))
		sendArray = append(sendArray, writeAddrMemHigh)
		sendArray = append(sendArray, addrByte[0:2]...)
		sendArray = append(sendArray, writeAddrMemLow)
		sendArray = append(sendArray, addrByte[2:4]...)
		sendArray = append(sendArray, readMemData)
		sendArray = append(sendArray, 0x00,0x00)
		sendArray = append(sendArray, requestRead...)

		sendBytes(sendArray, dataToSend)
		fmt.Printf(">> [Dump Mem %i] || % x || %d \n", start + i, sendArray, sendArray)

		memData[i] = reciveBytes(dataToRecv)
		fmt.Printf("<< [Dump Mem %i]  || % x || %d \n", start + i, memData[i], memData[i])
		sendArray = sendArray[:0] //Keep allocated memory
	}


	return memData
}

//=====================================================================================================================
/*
			str32bits to [4]byte

str:   "00000000  00000000  00000000  00000000"
       | byte[0] | byte[1] | byte[2] | byte[3]|
*/
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

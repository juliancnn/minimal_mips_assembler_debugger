package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// Sporta asm con instrucciones mayusculas o minusculas,
// comentarios del tipo // y multilineas con /* */
// Los registris pueden terner nosmbres como $R1 R1 $1 r01
// Detecta etiquetas duplicadas en la declaracion.
// detecta uso de etiquetas sin asignar.
// Las instrucciones se separan por endlines, el ; es opcional

func main() {

	var asmContent string
	var inputCode *string
	var fileInputName *string
	var fileOutputName *string
	var fdOutput *os.File
	var binaryCode []string;

	//Check the flags
	fileInputName = flag.String("i", "", "Input file to assemble")
	fileOutputName = flag.String("o", "", "text binary Output file")
	inputCode = flag.String("a", "", "ASM one line, [dont use with -i]")
	flag.Parse()

	// Input ASM
	if "" != *fileInputName{
		asmContent = loadASM(*fileInputName)
	}else if "" != *inputCode {
		asmContent = *inputCode
	}  else {
		flag.Usage()
		return
	}


	// Output ASM
	if *fileOutputName != ""{
		var err error
		fdOutput, err = os.Create(*fileOutputName)
		if err != nil {
			fmt.Println(err)
			return
		}

	}



	rawAsm := clearCode(asmContent)   //stg 1
	asm, tags := removeLabels(rawAsm) // stg2
	// Vervose stage 2
	fmt.Print("\n--------MAPA DE TAGS--------------\n")
	fmt.Print(tags)
	fmt.Print("\n------CODIGO LIMPIO---------------\n")
	for i := 0; i < len(asm); i++ {
		fmt.Printf("[%02d]: %v\n", i, asm[i])
	}
	fmt.Print("\n--------TOKENIZADO----------------\n")
	asmListTokens := tokenicer(asm)

	fmt.Print("\n---------RESOLUCION DE TAGS-------\n")
	asmListTokens = resolveTags(asmListTokens, tags)
	for i := 0; i < len(asmListTokens); i++ {
		fmt.Printf("[%02d]: %q\n", i, asmListTokens[i])
	}

	fmt.Print("\n--------GENERACION DEL ASM ------\n")
	for i, inst := range asmListTokens {
		binaryCode = append(binaryCode, generateLine(inst))
		fmt.Printf("%s\n",binaryCode[i] )
	}

	if fdOutput != nil{
		for _,line := range binaryCode{
			fmt.Fprintln(fdOutput,line)
		}
	}

}

/**********************************************
 	Read the file and return his content
@WARNING: Stop the program if don't be opened
***********************************************/
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

/************************************************
				First Stage
		Prepare code for the pre-processos
***********************************************/
func clearCode(code string) []string {

	var reComments *regexp.Regexp
	var codeClean string


	// Delete Inline comments //
	reComments = regexp.MustCompile(`(?m)s*//.*$`)
	codeClean = reComments.ReplaceAllString(code, "")
	// Delte multiline comments /* */
	reComments = regexp.MustCompile(`(?ms)(?mU)\/\*(\n|\r\n|\z|\s|.)*\*\/`)
	codeClean = reComments.ReplaceAllString(codeClean, "")
	// Left justify
	reComments = regexp.MustCompile(`(?m)^\s*`)
	codeClean = reComments.ReplaceAllString(codeClean, "")
	/* ------------------ BUILT IN SIM  START  ------------------  */
	// REMPLACE nop for sll r0, r0, 0
	reComments = regexp.MustCompile(`(?mi)^nop\s*`)
	codeClean = reComments.ReplaceAllString(codeClean, "sll r0, r0, 0\n")
	/* ------------------ BUILT IN SIM  START  ------------------  */
	// Generate halt
	reComments = regexp.MustCompile(`(?mi)^halt\s*`)
	codeClean = reComments.ReplaceAllString(codeClean, "halt 0\n")
	/*  ------------------ BUILT IN SIM  END  ------------------  */
	// Delete spaces after end int.
	reComments = regexp.MustCompile(`(?m)\s*$`)
	codeClean = reComments.ReplaceAllString(codeClean, "")
	// Delete semicolon;
	reComments = regexp.MustCompile(`(?m);*$`)
	codeClean = reComments.ReplaceAllString(codeClean, "")
	// Delete $;
	reComments = regexp.MustCompile(`(?m)\$`)
	codeClean = reComments.ReplaceAllString(codeClean, "")
	// delete blank lines
	reComments = regexp.MustCompile(`(?m)^\s*$[\r\n]*|[\r\n]+\s+\z`)
	codeClean = reComments.ReplaceAllString(codeClean, "")

	codeClean = strings.ToUpper(codeClean)
	return strings.Split(codeClean, "\n")

}

/************************************************
				Stage 2
	Extract the labels and theirs address
***********************************************/
func removeLabels(rawAsm []string) ([]string, map[string]int) {

	var reLabels *regexp.Regexp
	//tagMap := make([]tlabelDir, 0, 10)
	tagMap := make(map[string]int, 10)
	asm := make([]string, 0, len(rawAsm))

	reLabels = regexp.MustCompile(`(?m)^\w*:`)

	var count int
	for i := 0; i < len(rawAsm); i++ {
		if reLabels.MatchString(rawAsm[i]) {
			str := strings.Replace(rawAsm[i], ":", "", 1)
			tagMap[str] = i - count
			count++
		} else {
			asm = append(asm, rawAsm[i])
		}
	}
	if len(tagMap) != count {
		log.Fatal("Exist duplicated tag in the asm!")
	}

	return asm, tagMap
}

/************************************************
				Stage 3
           generate tokens!
***********************************************/

func tokenicer(asm []string) [][]string {
	//var branchRel = [...]string{"BNE", "BEQ"}
	//var jumpAbs = [...]string{"BNE", "BEQ"}
	var listTokens [][]string

	var regRules = [...]string{
		`(?m)(\w+)\s+(-{0,1}\w+)\s*,\s*(-{0,1}\w+)\s*,\s*(-{0,1}\w+)\s*$`,    // format3args
		`(?m)(\w+)\s+(-{0,1}\w+)\s*,\s*(-{0,1}\w+)\s*\(\s*(-{0,1}\w+)\)\s*$`, //format3argsWbracket
		`(?m)(\w+)\s+(-{0,1}\w+)\s*,\s*(-{0,1}\w+)\s*$`,                      // format2args
		`(?m)(\w+)\s+(-{0,1}\w+)\s*$`,                                        //format1args
	}
	var compileRegx [len(regRules)]*regexp.Regexp

	for i := 0; i < len(regRules); i++ {
		compileRegx[i] = regexp.MustCompile(regRules[i])
	}

	for i := 0; i < len(asm); i++ {
		for j := 0; j < len(compileRegx); j++ {
			res := compileRegx[j].FindStringSubmatch(asm[i])
			if nil != res {
				fmt.Printf("[%02d]: %q\n", i, res[1:])
				listTokens = append(listTokens, res[1:])
			} else if j-1 == len(compileRegx) {
				log.Fatal("\nNo hay regex para vos bebe![%02d] %v  || %v\n", i, asm[i], res)
			}

		}

	}

	return listTokens

}

/************************************************
				Stage 4
			Resolve tags!
***********************************************/

func resolveTags(asmTekenized [][]string, tags map[string]int) [][]string {
	var branchRel = [...]string{"BNE", "BEQ"} // Constantes x ser array inmutalbe
	var jumpAbs = [...]string{"JAL", "J"}   // Constantes x ser array inmutalbe
	const posOffset int = 3
	const posTag int = 1

	for i, val := range asmTekenized {
		if branchRel[0] == val[0] || branchRel[1] == val[0] {
			absJump, exist := tags[val[posOffset]]
			if !exist {
				log.Fatalf("Error en tags, '%v' se usa en el codigo |%v|"+
					"pero no se encuentra definida como tag", val[posOffset], val)
			}
			offset := absJump - i - 1
			asmTekenized[i][posOffset] = strconv.Itoa(offset)
		} else if jumpAbs[0] == val[0] || jumpAbs[1] == val[0] {
			absJump, exist := tags[val[posTag]]
			if !exist {
				log.Fatalf("Error en tags, '%v' se usa en el codigo |%v|"+
					"pero no se encuentra definida como tag", val[posTag], val)
			}
			asmTekenized[i][posTag] = strconv.Itoa(absJump)
		}
	}

	return asmTekenized
}

/************************************************
				Stage 4
			String to binary string!
***********************************************/

func str2binstr(str string, len int) string {

	const minInt32 uint32 = ^uint32(0)
	var strbin string

	rex := regexp.MustCompile(`R{0,1}(-{0,1}\d+)`)

	strNum := rex.FindStringSubmatch(str)
	if nil == strNum {
		log.Fatalf("Esto [%v] no es un numero ni un registro maestro\n", str)
	}

	num, _ := strconv.Atoi(strNum[1])

	if num < 0 {
		nunComp := (uint32(-num) ^ minInt32) + 1
		strbin = fmt.Sprintf("%032b", nunComp)
	} else {
		strbin = fmt.Sprintf("%032b", num)
	}

	//return "/" + strbin[32-len:] + "/"
	return strbin[32-len:]
}

/************************************************
				Stage 5
      	  Generacion del ASM
***********************************************/
// genera la instruccion
func generateLine(token []string) string {
	var inst_bin string
	inst_bin = "00000000000000000000000000000000"

	switch token[0] {
	case "SLL":
		inst_bin = setRD(inst_bin, token[1])
		inst_bin = setRT(inst_bin, token[2])
		inst_bin = setShamt(inst_bin, token[3])
		inst_bin = setFunc(inst_bin, "000000")
	case "SRL":
		inst_bin = setRD(inst_bin, token[1])
		inst_bin = setRT(inst_bin, token[2])
		inst_bin = setShamt(inst_bin, token[3])
		inst_bin = setFunc(inst_bin, "000010")
	case "SRA":
		inst_bin = setRD(inst_bin, token[1])
		inst_bin = setRT(inst_bin, token[2])
		inst_bin = setShamt(inst_bin, token[3])
		inst_bin = setFunc(inst_bin, "000011")
	case "SLLV":
		inst_bin = setRD(inst_bin, token[1])
		inst_bin = setRT(inst_bin, token[2])
		inst_bin = setRS(inst_bin, token[3])
		inst_bin = setFunc(inst_bin, "000100")
	case "SRLV":
		inst_bin = setRD(inst_bin, token[1])
		inst_bin = setRT(inst_bin, token[2])
		inst_bin = setRS(inst_bin, token[3])
		inst_bin = setFunc(inst_bin, "000110")
	case "SRAV":
		inst_bin = setRD(inst_bin, token[1])
		inst_bin = setRT(inst_bin, token[2])
		inst_bin = setRS(inst_bin, token[3])
		inst_bin = setFunc(inst_bin, "000111")
	case "ADDU":
		inst_bin = setRD(inst_bin, token[1])
		inst_bin = setRT(inst_bin, token[3])
		inst_bin = setRS(inst_bin, token[2])
		inst_bin = setFunc(inst_bin, "100001")
	case "SUBU":
		inst_bin = setRD(inst_bin, token[1])
		inst_bin = setRT(inst_bin, token[3])
		inst_bin = setRS(inst_bin, token[2])
		inst_bin = setFunc(inst_bin, "100011")
	case "AND":
		inst_bin = setRD(inst_bin, token[1])
		inst_bin = setRT(inst_bin, token[2])
		inst_bin = setRS(inst_bin, token[3])
		inst_bin = setFunc(inst_bin, "100100")
	case "OR":
		inst_bin = setRD(inst_bin, token[1])
		inst_bin = setRT(inst_bin, token[2])
		inst_bin = setRS(inst_bin, token[3])
		inst_bin = setFunc(inst_bin, "100101")
	case "XOR":
		inst_bin = setRD(inst_bin, token[1])
		inst_bin = setRT(inst_bin, token[3])
		inst_bin = setRS(inst_bin, token[2])
		inst_bin = setFunc(inst_bin, "100110")
	case "NOR":
		inst_bin = setRD(inst_bin, token[1])
		inst_bin = setRT(inst_bin, token[3])
		inst_bin = setRS(inst_bin, token[2])
		inst_bin = setFunc(inst_bin, "100111")
	case "SLT":
		inst_bin = setRD(inst_bin, token[1])
		inst_bin = setRT(inst_bin, token[3])
		inst_bin = setRS(inst_bin, token[2])
		inst_bin = setFunc(inst_bin, "101010")
	case "LB":
		inst_bin = setOpCode(inst_bin, "100000")
		inst_bin = setRT(inst_bin, token[1])
		inst_bin = setOffsetInmed(inst_bin, token[2])
		inst_bin = setRS(inst_bin, token[3])
	case "LH":
		inst_bin = setOpCode(inst_bin, "100001")
		inst_bin = setRT(inst_bin, token[1])
		inst_bin = setOffsetInmed(inst_bin, token[2])
		inst_bin = setRS(inst_bin, token[3])
	case "LW":
		inst_bin = setOpCode(inst_bin, "100011")
		inst_bin = setRT(inst_bin, token[1])
		inst_bin = setOffsetInmed(inst_bin, token[2])
		inst_bin = setRS(inst_bin, token[3])
	case "LWU":
		inst_bin = setOpCode(inst_bin, "100111")
		inst_bin = setRT(inst_bin, token[1])
		inst_bin = setOffsetInmed(inst_bin, token[2])
		inst_bin = setRS(inst_bin, token[3])
	case "LHU":
		inst_bin = setOpCode(inst_bin, "100101")
		inst_bin = setRT(inst_bin, token[1])
		inst_bin = setOffsetInmed(inst_bin, token[2])
		inst_bin = setRS(inst_bin, token[3])
	case "LBU":
		inst_bin = setOpCode(inst_bin, "100100")
		inst_bin = setRT(inst_bin, token[1])
		inst_bin = setOffsetInmed(inst_bin, token[2])
		inst_bin = setRS(inst_bin, token[3])
	case "SB":
		inst_bin = setOpCode(inst_bin, "101000")
		inst_bin = setRT(inst_bin, token[1])
		inst_bin = setOffsetInmed(inst_bin, token[2])
		inst_bin = setRS(inst_bin, token[3])
	case "SH":
		inst_bin = setOpCode(inst_bin, "101001")
		inst_bin = setRT(inst_bin, token[1])
		inst_bin = setOffsetInmed(inst_bin, token[2])
		inst_bin = setRS(inst_bin, token[3])
	case "SW":
		inst_bin = setOpCode(inst_bin, "101011")
		inst_bin = setRT(inst_bin, token[1])
		inst_bin = setOffsetInmed(inst_bin, token[2])
		inst_bin = setRS(inst_bin, token[3])

	case "ADDI":
		inst_bin = setOpCode(inst_bin, "001000")
		inst_bin = setRT(inst_bin, token[1])
		inst_bin = setOffsetInmed(inst_bin, token[3])
		inst_bin = setRS(inst_bin, token[2])
	case "ANDI":
		inst_bin = setOpCode(inst_bin, "001100")
		inst_bin = setRT(inst_bin, token[1])
		inst_bin = setOffsetInmed(inst_bin, token[3])
		inst_bin = setRS(inst_bin, token[2])
	case "ORI":
		inst_bin = setOpCode(inst_bin, "001101")
		inst_bin = setRT(inst_bin, token[1])
		inst_bin = setOffsetInmed(inst_bin, token[3])
		inst_bin = setRS(inst_bin, token[2])
	case "XORI":
		inst_bin = setOpCode(inst_bin, "001110")
		inst_bin = setRT(inst_bin, token[1])
		inst_bin = setOffsetInmed(inst_bin, token[3])
		inst_bin = setRS(inst_bin, token[2])
	case "LUI":
		inst_bin = setOpCode(inst_bin, "001111")
		inst_bin = setRT(inst_bin, token[1])
		inst_bin = setOffsetInmed(inst_bin, token[2])
	case "SLTI":
		inst_bin = setOpCode(inst_bin, "001010")
		inst_bin = setRT(inst_bin, token[1])
		inst_bin = setOffsetInmed(inst_bin, token[3])
		inst_bin = setRS(inst_bin, token[2])
	case "BEQ":
		inst_bin = setOpCode(inst_bin, "000100")
		inst_bin = setRT(inst_bin, token[1])
		inst_bin = setOffsetInmed(inst_bin, token[3])
		inst_bin = setRS(inst_bin, token[2])
	case "BNE":
		inst_bin = setOpCode(inst_bin, "000101")
		inst_bin = setRT(inst_bin, token[1])
		inst_bin = setOffsetInmed(inst_bin, token[3])
		inst_bin = setRS(inst_bin, token[2])
	case "J":
		inst_bin = setOpCode(inst_bin, "000010")
		inst_bin = setTarget(inst_bin, token[1])
	case "JAL":
		inst_bin = setOpCode(inst_bin, "000011")
		inst_bin = setTarget(inst_bin, token[1])
	case "JR":
		inst_bin = setFunc(inst_bin, "001000")
		inst_bin = setRS(inst_bin, token[1])
	case "JALR":
		inst_bin = setFunc(inst_bin, "001001")
		if len(token) > 1 {
			inst_bin = setRS(inst_bin, token[2])
			inst_bin = setRD(inst_bin, token[1])
		} else {
			inst_bin = setRS(inst_bin, token[1])
			inst_bin = setRD(inst_bin, "31")
		}
	case "HALT":
		inst_bin = "11111111111111111111111111111111"
	default:
		log.Fatalf("Instruccion no reconocida %q", token)

	}

	return inst_bin
}

func setOpCode(inst string, opcode string) string {
	return opcode + inst[6:]
}
func setRS(inst string, rs string) string {
	rs = str2binstr(rs, 5)
	return inst[0:6] + rs + inst[11:]
}
func setRT(inst string, rt string) string {
	rt = str2binstr(rt, 5)
	return inst[0:11] + rt + inst[16:]
}
func setRD(inst string, rd string) string {
	rd = str2binstr(rd, 5)
	return inst[0:16] + rd + inst[21:]
}
func setShamt(inst string, shamt string) string {
	shamt = str2binstr(shamt, 5)
	return inst[0:21] + shamt + inst[26:]
}
func setFunc(inst string, aluFunc string) string {
	return inst[0:26] + aluFunc
}
func setOffsetInmed(inst string, offset string) string {
	offset = str2binstr(offset, 16)
	return inst[0:16] + offset
}
func setTarget(inst string, target string) string {
	target = str2binstr(target, 26)
	return inst[0:6] + target
}

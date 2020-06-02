// Limpia los registros, returna siempre con el R31
        J start
        nop
cleanRegisterFile:
        LUI R00, 0
        LUI R01, 0
        LUI R02, 0
        LUI R03, 0
        LUI R04, 0
        LUI R05, 0
        LUI R06, 0
        LUI R07, 0
        LUI R08, 0
        LUI R09, 0
        LUI R10, 0
        LUI R11, 0
        LUI R12, 0
        LUI R13, 0
        LUI R14, 0
        LUI R15, 0
        LUI R16, 0
        LUI R17, 0
        LUI R18, 0
        LUI R19, 0
        LUI R20, 0
        LUI R21, 0
        LUI R22, 0
        LUI R23, 0
        LUI R24, 0
        LUI R25, 0
        LUI R26, 0
        LUI R27, 0
        LUI R28, 0
        LUI R29, 0
        LUI R30, 0
        JR  R31
        LUI R31, 0
start:
/**************************************
        Clear MEM Data
 Limpio primeros 10 registros de la rom
 y pruebo load stores
 - Tests: Load Stores, offset+base, loads store cortocircuito
**************************************/
        JAL cleanRegisterFile
        nop
        //Limpio primeras 10 pos de la rom
        // Probando distintas config de la instruccion
        SW    R0, 0(R0)    //0
        SW    R0, 4(R0)    //1
        SW    R0, 8(R0)    //2
        SW    R0, 12(R0)   //3
        SW    R0, 16(R0)   //4
        ADDI  R1, R1, 16   // Uso R1 como base
        SW    R0, 4(R1)    //5
        ADDI  R1, R1, 4
        SW    R0, 0(R1)    //6
        ADDI  R1, R1, 4
        SW    R0, 0(R1)    //7
        ADDI  R1, R1, 4
        SW    R0, 0(R1)    //8
        ADDI  R1, R1, 4
        SW    R0, 0(R1)    //9
        ADDI  R1, R1, 4
        SW    R0, 0(R1)    //10
        // Cargo en MEM
        // [Los stores por byte/half word]
        NOR   R2, R2, R2   // Cargo todos 1 en R2
        SB  R2,  4(R0) // 0x00_00_00_FF
        SH  R2,  8(R0) // 0x00_00_FF_FF
        SW  R2, 12(R0) // 0xFF_FF_FF_FF
        LB  R1,  4(R0) // R1 =  0xFF_FF_FF_FF
        LBU R2,  4(R0) // R2 =  0x00_00_00_FF
        LH  R3,  8(R0) // R3 =  0xFF_FF_FF_FF
        LHU R5,  8(R0) // R4 =  0x00_00_FF_FF
        LW  R6,  12(R0) // R5 =  0xFF_FF_FF_FF
        // Load USE
        ADDU R7, R6 , 0
        ADDU R7, R7 , 0


/**************************************
         Fibonnacci
- R3 iteraciones (fibo[13])   10
- R4 counter
- r2 = r1 + r0
- Tests: cortociurcuitos, BNE, Storage, caso->Use-storage, JAL
**************************************/
        JAL cleanRegisterFile
        nop
        ADDI R3,R3, 10
        ADDI R1,R1, 1
fiboFunc:
        ADDU r2, r1, r0
        ADDI r0, r1, 0
        ADDI r1, r2, 0
        ADDI R4, R4, 1
        BNE  R4, R3, fiboFunc
        ADDI R0, R2, 0 // <<-- GUARDO EN R5 EL RESULTADO DE FIBONACCI (PARA EN LA SIM SABER CUANDO CORTA)
        LUI  R1, 0
        SW   R0, 0(R1) // <<-- GUARDO en la posicion cero de la memoria el fibo result
        halt

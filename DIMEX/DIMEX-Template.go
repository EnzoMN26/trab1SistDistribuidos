/*  Construido como parte da disciplina: FPPD - PUCRS - Escola Politecnica
    Professor: Fernando Dotti  (https://fldotti.github.io/)
    Modulo representando Algoritmo de Exclusão Mútua Distribuída:
    Semestre 2023/1
	Aspectos a observar:
	   mapeamento de módulo para estrutura
	   inicializacao
	   semantica de concorrência: cada evento é atômico
	   							  módulo trata 1 por vez
	Q U E S T A O
	   Além de obviamente entender a estrutura ...
	   Implementar o núcleo do algoritmo ja descrito, ou seja, o corpo das
	   funcoes reativas a cada entrada possível:
	   			handleUponReqEntry()  // recebe do nivel de cima (app)
				handleUponReqExit()   // recebe do nivel de cima (app)
				handleUponDeliverRespOk(msgOutro)   // recebe do nivel de baixo
				handleUponDeliverReqEntry(msgOutro) // recebe do nivel de baixo
*/

package DIMEX

import (
	PP2PLink "SD/PP2PLink"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ------------------------------------------------------------------------------------
// ------- principais tipos
// ------------------------------------------------------------------------------------

type State int // enumeracao dos estados possiveis de um processo
const (
	noMX State = iota
	wantMX
	inMX
)

type dmxReq int // enumeracao dos estados possiveis de um processo
const (
	ENTER dmxReq = iota
	EXIT
	SNAPSHOT
)

type dmxResp struct { // mensagem do módulo DIMEX infrmando que pode acessar - pode ser somente um sinal (vazio)
	// mensagem para aplicacao indicando que pode prosseguir
}

type DIMEX_Module struct {
	Req       chan dmxReq  // canal para receber pedidos da aplicacao (REQ e EXIT)
	Ind       chan dmxResp // canal para informar aplicacao que pode acessar
	addresses []string     // endereco de todos, na mesma ordem
	id        int          // identificador do processo - é o indice no array de enderecos acima
	st        State        // estado deste processo na exclusao mutua distribuida
	waiting   []bool       // processos aguardando tem flag true
	lcl       int          // relogio logico local
	reqTs     int          // timestamp local da ultima requisicao deste processo
	nbrResps  int
	dbg       bool
	snapshotCount int
	currentSnapshot int
	channels  []bool
	messageChannel map[int]string
	idByIp 		map[string]int

	Pp2plink *PP2PLink.PP2PLink // acesso aa comunicacao enviar por PP2PLinq.Req  e receber por PP2PLinq.Ind
}

// ------------------------------------------------------------------------------------
// ------- inicializacao
// ------------------------------------------------------------------------------------

func NewDIMEX(_addresses []string, _id int, _dbg bool) *DIMEX_Module {

	p2p := PP2PLink.NewPP2PLink(_addresses[_id], _dbg)

	dmx := &DIMEX_Module{
		Req: make(chan dmxReq, 1),
		Ind: make(chan dmxResp, 1),

		addresses: _addresses,
		id:        _id,
		st:        noMX,
		waiting:   make([]bool, len(_addresses)),
		lcl:       0,
		reqTs:     0,
		dbg:       _dbg,
		snapshotCount: 1,
		currentSnapshot:  0,
		channels: make([]bool, len(_addresses)),
		messageChannel: make(map[int]string),
		idByIp: make(map[string]int),
		Pp2plink: p2p}

	for index := range dmx.channels {
		dmx.channels[index] = false
	}

	for index := range _addresses {
		dmx.idByIp[_addresses[index]] = index
	}

	for i := 0; i < len(dmx.waiting); i++ {
		dmx.waiting[i] = false
	}
	dmx.Start()
	dmx.outDbg("Init DIMEX!")
	return dmx
}

// ------------------------------------------------------------------------------------
// ------- nucleo do funcionamento
// ------------------------------------------------------------------------------------

func (module *DIMEX_Module) Start() {

	go func() {
		for {
			select {
			case dmxR := <-module.Req: // vindo da  aplicação
				if dmxR == ENTER {
					module.outDbg("app pede mx")
					module.handleUponReqEntry() // ENTRADA DO ALGORITMO

				} else if dmxR == EXIT {
					module.outDbg("app libera mx")
					module.handleUponReqExit() // ENTRADA DO ALGORITMO
				} else if dmxR == SNAPSHOT {
					module.outDbg("app solicita snapshot")
					module.createSnapshot()
				}

			case msgOutro := <-module.Pp2plink.Ind: // vindo de outro processo
				//fmt.Printf("dimex recebe da rede: ", msgOutro)
				module.saveMessageInChannel(msgOutro)
				if strings.Contains(msgOutro.Message, "respOk") {
					module.outDbg("         <<<---- responde! " + msgOutro.Message)
					module.handleUponDeliverRespOk() // ENTRADA DO ALGORITMO

				} else if strings.Contains(msgOutro.Message, "reqEntry") {
					module.outDbg("          <<<---- pede??  " + msgOutro.Message)
					module.handleUponDeliverReqEntry(msgOutro) // ENTRADA DO ALGORITMO

				} else if strings.Contains(msgOutro.Message, "take snapshot"){
					module.outDbg("snapshot pedido " + msgOutro.Message)
					module.replySnapshot(msgOutro);
				}
			}
		}
	}()
}

// ------------------------------------------------------------------------------------
// ------- tratamento de pedidos vindos da aplicacao
// ------- UPON ENTRY
// ------- UPON EXIT
// ------------------------------------------------------------------------------------

func (module *DIMEX_Module) handleUponReqEntry() {
	module.lcl = module.lcl + 1
	module.reqTs = module.lcl
	module.nbrResps = 0
	for index, endereco := range module.addresses {
		if(index != module.id && index != 0){
			module.sendToLink(endereco, "reqEntry " + strconv.Itoa(module.id) + " " + strconv.Itoa(module.reqTs), " ")
		}
	}
	module.st = wantMX
}

func (module *DIMEX_Module) handleUponReqExit() {
	for index, esperando := range module.waiting{
		if(esperando){
			module.sendToLink(module.addresses[index], "respOk", " ")
		}
	}
	module.st = noMX
	module.waiting = make([]bool, len(module.addresses))
}

// ------------------------------------------------------------------------------------
// ------- tratamento de mensagens de outros processos
// ------- UPON respOK
// ------- UPON reqEntry
// ------------------------------------------------------------------------------------

func (module *DIMEX_Module) handleUponDeliverRespOk(){
	module.nbrResps = module.nbrResps + 1
	if(module.nbrResps == len(module.addresses) - 2){
		module.Ind <- dmxResp{}
		module.st = inMX
	}
/*
upon event [ pl, Deliver | p, [ respOk, r ] ]
resps++
se resps = N
então trigger [ dmx, Deliver | free2Access ]
estado := estouNaSC

*/
}
func (module *DIMEX_Module) handleUponDeliverReqEntry(msgOutro PP2PLink.PP2PLink_Ind_Message) {
	// outro processo quer entrar na SC
	/*
						upon event [ pl, Deliver | p, [ reqEntry, r, rts ]  do
		     				se (estado == naoQueroSC)   OR
		        				 (estado == QueroSC AND  myTs >  ts)
							então  trigger [ pl, Send | p , [ respOk, r ]  ]
		 					senão
		        				então  postergados := postergados + [p, r ]
		     					lts.ts := max(lts.ts, rts.ts)
	*/
	reqIdReqTs := strings.Split(msgOutro.Message, " ")
	othId, _ := strconv.Atoi(reqIdReqTs[1]) 
	othReqTs, _ := strconv.Atoi(reqIdReqTs[2]) 

	if (module.st == noMX) || (module.st == wantMX && before(othId, othReqTs, module.id, module.reqTs)) {
		module.sendToLink(module.addresses[othId], "respOk", " ")
	} else {
		module.waiting[othId] = true;
	}
	module.lcl = max(module.lcl, othReqTs) //EXLUSAO MUTUA
}

func(module *DIMEX_Module) createSnapshot(){
	//somente id 0
	for index := range module.addresses{
		if index != 0{
			module.sendToLink(module.addresses[index], strconv.Itoa(module.id) +" take snapshot "+ strconv.Itoa(module.snapshotCount), " ")
		}
	}
	module.channels[module.id] = true
	module.snapshotCount = module.snapshotCount+1;
}

func(module *DIMEX_Module) replySnapshot(msgOutro PP2PLink.PP2PLink_Ind_Message){
	idSnapshot,_ := strconv.Atoi(strings.Split(msgOutro.Message, " ")[3])
	idFrom,_ := strconv.Atoi(strings.Split(msgOutro.Message, " ")[0])
	if idSnapshot == module.currentSnapshot && module.currentSnapshot != 0 {
		module.channels[idFrom] = true
		module.recordChannel(idFrom)
		count := 0
		for index := range module.channels{
			if !module.channels[index] {
				break;
			}
			count++
		}
		channelIstrue := (count == len(module.channels));
		if channelIstrue{
			module.channels = make([]bool, len(module.addresses))
			module.currentSnapshot = 0
		}
	} else if module.currentSnapshot == 0 {
		module.currentSnapshot = idSnapshot
		module.channels[module.id] = true
		module.recordLocalState(idFrom, idSnapshot)
		for index := range module.addresses{
			if index != module.id {
				module.sendToLink(module.addresses[index], strconv.Itoa(module.id) +" take snapshot "+ strconv.Itoa(idSnapshot), " ")
			}
		}
	} else {
		module.sendToLink(module.addresses[idFrom], strconv.Itoa(module.id) +" take snapshot "+ strconv.Itoa(idSnapshot), " ")
		module.channels[module.id] = true
	}
	//estado = st,waiting,lcl,reqTs, mensagens
	//grava estado local
	//estado do canal entre o receptor e o remetente é setado vazio
	//envia mensagem "take snapshot" em OUT
}

// ------------------------------------------------------------------------------------
// ------- funcoes de ajuda
// ------------------------------------------------------------------------------------

func (module *DIMEX_Module) sendToLink(address string, content string, space string) {
	module.outDbg(space + " ---->>>>   to: " + address + "     msg: " + content)
	module.Pp2plink.Req <- PP2PLink.PP2PLink_Req_Message{
		To:      address,
		Message: content}
}

func before(oneId, oneTs, othId, othTs int) bool {
	if oneTs < othTs {
		return true
	} else if oneTs > othTs {
		return false
	} else {
		return oneId < othId
	}
}

func (module *DIMEX_Module) outDbg(s string) {
	if module.dbg {
		fmt.Println(". . . . . . . . . . . . [ DIMEX : " + s + " ]")
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (module *DIMEX_Module) recordChannel(idFrom int) {
	fileName := "./p"+ strconv.Itoa(module.id) +".txt"
	file, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	messageChannelStr := fmt.Sprintf("p%d -> p%d: ", idFrom, module.id)
	_, err = file.WriteString(messageChannelStr)
	if err != nil {
        fmt.Println("Erro ao gravar no arquivo:", err)
    }
	messageChannelStr = fmt.Sprintf("%s\n", module.messageChannel[idFrom]) 
	_, err = file.WriteString(messageChannelStr)
    if err != nil {
        fmt.Println("Erro ao gravar no arquivo:", err)
    }
	defer file.Close() // Ensure the file is closed at the end of the function
	module.deleteMessagesInChannel()
}

func (module *DIMEX_Module) recordLocalState(idFrom int, idSnapshot int) {
	fileName := "./p"+ strconv.Itoa(module.id) +".txt"
	file, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	localState := fmt.Sprintf("INICIO SNAPSHOT %d\n", idSnapshot)
	localState += fmt.Sprintf("LocalState: p%d\n", module.id)
    localState += fmt.Sprintf("st: %v\n", module.st)
    localState += fmt.Sprintf("waiting: %v\n", module.waiting)
    localState += fmt.Sprintf("lcl: %v\n", module.lcl)
    localState += fmt.Sprintf("p%d -> p%d: ", idFrom, module.id)
	localState += fmt.Sprintf("%s\n", module.messageChannel[idFrom]) 

	_, err = file.WriteString(localState)
    if err != nil {
        fmt.Println("Erro ao gravar no arquivo:", err)
    }
	defer file.Close() // Ensure the file is closed at the end of the function
	fmt.Println("Estado local gravado com sucesso no arquivo:", fileName)
	module.deleteMessagesInChannel()
}

func (module *DIMEX_Module) saveMessageInChannel(msgOutro PP2PLink.PP2PLink_Ind_Message) {
	idFrom := module.idByIp[msgOutro.From]
	value, exists := module.messageChannel[idFrom]
	if exists{
		module.messageChannel[idFrom] = value + "\n" + msgOutro.Message
	} else{
		module.messageChannel[idFrom] = msgOutro.Message
	}
}

func (module *DIMEX_Module) deleteMessagesInChannel(){
	module.messageChannel = make(map[int]string)
}

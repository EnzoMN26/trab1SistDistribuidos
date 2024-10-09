import json
import os

def verificaInvariante1(dadosPorProcesso):
    inMXProcesses = []
    for pdata in dadosPorProcesso.values():
        if pdata.get('state') == 2:
            inMXProcesses.append(pdata)
   
    if len(inMXProcesses) > 1:
        print(f"Violação de Invariante 1: {len(inMXProcesses)} processos estão na zona crítica.")
        return False
    
    return True

def verificaInvariante2(dadosPorProcesso):
    allNoMX = all(pdata['state'] == 0 for pdata in dadosPorProcesso.values())
    
    if not allNoMX:
        return True
    
    for pdata in dadosPorProcesso.values():
        messages = pdata.get('messages')
        waitingList = pdata.get('waiting')
        
        if any(waitingList):
            print(f"Violação de Invariante 2: Processos estão esperando enquanto todos processos estão em noMX.")
            return False
        if any(messages):
            for idx, message in enumerate(messages):
                if message != f"p{idx}: ":
                    print(f"Violação de Invariante 2: Mensagens estão em trânsito enquanto todos processos estão em noMX.")
                    return False
    return True

def verificaInvariante3(dadosPorProcesso):
    for pid, pdata in dadosPorProcesso.items():
        for idx, isWaiting in enumerate(pdata['waiting']):
            if isWaiting:
                if(pdata['state'] == 0):
                    print(f"Violação de Invariante 3: Processo {idx} está esperando o processo {pid}, porém o processo {pid} não está querendo e não está na ZC.")
                    return False
    return True

def verificaInvariante(snapshots, numProcesses):
    for snapshotId, dadosPorProcesso in snapshots.items():
        print(f"\nChecando snapshot ID: {snapshotId}")
        
        # Check if we have data from all processes
        if len(dadosPorProcesso) < numProcesses:
            print(f"Dados Incompletos: experava-se {numProcesses} processos, {len(dadosPorProcesso)} recebidos")
            continue
        
        invariante1 = verificaInvariante1(dadosPorProcesso)
        invariante2 = verificaInvariante2(dadosPorProcesso)
        invariante3 = verificaInvariante3(dadosPorProcesso)
        if not all([invariante1, invariante2, invariante3]):
            print("Invariante violada.")
        else:
            print("Snapshot Correto!")


def main():
    numProcesses = 4
    snapshots = {}
    for processId in range(numProcesses):
        
        filename = f'p{processId}.txt'
        if not os.path.exists(filename):
            print(f"Arquivo de Snapshot {filename} não existe")
            continue
        
        with open(filename, 'r') as file:
            for line in file:
                line = line.strip()
                if not line:
                    continue
                
                try:
                    snapshotData = json.loads(line)
                except json.JSONDecodeError as e:
                    print(f"Error decoding JSON in file {filename}: {e}")
                    continue

                snapshotId = snapshotData.get('snapshotId')
                if snapshotId is None:
                    print(f"No 'snapshotId' in data from file {filename}.")
                    continue
                snapshots.setdefault(snapshotId, {})[processId] = snapshotData
    verificaInvariante(snapshots, numProcesses)

if __name__ == "__main__":
    main()
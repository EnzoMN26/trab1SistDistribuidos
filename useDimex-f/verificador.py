import json
import os

def read_snapshots(num_processes):
    snapshots = {}
    for process_id in range(num_processes):
        
        filename = f'p{process_id}.txt'
        if not os.path.exists(filename):
            print(f"Arquivo de Snapshot {filename} não existe")
            continue
        
        with open(filename, 'r') as file:
            for line in file:
                line = line.strip()
                if not line:
                    continue
                
                try:
                    snapshot_data = json.loads(line)
                except json.JSONDecodeError as e:
                    print(f"Error decoding JSON in file {filename}: {e}")
                    continue

                snapshot_id = snapshot_data.get('snapshotId')
                if snapshot_id is None:
                    print(f"No 'IdSnapshot' in data from file {filename}.")
                    continue
                snapshots.setdefault(snapshot_id, {})[process_id] = snapshot_data

    return snapshots

def evaluate_invariant_1(data_per_process):
    in_mx_processes = []
    for pdata in data_per_process.values():
        # print(f"pdata: {pdata}\n")
        # print(f"pdata.get(state): {pdata.get('state')}")
        # print(f"pdata.get(state) booleanzin: {pdata.get('state') == 2}")
        if pdata.get('state') == 2:
            in_mx_processes.append(pdata)
   
    if len(in_mx_processes) > 1:
        print(f"Violação de Invariante 1: {len(in_mx_processes)} processos estão na zona crítica.")
        return False
    return True

def evaluate_invariant_2(data_per_process):
    all_no_mx = all(pdata['state'] == 0 for pdata in data_per_process.values())
    
    if not all_no_mx:
        return True
    
    for pdata in data_per_process.values():
        messages = pdata.get('messages')
        waiting_list = pdata.get('waiting')
        
        if any(waiting_list):
            print(f"Violação de Invariante 2: Processos estão esperando enquanto todos processos estão em noMX.")
            return False
        if any(messages):
            for idx, message in enumerate(messages):
                if message != f"p{idx}: ":
                    print(f"Violação de Invariante 2: Mensagens estão em trânsito enquanto todos processos estão em noMX.")
                    return False
                    

    return True

def evaluate_invariant_3(data_per_process):
    for pid, pdata in data_per_process.items():
        for idx, is_waiting in enumerate(pdata['waiting']):
            if is_waiting:
                if(pdata['state'] == 0):
                    print(f"Violação de Invariante 3: Processo {pid} está esperando o processo {idx}, porém o processo {pid} não está querendo a ZC.")
                    return False
    return True
    
def evaluate_invariant_4(data_per_process):
    return

def evaluate_invariants(snapshots, num_processes):
    for snapshot_id, data_per_process in snapshots.items():
        print(f"\nEvaluating Snapshot ID: {snapshot_id}")
        
        # Check if we have data from all processes
        if len(data_per_process) < num_processes:
            print(f"  Incomplete data: expected {num_processes} processes, got {len(data_per_process)}")
            continue
        
        invariant1_ok = evaluate_invariant_1(data_per_process)
        invariant2_ok = evaluate_invariant_2(data_per_process)
        invariant3_ok = evaluate_invariant_3(data_per_process)
        if not all([invariant1_ok, invariant2_ok, invariant3_ok]):
            print("Some invariants violated.")


def main():
    num_processes = 4
    snapshots = read_snapshots(num_processes)
    evaluate_invariants(snapshots, num_processes)

if __name__ == "__main__":
    main()
import scipy
from scipy import stats

file = "./distributed_5_node.txt"

f = open(file, "r")
data = f.read()
f.close()

data = [[a.strip().replace(" ns/op", "") for a in x.split("\t")] for x in data.split("\n")][3:-2]

times = {}
for i, line in enumerate(data):
    data[i][2] = int(line[2])/10**9
    data[i].pop(1)
    data[i][0] = data[i][0].replace("BenchmarkServer/512x512x100-8", "").replace("-10", "")
    if data[i][0] not in times:
        times[data[i][0]] = []
    times[data[i][0]].append(data[i][1])

for k, v in times.items():
    average = round(scipy.mean(v), 2)
    error = round(scipy.stats.sem(v), 3)
    print(k, average, error)

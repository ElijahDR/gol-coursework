import scipy
from scipy import stats

times = {}
for i in range(1, 9):
    file = "./distributed_" + str(i) + "_node.txt"

    f = open(file, "r")
    data = f.read()
    f.close()

    data = [[a.strip().replace(" ns/op", "") for a in x.split("\t")] for x in data.split("\n")][3:-2]

    for j, line in enumerate(data):
        data[j][2] = int(line[2])/10**9
        print(line)
        data[j].pop(1)
        data[j].pop(0)
        if i not in times:
            times[i] = []
        times[i].append(data[j][0])

    print(times)

for k, v in times.items():
    average = round(scipy.mean(v), 2)
    error = round(scipy.stats.sem(v), 3)
    print(k, average, error)

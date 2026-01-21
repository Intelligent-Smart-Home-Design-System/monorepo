package actors

// Пожар, протечка и тд

// ЗАГОТОВКА:
//const (
//	N = 5
//	M = 5
//	FireDistributionTime = 1.0
//)
//
//type Cell struct {
//	x, y  int
//	burnt bool
//}
//
//func fireProcess(proc simgo.Process, grid [][]*Cell, cell *Cell) {
//	if cell.burnt {
//		return
//	}
//
//	cell.burnt = true
//	fmt.Printf("Time %.1f: fire at (%d, %d)\n", proc.Now(), cell.x, cell.y)
//
//	proc.Wait(proc.Timeout(FireDistributionTime))
//
//	directions := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
//
//	for _, d := range directions {
//		newx, newy := cell.x + d[0], cell.y + d[1]
//		if newx >= 0 && newx < N && newy >= 0 && newy < M {
//			neighbor := grid[newx][newy]
//			if !neighbor.burnt {
//				proc.Process(func(p simgo.Process) {
//					fireProcess(p, grid, neighbor)
//				})
//			}
//		}
//	}
//}
//
//func main() {
//	sim := simgo.NewSimulation()
//
//	grid := make([][]*Cell, N)
//	for i := 0; i < N; i++ {
//		grid[i] = make([]*Cell, M)
//		for j := 0; j < M; j++ {
//			grid[i][j] = &Cell{x: i, y: j}
//		}
//	}
//
//	start := grid[2][2]
//	sim.Process(func(proc simgo.Process) {
//		fireProcess(proc, grid, start)
//	})
//
//	sim.Run()
//}

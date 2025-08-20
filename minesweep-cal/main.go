package main

import (
	"fmt"
	"math"
	"time"
)

const (
	// UNOPENED 代表未打开的格子
	UNOPENED = -1
)

// Point 表示一个格子的坐标
type Point struct {
	R, C int
}

// 8个方向的邻居坐标偏移
var neighbors = [8][2]int{{-1, -1}, {-1, 0}, {-1, 1}, {0, -1}, {0, 1}, {1, -1}, {1, 0}, {1, 1}}

// validateBoard 检查棋盘格式是否有效
func validateBoard(board [][]int) bool {
	if len(board) == 0 || len(board[0]) == 0 {
		fmt.Println("错误: 棋盘不能为空")
		return false
	}

	rows := len(board)
	cols := len(board[0])

	for r, row := range board {
		// 检查每行长度是否一致
		if len(row) != cols {
			fmt.Printf("错误: 棋盘行长度不一致 (行 %d 长度为 %d, 应为 %d)\n", r, len(row), cols)
			return false
		}
		for c, val := range row {
			// 检查数值范围
			if val < UNOPENED || val > 8 {
				fmt.Printf("错误: 无效的格子值 %d at (%d, %d)\n", val, r, c)
				return false
			}

			// 检查数字线索是否符合逻辑
			if val >= 0 {
				neighborCount := 0
				for _, offset := range neighbors {
					nr, nc := r+offset[0], c+offset[1]
					if nr >= 0 && nr < rows && nc >= 0 && nc < cols {
						neighborCount++
					}
				}
				if val > neighborCount {
					fmt.Printf("逻辑错误: (%d, %d) 值为 %d, 但只有 %d 个邻居\n", r, c, val, neighborCount)
					return false
				}
			}
		}
	}
	return true
}

// Constraint 定义了一个围绕数字线索的约束
type Constraint struct {
	cluePos   Point // 线索格位置
	neighbors []int // 周围未打开格子的索引 (指向 unknowns)
	count     int   // 线索数
}

// Solver 是解决扫雷问题的主结构
type Solver struct {
	board            [][]int
	rows, cols       int
	unknowns         []Point       // 所有与线索相邻的未打开格子
	unknownMap       map[Point]int // 坐标到 unknowns 索引的映射
	constraints      []Constraint  // 所有约束
	isolatedUnknowns int           // 与任何线索都不相邻的孤立未打开格子数量
	minMines         int           // 相关未知区域的最小雷数
	maxMines         int           // 相关未知区域的最大雷数
	foundSolution    bool          // 是否至少找到一个解
	isPossible       bool          // 棋盘是否存在有效解
}

// newSolver 创建并初始化一个新的 solver
func newSolver(board [][]int) *Solver {
	rows := len(board)
	cols := len(board[0])

	// 创建棋盘副本
	boardCopy := make([][]int, rows)
	for i := range boardCopy {
		boardCopy[i] = make([]int, cols)
		copy(boardCopy[i], board[i])
	}

	s := &Solver{
		board:      boardCopy,
		rows:       rows,
		cols:       cols,
		unknownMap: make(map[Point]int),
		minMines:   math.MaxInt32,
		maxMines:   -1,
		isPossible: true, // 初始假定有解
	}
	s.prepare()
	return s
}

// prepare 预处理棋盘，找出未知格、约束和孤立格
func (s *Solver) prepare() {
	isRelevantUnknown := make(map[Point]bool)
	totalUnknowns := 0
	var clues []Point

	for r := 0; r < s.rows; r++ {
		for c := 0; c < s.cols; c++ {
			if s.board[r][c] >= 0 {
				clues = append(clues, Point{r, c})
			} else {
				totalUnknowns++
			}
		}
	}

	// 找出所有与数字线索相邻的未打开格子
	for _, cluePos := range clues {
		for _, offset := range neighbors {
			nr, nc := cluePos.R+offset[0], cluePos.C+offset[1]
			if s.inBounds(nr, nc) && s.board[nr][nc] == UNOPENED {
				p := Point{nr, nc}
				if !isRelevantUnknown[p] {
					isRelevantUnknown[p] = true
					s.unknownMap[p] = len(s.unknowns)
					s.unknowns = append(s.unknowns, p)
				}
			}
		}
	}

	s.buildConstraints(clues)
	if !s.isPossible {
		return
	}

	s.isolatedUnknowns = totalUnknowns - len(s.unknowns)
}

// buildConstraints 根据线索格构建约束条件
func (s *Solver) buildConstraints(clues []Point) {
	for _, cluePos := range clues {
		var neighborIndices []int
		for _, offset := range neighbors {
			nr, nc := cluePos.R+offset[0], cluePos.C+offset[1]
			p := Point{nr, nc}
			if idx, exists := s.unknownMap[p]; exists {
				neighborIndices = append(neighborIndices, idx)
			}
		}

		clueValue := s.board[cluePos.R][cluePos.C]
		if len(neighborIndices) == 0 {
			// 如果一个数字线索周围没有未打开的格子，那么它的值必须是0
			if clueValue > 0 {
				fmt.Printf("逻辑错误: (%d, %d) 值为 %d 但没有未打开的邻居\n", cluePos.R, cluePos.C, clueValue)
				s.isPossible = false
				return
			}
		} else {
			s.constraints = append(s.constraints, Constraint{
				cluePos:   cluePos,
				neighbors: neighborIndices,
				count:     clueValue,
			})
		}
	}
}

// inBounds 检查坐标是否在棋盘范围内
func (s *Solver) inBounds(r, c int) bool {
	return r >= 0 && r < s.rows && c >= 0 && c < s.cols
}

// run 通过分治法解决问题
func (s *Solver) run() {
	if !s.isPossible {
		return
	}
	if len(s.unknowns) == 0 {
		s.minMines = 0
		s.maxMines = 0
		s.foundSolution = true
		return
	}

	// 将问题分解为多个不相关的连通分量
	components := s.findComponents()

	totalMinMines := 0
	totalMaxMines := 0
	s.foundSolution = true // 乐观地假设有解

	for _, componentIndices := range components {
		subMin, subMax, subFound := s.solveComponent(componentIndices)
		if !subFound {
			s.foundSolution = false // 任何一个子问题无解，则整个问题无解
			return
		}
		totalMinMines += subMin
		totalMaxMines += subMax
	}

	s.minMines = totalMinMines
	s.maxMines = totalMaxMines
}

// findComponents 使用BFS找到所有关联的未知格组成的连通分量
func (s *Solver) findComponents() [][]int {
	if len(s.unknowns) == 0 {
		return nil
	}

	// 构建邻接表
	adj := make([][]int, len(s.unknowns))
	for _, c := range s.constraints {
		for i := 0; i < len(c.neighbors); i++ {
			for j := i + 1; j < len(c.neighbors); j++ {
				u, v := c.neighbors[i], c.neighbors[j]
				adj[u] = append(adj[u], v)
				adj[v] = append(adj[v], u)
			}
		}
	}

	var components [][]int
	visited := make([]bool, len(s.unknowns))
	for i := range s.unknowns {
		if !visited[i] {
			var currentComponent []int
			q := []int{i}
			visited[i] = true
			head := 0
			for head < len(q) {
				u := q[head]
				head++
				currentComponent = append(currentComponent, u)
				for _, v := range adj[u] {
					if !visited[v] {
						visited[v] = true
						q = append(q, v)
					}
				}
			}
			components = append(components, currentComponent)
		}
	}
	return components
}

// solveComponent 解决单个连通分量子问题
func (s *Solver) solveComponent(componentIndices []int) (min, max int, found bool) {
	if len(componentIndices) == 0 {
		return 0, 0, true
	}

	// 为子问题创建一个新的、更小的solver
	subSolver := &Solver{
		unknownMap: make(map[Point]int),
		minMines:   math.MaxInt32,
		maxMines:   -1,
		isPossible: true,
	}

	originalIndexToSubIndex := make(map[int]int)
	for i, originalIndex := range componentIndices {
		point := s.unknowns[originalIndex]
		subSolver.unknowns = append(subSolver.unknowns, point)
		subSolver.unknownMap[point] = i
		originalIndexToSubIndex[originalIndex] = i
	}

	// 筛选出与当前子问题相关的约束
	for _, constraint := range s.constraints {
		var subNeighbors []int
		isRelevant := false
		for _, neighborIndex := range constraint.neighbors {
			if subIndex, ok := originalIndexToSubIndex[neighborIndex]; ok {
				subNeighbors = append(subNeighbors, subIndex)
				isRelevant = true
			}
		}
		if isRelevant {
			subSolver.constraints = append(subSolver.constraints, Constraint{
				cluePos:   constraint.cluePos,
				neighbors: subNeighbors,
				count:     constraint.count,
			})
		}
	}

	assignment := make([]int, len(subSolver.unknowns))
	subSolver.backtrack(0, assignment)

	return subSolver.minMines, subSolver.maxMines, subSolver.foundSolution
}

// backtrack 核心回溯函数，尝试所有可能的地雷布局
func (s *Solver) backtrack(k int, assignment []int) {
	// 剪枝：检查当前的部分布局是否已经违反了约束
	if !s.isPartialAssignmentValid(k, assignment) {
		return
	}

	// 如果所有未知格子都已分配，检查是否为一个完整解
	if k == len(s.unknowns) {
		if s.isFullAssignmentValid(assignment) {
			currentMines := 0
			for _, val := range assignment {
				currentMines += val
			}

			if !s.foundSolution {
				s.minMines = currentMines
				s.maxMines = currentMines
				s.foundSolution = true
			} else {
				if currentMines < s.minMines {
					s.minMines = currentMines
				}
				if currentMines > s.maxMines {
					s.maxMines = currentMines
				}
			}
		}
		return
	}

	// 尝试不放雷
	assignment[k] = 0
	s.backtrack(k+1, assignment)

	// 尝试放雷
	assignment[k] = 1
	s.backtrack(k+1, assignment)
}

// isPartialAssignmentValid 剪枝检查
func (s *Solver) isPartialAssignmentValid(currentIndex int, assignment []int) bool {
	for _, constraint := range s.constraints {
		mineCount := 0
		unknownCount := 0

		for _, neighborIndex := range constraint.neighbors {
			if neighborIndex < currentIndex {
				mineCount += assignment[neighborIndex]
			} else {
				unknownCount++
			}
		}

		// 剪枝条件：
		// 1. 当前已确定的雷数 > 约束数
		// 2. 当前已确定的雷数 + 剩下所有未定格子都当成雷 < 约束数
		if mineCount > constraint.count || mineCount+unknownCount < constraint.count {
			return false
		}
	}
	return true
}

// isFullAssignmentValid 检查一个完整的布局是否满足所有约束
func (s *Solver) isFullAssignmentValid(assignment []int) bool {
	for _, constraint := range s.constraints {
		mineCount := 0
		for _, neighbor := range constraint.neighbors {
			mineCount += assignment[neighbor]
		}
		if mineCount != constraint.count {
			return false
		}
	}
	return true
}

// calculate 是对外暴露的主函数，负责整个计算流程
// 返回值: (最大雷数, 最小雷数)
func calculate(minesweeper [][]int) (int, int) {
	if !validateBoard(minesweeper) {
		return -1, -1
	}

	solver := newSolver(minesweeper)
	solver.run()

	if !solver.foundSolution {
		// 存在逻辑矛盾，无解
		return -1, -1
	}

	// 最终雷数是“相关区域雷数”和“孤立区域雷数”之和
	// 孤立区域的雷可以是0到其格子总数之间的任意值
	finalMinMines := solver.minMines
	finalMaxMines := solver.maxMines + solver.isolatedUnknowns

	return finalMaxMines, finalMinMines
}

func main() {
	minesweeper := [][]int{
		{-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1},
		{-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1},
		{-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1},
		{-1, -1, -1, -1, -1, -1, 4, -1, -1, -1, -1, -1, -1},
		{-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1},
		{-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, 4, -1},
		{-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1},
		{-1, -1, 3, 2, -1, -1, -1, -1, -1, -1, -1, -1, -1},
		{-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1},
		{-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1},
		{-1, -1, -1, -1, -1, -1, -1, -1, 4, -1, -1, -1, -1},
		{-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1},
		{-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1},
	}

	start := time.Now()
	maxMinesNum, minMinesNum := calculate(minesweeper)
	end := time.Now()

	fmt.Printf("maxMinesNum: %d\n", maxMinesNum)
	fmt.Printf("minMinesNum: %d\n", minMinesNum)
	fmt.Printf("useTime: %v\n", end.Sub(start))
}

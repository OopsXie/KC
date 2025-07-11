package main

import (
	"fmt"
	"math"
	"time"
)

const (
	UNOPENED = -1 // -1 代表未打开的格子
)

// Point 结构体用于表示格子的坐标
type Point struct {
	R, C int
}

// 8个方向的邻居坐标偏移
var neighbors = [8][2]int{{-1, -1}, {-1, 0}, {-1, 1}, {0, -1}, {0, 1}, {1, -1}, {1, 0}, {1, 1}}

// isValidBoard 检查输入的棋盘格式是否有效
func isValidBoard(board [][]int) bool {
	if len(board) == 0 || len(board[0]) == 0 {
		fmt.Println("输入错误: 棋盘不能为空。")
		return false
	}

	rows := len(board)
	cols := len(board[0])

	for r := 0; r < rows; r++ {
		if len(board[r]) != cols {
			fmt.Printf("输入格式错误: 棋盘的行长度不一致 (第 %d 行的长度为 %d, 期望为 %d)\n", r, len(board[r]), cols)
			return false
		}
		for c := 0; c < cols; c++ {
			val := board[r][c]
			if val < UNOPENED || val > 8 {
				fmt.Printf("输入格式错误: 在 (%d, %d) 处的格子值无效: %d (有效范围是 -1 到 8)\n", r, c, val)
				return false
			}
		}
	}
	return true
}

// Constraint 结构体定义了一个约束条件
type Constraint struct {
	cluePos   Point // 线索格子的位置
	neighbors []int // 周围未打开格子的索引 (指向 unknowns 数组)
	count     int   // 线索数字
}

// minesweeperSolver 是解决扫雷问题的主要结构体
type minesweeperSolver struct {
	board            [][]int
	rows, cols       int
	unknowns         []Point       // 所有与线索相邻的未打开格子
	unknownMap       map[Point]int // 从 Point 到 unknowns 索引的映射，用于快速查找
	constraints      []Constraint  // 所有的约束条件
	isolatedUnknowns int           // 与任何线索都不相邻的孤立未打开格子数量
	minMines         int           // 记录相关未知区域的最小雷数
	maxMines         int           // 记录相关未知区域的最大雷数
	foundSolution    bool          // 是否至少找到一个有效解
	isPossible       bool          // 棋盘是否存在有效解
}

// newSolver 创建并初始化一个新的 solver
func newSolver(board [][]int) *minesweeperSolver {
	rows := len(board)
	cols := len(board[0])

	boardCopy := make([][]int, rows)
	for i := range boardCopy {
		boardCopy[i] = make([]int, cols)
		copy(boardCopy[i], board[i])
	}

	s := &minesweeperSolver{
		board:      boardCopy,
		rows:       rows,
		cols:       cols,
		unknownMap: make(map[Point]int),
		minMines:   math.MaxInt32,
		maxMines:   -1,
		isPossible: true, // 初始假定棋盘有解
	}
	s.prepare()
	return s
}

// prepare 预处理棋盘，找出所有未知格子、约束和孤立格子
func (s *minesweeperSolver) prepare() {
	isRelevantUnknown := make(map[Point]bool)
	totalUnknowns := 0
	var clues []Point

	// 步骤 1: 找出所有线索格(数字0-8)和未打开的格(-1)
	for r := 0; r < s.rows; r++ {
		for c := 0; c < s.cols; c++ {
			if s.board[r][c] >= 0 {
				clues = append(clues, Point{r, c})
			} else { // s.board[r][c] == UNOPENED
				totalUnknowns++
			}
		}
	}

	// 步骤 2: 找出所有与线索相邻的"相关"未知格子
	for _, cluePos := range clues {
		for _, offset := range neighbors {
			nr, nc := cluePos.R+offset[0], cluePos.C+offset[1]
			if s.isValidCoordinate(nr, nc) && s.board[nr][nc] == UNOPENED {
				p := Point{nr, nc}
				if !isRelevantUnknown[p] {
					isRelevantUnknown[p] = true
					s.unknownMap[p] = len(s.unknowns)
					s.unknowns = append(s.unknowns, p)
				}
			}
		}
	}

	// 步骤 3: 构建约束，并进行早期合法性检查
	s.buildConstraints(clues)
	if !s.isPossible {
		return // 如果在构建约束时发现矛盾，则棋盘无解
	}

	// 步骤 4: 计算与任何线索都不相邻的"孤立"未知格子数量
	s.isolatedUnknowns = totalUnknowns - len(s.unknowns)
}

// buildConstraints 根据线索格构建约束条件
func (s *minesweeperSolver) buildConstraints(clues []Point) {
	for _, cluePos := range clues {
		var neighborIndices []int
		for _, offset := range neighbors {
			nr, nc := cluePos.R+offset[0], cluePos.C+offset[1]
			if s.isValidCoordinate(nr, nc) && s.board[nr][nc] == UNOPENED {
				p := Point{nr, nc}
				if idx, exists := s.unknownMap[p]; exists {
					neighborIndices = append(neighborIndices, idx)
				}
			}
		}

		// 【重要修正】如果一个线索的所有邻居都已打开，检查其数字是否为0
		// 因为输入中没有预设的雷，所以这种情况下线索数字必须是0
		if len(neighborIndices) == 0 {
			if s.board[cluePos.R][cluePos.C] > 0 {
				fmt.Printf("逻辑错误: 在 (%d, %d) 处的数字是 %d, 但它没有未打开的邻居, 因此该数字必须为0。\n", cluePos.R, cluePos.C, s.board[cluePos.R][cluePos.C])
				s.isPossible = false // 标记为无解
				return
			}
		} else {
			s.constraints = append(s.constraints, Constraint{
				cluePos:   cluePos,
				neighbors: neighborIndices,
				count:     s.board[cluePos.R][cluePos.C],
			})
		}
	}
}

// isValidCoordinate 检查坐标是否在棋盘范围内
func (s *minesweeperSolver) isValidCoordinate(r, c int) bool {
	return r >= 0 && r < s.rows && c >= 0 && c < s.cols
}

// solve 是解决问题的主入口
func (s *minesweeperSolver) solve() {
	// 如果预处理阶段发现棋盘无解，直接返回
	if !s.isPossible {
		return
	}

	// 如果没有与线索相关的未知区域，则最小雷和最大雷都为0
	if len(s.unknowns) == 0 {
		s.minMines = 0
		s.maxMines = 0
		s.foundSolution = true
		return
	}

	// 使用回溯法求解
	assignment := make([]int, len(s.unknowns))
	s.solveRecursive(0, assignment)
}

// solveRecursive 是核心的回溯函数
// k: 当前正在处理的未知格子的索引
// assignment: 存储对未知格子的布雷方案 (1表示雷, 0表示安全)
func (s *minesweeperSolver) solveRecursive(k int, assignment []int) {
	// 剪枝: 在深入递归前，检查当前的部分解是否已经违反了约束
	if !s.isPossiblePartialAssignment(k, assignment) {
		return
	}

	// 基本情况: 所有未知格子都已分配完毕
	if k == len(s.unknowns) {
		// 再次验证完整解是否满足所有约束
		if s.isValidFullAssignment(assignment) {
			currentMines := 0
			for _, val := range assignment {
				currentMines += val
			}

			// 更新全局的最小/最大雷数
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

	// 递归步骤:
	// 尝试1: 假设当前格子(k)是安全的(0)
	assignment[k] = 0
	s.solveRecursive(k+1, assignment)

	// 尝试2: 假设当前格子(k)是地雷(1)
	assignment[k] = 1
	s.solveRecursive(k+1, assignment)
}

// isPossiblePartialAssignment 是一个强大的剪枝函数
// 它检查到目前为止(到索引k-1)的分配是否可能导向一个有效解
func (s *minesweeperSolver) isPossiblePartialAssignment(currentIndex int, assignment []int) bool {
	for _, constraint := range s.constraints {
		mineCount := 0
		unknownCount := 0

		// 遍历约束中的所有邻居
		for _, neighborIndex := range constraint.neighbors {
			if neighborIndex < currentIndex {
				// 这个邻居已经被分配了
				mineCount += assignment[neighborIndex]
			} else {
				// 这个邻居还未被分配
				unknownCount++
			}
		}

		// 剪枝逻辑:
		// 1. 如果已确定的雷数已经超过了线索数，那么此路不通
		// 2. 如果已确定的雷数 + 剩下所有未分配的邻居都当成雷，还凑不够线索数，那么此路也不通
		if mineCount > constraint.count || mineCount+unknownCount < constraint.count {
			return false
		}
	}
	return true
}

// isValidFullAssignment 在找到一个完整解后，最终确认该解是否满足所有约束
func (s *minesweeperSolver) isValidFullAssignment(assignment []int) bool {
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
// 返回值: (最小雷数, 最大雷数, 是否有解)
func calculate(minesweeper [][]int) (int, int) {
	if !isValidBoard(minesweeper) {
		return -1, -1
	}

	solver := newSolver(minesweeper)
	solver.solve()

	if !solver.foundSolution {
		return -1, -1
	}

	// 最终的最小雷数 = 相关区域的最小雷数 (孤立区域假设没有雷)
	finalMinMines := solver.minMines
	// 最终的最大雷数 = 相关区域的最大雷数 + 所有孤立区域的格子数 (孤立区域假设全是雷)
	finalMaxMines := solver.maxMines + solver.isolatedUnknowns

	return finalMaxMines, finalMinMines
}

func main() {
	// 【修正】使用你在问题描述中给出的棋盘
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

	fmt.Printf("最多雷数 (maxMinesNum): %d\n", maxMinesNum)
	fmt.Printf("最少雷数 (minMinesNum): %d\n", minMinesNum)

	fmt.Printf("计算用时 (useTime): %v\n", end.Sub(start))
}

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
	//判断棋盘是否为空
	if len(board) == 0 || len(board[0]) == 0 {
		fmt.Println("输入错误: 棋盘不能为空。")
		return false
	}

	//获取棋盘的行数和列数
	rows := len(board)
	cols := len(board[0])

	//遍历棋盘
	for r := 0; r < rows; r++ {
		if len(board[r]) != cols {
			fmt.Printf("输入格式错误: 棋盘的行长度不一致 (第 %d 行的长度为 %d, 期望为 %d)\n", r, len(board[r]), cols)
			return false
		}
		//遍历棋盘的每一行
		for c := 0; c < cols; c++ {
			val := board[r][c]

			//判断格子值是否在-1到8之间
			if val < UNOPENED || val > 8 {
				fmt.Printf("输入格式错误: 在 (%d, %d) 处的格子值无效: %d (有效范围是 -1 到 8)\n", r, c, val)
				return false
			}

			//判断格子值是否大于等于0
			if val >= 0 {
				//获取格子的邻居数量
				neighborCount := 0
				for _, offset := range neighbors {
					nr, nc := r+offset[0], c+offset[1]

					if nr >= 0 && nr < rows && nc >= 0 && nc < cols {
						neighborCount++
					}
				}
				//判断格子值是否大于邻居数量
				if val > neighborCount {
					fmt.Printf("输入逻辑错误: 在 (%d, %d) 处的数字是 %d, 但该位置只有 %d 个邻居，因此数字不可能大于 %d。\n", r, c, val, neighborCount, neighborCount)
					return false
				}
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

	for r := 0; r < s.rows; r++ {
		for c := 0; c < s.cols; c++ {
			if s.board[r][c] >= 0 {
				clues = append(clues, Point{r, c})
			} else {
				totalUnknowns++
			}
		}
	}

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

	s.buildConstraints(clues)
	if !s.isPossible {
		return
	}

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

		if len(neighborIndices) == 0 {
			if s.board[cluePos.R][cluePos.C] > 0 {
				fmt.Printf("逻辑错误: 在 (%d, %d) 处的数字是 %d, 但它没有未打开的邻居, 因此该数字必须为0。\n", cluePos.R, cluePos.C, s.board[cluePos.R][cluePos.C])
				s.isPossible = false
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

// 通过分治法解决问题
func (s *minesweeperSolver) solve() {
	if !s.isPossible {
		return
	}
	if len(s.unknowns) == 0 {
		s.minMines = 0
		s.maxMines = 0
		s.foundSolution = true
		return
	}

	components := s.findConnectedComponents()

	totalMinMines := 0
	totalMaxMines := 0
	s.foundSolution = true // 乐观地假设有解

	for _, componentIndices := range components {
		subMin, subMax, subFound := s.solveComponent(componentIndices)

		if !subFound {
			s.foundSolution = false // 任何一个子问题无解，则整个棋盘无解
			return
		}
		totalMinMines += subMin
		totalMaxMines += subMax
	}

	s.minMines = totalMinMines
	s.maxMines = totalMaxMines
}

func (s *minesweeperSolver) findConnectedComponents() [][]int {
	if len(s.unknowns) == 0 {
		return nil
	}

	adj := make([][]int, len(s.unknowns))
	for _, constraint := range s.constraints {
		for i := 0; i < len(constraint.neighbors); i++ {
			for j := i + 1; j < len(constraint.neighbors); j++ {
				u, v := constraint.neighbors[i], constraint.neighbors[j]
				adj[u] = append(adj[u], v)
				adj[v] = append(adj[v], u)
			}
		}
	}

	var components [][]int
	visited := make([]bool, len(s.unknowns))
	for i := 0; i < len(s.unknowns); i++ {
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

func (s *minesweeperSolver) solveComponent(componentIndices []int) (min, max int, found bool) {
	if len(componentIndices) == 0 {
		return 0, 0, true
	}

	subSolver := &minesweeperSolver{
		rows:       s.rows,
		cols:       s.cols,
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

	for _, constraint := range s.constraints {
		var subNeighbors []int
		for _, neighborIndex := range constraint.neighbors {
			if subIndex, ok := originalIndexToSubIndex[neighborIndex]; ok {
				subNeighbors = append(subNeighbors, subIndex)
			}
		}
		if len(subNeighbors) > 0 {
			subSolver.constraints = append(subSolver.constraints, Constraint{
				cluePos:   constraint.cluePos,
				neighbors: subNeighbors,
				count:     constraint.count,
			})
		}
	}

	assignment := make([]int, len(subSolver.unknowns))
	subSolver.solveRecursive(0, assignment)

	return subSolver.minMines, subSolver.maxMines, subSolver.foundSolution
}

func (s *minesweeperSolver) solveRecursive(k int, assignment []int) {
	if !s.isPossiblePartialAssignment(k, assignment) {
		return
	}

	if k == len(s.unknowns) {
		if s.isValidFullAssignment(assignment) {
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

	assignment[k] = 0
	s.solveRecursive(k+1, assignment)

	assignment[k] = 1
	s.solveRecursive(k+1, assignment)
}

func (s *minesweeperSolver) isPossiblePartialAssignment(currentIndex int, assignment []int) bool {
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

		if mineCount > constraint.count || mineCount+unknownCount < constraint.count {
			return false
		}
	}
	return true
}

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
// 返回值: (最小雷数, 最大雷数)
func calculate(minesweeper [][]int) (int, int) {
	// 检查输入的棋盘格式是否有效
	// 如果无效，直接返回 (-1, -1)
	if !isValidBoard(minesweeper) {
		return -1, -1
	}

	solver := newSolver(minesweeper)
	solver.solve()

	if !solver.foundSolution {
		return -1, -1
	}

	finalMinMines := solver.minMines
	finalMaxMines := solver.maxMines + solver.isolatedUnknowns

	return finalMaxMines, finalMinMines
}

func main() {
	minesweeper := [][]int{
		{-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1},
		{-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, 4, -1},
		{-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1},
		{-1, -1, -1, -1, -1, -1, 4, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1},
		{-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, 2, -1, 2, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1},
		{-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, 4, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1},
		{-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, 2, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1},
		{-1, -1, 3, 2, -1, -1, -1, -1, -1, -1, -1, -1, 2, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1},
		{-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, 3, 2, -1, -1, -1, -1, -1, -1, -1, -1},
		{-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1},
		{-1, -1, -1, -1, -1, -1, -1, -1, 4, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1},
		{-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1},
		{-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1},
	}

	start := time.Now()
	maxMinesNum, minMinesNum := calculate(minesweeper)
	end := time.Now()

	fmt.Printf("maxMinesNum: %d\n", maxMinesNum)
	fmt.Printf("minMinesNum: %d\n", minMinesNum)
	fmt.Printf("useTime: %v\n", end.Sub(start))
}

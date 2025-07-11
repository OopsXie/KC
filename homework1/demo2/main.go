package main

import (
	"fmt"
	"math"
	"time"
)

// 定义棋盘上格子的状态
const (
	UNOPENED = -1 // 未翻开
	MINE     = 9  // 内部标记：假定为雷
	SAFE     = 10 // 内部标记：假定为安全
)

// Point 结构体用于表示坐标
type Point struct {
	R, C int
}

// 定义8个方向的偏移量，作为包级别的常量
var neighbors = [8][2]int{{-1, -1}, {-1, 0}, {-1, 1}, {0, -1}, {0, 1}, {1, -1}, {1, 0}, {1, 1}}

// isValidBoard 检查输入的棋盘格式和逻辑是否正确
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

			// 1. 基础格式验证：检查数值范围
			if val < UNOPENED || val > 8 {
				fmt.Printf("输入格式错误: 在 (%d, %d) 处的格子值无效: %d (有效范围是 -1 到 8)\n", r, c, val)
				return false
			}

			// 2. 逻辑验证：检查数字是否超过其邻居数量
			if val >= 0 {
				neighborCount := 0
				for _, offset := range neighbors {
					nr, nc := r+offset[0], c+offset[1]
					if nr >= 0 && nr < rows && nc >= 0 && nc < cols {
						neighborCount++
					}
				}
				if val > neighborCount {
					fmt.Printf("输入逻辑错误: 在 (%d, %d) 处的数字是 %d, 但该位置只有 %d 个邻居，因此数字不可能大于 %d。\n", r, c, val, neighborCount, neighborCount)
					return false
				}
			}
		}
	}
	return true
}

type minesweeperSolver struct {
	board            [][]int
	rows, cols       int
	clues            []Point
	unknowns         []Point
	isolatedUnknowns int
	minMines         int
	maxMines         int
	foundSolution    bool
}

func newSolver(board [][]int) *minesweeperSolver {
	rows := len(board)
	cols := len(board[0])

	boardCopy := make([][]int, rows)
	for i := range boardCopy {
		boardCopy[i] = make([]int, cols)
		copy(boardCopy[i], board[i])
	}

	return &minesweeperSolver{
		board:    boardCopy,
		rows:     rows,
		cols:     cols,
		minMines: math.MaxInt32,
		maxMines: -1,
	}
}

// *** 新增的优化函数 ***
// propagateConstraints 通过逻辑推理减少未知格子的数量
// 如果发现矛盾，返回 false
func (s *minesweeperSolver) propagateConstraints() bool {
	for { // 持续循环，直到在一整轮中没有任何新的推断发生
		deductionMade := false

		// 每次循环都重新收集线索
		s.clues = nil
		for r := 0; r < s.rows; r++ {
			for c := 0; c < s.cols; c++ {
				if s.board[r][c] >= 0 && s.board[r][c] <= 8 {
					s.clues = append(s.clues, Point{r, c})
				}
			}
		}

		for _, cluePos := range s.clues {
			clueVal := s.board[cluePos.R][cluePos.C]
			mineCount := 0
			unknownNeighbors := []Point{}

			for _, offset := range neighbors {
				nr, nc := cluePos.R+offset[0], cluePos.C+offset[1]
				if !s.isValidCoordinate(nr, nc) {
					continue
				}
				cellState := s.board[nr][nc]
				if cellState == MINE {
					mineCount++
				} else if cellState == UNOPENED {
					unknownNeighbors = append(unknownNeighbors, Point{nr, nc})
				}
			}

			if mineCount > clueVal || mineCount+len(unknownNeighbors) < clueVal {
				return false // 出现矛盾，此路不通
			}

			// 推理规则1: 剩余未打开格子必然是雷
			if len(unknownNeighbors) > 0 && mineCount+len(unknownNeighbors) == clueVal {
				for _, p := range unknownNeighbors {
					if s.board[p.R][p.C] == UNOPENED {
						s.board[p.R][p.C] = MINE
						deductionMade = true
					}
				}
			}

			// 推理规则2: 剩余未打开格子必然是安全的
			if len(unknownNeighbors) > 0 && mineCount == clueVal {
				for _, p := range unknownNeighbors {
					if s.board[p.R][p.C] == UNOPENED {
						s.board[p.R][p.C] = SAFE
						deductionMade = true
					}
				}
			}
		}

		if !deductionMade {
			break // 如果一整轮都没有新的推断，则退出循环
		}
	}
	return true // 没有发现矛盾
}

// prepare 负责在推断后，分析棋盘，找出剩余的未知格
func (s *minesweeperSolver) prepare() {
	isRelevantUnknown := make(map[Point]bool)
	totalUnknowns := 0
	s.clues = nil // 清空旧线索

	for r := 0; r < s.rows; r++ {
		for c := 0; c < s.cols; c++ {
			cell := s.board[r][c]
			if cell >= 0 && cell <= 8 {
				s.clues = append(s.clues, Point{r, c})
			}
			if cell == UNOPENED {
				totalUnknowns++
			}
		}
	}
	for _, cluePos := range s.clues {
		for _, offset := range neighbors {
			nr, nc := cluePos.R+offset[0], cluePos.C+offset[1]
			if s.isValidCoordinate(nr, nc) && s.board[nr][nc] == UNOPENED {
				p := Point{nr, nc}
				if !isRelevantUnknown[p] {
					isRelevantUnknown[p] = true
					s.unknowns = append(s.unknowns, p)
				}
			}
		}
	}
	s.isolatedUnknowns = totalUnknowns - len(s.unknowns)
}

func (s *minesweeperSolver) isValidCoordinate(r, c int) bool {
	return r >= 0 && r < s.rows && c >= 0 && c < s.cols
}

func (s *minesweeperSolver) checkConstraints() bool {
	for _, cluePos := range s.clues {
		clueVal := s.board[cluePos.R][cluePos.C]
		mineCount, unknownCount := 0, 0
		for _, offset := range neighbors {
			nr, nc := cluePos.R+offset[0], cluePos.C+offset[1]
			if !s.isValidCoordinate(nr, nc) {
				continue
			}
			switch s.board[nr][nc] {
			case MINE:
				mineCount++
			case UNOPENED:
				unknownCount++
			}
		}
		if mineCount > clueVal || mineCount+unknownCount < clueVal {
			return false
		}
	}
	return true
}

func (s *minesweeperSolver) solveRecursive(k int) {
	if k == len(s.unknowns) {
		currentMines := 0
		for _, p := range s.unknowns {
			if s.board[p.R][p.C] == MINE {
				currentMines++
			}
		}
		if !s.foundSolution {
			s.minMines, s.maxMines = currentMines, currentMines
			s.foundSolution = true
		} else {
			if currentMines < s.minMines {
				s.minMines = currentMines
			}
			if currentMines > s.maxMines {
				s.maxMines = currentMines
			}
		}
		return
	}
	p := s.unknowns[k]
	s.board[p.R][p.C] = MINE
	if s.checkConstraints() {
		s.solveRecursive(k + 1)
	}
	s.board[p.R][p.C] = SAFE
	if s.checkConstraints() {
		s.solveRecursive(k + 1)
	}
	s.board[p.R][p.C] = UNOPENED
}

// calculate 是对外暴露的主函数，负责整个计算流程
func calculate(minesweeper [][]int) (int, int) {
	if !isValidBoard(minesweeper) {
		return -1, -1
	}

	solver := newSolver(minesweeper)

	// 1. 执行约束传播优化
	if !solver.propagateConstraints() {
		return -1, -1 // 推理过程中发现矛盾，直接判定无解
	}

	// 2. 在优化后的棋盘上准备求解
	solver.prepare()

	// 3. 计算在传播阶段已经确定的雷数
	confirmedMines := 0
	for r := 0; r < solver.rows; r++ {
		for c := 0; c < solver.cols; c++ {
			if solver.board[r][c] == MINE {
				confirmedMines++
			}
		}
	}

	// 如果所有相关未知格都已被推理确定，直接计算结果
	if len(solver.unknowns) == 0 {
		return confirmedMines + solver.isolatedUnknowns, confirmedMines
	}

	// 4. 对剩余的硬骨头进行回溯搜索
	solver.solveRecursive(0)

	if !solver.foundSolution {
		return -1, -1
	}

	// 最终结果 = 传播阶段确定的雷 + 回溯阶段确定的雷
	finalMinMines := confirmedMines + solver.minMines
	finalMaxMines := confirmedMines + solver.maxMines + solver.isolatedUnknowns

	return finalMaxMines, finalMinMines
}

func main() { //主函数
	//定义一个二维数组
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
	var maxMinesNum, minMinesNum int                  //定义最大雷数和最小雷数
	start := time.Now()                               //开始时间
	maxMinesNum, minMinesNum = calculate(minesweeper) //计算
	end := time.Now()                                 //结束时间

	//打印结果
	if maxMinesNum == -1 && minMinesNum == -1 {
		fmt.Println("计算中止：输入棋盘无效或棋盘无解。")
	} else {
		fmt.Printf("maxMinesNum: %d\n", maxMinesNum)
		fmt.Printf("minMinesNum: %d\n", minMinesNum)
	}
	fmt.Printf("useTime: %v\n", end.Sub(start))
}

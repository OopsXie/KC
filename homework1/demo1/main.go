package main

import (
	"fmt"
	"math"
	"time"
)

const (
	UNOPENED = -1
	MINE     = 9
	SAFE     = 10
)

type Point struct {
	R, C int
}

var neighbors = [8][2]int{{-1, -1}, {-1, 0}, {-1, 1}, {0, -1}, {0, 1}, {1, -1}, {1, 0}, {1, 1}}

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

	s := &minesweeperSolver{
		board:    boardCopy,
		rows:     rows,
		cols:     cols,
		minMines: math.MaxInt32,
		maxMines: -1,
	}
	s.prepare()
	return s
}

func (s *minesweeperSolver) prepare() {
	isRelevantUnknown := make(map[Point]bool)
	totalUnknowns := 0

	for r := 0; r < s.rows; r++ {
		for c := 0; c < s.cols; c++ {
			if s.board[r][c] >= 0 && s.board[r][c] <= 8 {
				s.clues = append(s.clues, Point{r, c})
			}
			if s.board[r][c] == UNOPENED {
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
// 返回值: (最大雷数, 最小雷数)
func calculate(minesweeper [][]int) (int, int) {

	//判断棋盘是否合法，除边界外，每个格子值在-1到8之间
	//且每个数字格的值不超过其邻居数量
	//如果棋盘不合法，返回-1,-1
	if !isValidBoard(minesweeper) {
		return -1, -1
	}

	solver := newSolver(minesweeper)

	if len(solver.unknowns) == 0 {
		if solver.checkConstraints() {
			return solver.isolatedUnknowns, 0
		}
		return -1, -1
	}

	solver.solveRecursive(0)

	if !solver.foundSolution {
		return -1, -1
	}

	finalMinMines := solver.minMines
	finalMaxMines := solver.maxMines + solver.isolatedUnknowns

	return finalMaxMines, finalMinMines
}

func main() { //主函数

	//定义一个二维数组
	minesweeper := [][]int{
		{-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1},
		{-1, 2, -1, 2, -1, 2, -1, 2, -1, 2, -1, 2, -1},
		{-1, -1, 3, -1, 3, -1, 3, -1, 3, -1, 3, -1, -1},
		{-1, 2, -1, 3, -1, 3, -1, 3, -1, 3, -1, 2, -1},
		{-1, -1, 3, -1, 4, -1, 4, -1, 4, -1, 3, -1, -1},
		{-1, 2, -1, 3, -1, 4, -1, 4, -1, 4, -1, 2, -1},
		{-1, -1, 3, -1, 4, -1, 5, -1, 4, -1, 3, -1, -1},
		{-1, 2, -1, 3, -1, 4, -1, 4, -1, 4, -1, 2, -1},
		{-1, -1, 3, -1, 4, -1, 4, -1, 4, -1, 3, -1, -1},
		{-1, 2, -1, 3, -1, 3, -1, 3, -1, 3, -1, 2, -1},
		{-1, -1, 3, -1, 3, -1, 3, -1, 3, -1, 3, -1, -1},
		{-1, 2, -1, 2, -1, 2, -1, 2, -1, 2, -1, 2, -1},
		{-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1},
	}
	var maxMinesNum, minMinesNum int                  //定义最大雷数和最小雷数
	start := time.Now()                               //开始时间
	maxMinesNum, minMinesNum = calculate(minesweeper) //计算
	end := time.Now()                                 //结束时间

	//打印结果
	fmt.Printf("maxMinesNum: %d\n", maxMinesNum)
	fmt.Printf("minMinesNum: %d\n", minMinesNum)
	fmt.Printf("useTime: %v\n", end.Sub(start))
}

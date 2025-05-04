package main

import (
	"fmt"
	"time"
)

// Робот на полоске
type Robot struct {
	ID       int      // Идентификатор робота (1 или 2)
	Position int      // Текущая позиция
	Program  []string // Программа для робота
	PC       int      // Счетчик команд программы
}

// Полоска с роботами
type Stripe struct {
	Robots      []*Robot // Роботы на полоске
	BlackCell   int      // Позиция черной клетки
	Steps       int      // Количество выполненных шагов
	MaxSteps    int      // Максимальное количество шагов
	ShowDisplay bool     // Флаг для визуализации
	DelayMs     int      // Задержка между шагами в миллисекундах
}

// Создание новой полоски с роботами
func NewStripe(robot1Pos, robot2Pos, blackCellPos int, maxSteps int, showDisplay bool, delayMs int) *Stripe {
	// Программа для роботов
	program := []string{
		"MR",      // 1: Шаг вправо
		"IF FLAG", // 2: Проверяем, находимся ли мы на черной клетке
		"GOTO 7",  // 3: Если на черной клетке, перейти к строке 7
		"MR",      // 4: Если не на черной клетке, шаг вправо
		"ML",      // 5: Шаг влево
		"GOTO 1",  // 6: Возвращаемся к началу программы
		"MR",      // 7: Шаг вправо
		"GOTO 7",  // 8: Зацикливание движения вправо
	}

	robot1 := &Robot{
		ID:       1,
		Position: robot1Pos,
		Program:  program,
		PC:       0,
	}

	robot2 := &Robot{
		ID:       2,
		Position: robot2Pos,
		Program:  program,
		PC:       0,
	}

	return &Stripe{
		Robots:      []*Robot{robot1, robot2},
		BlackCell:   blackCellPos,
		Steps:       0,
		MaxSteps:    maxSteps,
		ShowDisplay: showDisplay,
		DelayMs:     delayMs,
	}
}

// Проверка, находится ли робот на черной клетке
func (s *Stripe) IsOnBlackCell(robotPos int) bool {
	return robotPos == s.BlackCell
}

// Выполнение одной команды для робота
func (s *Stripe) ExecuteCommand(robot *Robot) {
	cmd := robot.Program[robot.PC]

	switch cmd {
	case "ML": // Шаг влево
		robot.Position--
		robot.PC = (robot.PC + 1) % len(robot.Program)

		// Симуляция времени выполнения команды
		if s.ShowDisplay {
			time.Sleep(time.Duration(s.DelayMs) * time.Millisecond)
		}

	case "MR": // Шаг вправо
		robot.Position++
		robot.PC = (robot.PC + 1) % len(robot.Program)

		// Симуляция времени выполнения команды
		if s.ShowDisplay {
			time.Sleep(time.Duration(s.DelayMs) * time.Millisecond)
		}

	case "IF FLAG": // Проверка черной клетки
		if s.IsOnBlackCell(robot.Position) {
			// Если на черной клетке, переход к следующей команде (2)
			robot.PC = (robot.PC + 1) % len(robot.Program)
		} else {
			// Если не на черной клетке, переход к строке 3
			robot.PC = (robot.PC + 2) % len(robot.Program)
		}

		// Симуляция времени выполнения команды
		if s.ShowDisplay {
			time.Sleep(time.Duration(s.DelayMs) * time.Millisecond)
		}

	default: // GOTO N
		if len(cmd) >= 4 && cmd[:4] == "GOTO" {
			var lineNum int
			fmt.Sscanf(cmd, "GOTO %d", &lineNum)
			robot.PC = lineNum - 1 // Переход к нужной строке (индексация с 1)
		} else {
			robot.PC = (robot.PC + 1) % len(robot.Program)
		}
	}
}

// Выполнение одного шага симуляции
func (s *Stripe) Step() bool {
	// Обработка робота 1
	s.ExecuteCommand(s.Robots[0])

	// Проверка, встретились ли роботы
	if s.Robots[0].Position == s.Robots[1].Position {
		return true
	}

	// Обработка робота 2
	s.ExecuteCommand(s.Robots[1])

	// Снова проверка, встретились ли роботы
	if s.Robots[0].Position == s.Robots[1].Position {
		return true
	}

	s.Steps++
	return false
}

// DisplayState выводит текущее состояние полоски
func (s *Stripe) DisplayState() {
	minPos := min(s.Robots[0].Position, s.Robots[1].Position) - 5
	maxPos := max(s.Robots[0].Position, s.Robots[1].Position) + 5

	if s.BlackCell < minPos {
		minPos = s.BlackCell - 2
	} else if s.BlackCell > maxPos {
		maxPos = s.BlackCell + 2
	}

	fmt.Printf("Шаг: %d\n", s.Steps)

	// Отображение полоски
	fmt.Print("[")
	for i := minPos; i <= maxPos; i++ {
		if i == s.Robots[0].Position && i == s.Robots[1].Position {
			fmt.Print("R1+R2") // Оба робота на одной клетке
		} else if i == s.Robots[0].Position {
			fmt.Print("R1") // Робот 1
		} else if i == s.Robots[1].Position {
			fmt.Print("R2") // Робот 2
		} else if i == s.BlackCell {
			fmt.Print("■") // Черная клетка
		} else {
			fmt.Print("□") // Белая клетка
		}
		if i < maxPos {
			fmt.Print("][")
		}
	}
	fmt.Println("]")

	// Информация о роботах
	for i, robot := range s.Robots {
		fmt.Printf("Робот %d: позиция %d, программный счетчик %d, текущая команда %s\n",
			i+1, robot.Position, robot.PC+1, robot.Program[robot.PC])
	}
	fmt.Println()
}

// Запуск симуляции
func (s *Stripe) Run() bool {
	fmt.Println("Начальное состояние:")
	s.DisplayState()

	robotsMet := false

	for s.Steps < s.MaxSteps && !robotsMet {
		robotsMet = s.Step()

		if s.ShowDisplay {
			s.DisplayState()
			time.Sleep(time.Duration(s.DelayMs) * time.Millisecond)
		}
	}

	if robotsMet {
		fmt.Printf("Роботы встретились на позиции %d после %d шагов!\n",
			s.Robots[0].Position, s.Steps)
		return true
	} else {
		fmt.Printf("Роботы не встретились за %d шагов.\n", s.MaxSteps)
		return false
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func main() {
	// Начальные параметры
	robot1Pos := -5     // Позиция первого робота
	robot2Pos := 5      // Позиция второго робота
	blackCellPos := 0   // Позиция черной клетки
	maxSteps := 100     // Максимальное количество шагов
	showDisplay := true // Включить визуализацию
	delayMs := 200      // Задержка между шагами в миллисекундах

	// Создаем и запускаем симуляцию
	stripe := NewStripe(robot1Pos, robot2Pos, blackCellPos, maxSteps, showDisplay, delayMs)
	success := stripe.Run()

	if success {
		// Выводим объяснение алгоритма
		fmt.Println("\nОбъяснение алгоритма:")
		fmt.Println("1: MR      - Шаг вправо")
		fmt.Println("2: IF FLAG - Проверяем, находимся ли мы на черной клетке")
		fmt.Println("3: GOTO 7  - Если на черной клетке, перейти к строке 7")
		fmt.Println("4: MR      - Если не на черной клетке, шаг вправо")
		fmt.Println("5: ML      - Шаг влево")
		fmt.Println("6: GOTO 1  - Возвращаемся к началу программы")
		fmt.Println("7: MR      - Шаг вправо")
		fmt.Println("8: GOTO 7  - Зацикливание движения вправо")
		fmt.Println()
		fmt.Println("Принцип работы:")
		fmt.Println("- Первоначально робот делает шаг вправо (MR)")
		fmt.Println("- Затем робот проверяет, находится ли он на черной клетке (IF FLAG)")
		fmt.Println("- Если робот находится на черной клетке:")
		fmt.Println("  * Он переходит к строке 7, где начинает бесконечно двигаться вправо")
		fmt.Println("- Если робот не находится на черной клетке:")
		fmt.Println("  * Он делает шаг вправо (MR), затем шаг влево (ML), возвращаясь на исходную позицию")
		fmt.Println("  * После этого он снова возвращается к началу программы (GOTO 1)")
		fmt.Println("- За счет этого, роботы сначала делают шаг вправо, а затем:")
		fmt.Println("  * Если находят черную клетку, продолжают движение вправо")
		fmt.Println("  * Если не находят черную клетку, остаются примерно на месте, с небольшим отклонением")
		fmt.Println("- В результате оба робота в конечном итоге встречаются справа от черной клетки")
	}
}

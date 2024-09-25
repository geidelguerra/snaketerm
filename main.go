package main

import "golang.org/x/term"
import "os"
import "bytes"
import "fmt"
import "time"
import "sync"
import "math/rand"

const (
	LEFT int = iota
	RIGHT
	UP
	DOWN
)

type Player struct {
	x int 
	y int
	dir int
	next *Player
	size int
}

type Apple struct {
	x int
	y int
	spawned bool
}

func (p *Player) GrowPlayer() {
	tail := p
	
	for tail.next != nil {
		tail = tail.next
	}
	
	tail.next = &Player{ tail.x, tail.y, tail.dir, nil, 0 }
	p.size += 1
}

func (p *Player) DrawPlayer(out *bytes.Buffer) {
	tail := p.next
	for tail != nil {
		SetCursor(out, tail.x, tail.y)
		out.WriteString("+")
		tail = tail.next
	}
	
	SetCursor(out, p.x, p.y)
	switch p.dir {
	case UP:
		out.WriteRune(0x25b2)
	case DOWN:
		out.WriteRune(0x25bc)
	case LEFT:
		out.WriteRune(0x25c0)
	case RIGHT:
		out.WriteRune(0x25b6)
	default:
		out.WriteRune(0x25c6)
	}
}

func DrawGrid(out *bytes.Buffer, x, y, w, h int) {
	for i := x; i <= x + w; i += 1 {
		for j := y; j <= y + h; j += 1 {
			SetCursor(out, i, j)
			if i == x && j == y {
				out.WriteRune(0x2554)
			} else if i == x + w && j == y {
				out.WriteRune(0x2557)
			} else if i == x && j == y + h {
				out.WriteRune(0x255a)
			} else if i == x + w && j == y + h {
				out.WriteRune(0x255d)
			} else if j == y || j == y + h {
				out.WriteRune(0x2550)
			} else if i == x || i == x + w {
				out.WriteRune(0x2551)
			}
		}
	}
}

func (p *Player) MovePlayer() {
	switch p.dir {
		case LEFT:
			p.x -= 1
		case RIGHT:
			p.x += 1
		case UP:
			p.y -= 1
		case DOWN:
			p.y += 1
	}
	
	x, y := p.x, p.y

	tail := p.next
	
	for tail != nil {
		x2, y2 := tail.x, tail.y
		tail.x, tail.y = x, y
		x, y = x2, y2
		tail = tail.next
	}
}

func (p *Player) ResetPlayer(x, y, w, h int) {
	p.x = x + 1 + rand.Intn(w - 1)
	p.y = y + 1 + rand.Intn(h - 1)
	p.dir = -1
	p.next = nil
	p.size = 1
}

func (p *Player) CheckPlayerHitBounds(x, y, w, h int) (bool) {
	return p.x <= x || p.x >= x + w || p.y <= y || p.y >= y + h
}

func (p *Player) CheckPlayerHitItself() (bool) {
	// TODO: geidel do this check correctly
	return false
	if p.size > 3 {
		tail := p.next
		for tail != nil {
			if p.x == tail.x && p.y == tail.y {
				return true
			}
			tail = tail.next
		}
	}
	
	return false
}

func (p *Player) CheckPlayerHitApple(apple Apple) (bool) {
	return p.x == apple.x && p.y == apple.y
}

func (a *Apple) SpawnApple(x, y, w, h int) {
	a.x = x + 1 + rand.Intn(w - 1)
	a.y = y + 1 + rand.Intn(h - 1)
	a.spawned = true
}

func (a *Apple) DrawApple(out *bytes.Buffer) {
	SetCursor(out, a.x, a.y)
	out.WriteString("A")
}

func ClearScreen(out *bytes.Buffer) {
	out.WriteString("\033[2J")
}

func SetCursor(out *bytes.Buffer, x, y int) {
	out.WriteString(fmt.Sprintf("\033[%d;%dH", y, x))
}

func main () {
	state, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}
	defer term.Restore(int(os.Stdin.Fd()), state)
	
	w, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		panic(err)
	}

	var out bytes.Buffer
	var inputBuf [64]byte
	var inputNBytes int
	quit := false
	gameOver := false
	gridW, gridH := 40, 20
	gridX, gridY := w / 2 - gridW / 2, h / 2 - gridH / 2
	var m sync.Mutex

	go func (buf *[64]byte, nBytes *int, m *sync.Mutex) {
		for {
			m.Lock()
			
			var buf2 [64]byte
			nBytes2, err := os.Stdin.Read(buf2[:])
			if err != nil {
				panic(err)
			}

			*nBytes = nBytes2
			*buf = buf2

			if nBytes2 == 1 && buf2[0] == 0x03 {
				quit = true
				break
			}

			m.Unlock()
		}
	}(&inputBuf, &inputNBytes, &m)
	
	os.Stdout.WriteString("\x1b[?25l")
	
	player := Player{ 1, 1, -1, nil, 0 }
	player.ResetPlayer(gridX, gridY, gridW, gridH)
	
	apple := Apple{ 1, 1, false }
	apple.SpawnApple(gridX, gridY, gridW, gridH)
	for !quit {
		out.Reset()
		ClearScreen(&out)
		SetCursor(&out, 1, 1)
		
		if gameOver {
			if inputNBytes == 1 && inputBuf[0] == '\r' {
				apple.SpawnApple(gridX, gridY, gridW, gridH)
				player.ResetPlayer(gridX, gridY, gridW, gridH)
				gameOver = false
			}

			text := "Game Over (Press Enter to Restart)"
			textLen := len(text)
			SetCursor(&out, w / 2 - textLen / 2, h / 2 + 1)
			out.WriteString(text)
		} else {
			if inputNBytes > 1 {
				switch string(inputBuf[:inputNBytes]) {
					case "\x1b[A":
						player.dir = UP
					case "\x1b[B":
						player.dir = DOWN
					case "\x1b[C":
						player.dir = RIGHT
					case "\x1b[D":
						player.dir = LEFT
				}
			}

			DrawGrid(&out, gridX, gridY, gridW, gridH)

			if !apple.spawned {
				apple.SpawnApple(gridX, gridY, gridW, gridH)
			}

			player.MovePlayer()

			if player.CheckPlayerHitBounds(gridX, gridY, gridW, gridH) {
				gameOver = true
				continue
			}

			if player.CheckPlayerHitItself() {
				gameOver = true
				continue
			}

			if player.CheckPlayerHitApple(apple) {
				player.GrowPlayer()
				apple.SpawnApple(gridX, gridY, gridW, gridH)
			}

			apple.DrawApple(&out)
			player.DrawPlayer(&out)
		}

		out.WriteTo(os.Stdout)
		inputNBytes = 0

		time.Sleep(100 * time.Millisecond)
	}

	os.Stdout.WriteString("\x1b[?25h")
}

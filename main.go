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
		out.WriteString("*")
		tail = tail.next
	}
	
	SetCursor(out, p.x, p.y)
	out.WriteString("#")
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
	p.x = x + rand.Intn(w)
	p.y = y + rand.Intn(h)
	p.dir = -1
	p.next = nil
	p.size = 1
}

func (p *Player) CheckPlayerHitBounds(x, y, w, h int) (bool) {
	return p.x < x || p.x > w || p.y < y || p.y > h
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
	a.x = x + rand.Intn(w)
	a.y = y + rand.Intn(h)
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
	
	player := Player{ 1, 1, -1, nil, 0 }
	
	apple := Apple{ 1, 1, false }

	var out bytes.Buffer
	var inputBuf [64]byte
	var inputNBytes int
	quit := false
	gameOver := false
	
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

	for !quit {
		w, h, err := term.GetSize(int(os.Stdout.Fd()))
		if err != nil {
			panic(err)
		}

		out.Reset()
		ClearScreen(&out)
		SetCursor(&out, 1, 1)
		
		if gameOver {
			if inputNBytes == 1 && inputBuf[0] == '\r' {
				apple.SpawnApple(1, 1, w, h)
				player.ResetPlayer(1, 1, w, h)
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

			if !apple.spawned {
				apple.SpawnApple(1, 1, w, h)
			}

			player.MovePlayer()

			if player.CheckPlayerHitBounds(1, 1, w, h) {
				gameOver = true
				continue
			}

			if player.CheckPlayerHitItself() {
				gameOver = true
				continue
			}

			if player.CheckPlayerHitApple(apple) {
				player.GrowPlayer()
				apple.SpawnApple(1, 1, w, h)
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

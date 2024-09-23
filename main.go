package main

import "golang.org/x/term"
import "os"
import "bytes"
import "fmt"
import "time"
import "sync"

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
	
	tail.next = &Player{ tail.x, tail.y, tail.dir, nil }
}

func (p *Player) DrawPlayer(out *bytes.Buffer) {
	tail := p.next
	for tail != nil {
		out.WriteString(fmt.Sprintf("\033[%d;%dH", tail.y, tail.x))
		out.WriteString("*")
		tail = tail.next
	}
	
	out.WriteString(fmt.Sprintf("\033[%d;%dH", p.y, p.x))
	out.WriteString("#")
}

func (p *Player) MovePlayer(w, h int) {
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

	if p.x < 1 {
		p.x = 1
	} else if p.x > w {
		p.x = w
	}

	if p.y < 1 {
		p.y = 1
	} else if p.y > h {
		p.y = h
	}

	tail := p.next
	
	for tail != nil {
		x2, y2 := tail.x, tail.y
		tail.x, tail.y = x, y
		x, y = x2, y2
		tail = tail.next
	}
}

func main () {
	state, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}
	defer term.Restore(int(os.Stdin.Fd()), state)
	
	player := Player{ 1, 1, -1, nil }
	player.GrowPlayer()
	player.GrowPlayer()
	player.GrowPlayer()

	var out bytes.Buffer
	var inputBuf [64]byte
	var inputNBytes int
	quit := false
	
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

		player.MovePlayer(w, h)

		inputNBytes = 0
		out.Reset()
		out.WriteString("\033[2J")
		player.DrawPlayer(&out)
		out.WriteTo(os.Stdout)

		time.Sleep(100 * time.Millisecond)
	}

	os.Stdout.WriteString("\x1b[?25h")
}

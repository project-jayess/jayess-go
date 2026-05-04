package lexer

func (l *Lexer) current() rune {
	if l.pos >= len(l.input) {
		return 0
	}
	return l.input[l.pos]
}

func (l *Lexer) peek() rune {
	if l.pos+1 >= len(l.input) {
		return 0
	}
	return l.input[l.pos+1]
}

func (l *Lexer) peekSecond() rune {
	if l.pos+2 >= len(l.input) {
		return 0
	}
	return l.input[l.pos+2]
}

func (l *Lexer) peekThird() rune {
	if l.pos+3 >= len(l.input) {
		return 0
	}
	return l.input[l.pos+3]
}

func (l *Lexer) advance() {
	if l.pos >= len(l.input) {
		return
	}
	if l.input[l.pos] == '\n' {
		l.line++
		l.column = 1
	} else {
		l.column++
	}
	l.pos++
}

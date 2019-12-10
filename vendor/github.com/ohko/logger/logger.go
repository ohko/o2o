package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
)

// ...
const (
	LoggerLevel0Debug   = iota // 测试信息 绿色
	LoggerLevel1Warning        // 警告信息 黄色
	LoggerLevel2Error          // 错误信息 红色
	LoggerLevel3Fatal          // 严重信息 高亮红色
	LoggerLevel4Trace          // 打印信息 灰色
	LoggerLevel5Off            // 关闭信息
)

// Logger ...
type Logger struct {
	l      *log.Logger
	level  int
	color  bool
	prefix string
}

// NewLogger ...
func NewLogger(out io.Writer) *Logger {
	level, _ := strconv.Atoi(os.Getenv("LOG_LEVEL"))
	if out == nil {
		out = os.Stdout
	}

	o := &Logger{
		color: true,
		level: level,
		l:     log.New(out, "", log.Ldate|log.Ltime|log.Lshortfile),
	}
	return o
}

// LogCalldepth ...
func (o *Logger) LogCalldepth(calldepth int, level int, msg ...interface{}) {
	if level < o.level || o.level == LoggerLevel5Off {
		return
	}

	prefix := ""
	if o.color {
		switch level {
		case LoggerLevel0Debug:
			prefix = "\033[32m[" + o.prefix + ":D] \033[m"
		case LoggerLevel1Warning:
			prefix = "\033[33m[" + o.prefix + ":W] \033[m"
		case LoggerLevel2Error:
			prefix = "\033[31m[" + o.prefix + ":E] \033[m"
		case LoggerLevel3Fatal:
			prefix = "\033[31;1;7m[" + o.prefix + ":F] \033[m"
		case LoggerLevel4Trace:
			prefix = "\033[37m[" + o.prefix + ":T] \033[m"
		default:
			prefix = "[" + o.prefix + ":N] "
		}
	} else {
		prefix = "[" + o.prefix + ":N] "
	}

	o.l.Output(calldepth, prefix+fmt.Sprint(msg...))
}

// SetColor Enable/Disable color
func (o *Logger) SetColor(enable bool) {
	o.color = enable
}

// SetFlags ...
func (o *Logger) SetFlags(flag int) {
	o.l.SetFlags(flag)
}

// SetLevel ...
func (o *Logger) SetLevel(level int) {
	o.level = level
}

// SetPrefix ...
func (o *Logger) SetPrefix(prefix string) {
	o.prefix = prefix
}

// SetOutput ...
func (o *Logger) SetOutput(w io.Writer) {
	o.l.SetOutput(w)
}

// Log0Debug ...
func (o *Logger) Log0Debug(v ...interface{}) {
	o.LogCalldepth(3, LoggerLevel0Debug, fmt.Sprintln(v...))
}

// Log1Warn ...
func (o *Logger) Log1Warn(v ...interface{}) {
	o.LogCalldepth(3, LoggerLevel1Warning, fmt.Sprintln(v...))
}

// Log2Error ...
func (o *Logger) Log2Error(v ...interface{}) {
	o.LogCalldepth(3, LoggerLevel2Error, fmt.Sprintln(v...))
}

// Log3Fatal ...
func (o *Logger) Log3Fatal(v ...interface{}) {
	o.LogCalldepth(3, LoggerLevel3Fatal, fmt.Sprintln(v...))
	os.Exit(1)
}

// Log4Trace ...
func (o *Logger) Log4Trace(v ...interface{}) {
	o.LogCalldepth(3, LoggerLevel4Trace, fmt.Sprintln(v...))
}

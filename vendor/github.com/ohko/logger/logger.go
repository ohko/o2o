package logger

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
)

// ...
const (
	LoggerLevel0Debug   = iota // 测试信息 绿色
	LoggerLevel1Warning        // 警告信息 黄色
	LoggerLevel2Error          // 错误信息 红色
	LoggerLevel3Fatal          // 严重信息 高亮红色
	LoggerLevel4Trace          // 打印信息 灰色
	LoggerLevel5Off            // 关闭信息
	LoggerLevelNormal          // 无色
)

// Logger ...
type Logger struct {
	l      *log.Logger
	level  *int
	color  bool
	prefix string
	lock   sync.Mutex

	storePrefix map[int]string
	forks       []*Logger
}

// NewLogger ...
func NewLogger(out io.Writer) *Logger {
	level, _ := strconv.Atoi(os.Getenv("LOG_LEVEL"))
	if out == nil {
		out = os.Stdout
	}

	o := &Logger{
		level: &level,
		l:     log.New(out, "", log.Ldate|log.Ltime|log.Llongfile),
		storePrefix: map[int]string{
			LoggerLevel0Debug:   "",
			LoggerLevel1Warning: "",
			LoggerLevel2Error:   "",
			LoggerLevel3Fatal:   "",
			LoggerLevel4Trace:   "",
			LoggerLevelNormal:   "",
		},
	}
	o.updatePrefix()
	return o
}

// LogCalldepth ...
func (o *Logger) LogCalldepth(calldepth int, level int, msg ...interface{}) {
	if level < *o.level || *o.level == LoggerLevel5Off {
		return
	}

	if level > LoggerLevelNormal {
		level = LoggerLevelNormal
	}

	o.l.Output(calldepth, o.storePrefix[level]+fmt.Sprint(msg...))
}

// SetColor Enable/Disable color
func (o *Logger) SetColor(enable bool) {
	o.color = enable
	o.updatePrefix()
}

// SetFlags ...
func (o *Logger) SetFlags(flag int) {
	o.l.SetFlags(flag)
}

// SetLevel ...
func (o *Logger) SetLevel(level int) {
	*o.level = level
}

// SetPrefix ...
func (o *Logger) SetPrefix(prefix string) {
	o.prefix = prefix
	o.updatePrefix()
}

func (o *Logger) updatePrefix() {
	o.lock.Lock()
	defer o.lock.Unlock()
	if o.color {
		o.storePrefix[LoggerLevel0Debug] = "\033[32m[" + o.prefix + ":D] \033[m"
		o.storePrefix[LoggerLevel1Warning] = "\033[33m[" + o.prefix + ":W] \033[m"
		o.storePrefix[LoggerLevel2Error] = "\033[31m[" + o.prefix + ":E] \033[m"
		o.storePrefix[LoggerLevel3Fatal] = "\033[31;1;7m[" + o.prefix + ":F] \033[m"
		o.storePrefix[LoggerLevel4Trace] = "\033[37m[" + o.prefix + ":T] \033[m"
	} else {
		o.storePrefix[LoggerLevel0Debug] = "[" + o.prefix + ":D]"
		o.storePrefix[LoggerLevel1Warning] = "[" + o.prefix + ":W]"
		o.storePrefix[LoggerLevel2Error] = "[" + o.prefix + ":E]"
		o.storePrefix[LoggerLevel3Fatal] = "[" + o.prefix + ":F]"
		o.storePrefix[LoggerLevel4Trace] = "[" + o.prefix + ":T]"
	}
	o.storePrefix[LoggerLevelNormal] = "[" + o.prefix + ":N]"
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

// Fork ...
func (o *Logger) Fork(prefix string) *Logger {
	f := &Logger{
		level: o.level,
		l:     o.l,
		storePrefix: map[int]string{
			LoggerLevel0Debug:   o.storePrefix[LoggerLevel0Debug],
			LoggerLevel1Warning: o.storePrefix[LoggerLevel1Warning],
			LoggerLevel2Error:   o.storePrefix[LoggerLevel2Error],
			LoggerLevel3Fatal:   o.storePrefix[LoggerLevel3Fatal],
			LoggerLevel4Trace:   o.storePrefix[LoggerLevel4Trace],
			LoggerLevelNormal:   o.storePrefix[LoggerLevelNormal],
		},
	}
	o.forks = append(o.forks, f)
	return f
}

// Listen ...
func (o *Logger) Listen(addr string) {
	ms := http.NewServeMux()
	ms.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/html; charset=utf-8")

		if level, err := strconv.Atoi(r.FormValue("level")); err == nil {
			o.SetLevel(level)
			w.Write([]byte(`<script>location.href='/'</script>`))
			return
		}

		h := strings.ReplaceAll(htm, "{LEVEL}", strconv.Itoa(*o.level))
		w.Write([]byte(h))
	})
	o.Log4Trace("Logger listen:", addr)
	o.Log4Trace(http.ListenAndServe(addr, ms))
}

const (
	htm = `
<form>
	LOG_LEVEL: 
	<label><input type="radio" name="level" value=0> Debug</label>
	<label><input type="radio" name="level" value=1> Warning</label>
	<label><input type="radio" name="level" value=2> Error</label>
	<label><input type="radio" name="level" value=3> Fatal</label>
	<label><input type="radio" name="level" value=4> Trace</label>
	<label><input type="radio" name="level" value=5> Off</label>
	<button>Update</button>
	<script>document.querySelector("input[value='{LEVEL}']").checked=true</script>
</form>
`
)

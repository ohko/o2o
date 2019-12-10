package logger

import (
	"archive/zip"
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

// ...
const (
	ModeMonth = "month" // 按月压缩模式
	ModeDay   = "day"   // 按日压缩模式
)

// DefaultWriter ...
type DefaultWriter struct {
	fileHandle io.Writer
	lastHandle *os.File
	clone      io.Writer

	compressMode  string // 日志压缩模式 [month|day]
	compressCount int    // 日志压缩前几次 month模式只支持压缩上个月，day支持大于等于1的数字
	compressKeep  int    // 前多少次的压缩文件删除掉，支持month和day模式
}

// NewDefaultWriter ...
func NewDefaultWriter(clone io.Writer) *DefaultWriter {
	o := new(DefaultWriter)
	o.clone = clone
	o.next()

	go o.backend()

	return o
}

// SetCompressMode 设置日志模式
// mode: month=按月压缩，day=按日压缩
// count: >=1 ，仅在按日压缩模式下有效，设置为压缩几天前的日志。默认为1
// keep: >=1 ，设置为删除几次前的日志，同时支持month和day模式。默认为0，不删除。例如：1=保留最近1个压缩日志，2=保留最近2个压缩日志，依次类推。。。
func (o *DefaultWriter) SetCompressMode(mode string, count, keep int) {
	if mode == ModeMonth || mode == ModeDay {
		o.compressMode = mode
	}
	if count <= 1 {
		count = 1
	}
	if keep < 0 {
		keep = 0
	}
	if mode == ModeDay {
		keep += count
	}
	o.compressCount = count
	o.compressKeep = keep
}

func (o *DefaultWriter) next() {
	f := "./log/" + time.Now().Format("2006/01/2006-01-02") + ".log"
	os.MkdirAll(filepath.Dir(f), 0755)
	nc, err := os.OpenFile(f, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Println(err)
		return
	}

	// 一分钟后关闭文件句柄
	if o.lastHandle != nil {
		oldnc := o.lastHandle
		go func(f *os.File) {
			time.Sleep(time.Minute)
			f.Close()
		}(oldnc)
	}

	// 设置新文件句柄
	o.lastHandle = nc
	if o.clone != nil {
		o.fileHandle = io.MultiWriter(nc, o.clone)
	} else {
		o.fileHandle = nc
	}
}

func (o *DefaultWriter) backend() {
	for {
		// 等待明天
		t1 := time.Now()
		t2, _ := time.Parse("2006-01-02 -0700", t1.Add(time.Hour*24).Format("2006-01-02 -0700"))
		<-time.After(t2.Sub(t1))

		// 下一个日志文件
		o.next()

		// 每个月的第一天压缩上个月日志
		if o.compressMode == ModeMonth && time.Now().Format("02 15:04") == "01 00:00" {
			go func() {
				if err := compressAndRemoveDir("./log/"+t1.Format("2006/01/"), "./log/"+t1.Format("2006/2006-01.zip")); err != nil {
					log.Println(err)
				}

				// 删除过期日志
				if o.compressKeep > 0 {
					zipFile := subMoth(time.Now(), o.compressKeep).Format("./log/2006/2006-01.zip")
					if err := os.RemoveAll(zipFile); err != nil {
						log.Println(err)
					}
				}
			}()
		}

		// 压缩几天前的日志
		if o.compressMode == ModeDay && o.compressCount >= 1 {
			go func() {
				t := time.Now().Add(-time.Hour * time.Duration(24*o.compressCount))
				logFile := t.Format("./log/2006/01/2006-01-02.log")
				zipFile := t.Format("./log/2006/01/2006-01-02.zip")
				if err := compressAndRemoveFile(logFile, zipFile); err != nil {
					log.Println(err)
				}

				// 删除过期日志
				if o.compressKeep > 0 {
					zipFile := time.Now().Add(-time.Hour * time.Duration(24*(o.compressKeep+1))).Format("./log/2006/01/2006-01-02.zip")
					if err := os.RemoveAll(zipFile); err != nil {
						log.Println(err)
					}
				}
			}()
		}
	}
}

func (o *DefaultWriter) Write(p []byte) (n int, err error) {
	if o.fileHandle == nil {
		return 0, errors.New("io nil error")
	}
	return o.fileHandle.Write(p)
}

func compressAndRemoveDir(dir, zipFile string) error {
	fz, err := os.Create(zipFile)
	if err != nil {
		return err
	}
	defer fz.Close()

	w := zip.NewWriter(fz)
	defer w.Close()

	if err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			fDest, err := w.Create(info.Name())
			if err != nil {
				return err
			}
			fSrc, err := os.Open(path)
			if err != nil {
				return err
			}
			defer fSrc.Close()
			_, err = io.Copy(fDest, fSrc)
			if err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}

	// 删除上个月的日志目录
	return os.RemoveAll(dir)
}

func compressAndRemoveFile(file, zipFile string) error {
	fz, err := os.Create(zipFile)
	if err != nil {
		return err
	}
	defer fz.Close()

	w := zip.NewWriter(fz)
	defer w.Close()

	fDest, err := w.Create(filepath.Base(file))
	if err != nil {
		return err
	}
	fSrc, err := os.Open(file)
	if err != nil {
		return err
	}
	defer fSrc.Close()
	_, err = io.Copy(fDest, fSrc)
	if err != nil {
		return err
	}

	// 删除日志文件
	return os.RemoveAll(file)
}

// 返回几个月前的第一天时间
func subMoth(t time.Time, c int) time.Time {

	if c == 0 {
		t, _ = time.Parse("2006-01 -0700", t.Format("2006-01 -0700"))
		return t
	}

	for i := 0; i < c; i++ {
		// 这个月的第一天
		t, _ = time.Parse("2006-01 -0700", t.Format("2006-01 -0700"))
		// 减一小时
		t = t.Add(-time.Hour)
		// 上个月的第一天
		t, _ = time.Parse("2006-01 -0700", t.Format("2006-01 -0700"))
	}

	return t
}

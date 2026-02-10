package main

import (
	"encoding/xml"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Duration 是 time.Duration 的包装器，用于支持自定义 XML 和 JSON 编组。
type Duration time.Duration

// UnmarshalXML 实现 xml.Unmarshaler 接口。
// 它解析格式为 "h:mm:ss.ns" 或 "mm:ss.ns" 的时间字符串。
func (duration *Duration) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var content string
	if err := d.DecodeElement(&content, &start); err != nil {
		return err
	}

	s := strings.TrimSpace(content)
	if s == "" {
		*duration = Duration(0)
		return nil
	}

	parts := strings.SplitN(s, ":", 3)
	if len(parts) != 3 {
		return errors.New("invalid format")
	}

	h, err := strconv.Atoi(parts[0])
	if err != nil {
		return err
	}

	m, err := strconv.Atoi(parts[1])
	if err != nil {
		return err
	}

	secPart := parts[2]
	secParts := strings.SplitN(secPart, ".", 2)

	sec, err := strconv.Atoi(secParts[0])
	if err != nil {
		return err
	}

	var ns int
	if len(secParts) == 2 {
		frac := secParts[1]
		if len(frac) > 9 {
			frac = frac[:9]
		}

		for len(frac) < 9 {
			frac += "0"
		}

		ns, err = strconv.Atoi(frac)
		if err != nil {
			return err
		}
	}

	dur := time.Duration(h)*time.Hour +
		time.Duration(m)*time.Minute +
		time.Duration(sec)*time.Second +
		time.Duration(ns)*time.Nanosecond
	*duration = Duration(dur)

	return nil
}

// MarshalJSON 实现 json.Marshaler 接口。
// 它将持续时间格式化为字符串，例如 "mm:ss.xx" 或 "hh:mm:ss"。
func (duration Duration) MarshalJSON() ([]byte, error) {
	d := time.Duration(duration)

	neg := d < 0
	if neg {
		d = -d
	}

	h := d / time.Hour
	d -= h * time.Hour

	m := d / time.Minute
	d -= m * time.Minute

	s := d / time.Second
	d -= s * time.Second

	if h == 0 && m < 10 {
		ms := d / time.Millisecond / 10

		if neg {
			return fmt.Appendf(nil, `"-%d:%02d.%02d"`, m, s, ms), nil
		}

		return fmt.Appendf(nil, `"%d:%02d.%02d"`, m, s, ms), nil
	}

	if neg {
		return fmt.Appendf(nil, `"-%02d:%02d:%02d"`, h, m, s), nil
	}

	return fmt.Appendf(nil, `"%02d:%02d:%02d"`, h, m, s), nil
}

// String 返回底层 time.Duration 的字符串表示形式.
func (duration Duration) String() string {
	return time.Duration(duration).String()
}

// xmlRun 代表 LiveSplit .lss 文件的根元素.
type xmlRun struct {
	XMLName      xml.Name `xml:"Run"`
	GameName     string   `xml:"GameName"`
	CategoryName string   `xml:"CategoryName"`
	AttemptCount int
	Attempt      []*xmlAttempt `xml:"AttemptHistory>Attempt"`
	Segments     []*xmlSegment `xml:"Segments>Segment"`
}

// xmlAttempt 代表单个运行尝试历史记录.
type xmlAttempt struct {
	Id       int    `xml:"id,attr"`
	Started  string `xml:"started,attr"`
	Ended    string `xml:"ended,attr"`
	RealTime Duration
	GameTime Duration
}

// xmlSegment 代表一个分段.
type xmlSegment struct {
	Name            string
	BestSegmentTime xmlSegmentTime
	SegmentHistory  []*xmlAttempt `xml:"SegmentHistory>Time"`
}

// xmlSegmentTime 保存分段的最佳时间.
type xmlSegmentTime struct {
	RealTime Duration
	GameTime Duration
}
